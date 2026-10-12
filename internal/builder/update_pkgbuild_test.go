package builder

import (
	"os"
	"os/exec"
	"path/filepath"
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

// runUpdatePkgbuild runs update_pkgbuild.sh on the AUR PKGBUILD with pacman and updpkgsums stubbed out.
func runUpdatePkgbuild(t *testing.T, allowUnsupported string) string {
	t.Helper()
	bin := t.TempDir()
	os.WriteFile(filepath.Join(bin, "pacman"), []byte("#!/bin/sh\necho \"linux 7.2.9.arch1-1\"\n"), 0o755)
	os.WriteFile(filepath.Join(bin, "updpkgsums"), []byte("#!/bin/sh\n"), 0o755)

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "PKGBUILD"), []byte(aurZfsLinuxPkgbuild), 0o644)
	script := filepath.Join(t.TempDir(), "update_pkgbuild.sh")
	os.WriteFile(script, []byte(UpdatePkgbuildScript), 0o755)

	cmd := exec.Command("bash", script, "2.4.4", "unused")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "VARIANT=", "ALLOW_UNSUPPORTED_KERNEL="+allowUnsupported)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("update_pkgbuild.sh failed: %v: %s", err, out)
	}
	b, err := os.ReadFile(filepath.Join(dir, "PKGBUILD"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

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

func TestUpdatePkgbuildVerifiesTheReleaseSignature(t *testing.T) {
	pkgbuild := runUpdatePkgbuild(t, "false")

	sources := evalPkgbuild(t, pkgbuild, `printf '%s\n' "${source[@]}"`)
	want := "https://github.com/openzfs/zfs/releases/download/zfs-2.4.4/zfs-2.4.4.tar.gz\n" +
		"https://github.com/openzfs/zfs/releases/download/zfs-2.4.4/zfs-2.4.4.tar.gz.asc"
	if sources != want {
		t.Errorf("sources:\n%s\nwant:\n%s", sources, want)
	}
	keys := evalPkgbuild(t, pkgbuild, `echo "${validpgpkeys[@]}"`)
	if keys != "4F3BA9AB6D1F8D683DC2DFB56AD860EED4598027 C33DF142657ED1F7C328A2960AB9E991C6AF658B" {
		t.Errorf("validpgpkeys = %q", keys)
	}
	if v := evalPkgbuild(t, pkgbuild, `echo "$pkgver"`); v != "2.4.4_7.2.9.arch1.1" {
		t.Errorf("pkgver = %q", v)
	}
}

func TestUpdatePkgbuildKeepsTheKernelCheckByDefault(t *testing.T) {
	if strings.Contains(runUpdatePkgbuild(t, "false"), "--enable-linux-experimental") {
		t.Error("unsupported kernels must only be allowed on request")
	}
	if !strings.Contains(runUpdatePkgbuild(t, "true"), "--with-config=kernel --enable-linux-experimental") {
		t.Error("ALLOW_UNSUPPORTED_KERNEL=true must skip the kernel version check")
	}
}

func TestBuildScriptChecksSourceSignatures(t *testing.T) {
	if strings.Contains(CheckAndBuildScript, "--skippgpcheck") {
		t.Error("makepkg must verify the OpenZFS release signature")
	}
	for _, fp := range []string{"4F3BA9AB6D1F8D683DC2DFB56AD860EED4598027", "C33DF142657ED1F7C328A2960AB9E991C6AF658B"} {
		if !strings.Contains(UpdatePkgbuildScript, fp) {
			t.Errorf("update_pkgbuild.sh does not trust %s", fp)
		}
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
	for _, fp := range []string{"4F3BA9AB6D1F8D683DC2DFB56AD860EED4598027", "C33DF142657ED1F7C328A2960AB9E991C6AF658B"} {
		if !strings.Contains(string(out), "fpr:::::::::"+fp+":") {
			t.Errorf("openzfs-keys.asc lacks %s", fp)
		}
	}
}
