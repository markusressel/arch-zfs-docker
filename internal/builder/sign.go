package builder

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Signer signs packages and the repository database with the repository key, in a keyring of its own.
//
// The key must never be readable where AUR code runs: the build user may use sudo, so it is root in the
// build container. Kubernetes jobs therefore build in one container and sign in another one, which alone
// mounts the key (see docs/development/builder.md).
type Signer struct {
	Run Runner
	// Home is the GnuPG home directory holding the imported key.
	Home string
	// Fingerprint of the signing key.
	Fingerprint string
}

// NewSigner imports the armored private key in keyFile into a new keyring in home.
func NewSigner(ctx context.Context, run Runner, keyFile, home string) (*Signer, error) {
	if err := os.MkdirAll(home, 0o700); err != nil {
		return nil, err
	}
	if err := run.Run(ctx, "", "gpg", "--batch", "--homedir", home, "--import", keyFile); err != nil {
		return nil, fmt.Errorf("import signing key: %w", err)
	}
	out, err := run.Output(ctx, "", "gpg", "--batch", "--homedir", home, "--with-colons", "--list-secret-keys")
	if err != nil {
		return nil, err
	}
	fpr := secretKeyFingerprint(string(out))
	if fpr == "" {
		return nil, fmt.Errorf("%s holds no secret key", keyFile)
	}
	return &Signer{Run: run, Home: home, Fingerprint: fpr}, nil
}

// secretKeyFingerprint returns the fingerprint of the first secret key in `gpg --with-colons` output.
func secretKeyFingerprint(colons string) string {
	inSecretKey := false
	sc := bufio.NewScanner(strings.NewReader(colons))
	for sc.Scan() {
		fields := strings.Split(sc.Text(), ":")
		switch fields[0] {
		case "sec":
			inSecretKey = true
		case "ssb":
			inSecretKey = false
		case "fpr":
			if inSecretKey && len(fields) > 9 && fields[9] != "" {
				return fields[9]
			}
		}
	}
	return ""
}

// Sign writes the detached signature <file>.sig next to a file in dir, replacing an old one.
// Sign is for files the index step wrote itself (the database archives); packages go through SignPublished.
func (s *Signer) Sign(ctx context.Context, dir, file string) error {
	return s.signFile(ctx, dir, file, filepath.Join(dir, file))
}

// signFile signs input and stores the signature as dir/<file>.sig.
func (s *Signer) signFile(ctx context.Context, dir, file, input string) error {
	tmp := filepath.Join(dir, ".tmp-"+file+".sig")
	os.Remove(tmp)
	if err := s.Run.Run(ctx, dir, "gpg", "--batch", "--homedir", s.Home, "--local-user", s.Fingerprint,
		"--detach-sign", "--no-armor", "--output", tmp, "--", input); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, file+".sig"))
}

// Verify reports whether <file>.sig is a valid signature of the file by the repository key (the
// keyring holds no other key).
func (s *Signer) Verify(ctx context.Context, dir, file string) bool {
	if _, err := os.Stat(filepath.Join(dir, file+".sig")); err != nil {
		return false
	}
	return s.Run.Run(ctx, dir, "gpg", "--batch", "--homedir", s.Home, "--verify", "--", file+".sig", file) == nil
}

// Published is a package a build step copied into the repository, with the SHA-256 of what it copied.
type Published struct {
	File   string
	SHA256 string
}

var sha256Re = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ReadHandoff reads the packages a build step published ("<sha256> <file>" per line). A missing file means
// the step published nothing.
func ReadHandoff(path string) ([]Published, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var published []Published
	for _, line := range strings.Split(string(b), "\n") {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}
		sum, file, ok := strings.Cut(line, " ")
		if !ok || !sha256Re.MatchString(sum) {
			return nil, fmt.Errorf("malformed handoff line %q", line)
		}
		// The build container runs AUR code: accept nothing but plain package file names.
		if err := ValidatePackageFileName(file); err != nil {
			return nil, err
		}
		published = append(published, Published{File: file, SHA256: sum})
	}
	return published, nil
}

// WriteHandoff appends published packages for the index step.
func WriteHandoff(path string, published []Published) error {
	if len(published) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, p := range published {
		if _, err := io.WriteString(f, p.SHA256+" "+p.File+"\n"); err != nil {
			return err
		}
	}
	return nil
}

// SignPublished signs a package the build handed over. It copies the package out of the shared volume into
// the signer's own directory, checks it against the SHA-256 the build recorded, and signs that copy. A
// package swapped on the volume after the build is refused; one swapped after the check no longer matches
// its signature, which the database step and pacman check.
func (s *Signer) SignPublished(ctx context.Context, dir string, p Published) error {
	src, err := os.Open(filepath.Join(dir, p.File))
	if err != nil {
		return fmt.Errorf("published package %s: %w", p.File, err)
	}
	defer src.Close()
	private := filepath.Join(s.Home, "package")
	dst, err := os.OpenFile(private, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer os.Remove(private)
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(dst, h), src); err != nil {
		dst.Close()
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != p.SHA256 {
		return fmt.Errorf("%s changed after the build (sha256 %s, the build published %s); not signing it", p.File, got, p.SHA256)
	}
	return s.signFile(ctx, dir, p.File, private)
}

// IndexRepo is the step after a build or removal: it signs the packages the step handed over and
// rebuilds the database. Without a signer it only rebuilds the database, unsigned.
func IndexRepo(ctx context.Context, run Runner, out io.Writer, dir, repoName string, signer *Signer, published []Published) error {
	if signer != nil {
		for _, p := range published {
			fmt.Fprintf(out, "==> Signing %s\n", p.File)
			if err := signer.SignPublished(ctx, dir, p); err != nil {
				return err
			}
		}
	}
	fmt.Fprintf(out, "==> Updating pacman repository database: %s.db.tar.zst\n", repoName)
	return UpdateRepoDB(ctx, run, out, dir, repoName, signer)
}
