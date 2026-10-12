package scheduler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/markusressel/arch-zfs-docker/internal/compat"
	"github.com/markusressel/arch-zfs-docker/internal/k8s"
	"github.com/markusressel/arch-zfs-docker/internal/repo"
)

// fakeUpstream serves the Arch Linux package API for the given kernel packages (name -> pkgver-pkgrel).
func fakeUpstream(t *testing.T, versions map[string]string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pkg := strings.Split(strings.Trim(r.URL.Path, "/"), "/")[0]
		v, ok := versions[pkg]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		ver, rel, _ := strings.Cut(v, "-")
		w.Write([]byte(`{"pkgname":"` + pkg + `","pkgver":"` + ver + `","pkgrel":"` + rel + `","epoch":0}`))
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/%s/json/"
}

// repoWith creates a repository containing the given package files.
func repoWith(t *testing.T, files ...string) *repo.Indexer {
	t.Helper()
	dir := t.TempDir()
	arch := filepath.Join(dir, "zfslocal", "x86_64")
	if err := os.MkdirAll(arch, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(arch, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return repo.NewIndexer(dir, "zfslocal")
}

// fakeCompat returns a fixed verdict per kernel version and records observed jobs.
type fakeCompat struct {
	mu       sync.Mutex
	verdicts map[string]compat.State
	observed [][]k8s.JobSummary
}

func (f *fakeCompat) Observe(jobs []k8s.JobSummary) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.observed = append(f.observed, jobs)
}

func (f *fakeCompat) Assess(kernelVersion string) compat.Info {
	if s, ok := f.verdicts[kernelVersion]; ok {
		return compat.Info{State: s, ZfsVersion: "2.4.4", Reason: "test", Job: "zfs-build-old"}
	}
	return compat.Info{State: compat.StateOK, ZfsVersion: "2.4.4"}
}

const linuxPkg = "zfs-linux-2.4.4_7.2.8.arch1.2-1-x86_64.pkg.tar.zst"

func newTestScheduler(t *testing.T, client k8s.K8sClient, upstream map[string]string, files ...string) *Scheduler {
	t.Helper()
	s := NewScheduler(repoWith(t, files...), client, time.Hour)
	s.SetUpstreamURL(fakeUpstream(t, upstream))
	return s
}

func buildsOf(t *testing.T, c k8s.K8sClient) []k8s.JobSummary {
	t.Helper()
	jobs, err := c.ListJobs()
	if err != nil {
		t.Fatal(err)
	}
	var out []k8s.JobSummary
	for _, j := range jobs {
		if j.IsBuild() {
			out = append(out, j)
		}
	}
	return out
}

func TestCheckAllAndTriggerBuildsOnlyMissingVariants(t *testing.T) {
	client := &k8s.MockClient{}
	s := newTestScheduler(t, client, map[string]string{"linux": "7.2.8.arch1-2", "linux-lts": "6.18.55-1"}, linuxPkg)

	s.CheckAllAndTrigger()

	jobs := buildsOf(t, client)
	if len(jobs) != 1 || jobs[0].Variant != "lts" {
		t.Fatalf("expected one lts build, got %+v", jobs)
	}
	st := s.GetStatus()
	if !st["linux"].UpToDate || st["linux"].LocalKernel != "7.2.8.arch1.2" {
		t.Errorf("linux should be up to date: %+v", st["linux"])
	}
	if st["linux-lts"].UpToDate || st["linux-lts"].UpstreamKernel != "6.18.55-1" {
		t.Errorf("lts should need a build: %+v", st["linux-lts"])
	}
}

func TestCheckAllAndTriggerSkipsVariantsWithActiveBuild(t *testing.T) {
	client := &k8s.MockClient{}
	if _, err := client.TriggerBuild(k8s.BuildRequest{Variant: "lts"}); err != nil {
		t.Fatal(err)
	}
	// A running maintenance job must not keep the standard kernel from being built.
	if _, err := client.TriggerBuild(k8s.BuildRequest{Task: k8s.TaskRemovePackage, Package: linuxPkg}); err != nil {
		t.Fatal(err)
	}
	s := newTestScheduler(t, client, map[string]string{"linux": "7.2.9.arch1-1", "linux-lts": "6.18.55-1"}, linuxPkg)

	s.CheckAllAndTrigger()

	var std, lts int
	for _, j := range buildsOf(t, client) {
		if j.Variant == "lts" {
			lts++
		} else {
			std++
		}
	}
	if lts != 1 || std != 1 {
		t.Fatalf("expected the existing lts build and one new standard build, got lts=%d std=%d", lts, std)
	}
}

func TestCheckAllAndTriggerSkipsIncompatibleKernel(t *testing.T) {
	client := &k8s.MockClient{}
	s := newTestScheduler(t, client, map[string]string{"linux": "7.3.1.arch1-1", "linux-lts": "6.18.55-1"})
	fc := &fakeCompat{verdicts: map[string]compat.State{
		"7.3.1.arch1-1": compat.StateIncompatible,
		// A prediction only warns, the build is still attempted.
		"6.18.55-1": compat.StatePredictedUnsupported,
	}}
	s.SetCompat(fc)

	s.CheckAllAndTrigger()

	jobs := buildsOf(t, client)
	if len(jobs) != 1 || jobs[0].Variant != "lts" {
		t.Fatalf("expected only the lts build, got %+v", jobs)
	}
	if len(fc.observed) != 1 {
		t.Errorf("failed builds should be observed once per check, got %d", len(fc.observed))
	}
	if c := s.GetStatus()["linux"].Compat; c == nil || c.State != compat.StateIncompatible {
		t.Errorf("status should carry the verdict, got %+v", c)
	}
}

func TestCheckAllAndTriggerUpstreamUnavailable(t *testing.T) {
	client := &k8s.MockClient{}
	s := newTestScheduler(t, client, map[string]string{"linux": "7.2.9.arch1-1"}) // linux-lts answers 404

	s.CheckAllAndTrigger()

	if _, ok := s.GetStatus()["linux-lts"]; ok {
		t.Error("a failed upstream lookup must not produce a status")
	}
	if jobs := buildsOf(t, client); len(jobs) != 1 || jobs[0].Variant != "" {
		t.Fatalf("the reachable variant should still be built, got %+v", jobs)
	}
}

func TestGetStatusIsSafeDuringChecks(t *testing.T) {
	s := newTestScheduler(t, &k8s.MockClient{}, map[string]string{"linux": "7.3.1.arch1-1", "linux-lts": "6.18.55-1"})
	s.SetCompat(&fakeCompat{})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 20; i++ {
			s.CheckVariant("")
		}
	}()
	for {
		select {
		case <-done:
			return
		default:
			for _, st := range s.GetStatus() {
				_ = st.Compat
			}
		}
	}
}
