package builder

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

const pkgExt = ".pkg.tar.zst"

// PackageFile is a package file name split into its parts:
// <name>-<pkgver>-<pkgrel>-<arch>.pkg.tar.zst (pkgver and pkgrel never contain a hyphen).
type PackageFile struct {
	File, Name, Pkgver, Pkgrel, Arch string
}

// ParsePackageFile splits a package file name; ok is false for anything else.
func ParsePackageFile(file string) (p PackageFile, ok bool) {
	base, found := strings.CutSuffix(file, pkgExt)
	if !found {
		return p, false
	}
	parts := strings.Split(base, "-")
	if len(parts) < 4 {
		return p, false
	}
	n := len(parts)
	p = PackageFile{File: file, Name: strings.Join(parts[:n-3], "-"), Pkgver: parts[n-3], Pkgrel: parts[n-2], Arch: parts[n-1]}
	if p.Name == "" || p.Pkgver == "" || p.Pkgrel == "" || p.Arch == "" {
		return PackageFile{}, false
	}
	return p, true
}

// Version is the full pacman version, <pkgver>-<pkgrel>.
func (p PackageFile) Version() string { return p.Pkgver + "-" + p.Pkgrel }

// Newer reports whether p should be listed in the database instead of q (same package name).
//
// Kernel module packages have pkgver <zfsver>_<kernelver>: they are compared by kernel first, then by
// ZFS version. zfs-linux depends on linux=<kernel> exactly, so the database must follow the kernel Arch
// ships; a higher ZFS version built for an older kernel must not hide the build for the current kernel
// and block `pacman -Syu`. Other packages compare like pacman does.
func (p PackageFile) Newer(q PackageFile) bool {
	a, b := p.Version(), q.Version()
	zfsA, kernelA, okA := strings.Cut(a, "_")
	zfsB, kernelB, okB := strings.Cut(b, "_")
	if okA && okB {
		if c := Vercmp(kernelA, kernelB); c != 0 {
			return c > 0
		}
		return Vercmp(zfsA, zfsB) > 0
	}
	return Vercmp(a, b) > 0
}

