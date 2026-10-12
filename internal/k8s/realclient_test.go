package k8s

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeKube is a minimal Kubernetes API double for one namespace.
type fakeKube struct {
	mu       sync.Mutex
	jobs     string         // JSON of the job list
	created  map[string]any // last posted job
	deleted  []string
	pods     map[string]string // job name -> pod phase
	podLog   string
	authSeen string
	ownImage string // image of the service's own pod
}

func (f *fakeKube) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.authSeen = r.Header.Get("Authorization")
	const jobsPath = "/apis/batch/v1/namespaces/arch-repo/jobs"
	switch {
	case r.Method == http.MethodGet && r.URL.Path == jobsPath:
		if r.URL.Query().Get("labelSelector") != "app.kubernetes.io/name=zfs-repo-builder" {
			http.Error(w, "unexpected selector", http.StatusBadRequest)
			return
		}
		w.Write([]byte(f.jobs))
	case r.Method == http.MethodPost && r.URL.Path == jobsPath:
		json.NewDecoder(r.Body).Decode(&f.created)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{}`))
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, jobsPath+"/"):
		name := strings.TrimPrefix(r.URL.Path, jobsPath+"/")
		if name == "missing" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		f.deleted = append(f.deleted, name)
		w.Write([]byte(`{}`))
	case r.URL.Path == "/api/v1/namespaces/arch-repo/pods":
		job := strings.TrimPrefix(r.URL.Query().Get("labelSelector"), "job-name=")
		phase, ok := f.pods[job]
		if !ok {
			w.Write([]byte(`{"items":[]}`))
			return
		}
		w.Write([]byte(`{"items":[{"metadata":{"name":"` + job + `-pod"},"status":{"phase":"` + phase + `"}}]}`))
	case f.ownImage != "" && r.Method == http.MethodGet && r.URL.Path == "/api/v1/namespaces/arch-repo/pods/"+hostname():
		w.Write([]byte(`{"spec":{"containers":[{"name":"server","image":"` + f.ownImage + `"}]}}`))
	case strings.HasPrefix(r.URL.Path, "/api/v1/namespaces/arch-repo/pods/") && strings.HasSuffix(r.URL.Path, "/log"):
		if r.URL.Query().Get("follow") != "true" {
			http.Error(w, "expected follow", http.StatusBadRequest)
			return
		}
		w.Write([]byte(f.podLog))
	default:
		http.Error(w, r.Method+" "+r.URL.String(), http.StatusInternalServerError)
	}
}

func newTestKube(t *testing.T, f *fakeKube) *RealClient {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return &RealClient{
		httpClient: srv.Client(),
		baseURL:    srv.URL,
		token:      "secret",
		namespace:  "arch-repo",
		repoName:   "zfslocal",
		repoDir:    t.TempDir(),
	}
}

func TestRealClientListJobs(t *testing.T) {
	f := &fakeKube{jobs: `{"items":[
		{"metadata":{"name":"zfs-build-old","creationTimestamp":"2026-10-12T08:00:00Z","annotations":{"zfs-repo-builder/variant":"lts"}},
		 "status":{"startTime":"2026-10-12T08:00:00Z","completionTime":"2026-10-12T08:10:00Z","succeeded":1}},
		{"metadata":{"name":"zfs-build-new","creationTimestamp":"2026-10-12T09:00:00Z","annotations":{"zfs-repo-builder/task":"remove-package","zfs-repo-builder/package":"x.pkg.tar.zst"}},
		 "status":{"failed":3}}
	]}`}
	c := newTestKube(t, f)

	jobs, err := c.ListJobs()
	if err != nil {
		t.Fatal(err)
	}
	if f.authSeen != "Bearer secret" {
		t.Errorf("expected the service account token, got %q", f.authSeen)
	}
	if len(jobs) != 2 || jobs[0].Name != "zfs-build-new" {
		t.Fatalf("expected newest first, got %+v", jobs)
	}
	newest, old := jobs[0], jobs[1]
	if newest.Status != StatusFailed || newest.Task != TaskRemovePackage || newest.Package != "x.pkg.tar.zst" {
		t.Errorf("unexpected maintenance job %+v", newest)
	}
	if newest.StartTime == nil || !newest.StartTime.Equal(time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("a job without startTime should fall back to its creation time, got %v", newest.StartTime)
	}
	if old.Status != StatusSucceeded || old.Variant != "lts" || old.DurationSec != 600 || old.EndTime == nil {
		t.Errorf("unexpected build job %+v", old)
	}
}

func TestRealClientListJobsAPIError(t *testing.T) {
	c := newTestKube(t, &fakeKube{})
	c.namespace = "other" // the fake answers 500
	if _, err := c.ListJobs(); err == nil {
		t.Fatal("expected an error")
	}
}

func TestRealClientTriggerBuild(t *testing.T) {
	f := &fakeKube{}
	c := newTestKube(t, f)
	c.SetBuildNode("builder-1")

	job, err := c.TriggerBuild(BuildRequest{Task: TaskRemovePackage, Package: "x.pkg.tar.zst"})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != StatusPending || job.Task != TaskRemovePackage || !ValidJobName(job.Name) {
		t.Errorf("unexpected job %+v", job)
	}

	b, _ := json.Marshal(f.created)
	body := string(b)
	for _, want := range []string{
		`"name":"` + job.Name + `"`,
		`"zfs-repo-builder/task":"remove-package"`,
		`"zfs-repo-builder/package":"x.pkg.tar.zst"`,
		`"kubernetes.io/hostname":"builder-1"`,
		`"command":["/scripts/arch-repo","builder","remove-package"]`,
		`"image":"` + defaultBuilderImage + `"`,
		`cp /app/arch-repo /scripts/arch-repo`,
		`{"name":"PACKAGE_FILE","value":"x.pkg.tar.zst"}`,
		`"claimName":"arch-repo-data"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("job payload is missing %s:\n%s", want, body)
		}
	}

	// Without a build node there is no node selector.
	c.SetBuildNode("")
	if _, err := c.TriggerBuild(BuildRequest{Variant: "lts", KernelVersion: "6.18.55-1"}); err != nil {
		t.Fatal(err)
	}
	b, _ = json.Marshal(f.created)
	if strings.Contains(string(b), "nodeSelector") {
		t.Error("unexpected node selector")
	}
	if !strings.Contains(string(b), `"/scripts/check_and_build.sh"`) {
		t.Error("builds must run the build script")
	}
}

