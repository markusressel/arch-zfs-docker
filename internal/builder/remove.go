package builder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var safeFileRe = regexp.MustCompile(`^[A-Za-z0-9@._+-]+$`)

// ValidatePackageFileName refuses anything but a plain package file name, so a request cannot
// reach outside the repository directory.
func ValidatePackageFileName(file string) error {
	if file == "" || strings.HasPrefix(file, ".") || !safeFileRe.MatchString(file) {
		return fmt.Errorf("refusing unsafe package file name: %q", file)
	}
	for _, ext := range []string{".pkg.tar.zst", ".pkg.tar.xz", ".pkg.tar.gz"} {
		if strings.HasSuffix(file, ext) {
			return nil
		}
	}
	return fmt.Errorf("not a package file: %q", file)
}

// RemovePackage deletes one package file and its signature from the repository directory and
// regenerates the database.
func RemovePackage(ctx context.Context, run Runner, out io.Writer, dir, repoName, file string) error {
	if err := ValidatePackageFileName(file); err != nil {
		return err
	}
	path := filepath.Join(dir, file)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(out, "==> %s does not exist in %s (already removed?), refreshing the database anyway.\n", file, dir)
	} else {
		fmt.Fprintf(out, "==> Removing %s\n", file)
		for _, f := range []string{path, path + ".sig"} {
			if err := os.Remove(f); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	fmt.Fprintf(out, "==> Updating pacman repository database: %s.db.tar.zst\n", repoName)
	if err := UpdateRepoDB(ctx, run, out, dir, repoName); err != nil {
		return err
	}
	fmt.Fprintln(out, "==> Done.")
	return nil
}
