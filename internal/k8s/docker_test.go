package k8s

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// fakeDocker is a minimal Docker Engine API double.
type fakeDocker struct {
	mu         sync.Mutex
	imagePulls int
	hasImage   bool
	created    map[string]any
	calls      []string
	inspect    map[string]string // container name -> inspect JSON
}

func (f *fakeDocker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, r.Method+" "+r.URL.Path)

	switch {
	case r.URL.Path == "/_ping":
		w.Write([]byte("OK"))
	case r.URL.Path == "/containers/create":
		if !f.hasImage {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"No such image"}`))
			return
		}
		json.NewDecoder(r.Body).Decode(&f.created)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"Id":"abc"}`))
	case r.URL.Path == "/images/create":
		f.imagePulls++
		f.hasImage = true
		w.Write([]byte(`{"status":"Pulling"}` + "\n" + `{"status":"Done"}` + "\n"))
	case strings.HasSuffix(r.URL.Path, "/start"):
		w.WriteHeader(http.StatusNoContent)
	case r.URL.Path == "/containers/json":
		w.Write([]byte(`[{"Id":"id-ok"},{"Id":"id-fail"},{"Id":"id-run"}]`))
	case strings.HasPrefix(r.URL.Path, "/containers/") && strings.HasSuffix(r.URL.Path, "/json"):
		name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/containers/"), "/json")
		if body, ok := f.inspect[name]; ok {
			w.Write([]byte(body))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	case r.Method == http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

const labelled = `"Config":{"Labels":{"app.kubernetes.io/name":"zfs-repo-builder","zfs-repo-builder/variant":"lts","zfs-repo-builder/kernel-version":"6.18.55-1"}}`

func inspectJSON(name, status string, exit int) string {
	return `{"Id":"` + name + `","Name":"/` + name + `","Created":"2026-10-12T10:00:00Z",` +
		`"State":{"Status":"` + status + `","ExitCode":` + itoa(exit) + `,"StartedAt":"2026-10-12T10:00:05Z","FinishedAt":"2026-10-12T10:10:05Z"},` + labelled + `}`
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	return "1"
}

func newTestDocker(t *testing.T, f *fakeDocker) *DockerClient {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	c, err := NewDockerClient("tcp://"+strings.TrimPrefix(srv.URL, "http://"), "zfslocal", t.TempDir(), "/host/repo")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestDockerTriggerBuildPullsImageAndMountsRepo(t *testing.T) {
	f := &fakeDocker{}
	c := newTestDocker(t, f)

	job, err := c.TriggerBuild(BuildRequest{KernelVersion: "6.18.55-arch1-1", Variant: "lts", ZfsVersion: "2.3.9"})
	if err != nil {
		t.Fatal(err)
	}
	if f.imagePulls != 1 {
		t.Errorf("expected one image pull, got %d", f.imagePulls)
	}
	if job.Status != "Running" || job.Variant != "lts" {
		t.Errorf("unexpected job: %+v", job)
	}

	binds := f.created["HostConfig"].(map[string]any)["Binds"].([]any)
	if binds[0] != "/host/repo:/repo" {
		t.Errorf("unexpected bind: %v", binds)
	}
	env := strings.Join(toStrings(f.created["Env"].([]any)), "\n")
	for _, want := range []string{"JOB_NAME=" + job.Name, "VARIANT=lts", "KERNEL_VERSION=6.18.55.arch1-1", "ZFS_VERSION=2.3.9", "FORCE_BUILD=false"} {
		if !strings.Contains(env, want) {
			t.Errorf("env missing %q:\n%s", want, env)
		}
	}
	labels := f.created["Labels"].(map[string]any)
	if labels["zfs-repo-builder/variant"] != "lts" || labels["app.kubernetes.io/name"] != "zfs-repo-builder" {
		t.Errorf("unexpected labels: %v", labels)
	}
	cmd := f.created["Cmd"].([]any)[2].(string)
	if !strings.Contains(cmd, "/scripts/check_and_build.sh") {
		t.Errorf("command does not run the build script")
	}
}

func toStrings(in []any) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = v.(string)
	}
	return out
}

func TestDockerListJobsMapsStatus(t *testing.T) {
	f := &fakeDocker{hasImage: true, inspect: map[string]string{
		"id-ok":   inspectJSON("job-ok", "exited", 0),
		"id-fail": inspectJSON("job-fail", "exited", 2),
		"id-run":  inspectJSON("job-run", "running", 0),
	}}
	c := newTestDocker(t, f)

	jobs, err := c.ListJobs()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]JobSummary{}
	for _, j := range jobs {
		got[j.Name] = j
	}
	if got["job-ok"].Status != "Succeeded" || got["job-fail"].Status != "Failed" || got["job-run"].Status != "Running" {
		t.Errorf("unexpected statuses: %+v", got)
	}
	if got["job-ok"].DurationSec != 600 || got["job-ok"].KernelVersion != "6.18.55-1" || got["job-ok"].Variant != "lts" {
		t.Errorf("unexpected summary: %+v", got["job-ok"])
	}
}

func TestDockerRefusesForeignContainers(t *testing.T) {
	f := &fakeDocker{hasImage: true, inspect: map[string]string{
		"other": `{"Id":"x","Name":"/other","State":{"Status":"running"},"Config":{"Labels":{}}}`,
	}}
	c := newTestDocker(t, f)

	if err := c.DeleteJob("other"); err != ErrJobNotFound {
		t.Errorf("expected ErrJobNotFound, got %v", err)
	}
	if _, err := c.StreamLogs(context.Background(), "other"); err != ErrJobNotFound {
		t.Errorf("expected ErrJobNotFound for logs, got %v", err)
	}
	for _, call := range f.calls {
		if strings.HasPrefix(call, "DELETE") {
			t.Errorf("foreign container was deleted: %s", call)
		}
	}
}
