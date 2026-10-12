package builder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
)

// BuildJob is one build of the ZFS packages for a kernel, configured like the build jobs via environment
// (see k8s.buildEnv).
type BuildJob struct {
	JobName  string
	RepoName string
	// Variant is "" for linux, "lts" for linux-lts.
	Variant string
	// KernelVersion is a specific kernel from the Arch Linux Archive, or "" for the current one.
	KernelVersion string
	// ZfsVersion overrides the OpenZFS version of the AUR PKGBUILD ("" keeps it).
	ZfsVersion             string
	ForceBuild             bool
	ForceRebuildUtils      bool
	AllowUnsupportedKernel bool
}

// BuildSystem is everything a build touches outside of the Go process; tests replace all of it.
type BuildSystem struct {
	// Root runs commands as root, BuildUser runs them as the unprivileged build user (makepkg refuses root).
	Root      Runner
	BuildUser func(uid, gid int) Runner
	Out       io.Writer
	// Head checks that a URL exists (HTTP HEAD, following redirects).
	Head func(ctx context.Context, url string) error
	// LookupUser returns uid and gid of a user, or an error if it does not exist.
	LookupUser func(name string) (uid, gid int, err error)

	RepoRoot    string // /repo: packages in <root>/<repo>/x86_64, sources and pacman cache in <root>/cache
	PacmanConf  string // /etc/pacman.conf
	MakepkgConf string // /etc/makepkg.conf
	SudoersFile string // /etc/sudoers.d/build
	BuildHome   string // /home/build
}

const (
	buildUser  = "build"
	archiveURL = "https://archive.archlinux.org/packages/l"
	aurURL     = "https://aur.archlinux.org"
)

// Log lines the server parses from build logs (internal/compat): keep them unchanged.
const (
	logRequestedKernel = "==> Requested %s version: %s\n"
	logLatestKernel    = "==> Latest available %s in official repos: %s\n"
	logRequestedZfs    = "==> Requested OpenZFS version: %s (overriding the AUR default)\n"
	logUsingZfs        = "==> Using OpenZFS version: %s\n"
	logModulePhase     = "==> 2/2: Building zfs-linux from AUR...\n"
)

func (s *BuildSystem) repoDir(repoName string) string {
	return filepath.Join(s.RepoRoot, repoName, "x86_64")
}
func (s *BuildSystem) cacheDir() string   { return filepath.Join(s.RepoRoot, "cache") }
func (s *BuildSystem) sourcesDir() string { return filepath.Join(s.cacheDir(), "sources") }

