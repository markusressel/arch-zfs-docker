package builder

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeSystem simulates the build container: it records every command (as "root: ..." or "build: ...")
// and creates the files the real tools would (git clones, packages).
type fakeSystem struct {
	t        *testing.T
	calls    []string
	heads    []string
	missing  map[string]bool // URLs Head reports as missing
	upstream string          // version `pacman -Si` reports
	utilsVer string          // pkgver of the zfs-utils PKGBUILD
	home     string
	repoDir  string
}

type fakeUser struct {
	sys  *fakeSystem
	name string
}

func (u fakeUser) Run(_ context.Context, dir, name string, args ...string) error {
	u.sys.calls = append(u.sys.calls, u.name+": "+strings.Join(append([]string{name}, args...), " "))
	return u.sys.simulate(dir, name, args)
}

func (u fakeUser) Output(_ context.Context, dir, name string, args ...string) ([]byte, error) {
	line := strings.Join(append([]string{name}, args...), " ")
	u.sys.calls = append(u.sys.calls, u.name+": "+line)
	switch {
	case strings.HasPrefix(line, "pacman -Si "):
		return []byte("Repository      : core\nName            : " + args[1] + "\nVersion         : " + u.sys.upstream + "\n"), nil
	case strings.HasPrefix(line, "pacman -Q "):
		return []byte(args[1] + " " + u.sys.upstream + "\n"), nil
	case line == "makepkg --printsrcinfo":
		b, _ := os.ReadFile(filepath.Join(dir, "PKGBUILD"))
		v := strings.TrimPrefix(regexpLine(string(b), "pkgver="), "pkgver=")
		return []byte("pkgbase = zfs-utils\n\tpkgver = " + v + "\n\tpkgrel = 1\n"), nil
	}
	return nil, fmt.Errorf("unexpected %s", line)
}

func regexpLine(s, prefix string) string {
	for _, l := range strings.Split(s, "\n") {
		if strings.HasPrefix(l, prefix) {
			return l
		}
	}
	return ""
}

func (f *fakeSystem) simulate(dir, name string, args []string) error {
	switch {
	case name == "git" && args[0] == "clone":
		dest := args[len(args)-1]
		os.MkdirAll(dest, 0o755)
		if strings.HasSuffix(dest, "zfs-utils") {
			return os.WriteFile(filepath.Join(dest, "PKGBUILD"), []byte("pkgname=zfs-utils\npkgver="+f.utilsVer+"\npkgrel=3\n"), 0o644)
		}
		return os.WriteFile(filepath.Join(dest, "PKGBUILD"), []byte(aurZfsLinuxPkgbuild), 0o644)
	case name == "makepkg" && args[0] != "--verifysource":
		// Write the packages a build produces.
		base := filepath.Base(dir)
		files := []string{"zfs-utils-" + f.utilsVer + "-1-x86_64.pkg.tar.zst"}
		if base != "zfs-utils" {
			kernel := strings.ReplaceAll(f.upstream, "-", ".")
			files = []string{base + "-2.4.4_" + kernel + "-1-x86_64.pkg.tar.zst", base + "-headers-2.4.4_" + kernel + "-1-x86_64.pkg.tar.zst"}
		}
		for _, file := range files {
			if err := os.WriteFile(filepath.Join(dir, file), []byte("pkg"), 0o644); err != nil {
				return err
			}
		}
	case name == "repo-add":
		base := strings.TrimSuffix(args[0], ".db.tar.zst")
		os.WriteFile(filepath.Join(dir, base+".db.tar.zst"), nil, 0o644)
		os.WriteFile(filepath.Join(dir, base+".files.tar.zst"), nil, 0o644)
	}
	return nil
}

func newFakeSystem(t *testing.T) (*fakeSystem, *BuildSystem, *bytes.Buffer) {
	t.Helper()
	root := t.TempDir()
	f := &fakeSystem{t: t, missing: map[string]bool{}, upstream: "7.2.9.arch1-1", utilsVer: "2.4.4", home: filepath.Join(root, "home", "build")}
	etc := filepath.Join(root, "etc")
	os.MkdirAll(etc, 0o755)
	os.WriteFile(filepath.Join(etc, "pacman.conf"), []byte("[options]\n#CacheDir    = /var/cache/pacman/pkg/\n"), 0o644)
	os.WriteFile(filepath.Join(etc, "makepkg.conf"), []byte("#MAKEFLAGS=\"-j2\"\n#SRCDEST=/home/srcdest\nPKGEXT='.pkg.tar.zst'\nCOMPRESSZST=(zstd -c -T0 -)\nOPTIONS=(strip docs !libtool !staticlibs emptydirs zipman purge debug lto)\n"), 0o644)
	out := &bytes.Buffer{}
	uid, gid := os.Getuid(), os.Getgid()
	sys := &BuildSystem{
		Root:      fakeUser{f, "root"},
		BuildUser: func(int, int) Runner { return fakeUser{f, "build"} },
		Out:       out,
		Head: func(_ context.Context, url string) error {
			f.heads = append(f.heads, url)
			if f.missing[url] {
				return fmt.Errorf("HTTP 404")
			}
			return nil
		},
		LookupUser:  func(string) (int, int, error) { return uid, gid, nil },
		RepoRoot:    filepath.Join(root, "repo"),
		PacmanConf:  filepath.Join(etc, "pacman.conf"),
		MakepkgConf: filepath.Join(etc, "makepkg.conf"),
		SudoersFile: filepath.Join(etc, "sudoers.d", "build"),
		BuildHome:   f.home,
	}
	f.repoDir = sys.repoDir("testrepo")
	return f, sys, out
}

