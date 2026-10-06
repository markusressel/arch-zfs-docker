package builder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakePackage writes a minimal package archive repo-add accepts.
func fakePackage(t *testing.T, dir, name, pkgver string) {
	t.Helper()
	work := t.TempDir()
	info := fmt.Sprintf("pkgname = %s\npkgver = %s-1\npkgdesc = x\narch = x86_64\nsize = 1\nbuilddate = 1\n", name, pkgver)
	if err := os.WriteFile(filepath.Join(work, ".PKGINFO"), []byte(info), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, fmt.Sprintf("%s-%s-1-x86_64.pkg.tar.zst", name, pkgver))
	if b, err := exec.Command("tar", "--zstd", "-cf", out, "-C", work, ".PKGINFO").CombinedOutput(); err != nil {
		t.Fatalf("tar: %v: %s", err, b)
	}
}

func TestUpdateRepoDbListsNewestKernelFirst(t *testing.T) {
	for _, tool := range []string{"repo-add", "vercmp", "flock", "bsdtar", "tar"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not available", tool)
		}
	}

	dir := t.TempDir()
	// Higher ZFS version on an older kernel must not hide the build for the newest kernel.
	fakePackage(t, dir, "zfs-linux", "2.4.5_7.2.8.arch1.2")
	fakePackage(t, dir, "zfs-linux", "2.4.4_7.2.9.arch1.1")
	fakePackage(t, dir, "zfs-linux", "2.3.9_7.2.9.arch1.1")
	fakePackage(t, dir, "zfs-linux-lts", "2.4.4_6.18.55.1")
	fakePackage(t, dir, "zfs-utils", "2.4.3")
	fakePackage(t, dir, "zfs-utils", "2.4.4")

	script := filepath.Join(t.TempDir(), "update_repo_db.sh")
	if err := os.WriteFile(script, []byte(UpdateRepoDbScript), 0o755); err != nil {
		t.Fatal(err)
	}
	if b, err := exec.Command("bash", script, dir, "testrepo").CombinedOutput(); err != nil {
		t.Fatalf("script failed: %v: %s", err, b)
	}

	listing, err := exec.Command("bsdtar", "-tf", filepath.Join(dir, "testrepo.db")).Output()
	if err != nil {
		t.Fatal(err)
	}
	entries := string(listing)
	for _, want := range []string{
		"zfs-linux-2.4.4_7.2.9.arch1.1-1/desc",
		"zfs-linux-lts-2.4.4_6.18.55.1-1/desc",
		"zfs-utils-2.4.4-1/desc",
	} {
		if !strings.Contains(entries, want) {
			t.Errorf("db is missing %s; got:\n%s", want, entries)
		}
	}
	for _, unwanted := range []string{"2.4.5_7.2.8", "2.3.9_7.2.9", "zfs-utils-2.4.3"} {
		if strings.Contains(entries, unwanted) {
			t.Errorf("db must not list %s; got:\n%s", unwanted, entries)
		}
	}
	// Old package files stay on disk.
	if _, err := os.Stat(filepath.Join(dir, "zfs-linux-2.4.5_7.2.8.arch1.2-1-x86_64.pkg.tar.zst")); err != nil {
		t.Errorf("old package file was removed: %v", err)
	}
}
