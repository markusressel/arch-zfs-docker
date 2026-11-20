package builder

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
func (s *Signer) Sign(ctx context.Context, dir, file string) error {
	tmp := ".tmp-" + file + ".sig"
	os.Remove(filepath.Join(dir, tmp))
	if err := s.Run.Run(ctx, dir, "gpg", "--batch", "--homedir", s.Home, "--local-user", s.Fingerprint,
		"--detach-sign", "--no-armor", "--output", tmp, "--", file); err != nil {
		os.Remove(filepath.Join(dir, tmp))
		return err
	}
	return os.Rename(filepath.Join(dir, tmp), filepath.Join(dir, file+".sig"))
}

// Verify reports whether <file>.sig is a valid signature of the file by the repository key (the
// keyring holds no other key).
func (s *Signer) Verify(ctx context.Context, dir, file string) bool {
	if _, err := os.Stat(filepath.Join(dir, file+".sig")); err != nil {
		return false
	}
	return s.Run.Run(ctx, dir, "gpg", "--batch", "--homedir", s.Home, "--verify", "--", file+".sig", file) == nil
}

// ReadHandoff reads the package file names a build step published (one per line). A missing file means
// the step published nothing.
func ReadHandoff(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(string(b), "\n") {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}
		// The build container runs AUR code: accept nothing but plain package file names.
		if err := ValidatePackageFileName(line); err != nil {
			return nil, err
		}
		files = append(files, line)
	}
	return files, nil
}

// WriteHandoff appends published package file names for the index step.
func WriteHandoff(path string, files []string) error {
	if len(files) == 0 {
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
	for _, file := range files {
		if _, err := io.WriteString(f, file+"\n"); err != nil {
			return err
		}
	}
	return nil
}

// IndexRepo is the step after a build or removal: it signs the packages the step handed over and
// rebuilds the database. Without a signer it only rebuilds the database, unsigned.
func IndexRepo(ctx context.Context, run Runner, out io.Writer, dir, repoName string, signer *Signer, published []string) error {
	if signer != nil {
		for _, file := range published {
			if _, err := os.Stat(filepath.Join(dir, file)); err != nil {
				return fmt.Errorf("published package %s: %w", file, err)
			}
			fmt.Fprintf(out, "==> Signing %s\n", file)
			if err := signer.Sign(ctx, dir, file); err != nil {
				return err
			}
		}
	}
	fmt.Fprintf(out, "==> Updating pacman repository database: %s.db.tar.zst\n", repoName)
	return UpdateRepoDB(ctx, run, out, dir, repoName, signer)
}
