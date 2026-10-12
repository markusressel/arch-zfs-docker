package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/markusressel/arch-zfs-docker/internal/compat"
	"github.com/markusressel/arch-zfs-docker/internal/k8s"
)

const fixturePkg = "zfs-linux-2.4.4_7.2.8.arch1.2-1-x86_64.pkg.tar.zst"

// okCompat never consults external services.
type okCompat struct{}

func (okCompat) Observe([]k8s.JobSummary)  {}
func (okCompat) Assess(string) compat.Info { return compat.Info{State: compat.StateOK} }

// useUpstream points the scheduler at a fake Arch Linux package API (kernel package -> pkgver-pkgrel).
func useUpstream(t *testing.T, srv *Server, versions map[string]string) {
	t.Helper()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pkg := strings.Split(strings.Trim(r.URL.Path, "/"), "/")[0]
		v, ok := versions[pkg]
		if !ok {
			http.NotFound(w, r)
			return
		}
		ver, rel, _ := strings.Cut(v, "-")
		w.Write([]byte(`{"pkgname":"` + pkg + `","pkgver":"` + ver + `","pkgrel":"` + rel + `"}`))
	}))
	t.Cleanup(api.Close)
	srv.scheduler.SetUpstreamURL(api.URL + "/%s/json/")
	srv.scheduler.SetCompat(okCompat{})
}

func deletePackage(srv *Server, name string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, httptest.NewRequest("DELETE", "/api/packages/"+name, nil))
	return w
}

func TestDeletePackageValidatesName(t *testing.T) {
	srv, _ := setupTestServer(t)
	for _, name := range []string{"zfslocal.db.tar.zst", "notes.txt", "..zfs.pkg.tar.zst", "zfs%2F..%2Fx.pkg.tar.zst"} {
		if w := deletePackage(srv, name); w.Code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", name, w.Code)
		}
	}
	if w := deletePackage(srv, "zfs-linux-9.9.9_1.2.3-1-x86_64.pkg.tar.zst"); w.Code != http.StatusNotFound {
		t.Errorf("unknown package: expected 404, got %d", w.Code)
	}
}

func TestDeletePackageStartsRemovalJob(t *testing.T) {
	srv, _ := setupTestServer(t)
	useUpstream(t, srv, map[string]string{"linux": "7.2.8.arch1-2"})
	srv.jobPollInterval = time.Millisecond

	w := deletePackage(srv, fixturePkg)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
	var job k8s.JobSummary
	if err := json.Unmarshal(w.Body.Bytes(), &job); err != nil {
		t.Fatal(err)
	}
	if job.Task != k8s.TaskRemovePackage || job.Package != fixturePkg {
		t.Fatalf("expected a removal job for %s, got %+v", fixturePkg, job)
	}
}

func TestSyncAfterJob(t *testing.T) {
	for _, tc := range []struct {
		status      k8s.JobStatus
		wantRebuild bool
	}{
		{k8s.StatusSucceeded, true},
		{k8s.StatusFailed, false},
	} {
		t.Run(string(tc.status), func(t *testing.T) {
			srv, _ := setupTestServer(t)
			// The fixture package is the current kernel's, so the check rebuilds once it is gone.
			useUpstream(t, srv, map[string]string{"linux": "7.2.9.arch1-1"})
			srv.jobPollInterval = time.Millisecond
			mock := srv.k8sClient.(*k8s.MockClient)

			job, _ := mock.TriggerBuild(k8s.BuildRequest{Task: k8s.TaskRemovePackage, Package: fixturePkg})
			done := make(chan struct{})
			go func() {
				srv.syncAfterJob(context.Background(), job.Name)
				close(done)
			}()
			time.Sleep(5 * time.Millisecond) // still running: nothing happens
			if builds := countBuilds(t, mock); builds != 0 {
				t.Fatalf("no build expected while the job runs, got %d", builds)
			}
			mock.SetStatus(job.Name, tc.status)

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("syncAfterJob did not return")
			}
			if got := countBuilds(t, mock) > 0; got != tc.wantRebuild {
				t.Errorf("rebuild triggered = %v, want %v", got, tc.wantRebuild)
			}
		})
	}
}

// countBuilds counts active build jobs (the fixture's finished demo job and maintenance jobs do not count).
func countBuilds(t *testing.T, c k8s.K8sClient) int {
	t.Helper()
	jobs, err := c.ListJobs()
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, j := range jobs {
		if j.IsBuild() && j.IsActive() {
			n++
		}
	}
	return n
}

func TestCloseStopsWaitingForJobs(t *testing.T) {
	srv, _ := setupTestServer(t)
	useUpstream(t, srv, map[string]string{"linux": "7.2.8.arch1-2"})
	srv.jobPollInterval = time.Hour

	if w := deletePackage(srv, fixturePkg); w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}
	closed := make(chan struct{})
	go func() {
		srv.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not stop the background wait for the removal job")
	}
}