// Run builds zfs-utils (unless the repository has it already) and the kernel module package for the job's
// kernel, and publishes them to the repository.
func (j BuildJob) Run(ctx context.Context, s *BuildSystem) error {
	out := s.Out
	kernelPkg := KernelPackage(j.Variant)
	repoDir := s.repoDir(j.RepoName)
	for _, d := range []string{repoDir, s.sourcesDir(), filepath.Join(s.cacheDir(), "pacman")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}

	fmt.Fprintln(out, "==> Initializing pacman keyring...")
	// Best effort, as the keyring of the image is usually initialized already.
	_ = s.Root.Run(ctx, "", "pacman-key", "--init")
	_ = s.Root.Run(ctx, "", "pacman-key", "--populate", "archlinux")
	fmt.Fprintln(out, "==> Refreshing Arch pacman database...")
	if err := s.Root.Run(ctx, "", "pacman", "-Sy", "--noconfirm", "archlinux-keyring"); err != nil {
		return err
	}

	target, pinned, err := j.targetKernel(ctx, s, kernelPkg)
	if err != nil {
		return err
	}

	if existing := existingModulePackage(repoDir, kernelPkg, j.ZfsVersion, target); existing != "" && !j.ForceBuild {
		fmt.Fprintf(out, "==> Package for %s (%s) already exists in %s:\n", kernelPkg, target, repoDir)
		fmt.Fprintf(out, "    %s\n", existing)
		fmt.Fprintln(out, "==> Repository is up to date. Exiting cleanly.")
		return nil
	}

	fmt.Fprintln(out, "==> New kernel detected or FORCE_BUILD=true. Preparing build environment...")
	uid, gid, err := s.prepare(ctx, kernelPkg, pinned)
	if err != nil {
		return err
	}
	build := s.BuildUser(uid, gid)
	work := filepath.Join(s.BuildHome, "work")

	keys := filepath.Join(work, "openzfs-keys.asc")
	if err := os.WriteFile(keys, []byte(OpenZFSKeys), 0o644); err != nil {
		return err
	}
	// makepkg verifies the OpenZFS release tarball against its detached signature (validpgpkeys), so a
	// tampered or truncated download fails the build instead of being packaged.
	fmt.Fprintln(out, "==> Importing OpenZFS release signing keys...")
	if err := build.Run(ctx, work, "gpg", "--batch", "--import", keys); err != nil {
		return err
	}

	zfsVersion, err := j.buildUtils(ctx, s, build, work)
	if err != nil {
		return err
	}

	fmt.Fprint(out, logModulePhase)
	moduleDir := filepath.Join(work, "zfs-"+kernelPkg)
	if err := build.Run(ctx, work, "git", "clone", "--depth", "1", "--quiet", aurURL+"/zfs-"+kernelPkg+".git", moduleDir); err != nil {
		return err
	}
	pkgbuild := PkgbuildTarget{ZfsVersion: zfsVersion, Variant: j.Variant, AllowUnsupportedKernel: j.AllowUnsupportedKernel}
	if err := s.withSourceLock(func() error { return UpdatePkgbuild(ctx, build, out, moduleDir, pkgbuild) }); err != nil {
		return err
	}
	// The kernel build system signs modules when a key is present; a fresh self-signed one per build.
	if err := build.Run(ctx, moduleDir, "openssl", "req", "-new", "-nodes", "-utf8", "-sha512", "-days", "36500", "-batch", "-x509",
		"-subj", "/C=US/ST=State/L=City/O=Organization/OU=Unit/CN=Regenerated-ZFS-Key",
		"-outform", "DER", "-out", "signing_key.x509", "-keyout", "signing_key.pem"); err != nil {
		return err
	}
	if err := s.makepkg(ctx, build, moduleDir, "--syncdeps", "--clean", "--rmdeps", "--noconfirm"); err != nil {
		return err
	}
	fmt.Fprintln(out, "==> Build completed successfully.")

	fmt.Fprintf(out, "==> Publishing packages to repository directory: %s\n", repoDir)
	if err := publishPackages(out, repoDir, filepath.Join(work, "zfs-utils"), moduleDir); err != nil {
		return err
	}
	fmt.Fprintf(out, "==> Updating pacman repository database: %s.db.tar.zst\n", j.RepoName)
	if err := UpdateRepoDB(ctx, s.Root, out, repoDir, j.RepoName); err != nil {
		return err
	}
	fmt.Fprintf(out, "==> Repository successfully updated for %s (%s)!\n", kernelPkg, target)
	return nil
}

// targetKernel returns the kernel version to build for and, for a requested version, the archive
// packages to install.
func (j BuildJob) targetKernel(ctx context.Context, s *BuildSystem, kernelPkg string) (string, []string, error) {
	if j.KernelVersion != "" {
		// Archive file names use ".arch" (7.2.8.arch1-2), requests may use "-arch".
		target := strings.Replace(j.KernelVersion, "-arch", ".arch", 1)
		fmt.Fprintf(s.Out, logRequestedKernel, kernelPkg, target)
		pinned := []string{
			fmt.Sprintf("%s/%s/%s-%s-x86_64.pkg.tar.zst", archiveURL, kernelPkg, kernelPkg, target),
			fmt.Sprintf("%s/%s-headers/%s-headers-%s-x86_64.pkg.tar.zst", archiveURL, kernelPkg, kernelPkg, target),
		}
		for _, url := range pinned {
			if err := s.Head(ctx, url); err != nil {
				return "", nil, fmt.Errorf("%s not found (%v). Kernel version %s is not available in the Arch Linux Archive", url, err, target)
			}
		}
		return target, pinned, nil
	}

	info, err := s.Root.Output(ctx, "", "pacman", "-Si", kernelPkg)
	if err != nil {
		return "", nil, err
	}
	target := pacmanField(string(info), "Version")
	if target == "" {
		return "", nil, fmt.Errorf("unable to determine the latest version of %s", kernelPkg)
	}
	fmt.Fprintf(s.Out, logLatestKernel, kernelPkg, target)
	return target, nil, nil
}

// pacmanField reads "Key : value" from `pacman -Si` output.
func pacmanField(info, key string) string {
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `\s*:\s*(\S+)`)
	if m := re.FindStringSubmatch(info); m != nil {
		return m[1]
	}
	return ""
}

// existingModulePackage returns a module package for the kernel (and ZFS version, if given) that the
// repository already has. Package file names use dots only (zfs-linux-2.4.4_7.2.8.arch1.2-1-x86_64.pkg.tar.zst).
func existingModulePackage(repoDir, kernelPkg, zfsVersion, kernel string) string {
	zfs := zfsVersion
	if zfs == "" {
		zfs = "[0-9]*"
	}
	matches, _ := filepath.Glob(filepath.Join(repoDir, "zfs-"+kernelPkg+"-"+zfs+"_"+strings.ReplaceAll(kernel, "-", ".")+"-*.pkg.tar*"))
	for _, m := range matches {
		if !strings.HasSuffix(m, ".sig") {
			return m
		}
	}
	return ""
}