func indexOfCall(calls []string, prefix string) int {
	for i, c := range calls {
		if strings.HasPrefix(c, prefix) {
			return i
		}
	}
	return -1
}

func TestBuildPublishesUtilsAndModule(t *testing.T) {
	f, sys, out := newFakeSystem(t)
	if err := (BuildJob{JobName: "job", RepoName: "testrepo"}).Run(context.Background(), sys); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}

	for _, want := range []string{
		"root: pacman -Si linux",
		"root: pacman -S --noconfirm --needed linux linux-headers",
		"build: gpg --batch --import " + filepath.Join(f.home, "work", "openzfs-keys.asc"),
		"build: makepkg --install --syncdeps --clean --rmdeps --noconfirm",
		"build: pacman -Q linux",
		"build: updpkgsums",
		"build: makepkg --syncdeps --clean --rmdeps --noconfirm",
		"root: repo-add tmp-testrepo.db.tar.zst zfs-linux-2.4.4_7.2.9.arch1.1-1-x86_64.pkg.tar.zst zfs-linux-headers-2.4.4_7.2.9.arch1.1-1-x86_64.pkg.tar.zst zfs-utils-2.4.4-1-x86_64.pkg.tar.zst",
	} {
		if indexOfCall(f.calls, want) < 0 {
			t.Errorf("missing call %q in:\n%s", want, strings.Join(f.calls, "\n"))
		}
	}
	// makepkg runs as the build user only, never skips signature checks, and every build downloads its
	// sources first (under the source cache lock).
	verified := false
	for _, c := range f.calls {
		if strings.Contains(c, "makepkg") && !strings.HasPrefix(c, "build: ") {
			t.Errorf("makepkg must not run as root: %s", c)
		}
		if strings.Contains(c, "--skippgpcheck") {
			t.Errorf("signature checks skipped: %s", c)
		}
		switch {
		case strings.HasPrefix(c, "build: makepkg --verifysource"):
			verified = true
		case strings.HasPrefix(c, "build: makepkg --") && !strings.Contains(c, "--printsrcinfo"):
			if !verified {
				t.Errorf("build without downloading its sources first: %s", c)
			}
			verified = false
		}
	}
	for _, f := range []string{"zfs-utils-2.4.4-1-x86_64.pkg.tar.zst", "zfs-linux-2.4.4_7.2.9.arch1.1-1-x86_64.pkg.tar.zst"} {
		if _, err := os.Stat(filepath.Join(sys.repoDir("testrepo"), f)); err != nil {
			t.Errorf("%s not published: %v", f, err)
		}
	}
	if !strings.Contains(out.String(), "==> Repository successfully updated for linux (7.2.9.arch1-1)!") {
		t.Errorf("unexpected log:\n%s", out)
	}

	// The configuration files point at the shared caches.
	conf, _ := os.ReadFile(sys.MakepkgConf)
	for _, want := range []string{"SRCDEST=" + sys.sourcesDir(), `MAKEFLAGS="-j$(nproc)"`, "OPTIONS=(strip docs !libtool !staticlibs emptydirs zipman purge !debug lto)"} {
		if !strings.Contains(string(conf), want) {
			t.Errorf("makepkg.conf lacks %q:\n%s", want, conf)
		}
	}
	pacmanConf, _ := os.ReadFile(sys.PacmanConf)
	if !strings.Contains(string(pacmanConf), "\nCacheDir = "+filepath.Join(sys.cacheDir(), "pacman")+" /var/cache/pacman/pkg\n") {
		t.Errorf("pacman.conf:\n%s", pacmanConf)
	}
}

