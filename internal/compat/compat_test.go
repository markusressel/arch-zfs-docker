package compat

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/markusressel/arch-zfs-docker/internal/k8s"
)

const header = `==> Requested linux version: 7.3.1.arch1-1
==> Using OpenZFS version: 2.4.4
==> 1/2: Checking zfs-utils from AUR...
==> 2/2: Building zfs-linux from AUR...
`

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		log  string
		want bool
	}{
		{"configure refuses kernel", header + "checking kernel source...\nconfigure: error: \n\t*** Cannot build against the target kernel.\n\t*** The maximum supported kernel version is 7.2.\n", true},
		{"compile error", header + "  CC [M]  /build/zfs/module/os/linux/zfs/zpl_file.o\n/build/zfs/module/os/linux/zfs/zpl_file.c:812:24: error: implicit declaration of function 'foo'\nmake[3]: *** [scripts/Makefile.build:243: zpl_file.o] Error 1\n", true},
		{"network failure in module phase", header + "==> ERROR: Failure while downloading https://github.com/openzfs/zfs/releases/download/zfs-2.4.4/zfs-2.4.4.tar.gz\n", false},
		{"failure before module build", "==> Refreshing Arch pacman database...\nerror: failed retrieving file 'core.db'\n/x/y.c:1:1: error: noise\n", false},
	}
	for _, c := range cases {
		got := Classify(c.log)
		if got.Incompatible != c.want {
			t.Errorf("%s: Incompatible=%v, want %v (%+v)", c.name, got.Incompatible, c.want, got)
		}
		if c.want && got.Evidence == "" {
			t.Errorf("%s: missing evidence", c.name)
		}
	}
}

func TestParseVersions(t *testing.T) {
	k, z := ParseVersions(header)
	if k != "7.3.1.arch1-1" || z != "2.4.4" {
		t.Errorf("got %q %q", k, z)
	}
	k, z = ParseVersions("==> Latest available linux-lts in official repos: 6.18.55-1\n==> Requested OpenZFS version: 2.3.9 (overriding the AUR default)\n")
	if k != "6.18.55-1" || z != "2.3.9" {
		t.Errorf("got %q %q", k, z)
	}
}

func fakeServices(t *testing.T, aurVersion, linuxMax string) *Sources {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/meta/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Name: zfs\nVersion: 2.4.4\nLinux-Maximum: " + linuxMax + "\nLinux-Minimum: 4.18\n"))
	})
	mux.HandleFunc("/aur", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"resultcount":1,"results":[{"Name":"zfs-utils","Version":"` + aurVersion + `"}]}`))
	})
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"items":[{"number":1,"title":"Linux 7.3 compat: build fails","html_url":"https://github.com/openzfs/zfs/issues/1","state":"open"},{"number":2,"title":"Unrelated 7.3 thing","html_url":"x","state":"open"}]}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	s := NewSources()
	s.metaURL = srv.URL + "/meta/%s"
	s.aurURL = srv.URL + "/aur"
	s.searchURL = srv.URL + "/search"
	return s
}

func TestAssessPredictsFromMeta(t *testing.T) {
	tr := &Tracker{store: NewStore(""), sources: fakeServices(t, "2.4.4-1", "7.2")}

	if info := tr.Assess("7.2.9.arch1-1"); info.State != StateOK || info.ZfsVersion != "2.4.4" || info.LinuxMax != "7.2" {
		t.Errorf("7.2.9: %+v", info)
	}
	info := tr.Assess("7.3.1.arch1-1")
	if info.State != StatePredictedUnsupported {
		t.Fatalf("7.3.1: %+v", info)
	}
	if len(info.Issues) != 1 || info.Issues[0].Number != 1 || !strings.Contains(info.IssuesURL, "github.com/openzfs/zfs/issues") {
		t.Errorf("issues: %+v", info)
	}
}

func TestObserveLearnsFromFailedBuild(t *testing.T) {
	dir := t.TempDir()
	logs := filepath.Join(dir, "logs")
	os.MkdirAll(logs, 0o755)
	failing := header + "/b/module/zpl.c:1:1: error: boom\n"
	os.WriteFile(filepath.Join(logs, "job-1.log"), []byte(failing), 0o644)

	tr := &Tracker{store: NewStore(filepath.Join(dir, "compat.json")), sources: fakeServices(t, "2.4.4-1", "7.3"), logsDir: logs}
	end := time.Now()
	tr.Observe([]k8s.JobSummary{{Name: "job-1", Status: "Failed", EndTime: &end}, {Name: "job-2", Status: "Succeeded"}})

	info := tr.Assess("7.3.1.arch1-1")
	if info.State != StateIncompatible || info.Job != "job-1" {
		t.Fatalf("expected incompatible via job-1, got %+v", info)
	}
	// Another kernel or another ZFS release is not affected.
	if info := tr.Assess("7.3.2.arch1-1"); info.State == StateIncompatible {
		t.Errorf("other kernel must not be blocked: %+v", info)
	}
	tr.sources = fakeServices(t, "2.4.5-1", "7.3")
	if info := tr.Assess("7.3.1.arch1-1"); info.State == StateIncompatible {
		t.Errorf("new ZFS release must lift the block: %+v", info)
	}

	// Persisted across restarts.
	reloaded := &Tracker{store: NewStore(filepath.Join(dir, "compat.json")), sources: fakeServices(t, "2.4.4-1", "7.3")}
	if info := reloaded.Assess("7.3.1.arch1-1"); info.State != StateIncompatible {
		t.Errorf("not persisted: %+v", info)
	}
}

func TestIncompatibilityExpires(t *testing.T) {
	s := NewStore("")
	s.record(Entry{Kernel: "7.3.1.arch1.1", ZfsVersion: "2.4.4", At: time.Now().Add(-incompatibleTTL - time.Hour)})
	if _, ok := s.lookup("7.3.1.arch1.1", "2.4.4"); ok {
		t.Error("expired entry must not block builds")
	}
}

func TestObserveWritesOnlyWhenSomethingChanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compat.json")
	logs := filepath.Join(dir, "logs")
	os.MkdirAll(logs, 0o755)
	tr := &Tracker{store: NewStore(path), sources: fakeServices(t, "2.4.4-1", "7.3"), logsDir: logs}

	// Running and succeeded jobs are polled constantly; they must not touch the disk.
	tr.Observe([]k8s.JobSummary{{Name: "job-1", Status: k8s.StatusRunning}, {Name: "job-2", Status: k8s.StatusSucceeded}})
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("nothing to remember, but %s was written (err=%v)", path, err)
	}

	// A failed job without a log yet is retried later, nothing to persist either.
	tr.Observe([]k8s.JobSummary{{Name: "job-3", Status: k8s.StatusFailed}})
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("unclassified job must not be persisted (err=%v)", err)
	}

	os.WriteFile(filepath.Join(logs, "job-3.log"), []byte("unrelated failure\n"), 0o644)
	tr.Observe([]k8s.JobSummary{{Name: "job-3", Status: k8s.StatusFailed}})
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("classified job must be remembered: %v", err)
	}

	// Seen again: no rewrite.
	before := st.ModTime()
	time.Sleep(10 * time.Millisecond)
	tr.Observe([]k8s.JobSummary{{Name: "job-3", Status: k8s.StatusFailed}})
	if st, _ := os.Stat(path); !st.ModTime().Equal(before) {
		t.Error("already seen job must not rewrite the store")
	}
}
