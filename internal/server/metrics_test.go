package server

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/markusressel/arch-zfs-docker/internal/k8s"
)

func TestSummarizeBuilds(t *testing.T) {
	at := func(h int) *time.Time { v := time.Date(2026, 11, 19, h, 0, 0, 0, time.UTC); return &v }
	// Newest first, like ListJobs.
	jobs := []k8s.JobSummary{
		{Name: "running", Status: k8s.StatusRunning, Variant: "lts"},
		{Name: "f2", Status: k8s.StatusFailed, Variant: "lts", EndTime: at(12)},
		{Name: "other", Status: k8s.StatusSucceeded, Variant: "", EndTime: at(11)},
		{Name: "f1", Status: k8s.StatusFailed, Variant: "lts", EndTime: at(10)},
		{Name: "remove", Status: k8s.StatusFailed, Variant: "lts", Task: k8s.TaskRemovePackage},
		{Name: "ok", Status: k8s.StatusSucceeded, Variant: "lts", EndTime: at(8)},
		{Name: "f0", Status: k8s.StatusFailed, Variant: "lts", EndTime: at(6)},
	}
	b := summarizeBuilds(jobs)
	lts := b["linux-lts"]
	if lts.failStreak != 2 || lts.running != 1 || lts.lastFinished.Name != "f2" || lts.lastSuccess.Name != "ok" {
		t.Errorf("linux-lts: %+v", lts)
	}
	if lts.byStatus[k8s.StatusFailed] != 3 {
		t.Errorf("maintenance jobs must not count as builds: %+v", lts.byStatus)
	}
	if l := b["linux"]; l.failStreak != 0 || l.lastSuccess.Name != "other" {
		t.Errorf("linux: %+v", l)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	srv, _ := setupTestServer(t)
	w := httptest.NewRecorder()
	srv.handleMetrics(w, httptest.NewRequest("GET", "/metrics", nil))
	body := w.Body.String()
	for _, want := range []string{
		"# TYPE arch_repo_packages gauge\narch_repo_packages 1\n",
		"arch_repo_jobs_list_ok 1\n",
		`arch_repo_build_fail_streak{kernel_pkg="linux"} 0`,
		`arch_repo_build_last_failed{kernel_pkg="linux",category="environment"} 0`,
		`arch_repo_build_jobs{kernel_pkg="linux",status="Succeeded"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in:\n%s", want, body)
		}
	}
	if strings.Count(body, "# HELP arch_repo_build_jobs ") != 1 {
		t.Errorf("HELP must be written once per family:\n%s", body)
	}
}
