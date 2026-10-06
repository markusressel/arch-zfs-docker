package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/markusressel/arch-zfs-docker/internal/config"
	"github.com/markusressel/arch-zfs-docker/internal/k8s"
	"github.com/markusressel/arch-zfs-docker/internal/repo"
	"github.com/markusressel/arch-zfs-docker/internal/ui"
)

// Server handles HTTP API, UI, and pacman repository file serving.
type Server struct {
	cfg       *config.Config
	indexer   *repo.Indexer
	k8sClient k8s.K8sClient
	mux       *http.ServeMux
}

// NewServer initializes the HTTP server and routes.
func NewServer(cfg *config.Config, k8sClient k8s.K8sClient) *Server {
	s := &Server{
		cfg:       cfg,
		indexer:   repo.NewIndexer(cfg.RepoDir, cfg.RepoName),
		k8sClient: k8sClient,
		mux:       http.NewServeMux(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// 1. Health check
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 2. UI static files
	uiFileServer := http.FileServer(ui.GetFileSystem())
	s.mux.Handle("GET /ui/", http.StripPrefix("/ui/", uiFileServer))
	s.mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/ui/", http.StatusFound)
			return
		}
		// If path doesn't start with /ui/ or /api/, pass to pacman static file handler
		s.handlePacmanFiles(w, r)
	})

	// 3. JSON API
	s.mux.HandleFunc("GET /api/packages", s.handleAPIPackages)
	s.mux.HandleFunc("GET /api/builds", s.handleAPIListBuilds)
	s.mux.HandleFunc("POST /api/builds", s.handleAPITriggerBuild)
	s.mux.HandleFunc("GET /api/builds/{name}/logs", s.handleAPILogsSSE)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	s.mux.ServeHTTP(w, r)
	if !strings.HasSuffix(r.URL.Path, "/logs") { // Don't log endless SSE requests
		log.Printf("%s %s %s (%s)\n", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
	}
}

// Start runs the HTTP server.
func (s *Server) Start(ctx context.Context) error {
	srv := &http.Server{
		Addr:    s.cfg.ListenAddr,
		Handler: s,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Printf("[server] Listening on %s serving repo %s from %s\n", s.cfg.ListenAddr, s.cfg.RepoName, s.cfg.RepoDir)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// --- Handlers ---

func (s *Server) handleAPIPackages(w http.ResponseWriter, r *http.Request) {
	arch := r.URL.Query().Get("arch")
	if arch == "" {
		arch = "x86_64"
	}

	summary, err := s.indexer.GetSummary(arch)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func (s *Server) handleAPIListBuilds(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.k8sClient.ListJobs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

func (s *Server) handleAPITriggerBuild(w http.ResponseWriter, r *http.Request) {
	var req k8s.BuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	job, err := s.k8sClient.TriggerBuild(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to trigger build: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(job)
}

func (s *Server) handleAPILogsSSE(w http.ResponseWriter, r *http.Request) {
	jobName := r.PathValue("name")
	if jobName == "" {
		http.Error(w, "missing job name", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	lines, err := s.k8sClient.StreamLogs(jobName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case line, ok := <-lines:
			if !ok {
				fmt.Fprintf(w, "event: close\ndata: end\n\n")
				flusher.Flush()
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", line)
			flusher.Flush()
		}
	}
}

func (s *Server) handlePacmanFiles(w http.ResponseWriter, r *http.Request) {
	// Map clean URL paths: e.g. /zfslocal/x86_64/... or /x86_64/... to disk
	cleanPath := strings.TrimPrefix(r.URL.Path, "/")
	cleanPath = filepath.Clean(cleanPath)

	filePath := filepath.Join(s.cfg.RepoDir, cleanPath)

	// If not found, try stripping repo name prefix if repo dir is already pointing to repo/arch
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		trimmed := strings.TrimPrefix(cleanPath, s.cfg.RepoName+"/")
		altPath := filepath.Join(s.cfg.RepoDir, trimmed)
		if _, err := os.Stat(altPath); err == nil {
			filePath = altPath
		}
	}

	fi, err := os.Stat(filePath)
	if err != nil || fi.IsDir() {
		http.NotFound(w, r)
		return
	}

	filename := fi.Name()

	// Pacman Cache Control Headers
	if strings.Contains(filename, ".db") || strings.Contains(filename, ".files") {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
		w.Header().Set("Expires", "0")
	} else if strings.Contains(filename, ".pkg.tar") {
		w.Header().Set("Cache-Control", "public, max-age=2592000, immutable")
	}

	http.ServeFile(w, r, filePath)
}
