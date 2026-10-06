package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

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
