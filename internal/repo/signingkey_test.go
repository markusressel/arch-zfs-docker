package repo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The fingerprint must match the one gpg shows, which clients pass to pacman-key --lsign-key.
func TestArmoredKeyFingerprintMatchesGpg(t *testing.T) {
	if _, err := exec.LookPath("gpg"); err != nil {
		t.Skip("gpg not available")
	}
	home, err := os.MkdirTemp("", "gpg-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		exec.Command("gpgconf", "--homedir", home, "--kill", "all").Run()
		os.RemoveAll(home)
	})
	for _, algo := range []string{"ed25519", "rsa2048"} {
		if b, err := exec.Command("gpg", "--batch", "--homedir", home, "--passphrase", "", "--quick-gen-key", "test "+algo, algo, "sign", "never").CombinedOutput(); err != nil {
			t.Skipf("gpg cannot create a key here: %v: %s", err, b)
		}
		colons, err := exec.Command("gpg", "--batch", "--homedir", home, "--with-colons", "--list-keys", "test "+algo).Output()
		if err != nil {
			t.Fatal(err)
		}
		var want string
		for _, line := range strings.Split(string(colons), "\n") {
			if f := strings.Split(line, ":"); f[0] == "fpr" {
				want = f[9]
				break
			}
		}
		armored, err := exec.Command("gpg", "--batch", "--homedir", home, "--armor", "--export", want).Output()
		if err != nil {
			t.Fatal(err)
		}
		got, err := ArmoredKeyFingerprint(string(armored))
		if err != nil || got != want {
			t.Errorf("%s: got %q, %v, want %q", algo, got, err, want)
		}

		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, SigningKeyFile("zfslocal")), armored, 0o644); err != nil {
			t.Fatal(err)
		}
		if fpr, ok := SigningKey(dir, "zfslocal"); !ok || fpr != want {
			t.Errorf("%s: SigningKey got %q, %v", algo, fpr, ok)
		}
	}
}

func TestArmoredKeyFingerprintRejectsGarbage(t *testing.T) {
	for _, in := range []string{
		"",
		"not a key",
		"-----BEGIN PGP PUBLIC KEY BLOCK-----\n\n!!!\n-----END PGP PUBLIC KEY BLOCK-----\n",
		"-----BEGIN PGP PUBLIC KEY BLOCK-----\n\nmQ==\n-----END PGP PUBLIC KEY BLOCK-----\n",
		// A signature packet (tag 2) instead of a key.
		"-----BEGIN PGP PUBLIC KEY BLOCK-----\n\niAQEAAAA\n-----END PGP PUBLIC KEY BLOCK-----\n",
	} {
		if fpr, err := ArmoredKeyFingerprint(in); err == nil {
			t.Errorf("%q: expected an error, got %q", in, fpr)
		}
	}
	if _, ok := SigningKey(t.TempDir(), "zfslocal"); ok {
		t.Error("no key file must mean no key")
	}
}
