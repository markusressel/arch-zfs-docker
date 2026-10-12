package builder

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The relevant parts of the AUR zfs-linux PKGBUILD.
const aurZfsLinuxPkgbuild = `pkgbase="zfs-linux"
pkgname=("zfs-linux" "zfs-linux-headers")
_zfsver="2.4.3"
_kernelver="7.2.8.arch1-1"
_kernelver_full="7.2.8.arch1-1"
_extramodules="${_kernelver_full/.arch/-arch}"

pkgver="${_zfsver}_$(echo ${_kernelver} | sed s/-/./g)"
pkgrel=1
source=("https://github.com/openzfs/zfs/releases/download/zfs-${_zfsver}/zfs-${_zfsver}.tar.gz")
sha256sums=('0000000000000000000000000000000000000000000000000000000000000000')

build() {
    ./configure --prefix=/usr --with-config=kernel \
                --with-linux=/usr/lib/modules/${_extramodules}/build
}
`

// evalPkgbuild sources the PKGBUILD like makepkg does and prints the expression.
func evalPkgbuild(t *testing.T, pkgbuild, expr string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "PKGBUILD")
	os.WriteFile(path, []byte(pkgbuild), 0o644)
	out, err := exec.Command("bash", "-c", `source "$1"; `+expr, "bash", path).CombinedOutput()
	if err != nil {
		t.Fatalf("sourcing PKGBUILD failed: %v: %s", err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestPatchPkgbuildTargetsKernelAndZfsVersion(t *testing.T) {
	pkgbuild, err := PatchPkgbuild(aurZfsLinuxPkgbuild, PkgbuildTarget{ZfsVersion: "2.4.4", KernelVersion: "7.2.9.arch1-1"})
	if err != nil {
		t.Fatal(err)
	}
	if v := evalPkgbuild(t, pkgbuild, `echo "$pkgver $_extramodules"`); v != "2.4.4_7.2.9.arch1.1 7.2.9-arch1-1" {
		t.Errorf("pkgver and module dir = %q", v)
	}
	if strings.Contains(pkgbuild, "--enable-linux-experimental") {
		t.Error("unsupported kernels must only be allowed on request")
	}
}

func TestPatchPkgbuildVerifiesTheReleaseSignature(t *testing.T) {
	pkgbuild, err := PatchPkgbuild(aurZfsLinuxPkgbuild, PkgbuildTarget{ZfsVersion: "2.4.4", KernelVersion: "7.2.9.arch1-1"})
	if err != nil {
		t.Fatal(err)
	}
	sources := evalPkgbuild(t, pkgbuild, `printf '%s\n' "${source[@]}"`)
	want := "https://github.com/openzfs/zfs/releases/download/zfs-2.4.4/zfs-2.4.4.tar.gz\n" +
		"https://github.com/openzfs/zfs/releases/download/zfs-2.4.4/zfs-2.4.4.tar.gz.asc"
	if sources != want {
		t.Errorf("sources:\n%s\nwant:\n%s", sources, want)
	}
	if keys := evalPkgbuild(t, pkgbuild, `echo "${validpgpkeys[@]}"`); keys != strings.Join(OpenZFSSigningKeys, " ") {
		t.Errorf("validpgpkeys = %q", keys)
	}
}

func TestPatchPkgbuildLtsModuleDir(t *testing.T) {
	pkgbuild, err := PatchPkgbuild(aurZfsLinuxPkgbuild, PkgbuildTarget{ZfsVersion: "2.4.4", KernelVersion: "6.18.55-1", Variant: "lts"})
	if err != nil {
		t.Fatal(err)
	}
	if v := evalPkgbuild(t, pkgbuild, `echo "$pkgver $_extramodules"`); v != "2.4.4_6.18.55.1 6.18.55-1-lts" {
		t.Errorf("pkgver and module dir = %q", v)
	}
}

func TestPatchPkgbuildAllowsUnsupportedKernelsOnRequest(t *testing.T) {
	pkgbuild, err := PatchPkgbuild(aurZfsLinuxPkgbuild, PkgbuildTarget{ZfsVersion: "2.4.4", KernelVersion: "7.2.9.arch1-1", AllowUnsupportedKernel: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pkgbuild, "--with-config=kernel --enable-linux-experimental") {
		t.Error("the kernel version check must be skipped")
	}
}

// A changed AUR PKGBUILD must stop the build instead of silently building something else.
func TestPatchPkgbuildFailsOnUnexpectedPkgbuild(t *testing.T) {
	broken := strings.Replace(aurZfsLinuxPkgbuild, "_kernelver=", "_kernel_version=", 1)
	if _, err := PatchPkgbuild(broken, PkgbuildTarget{ZfsVersion: "2.4.4", KernelVersion: "7.2.9.arch1-1"}); err == nil || !strings.Contains(err.Error(), "_kernelver=") {
		t.Errorf("expected a missing _kernelver error, got %v", err)
	}
	noConfigure := strings.Replace(aurZfsLinuxPkgbuild, "--with-config=kernel", "", 1)
	if _, err := PatchPkgbuild(noConfigure, PkgbuildTarget{ZfsVersion: "2.4.4", KernelVersion: "7.2.9.arch1-1", AllowUnsupportedKernel: true}); err == nil {
		t.Error("expected an error when the configure call is not found")
	}
}

func TestUpdatePkgbuildUsesTheInstalledKernel(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "PKGBUILD"), []byte(aurZfsLinuxPkgbuild), 0o644)
	run := &fakeRunner{outputs: map[string]string{"pacman -Q linux-lts": "linux-lts 6.18.55-1\n"}}

	if err := UpdatePkgbuild(context.Background(), run, io.Discard, dir, PkgbuildTarget{ZfsVersion: "2.4.4", Variant: "lts"}); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"pacman", "-Q", "linux-lts"}, {"updpkgsums"}}
	if !reflect.DeepEqual(run.calls, want) {
		t.Errorf("got %v", run.calls)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "PKGBUILD"))
	if !strings.Contains(string(b), `_kernelver="6.18.55-1"`) {
		t.Errorf("kernel version not applied:\n%s", b)
	}

	// Without the kernel installed nothing is changed.
	run = &fakeRunner{outputs: map[string]string{}}
	if err := UpdatePkgbuild(context.Background(), run, io.Discard, t.TempDir(), PkgbuildTarget{ZfsVersion: "2.4.4"}); err == nil {
		t.Error("expected an error")
	}
}

// The embedded keys must be the ones validpgpkeys names, or every build fails signature verification.
func TestOpenZFSKeysMatchValidPgpKeys(t *testing.T) {
	if _, err := exec.LookPath("gpg"); err != nil {
		t.Skip("gpg not available")
	}
	cmd := exec.Command("gpg", "--batch", "--with-colons", "--show-keys")
	cmd.Stdin = strings.NewReader(OpenZFSKeys)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, fp := range OpenZFSSigningKeys {
		if !strings.Contains(string(out), "fpr:::::::::"+fp+":") {
			t.Errorf("openzfs-keys.asc lacks %s", fp)
		}
	}
}
