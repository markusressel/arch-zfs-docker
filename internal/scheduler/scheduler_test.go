package scheduler

import (
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
