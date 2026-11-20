package builder

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecretKeyFingerprint(t *testing.T) {
	colons := "sec:u:255:22:A884D5C0DE99B191:1795163410:::u:::scSC:::+:::ed25519:::0:\n" +
		"fpr:::::::::5B2D584D2A4D2580DC74A8A2A884D5C0DE99B191:\n" +
		"uid:u::::1795163410::X::zfslocal repository signing key::::::::::0:\n" +
		"ssb:u:255:18:1111111111111111:1795163410::::::e:::+:::cv25519::\n" +
		"fpr:::::::::0000000000000000000000001111111111111111:\n"
	if got := secretKeyFingerprint(colons); got != "5B2D584D2A4D2580DC74A8A2A884D5C0DE99B191" {
		t.Errorf("got %q", got)
	}
	if got := secretKeyFingerprint("ssb:u:255:18:1:1::::::e:::+:::cv25519::\nfpr:::::::::ABC:\n"); got != "" {
		t.Errorf("a subkey fingerprint must not count, got %q", got)
	}
}

func TestHandoffRoundTripAndUnsafeNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "handoff", "published")
	if files, err := ReadHandoff(path); err != nil || files != nil {
		t.Fatalf("a missing handoff file means nothing was published, got %v, %v", files, err)
	}
	if err := WriteHandoff(path, []string{"zfs-utils-2.4.4-1-x86_64.pkg.tar.zst", "zfs-linux-2.4.4_7.2.9.arch1.1-1-x86_64.pkg.tar.zst"}); err != nil {
		t.Fatal(err)
	}
	files, err := ReadHandoff(path)
	if err != nil || len(files) != 2 {
		t.Fatalf("got %v, %v", files, err)
	}
	// The build container runs AUR code, so the index step must not sign anything outside the repository.
	if err := WriteHandoff(path, []string{"../../etc/passwd.pkg.tar.zst"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadHandoff(path); err == nil {
		t.Error("an unsafe file name must be refused")
	}
}

// newTestKey creates a signing key without passphrase and returns the armored private key file.
func newTestKey(t *testing.T) string {
	t.Helper()
	home, err := os.MkdirTemp("", "gpg-") // short path: gpg-agent sockets have a length limit
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		exec.Command("gpgconf", "--homedir", home, "--kill", "all").Run()
		os.RemoveAll(home)
	})
	if b, err := exec.Command("gpg", "--batch", "--homedir", home, "--passphrase", "", "--quick-gen-key", "test repository key", "ed25519", "sign", "never").CombinedOutput(); err != nil {
		t.Skipf("gpg cannot create a key here: %v: %s", err, b)
	}
	key, err := exec.Command("gpg", "--batch", "--homedir", home, "--armor", "--export-secret-keys").Output()
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "key.asc")
	if err := os.WriteFile(file, key, 0o400); err != nil {
		t.Fatal(err)
	}
	return file
}

func newTestSigner(t *testing.T) *Signer {
	t.Helper()
	home, err := os.MkdirTemp("", "gpg-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		exec.Command("gpgconf", "--homedir", home, "--kill", "all").Run()
		os.RemoveAll(home)
	})
	signer, err := NewSigner(context.Background(), ExecRunner{Out: io.Discard}, newTestKey(t), home)
	if err != nil {
		t.Fatal(err)
	}
	return signer
}

// With the real gpg and repo-add: the index step signs only what the build handed over, lists only
// packages with a valid signature and signs the database.
func TestIndexRepoSignsPackagesAndDatabase(t *testing.T) {
	requireTools(t, "gpg", "gpgconf", "repo-add", "bsdtar", "tar")
	signer := newTestSigner(t)
	ctx := context.Background()
	run := ExecRunner{Out: io.Discard}

	dir := t.TempDir()
	fakePackage(t, dir, "zfs-utils", "2.4.4")
	fakePackage(t, dir, "zfs-linux", "2.4.4_7.2.9.arch1.1")
	// On the disk, but not handed over by the build: e.g. swapped in through the storage.
	fakePackage(t, dir, "zfs-linux-lts", "2.4.4_6.18.55.1")
	published := []string{"zfs-utils-2.4.4-1-x86_64.pkg.tar.zst", "zfs-linux-2.4.4_7.2.9.arch1.1-1-x86_64.pkg.tar.zst"}

	if err := IndexRepo(ctx, run, io.Discard, dir, "testrepo", signer, published); err != nil {
		t.Fatal(err)
	}

	for _, f := range append(published, "testrepo.db", "testrepo.files") {
		if !signer.Verify(ctx, dir, f) {
			t.Errorf("%s has no valid signature", f)
		}
	}
	entries := dbEntries(t, filepath.Join(dir, "testrepo.db"))
	if !strings.Contains(entries, "zfs-linux-2.4.4_7.2.9.arch1.1-1/desc") || !strings.Contains(entries, "zfs-utils-2.4.4-1/desc") {
		t.Errorf("db is missing the signed packages:\n%s", entries)
	}
	if strings.Contains(entries, "zfs-linux-lts") {
		t.Errorf("db must not list an unsigned package:\n%s", entries)
	}
	if _, err := os.Stat(filepath.Join(dir, "zfs-linux-lts-2.4.4_6.18.55.1-1-x86_64.pkg.tar.zst.sig")); err == nil {
		t.Error("a package the build did not hand over must not be signed")
	}

	// A package swapped after signing loses its place in the database at the next run.
	f, err := os.OpenFile(filepath.Join(dir, published[0]), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("tampered")
	f.Close()
	if err := IndexRepo(ctx, run, io.Discard, dir, "testrepo", signer, nil); err != nil {
		t.Fatal(err)
	}
	if entries := dbEntries(t, filepath.Join(dir, "testrepo.db")); strings.Contains(entries, "zfs-utils") {
		t.Errorf("db must not list a package whose signature no longer matches:\n%s", entries)
	}

	// Unsigned again: the database signatures go, pacman rejects a mismatching one. (Without the
	// tampered file, which is no valid archive for repo-add.)
	os.Remove(filepath.Join(dir, published[0]))
	if err := UpdateRepoDB(ctx, run, io.Discard, dir, "testrepo", nil); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"testrepo.db.sig", "testrepo.db.tar.zst.sig", "testrepo.files.sig"} {
		if _, err := os.Lstat(filepath.Join(dir, f)); err == nil {
			t.Errorf("%s must be removed with an unsigned database", f)
		}
	}
}

func TestIndexRepoSignsAnEmptyDatabase(t *testing.T) {
	requireTools(t, "gpg", "gpgconf", "tar")
	signer := newTestSigner(t)
	dir := t.TempDir()
	if err := IndexRepo(context.Background(), ExecRunner{Out: io.Discard}, io.Discard, dir, "testrepo", signer, nil); err != nil {
		t.Fatal(err)
	}
	if !signer.Verify(context.Background(), dir, "testrepo.db") {
		t.Error("the empty database must be signed too")
	}
}
