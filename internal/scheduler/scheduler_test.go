package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/markusressel/arch-zfs-docker/internal/k8s"
	"github.com/markusressel/arch-zfs-docker/internal/repo"
)

func TestSchedulerFetchUpstream(t *testing.T) {
	s := NewScheduler(repo.NewIndexer("/tmp", "zfslocal"), &k8s.MockClient{}, 1*time.Hour)
	pkg, err := s.FetchUpstreamKernel("linux")
	if err != nil {
		t.Skipf("Network unavailable or upstream rate-limited: %v", err)
	}

	if pkg.PkgName != "linux" {
		t.Errorf("Expected pkgname linux, got %s", pkg.PkgName)
	}
	if pkg.PkgVer == "" {
		t.Errorf("Expected non-empty pkgver")
	}
}

func TestArchPkgResponseFullVersion(t *testing.T) {
	pkg := &ArchPkgResponse{
		PkgName: "linux",
		PkgVer:  "7.2.8.arch1",
		PkgRel:  "2",
		Epoch:   0,
	}

	if pkg.FullVersion() != "7.2.8.arch1-2" {
		t.Errorf("Expected 7.2.8.arch1-2, got %s", pkg.FullVersion())
	}

	pkg.Epoch = 1
	if pkg.FullVersion() != "1:7.2.8.arch1-2" {
		t.Errorf("Expected 1:7.2.8.arch1-2, got %s", pkg.FullVersion())
	}
}

func TestSchedulerDynamicInterval(t *testing.T) {
	s := NewScheduler(repo.NewIndexer("/tmp", "zfslocal"), &k8s.MockClient{}, 1*time.Hour)
	if s.GetInterval() != 1*time.Hour {
		t.Fatalf("expected 1h interval, got %v", s.GetInterval())
	}

	s.SetInterval(12 * time.Hour)
	if s.GetInterval() != 12*time.Hour {
		t.Fatalf("expected 12h interval, got %v", s.GetInterval())
	}
}

func TestSchedulerGetStatus(t *testing.T) {
	s := NewScheduler(repo.NewIndexer("/tmp", "zfslocal"), &k8s.MockClient{}, 1*time.Hour)
	now := time.Now()
	s.mu.Lock()
	s.statuses["linux"] = &VariantStatus{
		Variant:        "",
		KernelPkg:      "linux",
		UpstreamKernel: "7.2.8.arch1-2",
		LocalKernel:    "7.2.8.arch1-2",
		UpToDate:       true,
		CheckedAt:      now,
	}
	s.mu.Unlock()

	status := s.GetStatus()
	if len(status) != 1 {
		t.Fatalf("expected 1 status entry, got %d", len(status))
	}
	if status["linux"].UpstreamKernel != "7.2.8.arch1-2" {
		t.Errorf("unexpected status: %v", status["linux"])
	}
}

func TestSchedulerStartAndCancel(t *testing.T) {
	s := NewScheduler(repo.NewIndexer("/tmp", "zfslocal"), &k8s.MockClient{}, 100*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	// Update interval dynamically while running
	s.SetInterval(200 * time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	s.SetInterval(0) // disable
	time.Sleep(50 * time.Millisecond)
	cancel()
}

func TestLocalKernelState(t *testing.T) {
	pkgs := []repo.PackageInfo{
		{PackageName: "zfs-linux", KernelVersion: "7.2.6.arch1.1"},
		{PackageName: "zfs-linux", KernelVersion: "7.2.9.arch1.1"},
		{PackageName: "zfs-linux-headers", KernelVersion: "7.9.9.arch1.1"}, // headers alone never count
		{PackageName: "zfs-linux", KernelVersion: "7.1.8.arch1.3"},
		{PackageName: "zfs-linux-lts", KernelVersion: "6.18.55.1"},
		{PackageName: "zfs-utils"},
	}

	latest, match := localKernelState(pkgs, "linux", "7.2.9.arch1-1")
	if latest != "7.2.9.arch1.1" || !match {
		t.Errorf("linux: got %q match=%v", latest, match)
	}

	latest, match = localKernelState(pkgs, "linux", "7.2.10.arch1-1")
	if latest != "7.2.9.arch1.1" || match {
		t.Errorf("linux newer upstream: got %q match=%v", latest, match)
	}

	latest, match = localKernelState(pkgs, "linux-lts", "6.18.55-1")
	if latest != "6.18.55.1" || !match {
		t.Errorf("lts: got %q match=%v", latest, match)
	}
}

func TestHasActiveJob(t *testing.T) {
	jobs := []k8s.JobSummary{
		{Name: "a", Variant: "", Status: "Succeeded"},
		{Name: "b", Variant: "lts", Status: "Running"},
		{Name: "c", Variant: "", Status: "Failed"},
	}
	if hasActiveJob(jobs, "") {
		t.Error("finished jobs must not count as active")
	}
	if !hasActiveJob(jobs, "lts") {
		t.Error("running lts job must count as active")
	}
	jobs = append(jobs, k8s.JobSummary{Name: "t", Variant: "", Status: "Running", Task: k8s.TaskRemovePackage})
	if hasActiveJob(jobs, "") {
		t.Error("maintenance tasks must not block builds")
	}
	jobs = append(jobs, k8s.JobSummary{Name: "d", Variant: "", Status: "Pending"})
	if !hasActiveJob(jobs, "") {
		t.Error("pending job must count as active")
	}
}