// prepare installs the build tools and the kernel and sets up the build user, its work directory and the
// caches. It returns the build user's uid and gid.
func (s *BuildSystem) prepare(ctx context.Context, kernelPkg string, pinned []string) (int, int, error) {
	if _, _, err := s.LookupUser(buildUser); err != nil {
		if err := s.Root.Run(ctx, "", "useradd", "--create-home", "--shell", "/bin/bash", buildUser); err != nil {
			return 0, 0, err
		}
	}
	uid, gid, err := s.LookupUser(buildUser)
	if err != nil {
		return 0, 0, err
	}
	// makepkg installs dependencies with sudo.
	if err := os.MkdirAll(filepath.Dir(s.SudoersFile), 0o750); err != nil {
		return 0, 0, err
	}
	if err := os.WriteFile(s.SudoersFile, []byte(buildUser+" ALL=(ALL) NOPASSWD: ALL\n"), 0o440); err != nil {
		return 0, 0, err
	}

	// Package and source caches live on the repository volume, shared by all jobs.
	if err := editConf(s.PacmanConf, map[string]string{"CacheDir": " = " + filepath.Join(s.cacheDir(), "pacman") + " /var/cache/pacman/pkg"}); err != nil {
		return 0, 0, err
	}
	if err := s.Root.Run(ctx, "", "pacman", "-S", "--noconfirm", "--needed", "base-devel", "git", "pacman-contrib", "openssl", "curl"); err != nil {
		return 0, 0, err
	}
	if len(pinned) > 0 {
		err = s.Root.Run(ctx, "", "pacman", append([]string{"-U", "--noconfirm"}, pinned...)...)
	} else {
		err = s.Root.Run(ctx, "", "pacman", "-S", "--noconfirm", "--needed", kernelPkg, kernelPkg+"-headers")
	}
	if err != nil {
		return 0, 0, err
	}

	// makepkg: all CPU cores, the shared source cache, fast compression, no debug package.
	if err := editConf(s.MakepkgConf, map[string]string{
		"MAKEFLAGS":   `="-j$(nproc)"`,
		"SRCDEST":     "=" + s.sourcesDir(),
		"PKGEXT":      `=".pkg.tar.zst"`,
		"COMPRESSZST": "=(zstd -c -T0 -3 -)",
		"OPTIONS":     "=(strip docs !libtool !staticlibs emptydirs zipman purge !debug lto)",
	}); err != nil {
		return 0, 0, err
	}

	work := filepath.Join(s.BuildHome, "work")
	if err := os.RemoveAll(work); err != nil {
		return 0, 0, err
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		return 0, 0, err
	}
	// Only the source cache is written by the build user. The pacman cache is managed by root (and its
	// sandboxed download user): recursively chowning it breaks downloads of concurrently running jobs.
	for _, dir := range []string{s.BuildHome, s.sourcesDir()} {
		if err := chownTree(dir, uid, gid); err != nil {
			return 0, 0, err
		}
	}
	return uid, gid, nil
}

// buildUtils builds and installs zfs-utils (a dependency of the module package), or installs the
// repository's package for the OpenZFS version. It returns the OpenZFS version.
func (j BuildJob) buildUtils(ctx context.Context, s *BuildSystem, build Runner, work string) (string, error) {
	out := s.Out
	fmt.Fprintln(out, "==> 1/2: Checking zfs-utils from AUR...")
	dir := filepath.Join(work, "zfs-utils")
	if err := build.Run(ctx, work, "git", "clone", "--depth", "1", "--quiet", aurURL+"/zfs-utils.git", dir); err != nil {
		return "", err
	}
	if j.ZfsVersion != "" {
		fmt.Fprintf(out, logRequestedZfs, j.ZfsVersion)
		if err := setPkgver(filepath.Join(dir, "PKGBUILD"), j.ZfsVersion); err != nil {
			return "", err
		}
		// updpkgsums downloads the sources of the new version.
		if err := s.withSourceLock(func() error { return build.Run(ctx, dir, "updpkgsums") }); err != nil {
			return "", err
		}
	}
	srcinfo, err := build.Output(ctx, dir, "makepkg", "--printsrcinfo")
	if err != nil {
		return "", err
	}
	version := srcinfoField(string(srcinfo), "pkgver")
	if version == "" {
		return "", fmt.Errorf("zfs-utils PKGBUILD has no pkgver")
	}
	fmt.Fprintf(out, logUsingZfs, version)

	if existing := existingUtilsPackage(s.repoDir(j.RepoName), version); existing != "" && !j.ForceRebuildUtils {
		fmt.Fprintf(out, "==> Existing zfs-utils package found: %s\n", existing)
		fmt.Fprintln(out, "==> Skipping zfs-utils compilation and installing from repository cache...")
		return version, s.Root.Run(ctx, "", "pacman", "-U", "--noconfirm", existing)
	}
	fmt.Fprintln(out, "==> Building zfs-utils from AUR...")
	return version, s.makepkg(ctx, build, dir, "--install", "--syncdeps", "--clean", "--rmdeps", "--noconfirm")
}

