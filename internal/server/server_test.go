package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/markusressel/arch-zfs-docker/internal/config"
	"github.com/markusressel/arch-zfs-docker/internal/k8s"
)

func setupTestServer(t *testing.T) (*Server, string) {
	tempDir := t.TempDir()
	repoArchDir := filepath.Join(tempDir, "zfslocal", "x86_64")
	if err := os.MkdirAll(repoArchDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test files
	pkgFile := filepath.Join(repoArchDir, "zfs-linux-2.4.4_7.2.8.arch1.2-1-x86_64.pkg.tar.zst")
	dbFile := filepath.Join(repoArchDir, "zfslocal.db.tar.zst")
	if err := os.WriteFile(pkgFile, []byte("pkg-bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dbFile, []byte("db-bytes"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		RepoDir:   tempDir,
		RepoName:  "zfslocal",
		Namespace: "arch-repo",
	}

	mockK8s := k8s.NewMockClient()
	return NewServer(cfg, mockK8s), tempDir
}

func TestAPIPackages(t *testing.T) {
	srv, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/packages", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	if resp["repoName"] != "zfslocal" {
		t.Errorf("expected repoName zfslocal, got %v", resp["repoName"])
	}
	if resp["packageCount"].(float64) != 1 {
		t.Errorf("expected 1 package, got %v", resp["packageCount"])
	}
}

func TestAPIBuilds(t *testing.T) {
	srv, _ := setupTestServer(t)

	// 1. List builds
	req := httptest.NewRequest("GET", "/api/builds", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var builds []k8s.JobSummary
	if err := json.Unmarshal(w.Body.Bytes(), &builds); err != nil {
		t.Fatal(err)
	}

	if len(builds) == 0 {
		t.Fatal("expected at least 1 mock build job")
	}

	// 2. Trigger build
	body, _ := json.Marshal(k8s.BuildRequest{
		KernelVersion: "7.2.8.arch1-2",
		ForceBuild:    true,
	})
	postReq := httptest.NewRequest("POST", "/api/builds", bytes.NewReader(body))
	postW := httptest.NewRecorder()
	srv.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", postW.Code)
	}

	var created k8s.JobSummary
	if err := json.Unmarshal(postW.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	if created.Status != "Running" {
		t.Errorf("expected status Running, got %s", created.Status)
	}
}

func TestPacmanStaticFiles(t *testing.T) {
	srv, _ := setupTestServer(t)

	// Test .db has no-cache
	dbReq := httptest.NewRequest("GET", "/zfslocal/x86_64/zfslocal.db.tar.zst", nil)
	dbW := httptest.NewRecorder()
	srv.ServeHTTP(dbW, dbReq)

	if dbW.Code != http.StatusOK {
		t.Fatalf("expected status 200 for DB, got %d", dbW.Code)
	}

	cc := dbW.Header().Get("Cache-Control")
	if cc == "" || !bytes.Contains([]byte(cc), []byte("no-cache")) {
		t.Errorf("expected no-cache header on DB, got: %s", cc)
	}

	// Test .pkg.tar.zst has immutable cache
	pkgReq := httptest.NewRequest("GET", "/zfslocal/x86_64/zfs-linux-2.4.4_7.2.8.arch1.2-1-x86_64.pkg.tar.zst", nil)
	pkgW := httptest.NewRecorder()
	srv.ServeHTTP(pkgW, pkgReq)

	if pkgW.Code != http.StatusOK {
		t.Fatalf("expected status 200 for pkg, got %d", pkgW.Code)
	}

	pkgCC := pkgW.Header().Get("Cache-Control")
	if !bytes.Contains([]byte(pkgCC), []byte("immutable")) {
		t.Errorf("expected immutable cache header on pkg, got: %s", pkgCC)
	}
}

func TestAPISettings(t *testing.T) {
	srv, _ := setupTestServer(t)

	// 1. GET /api/settings
	req := httptest.NewRequest("GET", "/api/settings", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &settings); err != nil {
		t.Fatal(err)
	}
	if settings["repoName"] != "zfslocal" {
		t.Errorf("expected repoName zfslocal, got %v", settings["repoName"])
	}

	// 2. POST /api/settings update valid
	body, _ := json.Marshal(map[string]interface{}{
		"autoCheckInterval": "12h",
		"buildNode":         "worker-node-1",
	})
	postReq := httptest.NewRequest("POST", "/api/settings", bytes.NewReader(body))
	postW := httptest.NewRecorder()
	srv.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", postW.Code, postW.Body.String())
	}

	// 3. GET /api/settings again to verify changes
	getReq := httptest.NewRequest("GET", "/api/settings", nil)
	getW := httptest.NewRecorder()
	srv.ServeHTTP(getW, getReq)

	var updated map[string]interface{}
	json.Unmarshal(getW.Body.Bytes(), &updated)
	if updated["autoCheckInterval"] != "12h" {
		t.Errorf("expected autoCheckInterval 12h, got %v", updated["autoCheckInterval"])
	}
	if updated["buildNode"] != "worker-node-1" {
		t.Errorf("expected buildNode worker-node-1, got %v", updated["buildNode"])
	}

	// 4. POST /api/settings invalid interval
	badBody, _ := json.Marshal(map[string]interface{}{
		"autoCheckInterval": "invalid-duration",
	})
	badReq := httptest.NewRequest("POST", "/api/settings", bytes.NewReader(badBody))
	badW := httptest.NewRecorder()
	srv.ServeHTTP(badW, badReq)

	if badW.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for bad interval, got %d", badW.Code)
	}
}

