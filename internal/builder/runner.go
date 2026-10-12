package builder

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Runner runs the external tools the builder relies on (repo-add, tar, pacman, makepkg, ...).
// Tests replace it to check which commands would run, without touching the system.
type Runner interface {
	Run(ctx context.Context, dir string, name string, args ...string) error
}

// ExecRunner runs commands for real; their output goes to Out.
type ExecRunner struct {
	Out io.Writer
}

func (r ExecRunner) Run(ctx context.Context, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = r.Out
	cmd.Stderr = r.Out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}
