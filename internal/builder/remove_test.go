package builder

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePackageFileName(t *testing.T) {
	for _, ok := range []string{"zfs-linux-2.4.4_7.2.9.arch1.1-1-x86_64.pkg.tar.zst", "zfs-utils-2.4.4-1-x86_64.pkg.tar.xz", "a+b@c-1-1-any.pkg.tar.gz"} {
		if err := ValidatePackageFileName(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	// Path tricks and other files are refused.
	for _, bad := range []string{"", "../x.pkg.tar.zst", "a/b.pkg.tar.zst", ".hidden.pkg.tar.zst", "notapackage.txt", "x;rm.pkg.tar.zst", "zfslocal.db.tar.zst", "x.pkg.tar.zst.sig"} {
		if err := ValidatePackageFileName(bad); err == nil {
			t.Errorf("%q must be refused", bad)
		}
	}
}

func TestRemovePackageDeletesFileAndSignature(t *testing.T) {
	dir := t.TempDir()
	file := "zfs-utils-2.4.4-1-x86_64.pkg.tar.zst"
	touch(t, dir, file, file+".sig", "zfs-utils-2.4.3-1-x86_64.pkg.tar.zst")
	run := &fakeRunner{}
	if err := RemovePackage(context.Background(), run, io.Discard, dir, "testrepo", file); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{file, file + ".sig"} {
		if _, err := os.Stat(filepath.Join(dir, f)); !os.IsNotExist(err) {
			t.Errorf("%s still exists", f)
		}
	}
	// The database falls back to the remaining version.
	if len(run.calls) != 1 || run.calls[0][2] != "zfs-utils-2.4.3-1-x86_64.pkg.tar.zst" {
		t.Errorf("got %v", run.calls)
	}
}

func TestRemovePackageRefusesUnsafeNames(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "keep.pkg.tar.zst")
	run := &fakeRunner{}
	if err := RemovePackage(context.Background(), run, io.Discard, dir, "testrepo", "../"+filepath.Base(dir)+"/keep.pkg.tar.zst"); err == nil {
		t.Fatal("expected an error")
	}
	if len(run.calls) != 0 {
		t.Errorf("nothing may run: %v", run.calls)
	}
	if _, err := os.Stat(filepath.Join(dir, "keep.pkg.tar.zst")); err != nil {
		t.Error("file was touched")
	}
}

// The same with the real repo-add: removing the newest build falls back to the older one, removing the last
// package leaves an empty database instead of a stale one.
func TestRemovePackageWithRepoAdd(t *testing.T) {
	requireTools(t, "repo-add", "bsdtar", "tar")

	dir := t.TempDir()
	fakePackage(t, dir, "zfs-linux", "2.4.4_7.2.9.arch1.1")
	fakePackage(t, dir, "zfs-linux", "2.4.4_7.2.8.arch1.2")
	run := ExecRunner{Out: io.Discard}
	ctx := context.Background()

	if err := RemovePackage(ctx, run, io.Discard, dir, "testrepo", "zfs-linux-2.4.4_7.2.9.arch1.1-1-x86_64.pkg.tar.zst"); err != nil {
		t.Fatal(err)
	}
	entries := dbEntries(t, filepath.Join(dir, "testrepo.db"))
	if !strings.Contains(entries, "zfs-linux-2.4.4_7.2.8.arch1.2-1/desc") || strings.Contains(entries, "7.2.9") {
		t.Errorf("unexpected database entries:\n%s", entries)
	}

	if err := RemovePackage(ctx, run, io.Discard, dir, "testrepo", "zfs-linux-2.4.4_7.2.8.arch1.2-1-x86_64.pkg.tar.zst"); err != nil {
		t.Fatal(err)
	}
	if entries := dbEntries(t, filepath.Join(dir, "testrepo.db")); strings.Contains(entries, "zfs-linux") {
		t.Errorf("stale database entries:\n%s", entries)
	}
}