func TestAPIUpstream(t *testing.T) {
	srv, _ := setupTestServer(t)

	// GET /api/upstream
	req := httptest.NewRequest("GET", "/api/upstream", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// POST /api/upstream/check
	postReq := httptest.NewRequest("POST", "/api/upstream/check", nil)
	postW := httptest.NewRecorder()
	srv.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", postW.Code)
	}
}

func TestAPILogsSSE(t *testing.T) {
	srv, _ := setupTestServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest("GET", "/api/builds/zfs-repo-builder-scheduled-demo/logs", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		srv.ServeHTTP(w, req)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("expected text/event-stream content type, got %s", w.Header().Get("Content-Type"))
	}
}

func TestTriggerBuildConflictsWhenAlreadyBuilt(t *testing.T) {
	srv, _ := setupTestServer(t)

	post := func(req k8s.BuildRequest) *httptest.ResponseRecorder {
		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, httptest.NewRequest("POST", "/api/builds", bytes.NewReader(body)))
		return w
	}

	// Hyphenated user input must match the dotted package file name.
	if w := post(k8s.BuildRequest{KernelVersion: "7.2.8-arch1-2"}); w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for existing build, got %d", w.Code)
	}
	if w := post(k8s.BuildRequest{KernelVersion: "7.2.8-arch1-2", ForceBuild: true}); w.Code != http.StatusAccepted {
		t.Fatalf("expected 202 when overwrite confirmed, got %d", w.Code)
	}
	if w := post(k8s.BuildRequest{KernelVersion: "7.2.9-arch1-1"}); w.Code != http.StatusAccepted {
		t.Fatalf("expected 202 for a new kernel, got %d", w.Code)
	}
}

func TestDeleteBuilds(t *testing.T) {
	srv, _ := setupTestServer(t)

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, httptest.NewRequest("DELETE", "/api/builds/zfs-repo-builder-scheduled-demo", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	srv.ServeHTTP(w, httptest.NewRequest("DELETE", "/api/builds/zfs-repo-builder-scheduled-demo", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing job, got %d", w.Code)
	}

	// Bulk clear leaves running jobs alone.
	body, _ := json.Marshal(k8s.BuildRequest{KernelVersion: "7.2.9-arch1-1"})
	srv.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/builds", bytes.NewReader(body)))
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, httptest.NewRequest("DELETE", "/api/builds", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	listW := httptest.NewRecorder()
	srv.ServeHTTP(listW, httptest.NewRequest("GET", "/api/builds", nil))
	var jobs []k8s.JobSummary
	json.Unmarshal(listW.Body.Bytes(), &jobs)
	if len(jobs) != 1 || jobs[0].Status != "Running" {
		t.Fatalf("expected only the running job to remain, got %+v", jobs)
	}
}
