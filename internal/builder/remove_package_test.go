package builder

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemovePackageScript(t *testing.T) {
	for _, tool := range []string{"repo-add", "vercmp", "flock", "bsdtar", "tar"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not available", tool)
		}
	}

	root := t.TempDir()
	dir := filepath.Join(root, "testrepo", "x86_64")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	fakePackage(t, dir, "zfs-linux", "2.4.4_7.2.9.arch1.1")
	fakePackage(t, dir, "zfs-linux", "2.4.4_7.2.8.arch1.2")
	os.WriteFile(filepath.Join(dir, "zfs-linux-2.4.4_7.2.9.arch1.1-1-x86_64.pkg.tar.zst.sig"), []byte("sig"), 0o644)

	scripts := t.TempDir()
	os.WriteFile(filepath.Join(scripts, "remove_package.sh"), []byte(RemovePackageScript), 0o755)
	os.WriteFile(filepath.Join(scripts, "update_repo_db.sh"), []byte(UpdateRepoDbScript), 0o755)

	run := func(file string) (string, error) {
		cmd := exec.Command("bash", filepath.Join(scripts, "remove_package.sh"))
		cmd.Env = append(os.Environ(), "REPO_ROOT="+root, "REPO_NAME=testrepo", "JOB_NAME=job-1", "PACKAGE_FILE="+file)
		b, err := cmd.CombinedOutput()
		return string(b), err
	}

	// Path tricks are refused and nothing is touched.
	for _, bad := range []string{"../x.pkg.tar.zst", "a/b.pkg.tar.zst", "notapackage.txt", "x;rm.pkg.tar.zst"} {
		if out, err := run(bad); err == nil {
			t.Errorf("%q must be refused, got: %s", bad, out)
		}
	}

	newest := "zfs-linux-2.4.4_7.2.9.arch1.1-1-x86_64.pkg.tar.zst"
	if out, err := run(newest); err != nil {
		t.Fatalf("remove failed: %v: %s", err, out)
	}
	if _, err := os.Stat(filepath.Join(dir, newest)); !os.IsNotExist(err) {
		t.Error("package file still exists")
	}
	if _, err := os.Stat(filepath.Join(dir, newest+".sig")); !os.IsNotExist(err) {
		t.Error("signature still exists")
	}

	// The database now falls back to the remaining (older kernel) build.
	listing, err := exec.Command("bsdtar", "-tf", filepath.Join(dir, "testrepo.db")).Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(listing), "zfs-linux-2.4.4_7.2.8.arch1.2-1/desc") || strings.Contains(string(listing), "7.2.9") {
		t.Errorf("unexpected database entries:\n%s", listing)
	}

	// Removing the last package leaves an empty database instead of a stale one.
	if out, err := run("zfs-linux-2.4.4_7.2.8.arch1.2-1-x86_64.pkg.tar.zst"); err != nil {
		t.Fatalf("remove failed: %v: %s", err, out)
	}
	listing, err = exec.Command("bsdtar", "-tf", filepath.Join(dir, "testrepo.db")).Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(listing), "zfs-linux") {
		t.Errorf("stale database entries:\n%s", listing)
	}
}