// makepkg downloads and verifies the sources under the source lock, then builds.
func (s *BuildSystem) makepkg(ctx context.Context, build Runner, dir string, args ...string) error {
	if err := s.withSourceLock(func() error { return build.Run(ctx, dir, "makepkg", "--verifysource", "--noconfirm") }); err != nil {
		return err
	}
	return build.Run(ctx, dir, "makepkg", args...)
}

// withSourceLock runs fn holding the lock of the shared source cache. makepkg downloads to <file>.part and
// renames it when done, so two jobs fetching the same file at once break each other's download. Everything
// that downloads sources holds this lock; the builds afterwards only read finished files.
func (s *BuildSystem) withSourceLock(fn func() error) error {
	f, err := os.OpenFile(filepath.Join(s.sourcesDir(), ".download.lock"), os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		return fmt.Errorf("open source cache lock: %w", err)
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock source cache: %w", err)
	}
	return fn()
}

func existingUtilsPackage(repoDir, version string) string {
	matches, _ := filepath.Glob(filepath.Join(repoDir, "zfs-utils-"+version+"-*.pkg.tar*"))
	sort.Strings(matches)
	for _, m := range matches {
		if !strings.HasSuffix(m, ".sig") && !strings.Contains(filepath.Base(m), "debug") {
			return m
		}
	}
	return ""
}

// srcinfoField reads "key = value" from `makepkg --printsrcinfo` output.
func srcinfoField(srcinfo, key string) string {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + ` = (\S+)`)
	if m := re.FindStringSubmatch(srcinfo); m != nil {
		return m[1]
	}
	return ""
}

// setPkgver switches a PKGBUILD to another version; the pkgrel restarts at 1.
func setPkgver(path, version string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := string(b)
	for key, value := range map[string]string{"pkgver": version, "pkgrel": "1"} {
		re := regexp.MustCompile(`(?m)^` + key + `=.*$`)
		if !re.MatchString(s) {
			return fmt.Errorf("%s has no %s= line", path, key)
		}
		s = re.ReplaceAllLiteralString(s, key+"="+value)
	}
	return os.WriteFile(path, []byte(s), 0o644)
}

// editConf sets "KEY<rest>" lines of a config file such as makepkg.conf: an existing (possibly commented
// out) line is replaced, a missing one is appended.
func editConf(path string, settings map[string]string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s := string(b)
	keys := make([]string, 0, len(settings))
	for k := range settings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		line := key + settings[key]
		re := regexp.MustCompile(`(?m)^#*\s*` + regexp.QuoteMeta(key) + `\s*[=(].*$`)
		if loc := re.FindStringIndex(s); loc != nil {
			s = s[:loc[0]] + line + s[loc[1]:]
		} else {
			s = strings.TrimRight(s, "\n") + "\n" + line + "\n"
		}
	}
	return os.WriteFile(path, []byte(s), 0o644)
}

// chownTree hands a directory tree to a user. Files that disappear meanwhile (e.g. a concurrent job's
// finished download) are ignored.
func chownTree(dir string, uid, gid int) error {
	return filepath.WalkDir(dir, func(path string, _ os.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if err := os.Lchown(path, uid, gid); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	})
}

// publishPackages copies the built packages into the repository. Each file is written under a temporary
// name and renamed, so clients never download a partially copied package.
func publishPackages(out io.Writer, repoDir string, buildDirs ...string) error {
	for _, dir := range buildDirs {
		files, _ := filepath.Glob(filepath.Join(dir, "*.pkg.tar.zst"))
		for _, f := range files {
			dst := filepath.Join(repoDir, filepath.Base(f))
			if err := copyFile(f, dst); err != nil {
				return err
			}
			fmt.Fprintf(out, "'%s' -> '%s'\n", f, dst)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := filepath.Join(filepath.Dir(dst), ".tmp-"+filepath.Base(dst))
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

// httpHead checks that a URL exists, following redirects (like curl -fsIL).
func httpHead(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}