func TestBuildSkipsKernelsThatAreBuiltAlready(t *testing.T) {
	f, sys, out := newFakeSystem(t)
	os.MkdirAll(sys.repoDir("testrepo"), 0o755)
	touch(t, sys.repoDir("testrepo"), "zfs-linux-2.4.4_7.2.9.arch1.1-1-x86_64.pkg.tar.zst")

	if err := (BuildJob{JobName: "job", RepoName: "testrepo"}).Run(context.Background(), sys); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Repository is up to date") || indexOfCall(f.calls, "build: ") >= 0 {
		t.Errorf("must not build:\n%s\n%s", out, strings.Join(f.calls, "\n"))
	}

	// FORCE_BUILD rebuilds anyway.
	f.calls = nil
	if err := (BuildJob{JobName: "job", RepoName: "testrepo", ForceBuild: true}).Run(context.Background(), sys); err != nil {
		t.Fatal(err)
	}
	if indexOfCall(f.calls, "build: makepkg --syncdeps") < 0 {
		t.Error("forced build did not build")
	}
}

func TestBuildInstallsARequestedKernelFromTheArchive(t *testing.T) {
	f, sys, _ := newFakeSystem(t)
	if err := (BuildJob{JobName: "job", RepoName: "testrepo", Variant: "lts", KernelVersion: "6.18.55-1"}).Run(context.Background(), sys); err != nil {
		t.Fatal(err)
	}
	pkg := "https://archive.archlinux.org/packages/l/linux-lts/linux-lts-6.18.55-1-x86_64.pkg.tar.zst"
	headers := "https://archive.archlinux.org/packages/l/linux-lts-headers/linux-lts-headers-6.18.55-1-x86_64.pkg.tar.zst"
	if len(f.heads) != 2 || f.heads[0] != pkg || f.heads[1] != headers {
		t.Errorf("heads = %v", f.heads)
	}
	if indexOfCall(f.calls, "root: pacman -U --noconfirm "+pkg+" "+headers) < 0 || indexOfCall(f.calls, "root: pacman -Si") >= 0 {
		t.Errorf("calls:\n%s", strings.Join(f.calls, "\n"))
	}

	// A kernel missing from the archive fails before anything is installed.
	f, sys, _ = newFakeSystem(t)
	f.missing[headers] = true
	err := (BuildJob{JobName: "job", RepoName: "testrepo", Variant: "lts", KernelVersion: "6.18.55-1"}).Run(context.Background(), sys)
	if err == nil || !strings.Contains(err.Error(), "not available in the Arch Linux Archive") {
		t.Errorf("expected an archive error, got %v", err)
	}
	if indexOfCall(f.calls, "root: pacman -S --noconfirm --needed base-devel") >= 0 {
		t.Error("installed packages despite the missing kernel")
	}
}

func TestBuildReusesThePublishedUtilsPackage(t *testing.T) {
	f, sys, _ := newFakeSystem(t)
	os.MkdirAll(sys.repoDir("testrepo"), 0o755)
	touch(t, sys.repoDir("testrepo"), "zfs-utils-2.4.4-1-x86_64.pkg.tar.zst", "zfs-utils-2.4.4-1-x86_64.pkg.tar.zst.sig")

	if err := (BuildJob{JobName: "job", RepoName: "testrepo"}).Run(context.Background(), sys); err != nil {
		t.Fatal(err)
	}
	if indexOfCall(f.calls, "root: pacman -U --noconfirm "+filepath.Join(sys.repoDir("testrepo"), "zfs-utils-2.4.4-1-x86_64.pkg.tar.zst")) < 0 {
		t.Errorf("published zfs-utils not installed:\n%s", strings.Join(f.calls, "\n"))
	}
	if indexOfCall(f.calls, "build: makepkg --install") >= 0 {
		t.Error("zfs-utils rebuilt")
	}
}

func TestBuildOverridesTheZfsVersion(t *testing.T) {
	f, sys, _ := newFakeSystem(t)
	if err := (BuildJob{JobName: "job", RepoName: "testrepo", ZfsVersion: "2.3.9"}).Run(context.Background(), sys); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(f.home, "work", "zfs-utils", "PKGBUILD"))
	if !strings.Contains(string(b), "pkgver=2.3.9\npkgrel=1") {
		t.Errorf("zfs-utils PKGBUILD:\n%s", b)
	}
	module, _ := os.ReadFile(filepath.Join(f.home, "work", "zfs-linux", "PKGBUILD"))
	if !strings.Contains(string(module), "_zfsver=2.3.9") {
		t.Errorf("module PKGBUILD does not use the requested version:\n%s", module)
	}
}

func TestEditConf(t *testing.T) {
	path := filepath.Join(t.TempDir(), "makepkg.conf")
	os.WriteFile(path, []byte("#MAKEFLAGS=\"-j2\"\nPKGEXT='.pkg.tar.xz'\n#-- OPTIONS below\n"), 0o644)
	if err := editConf(path, map[string]string{"MAKEFLAGS": `="-j8"`, "PKGEXT": `=".pkg.tar.zst"`, "SRCDEST": "=/cache"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	want := "MAKEFLAGS=\"-j8\"\nPKGEXT=\".pkg.tar.zst\"\n#-- OPTIONS below\nSRCDEST=/cache\n"
	if string(b) != want {
		t.Errorf("got:\n%s\nwant:\n%s", b, want)
	}
}
