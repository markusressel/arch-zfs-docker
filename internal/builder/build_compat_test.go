package builder_test

import (
	"context"
	"testing"

	"github.com/markusressel/arch-zfs-docker/internal/builder"
	"github.com/markusressel/arch-zfs-docker/internal/compat"
)

// The server learns kernel/ZFS incompatibilities from build logs (internal/compat): the log lines it
// parses must stay intact.
func TestBuildLogIsUnderstoodByTheCompatClassifier(t *testing.T) {
	_, sys, out := builder.NewFakeSystem(t)
	if err := (builder.BuildJob{JobName: "job", RepoName: "testrepo", KernelVersion: "7.2.8-arch1-2", ZfsVersion: "2.3.9"}).Run(context.Background(), sys); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	kernel, zfs := compat.ParseVersions(out.String())
	if kernel != "7.2.8.arch1-2" || zfs != "2.3.9" {
		t.Errorf("ParseVersions = %q, %q from:\n%s", kernel, zfs, out)
	}
	// A configure failure after the module phase marker counts as a kernel incompatibility.
	log := out.String() + "configure: error: *** Cannot build against kernel version 7.3. The maximum supported kernel version is 7.2.\n"
	if v := compat.Classify(log); !v.Incompatible {
		t.Errorf("not classified as incompatible: %+v", v)
	}
}