// NewestPackages picks the package file the database lists for every package name, sorted by file name.
func NewestPackages(files []string) []PackageFile {
	newest := map[string]PackageFile{}
	for _, f := range files {
		p, ok := ParsePackageFile(f)
		if !ok {
			continue
		}
		if cur, seen := newest[p.Name]; !seen || p.Newer(cur) {
			newest[p.Name] = p
		}
	}
	out := make([]PackageFile, 0, len(newest))
	for _, p := range newest {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out
}

// UpdateRepoDB rebuilds the pacman database of a repository directory.
//
// Old package files stay on disk, so users on an older kernel can still `pacman -U` a matching build,
// but a pacman database lists only ONE version per package name and it must be the newest (see Newer):
// repo-add alone would let the most recently added package win, even if it is older.
//
// With a signer, the database lists only packages with a valid signature of the repository key, and the
// database itself is signed. A package without one (unsigned, or swapped on the storage) is left out with
// a warning: signing it here would sign whatever is on the disk, so only the step that built a package
// signs it (IndexRepo). Without a signer, the database is unsigned and old database signatures are removed.
func UpdateRepoDB(ctx context.Context, run Runner, out io.Writer, dir, repoName string, signer *Signer) error {
	// Serialize with concurrent jobs: repo-add fails on a held lock instead of waiting.
	lock, err := os.OpenFile(filepath.Join(dir, ".repo-db.lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open lock: %w", err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock database: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*"+pkgExt))
	if err != nil {
		return err
	}
	var listable []string
	for _, f := range files {
		file := filepath.Base(f)
		// Anyone who can write the volume can name a file; repo-add and gpg must never take one for an option.
		if err := ValidatePackageFileName(file); err != nil {
			fmt.Fprintf(out, "==> WARNING: ignoring %s: %v\n", file, err)
			continue
		}
		if signer != nil && !signer.Verify(ctx, dir, file) {
			fmt.Fprintf(out, "==> WARNING: %s has no valid signature, the database leaves it out. Rebuild it to sign it.\n", file)
			continue
		}
		listable = append(listable, file)
	}
	newest := NewestPackages(listable)
	tmp := "tmp-" + repoName
	signed := signer != nil
	if err := publishKey(ctx, dir, repoName, signer); err != nil {
		return err
	}

	if len(newest) == 0 {
		// E.g. after the last package was removed: clients must not see stale entries.
		fmt.Fprintf(out, "==> No packages found in %s, writing an empty database.\n", dir)
		for _, kind := range []string{"db", "files"} {
			archive := tmp + "." + kind + ".tar.zst"
			if err := run.Run(ctx, dir, "tar", "--zstd", "-cf", archive, "-T", "/dev/null"); err != nil {
				return err
			}
			if signed {
				if err := signer.Sign(ctx, dir, archive); err != nil {
					return err
				}
			}
			if err := publish(dir, tmp, repoName, kind, signed); err != nil {
				return err
			}
		}
		return nil
	}

	// Build under a temporary name and move into place, so clients never see a missing database.
	removeTmp(dir, tmp)
	cmd, args := "repo-add", []string{tmp + ".db.tar.zst"}
	if signed {
		// repo-add calls gpg itself and finds the keyring only through GNUPGHOME.
		cmd, args = "env", []string{"GNUPGHOME=" + signer.Home, "repo-add", "--sign", "--key", signer.Fingerprint, "--", tmp + ".db.tar.zst"}
	} else {
		args = append([]string{"--"}, args...)
	}
	for _, p := range newest {
		args = append(args, p.File)
	}
	if err := run.Run(ctx, dir, cmd, args...); err != nil {
		return err
	}
	for _, kind := range []string{"db", "files"} {
		if err := publish(dir, tmp, repoName, kind, signed); err != nil {
			return err
		}
	}
	removeTmp(dir, tmp)

	fmt.Fprintf(out, "==> Database %s.db.tar.zst lists:\n", repoName)
	for _, p := range newest {
		fmt.Fprintf(out, "    %s\n", p.File)
	}
	return nil
}

// publishKey writes the public signing key next to the database (<repo>.asc, see repo.SigningKeyFile), for
// clients to import; the web UI shows its fingerprint. Without a signer it removes an old key file, so the
// UI does not offer a key the database is not signed with.
func publishKey(ctx context.Context, dir, repoName string, signer *Signer) error {
	path := filepath.Join(dir, repoName+".asc")
	if signer == nil {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	key, err := signer.Run.Output(ctx, "", "gpg", "--batch", "--homedir", signer.Home, "--armor", "--export", "--", signer.Fingerprint)
	if err != nil {
		return err
	}
	if len(key) == 0 {
		return fmt.Errorf("gpg exported no public key for %s", signer.Fingerprint)
	}
	tmp := filepath.Join(dir, ".tmp-"+repoName+".asc")
	if err := os.WriteFile(tmp, key, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// publish moves the temporary <kind> archive into place and points the <repo>.<kind> link pacman reads at it.
// A signed archive brings its signature and the link <repo>.<kind>.sig. An unsigned one removes both: pacman
// rejects a database whose signature does not match, also when signatures are optional.
func publish(dir, tmp, repoName, kind string, signed bool) error {
	archive := repoName + "." + kind + ".tar.zst"
	link := repoName + "." + kind
	if signed {
		// The signature first: pacman fetching between the two renames sees a new signature with the old
		// database and fails once, instead of trusting a database without a matching signature.
		if err := os.Rename(filepath.Join(dir, tmp+"."+kind+".tar.zst.sig"), filepath.Join(dir, archive+".sig")); err != nil {
			return fmt.Errorf("publish %s.sig: %w", archive, err)
		}
		if err := replaceLink(dir, link+".sig", archive+".sig"); err != nil {
			return err
		}
	} else {
		for _, f := range []string{archive + ".sig", link + ".sig"} {
			if err := os.Remove(filepath.Join(dir, f)); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	if err := os.Rename(filepath.Join(dir, tmp+"."+kind+".tar.zst"), filepath.Join(dir, archive)); err != nil {
		return fmt.Errorf("publish %s: %w", archive, err)
	}
	return replaceLink(dir, link, archive)
}

// replaceLink points the symlink dir/link at target atomically: created under a temporary name, then
// renamed over the old one.
func replaceLink(dir, link, target string) error {
	path := filepath.Join(dir, link)
	tmpLink := path + ".tmp"
	os.Remove(tmpLink)
	if err := os.Symlink(target, tmpLink); err != nil {
		return fmt.Errorf("link %s: %w", link, err)
	}
	if err := os.Rename(tmpLink, path); err != nil {
		return fmt.Errorf("link %s: %w", link, err)
	}
	return nil
}

// removeTmp deletes leftovers of an interrupted run (repo-add also writes .old backups).
func removeTmp(dir, tmp string) {
	for _, pattern := range []string{tmp + ".db*", tmp + ".files*"} {
		matches, _ := filepath.Glob(filepath.Join(dir, pattern))
		for _, m := range matches {
			os.Remove(m)
		}
	}
}
