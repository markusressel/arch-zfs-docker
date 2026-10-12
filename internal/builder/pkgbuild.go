package builder

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// OpenZFSSigningKeys are the fingerprints of the keys that sign OpenZFS releases (the validpgpkeys of the
// AUR zfs-utils PKGBUILD); openzfs-keys.asc holds the keys themselves.
var OpenZFSSigningKeys = []string{
	"4F3BA9AB6D1F8D683DC2DFB56AD860EED4598027", // Tony Hutter (GPG key for signing ZFS releases)
	"C33DF142657ED1F7C328A2960AB9E991C6AF658B", // Brian Behlendorf
}

// PkgbuildTarget is what the AUR zfs-linux(-lts) PKGBUILD is adapted to.
type PkgbuildTarget struct {
	// ZfsVersion is the OpenZFS release, e.g. "2.4.4".
	ZfsVersion string
	// KernelVersion is the installed kernel package version, e.g. "7.2.9.arch1-1" or "6.18.55-1".
	KernelVersion string
	// Variant is "" for linux, "lts" for linux-lts.
	Variant string
	// AllowUnsupportedKernel skips OpenZFS' check for the newest supported kernel (--enable-linux-experimental).
	AllowUnsupportedKernel bool
}

// KernelPackage is the Arch kernel package of a variant: "linux" or "linux-<variant>".
func KernelPackage(variant string) string {
	if variant == "" {
		return "linux"
	}
	return "linux-" + variant
}

// PatchPkgbuild adapts the AUR zfs-linux PKGBUILD to the target. It fails when a line it has to change is
// missing, so a changed AUR PKGBUILD stops the build instead of silently building the wrong version.
func PatchPkgbuild(pkgbuild string, t PkgbuildTarget) (string, error) {
	assignments := map[string]string{
		"_zfsver":         t.ZfsVersion,
		"_kernelver":      `"` + t.KernelVersion + `"`,
		"_kernelver_full": `"` + t.KernelVersion + `"`,
	}
	// linux-lts installs its modules into /usr/lib/modules/<version>-lts.
	if t.Variant == "lts" {
		assignments["_extramodules"] = `"` + t.KernelVersion + `-lts"`
	}
	for name, value := range assignments {
		re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(name) + `=.*$`)
		if !re.MatchString(pkgbuild) {
			return "", fmt.Errorf("PKGBUILD has no %s= line", name)
		}
		pkgbuild = re.ReplaceAllLiteralString(pkgbuild, name+"="+value)
	}

	// OpenZFS refuses kernels newer than its META Linux-Maximum. Only an explicit request skips that check:
	// such modules are untested by OpenZFS, and pacman would install them on every client.
	if t.AllowUnsupportedKernel {
		if !strings.Contains(pkgbuild, "--with-config=kernel") {
			return "", fmt.Errorf("PKGBUILD does not configure with --with-config=kernel")
		}
		pkgbuild = strings.ReplaceAll(pkgbuild, "--with-config=kernel", "--with-config=kernel --enable-linux-experimental")
	}

	// The AUR PKGBUILD pins only the tarball's checksum, which has to be recomputed for other OpenZFS versions.
	// Verify the release signature instead, like the zfs-utils PKGBUILD does.
	var keys []string
	for _, k := range OpenZFSSigningKeys {
		keys = append(keys, "'"+k+"'")
	}
	pkgbuild = strings.TrimRight(pkgbuild, "\n") + "\n\n" +
		"# Added by arch-repo: verify the OpenZFS release signature.\n" +
		`source+=("${source[0]}.asc")` + "\n" +
		"validpgpkeys=(" + strings.Join(keys, " ") + ")\n"
	return pkgbuild, nil
}

// UpdatePkgbuild adapts the PKGBUILD in dir to the installed kernel of the variant and the OpenZFS version,
// then recomputes its checksums (updpkgsums downloads the sources).
func UpdatePkgbuild(ctx context.Context, run Runner, out io.Writer, dir string, t PkgbuildTarget) error {
	kernelPkg := KernelPackage(t.Variant)
	q, err := run.Output(ctx, dir, "pacman", "-Q", kernelPkg)
	if err != nil {
		return fmt.Errorf("installed version of %s: %w", kernelPkg, err)
	}
	// "linux 7.2.9.arch1-1"
	fields := strings.Fields(string(q))
	if len(fields) != 2 || fields[0] != kernelPkg {
		return fmt.Errorf("unexpected pacman -Q output: %q", q)
	}
	t.KernelVersion = fields[1]

	fmt.Fprintln(out, "==> Updating PKGBUILD:")
	fmt.Fprintf(out, "    Kernel Package: %s\n", kernelPkg)
	fmt.Fprintf(out, "    Kernel Version: %s\n", t.KernelVersion)
	fmt.Fprintf(out, "    ZFS pkgver:     %s\n", t.ZfsVersion)
	if t.AllowUnsupportedKernel {
		fmt.Fprintln(out, "==> WARNING: building even if OpenZFS does not support this kernel (--enable-linux-experimental)")
	}

	path := filepath.Join(dir, "PKGBUILD")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	patched, err := PatchPkgbuild(string(b), t)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(patched), 0o644); err != nil {
		return err
	}
	// Recompute the checksums for the new version; makepkg then checks the tarball against the signature.
	if err := run.Run(ctx, dir, "updpkgsums"); err != nil {
		return err
	}
	fmt.Fprintln(out, "==> PKGBUILD updated successfully.")
	return nil
}