func TestRealClientDeleteJob(t *testing.T) {
	f := &fakeKube{}
	c := newTestKube(t, f)

	if err := c.DeleteJob("zfs-build-1"); err != nil {
		t.Fatal(err)
	}
	if len(f.deleted) != 1 || f.deleted[0] != "zfs-build-1" {
		t.Errorf("expected the job to be deleted, got %v", f.deleted)
	}
	if err := c.DeleteJob("missing"); err != ErrJobNotFound {
		t.Errorf("expected ErrJobNotFound, got %v", err)
	}
	if err := c.DeleteJob("../etc"); err == nil {
		t.Error("invalid names must be rejected")
	}
}

func TestRealClientStreamLogsFollowsRunningPod(t *testing.T) {
	f := &fakeKube{pods: map[string]string{"zfs-build-1": "Running"}, podLog: "one\r\ntwo\n"}
	c := newTestKube(t, f)
	// A partial log file exists while the build runs; the live pod log wins.
	os.MkdirAll(filepath.Join(c.repoDir, "logs"), 0o755)
	os.WriteFile(filepath.Join(c.repoDir, "logs", "zfs-build-1.log"), []byte("partial\n"), 0o644)

	ch, err := c.StreamLogs(context.Background(), "zfs-build-1")
	if err != nil {
		t.Fatal(err)
	}
	if got := collect(ch); strings.Join(got, "|") != "one|two" {
		t.Errorf("unexpected lines %q", got)
	}
}

func TestRealClientStreamLogsRejectsInvalidNames(t *testing.T) {
	c := newTestKube(t, &fakeKube{})
	// A log file outside of the logs directory must not be reachable.
	os.WriteFile(filepath.Join(c.repoDir, "secret.log"), []byte("secret\n"), 0o644)

	for _, name := range []string{"../secret", "a&labelSelector=x", ""} {
		if ch, err := c.StreamLogs(context.Background(), name); err == nil {
			t.Errorf("%q: expected an error, streamed %q", name, collect(ch))
		}
	}
}

func TestStreamLinesHandlesLongLines(t *testing.T) {
	long := strings.Repeat("x", 200*1024) // e.g. pacman progress redrawn with \r
	ch := streamLines(context.Background(), io.NopCloser(strings.NewReader(long+"\r\nafter\nlast")))
	got := collect(ch)
	if len(got) != 3 || len(got[0]) != len(long) || got[1] != "after" || got[2] != "last" {
		t.Fatalf("expected 3 lines with the long one intact, got %d lines", len(got))
	}
}

func TestStreamLinesTruncatesHugeLines(t *testing.T) {
	huge := strings.Repeat("y", maxLogLine+10)
	got := collect(streamLines(context.Background(), io.NopCloser(strings.NewReader(huge+"\nnext\n"))))
	if len(got) != 2 || len(got[0]) > maxLogLine+64 || !strings.HasSuffix(got[0], truncatedMarker) || got[1] != "next" {
		t.Fatalf("expected a truncated line and the next one, got %d lines", len(got))
	}
}

// blockingBody never returns data until closed.
type blockingBody struct {
	closed chan struct{}
	once   sync.Once
}

func (b *blockingBody) Read([]byte) (int, error) { <-b.closed; return 0, io.EOF }
func (b *blockingBody) Close() error {
	b.once.Do(func() { close(b.closed) })
	return nil
}

func TestStreamLinesClosesBodyOnCancel(t *testing.T) {
	body := &blockingBody{closed: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	ch := streamLines(ctx, body)
	cancel()
	select {
	case <-body.closed:
	case <-time.After(2 * time.Second):
		t.Fatal("upstream body was not closed after the client went away")
	}
	for range ch {
	}
}

func collect(ch <-chan string) []string {
	var out []string
	for l := range ch {
		out = append(out, l)
	}
	return out
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}

// In the cluster, build jobs take the builder from the image the service itself runs.
func TestRealClientUsesItsOwnImageForTheBuilder(t *testing.T) {
	f := &fakeKube{ownImage: "ghcr.io/example/arch-zfs-docker:v1.2.3"}
	c := newTestKube(t, f)
	c.inCluster = true

	if _, err := c.TriggerBuild(BuildRequest{}); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(f.created)
	if !strings.Contains(string(b), `"image":"ghcr.io/example/arch-zfs-docker:v1.2.3"`) {
		t.Errorf("init container does not use the service's image:\n%s", b)
	}

	// A configured image wins.
	c2 := newTestKube(t, f)
	c2.inCluster = true
	c2.builderImage = "registry.local/arch-repo:dev"
	if _, err := c2.TriggerBuild(BuildRequest{}); err != nil {
		t.Fatal(err)
	}
	b, _ = json.Marshal(f.created)
	if !strings.Contains(string(b), `"image":"registry.local/arch-repo:dev"`) {
		t.Errorf("configured builder image ignored:\n%s", b)
	}
}
