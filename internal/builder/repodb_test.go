package builder

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// fakeRunner records commands instead of running them.
type fakeRunner struct {
	calls [][]string
	// outputs answers Output by command line ("pacman -Q linux").
	outputs map[string]string
}

func (f *fakeRunner) Output(_ context.Context, _, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	line := strings.Join(append([]string{name}, args...), " ")
	out, ok := f.outputs[line]
	if !ok {
		return nil, fmt.Errorf("%s: unexpected command", line)
	}
	return []byte(out), nil
}

func (f *fakeRunner) Run(_ context.Context, dir, name string, args ...string) error {
	f.calls = append(f.calls, append([]string{name}, args...))
	// Create the archives the real tools would write; the caller moves them into place.
	var created []string
	switch name {
	case "repo-add": // repo-add <name>.db.tar.zst <packages...> also writes <name>.files.tar.zst
		base := strings.TrimSuffix(args[0], ".db.tar.zst")
		created = []string{base + ".db.tar.zst", base + ".files.tar.zst"}
	case "tar": // tar --zstd -cf <archive> ...
		created = []string{args[2]}
	}
	for _, f := range created {
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func touch(t *testing.T, dir string, files ...string) {
	t.Helper()
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestParsePackageFile(t *testing.T) {
	p, ok := ParsePackageFile("zfs-linux-lts-headers-2.4.4_6.18.55.1-1-x86_64.pkg.tar.zst")
	want := PackageFile{File: "zfs-linux-lts-headers-2.4.4_6.18.55.1-1-x86_64.pkg.tar.zst", Name: "zfs-linux-lts-headers", Pkgver: "2.4.4_6.18.55.1", Pkgrel: "1", Arch: "x86_64"}
	if !ok || p != want {
		t.Errorf("got %+v, %v", p, ok)
	}
	for _, bad := range []string{"zfslocal.db.tar.zst", "x-1-x86_64.pkg.tar.zst", "zfs-utils-2.4.4-1-x86_64.pkg.tar.zst.sig", "-2.4.4-1-x86_64.pkg.tar.zst"} {
		if _, ok := ParsePackageFile(bad); ok {
			t.Errorf("%q must not parse", bad)
		}
	}
}

func TestNewestPackagesPrefersTheNewestKernel(t *testing.T) {
	got := NewestPackages([]string{
		// A higher ZFS version on an older kernel must not hide the build for the newest kernel.
		"zfs-linux-2.4.5_7.2.8.arch1.2-1-x86_64.pkg.tar.zst",
		"zfs-linux-2.4.4_7.2.9.arch1.1-1-x86_64.pkg.tar.zst",
		"zfs-linux-2.3.9_7.2.9.arch1.1-1-x86_64.pkg.tar.zst",
		// Kernel 7.2.10 is newer than 7.2.9, also though it sorts lower as text.
		"zfs-linux-lts-2.4.4_6.18.9.1-1-x86_64.pkg.tar.zst",
		"zfs-linux-lts-2.4.4_6.18.10.1-1-x86_64.pkg.tar.zst",
		"zfs-utils-2.4.10-1-x86_64.pkg.tar.zst",
		"zfs-utils-2.4.9-2-x86_64.pkg.tar.zst",
		"zfslocal.db.tar.zst",
	})
	var files []string
	for _, p := range got {
		files = append(files, p.File)
	}
	want := []string{
		"zfs-linux-2.4.4_7.2.9.arch1.1-1-x86_64.pkg.tar.zst",
		"zfs-linux-lts-2.4.4_6.18.10.1-1-x86_64.pkg.tar.zst",
		"zfs-utils-2.4.10-1-x86_64.pkg.tar.zst",
	}
	if !reflect.DeepEqual(files, want) {
		t.Errorf("got %v\nwant %v", files, want)
	}
}

func TestNewerComparesPkgrelOfTheSameBuild(t *testing.T) {
	a, _ := ParsePackageFile("zfs-linux-2.4.4_7.2.9.arch1.1-2-x86_64.pkg.tar.zst")
	b, _ := ParsePackageFile("zfs-linux-2.4.4_7.2.9.arch1.1-1-x86_64.pkg.tar.zst")
	if !a.Newer(b) || b.Newer(a) {
		t.Error("a higher pkgrel of the same kernel and ZFS version must win")
	}
}

func TestUpdateRepoDBAddsOnlyTheNewestPackages(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "zfs-utils-2.4.3-1-x86_64.pkg.tar.zst", "zfs-utils-2.4.4-1-x86_64.pkg.tar.zst", "zfs-utils-2.4.4-1-x86_64.pkg.tar.zst.sig")
	run := &fakeRunner{}
	if err := UpdateRepoDB(context.Background(), run, io.Discard, dir, "testrepo"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"repo-add", "tmp-testrepo.db.tar.zst", "zfs-utils-2.4.4-1-x86_64.pkg.tar.zst"}}
	if !reflect.DeepEqual(run.calls, want) {
		t.Errorf("got %v", run.calls)
	}
	for _, kind := range []string{"db", "files"} {
		target, err := os.Readlink(filepath.Join(dir, "testrepo."+kind))
		if err != nil || target != "testrepo."+kind+".tar.zst" {
			t.Errorf("testrepo.%s -> %q (%v)", kind, target, err)
		}
		if _, err := os.Stat(filepath.Join(dir, "tmp-testrepo."+kind+".tar.zst")); !os.IsNotExist(err) {
			t.Errorf("temporary %s archive left behind", kind)
		}
	}
}

func TestUpdateRepoDBWritesAnEmptyDatabase(t *testing.T) {
	dir := t.TempDir()
	run := &fakeRunner{}
	if err := UpdateRepoDB(context.Background(), run, io.Discard, dir, "testrepo"); err != nil {
		t.Fatal(err)
	}
	if len(run.calls) != 2 || run.calls[0][0] != "tar" || run.calls[1][3] != "tmp-testrepo.files.tar.zst" {
		t.Errorf("got %v", run.calls)
	}
	if _, err := os.Readlink(filepath.Join(dir, "testrepo.db")); err != nil {
		t.Error(err)
	}
}

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

func requireTools(t *testing.T, tools ...string) {
	t.Helper()
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not available", tool)
		}
	}
}

// dbEntries lists the packages in a database archive.
func dbEntries(t *testing.T, db string) string {
	t.Helper()
	listing, err := exec.Command("bsdtar", "-tf", db).Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(listing)
}

// The same with the real repo-add.
func TestUpdateRepoDBWithRepoAdd(t *testing.T) {
	requireTools(t, "repo-add", "bsdtar", "tar")

	dir := t.TempDir()
	fakePackage(t, dir, "zfs-linux", "2.4.5_7.2.8.arch1.2")
	fakePackage(t, dir, "zfs-linux", "2.4.4_7.2.9.arch1.1")
	fakePackage(t, dir, "zfs-linux", "2.3.9_7.2.9.arch1.1")
	fakePackage(t, dir, "zfs-linux-lts", "2.4.4_6.18.55.1")
	fakePackage(t, dir, "zfs-utils", "2.4.3")
	fakePackage(t, dir, "zfs-utils", "2.4.4")

	if err := UpdateRepoDB(context.Background(), ExecRunner{Out: io.Discard}, io.Discard, dir, "testrepo"); err != nil {
		t.Fatal(err)
	}

	entries := dbEntries(t, filepath.Join(dir, "testrepo.db"))
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
