package builder

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// Runner runs the external tools the builder relies on (repo-add, tar, pacman, makepkg, ...).
// Tests replace it to check which commands would run, without touching the system.
type Runner interface {
	Run(ctx context.Context, dir string, name string, args ...string) error
	// Output runs a command and returns its standard output.
	Output(ctx context.Context, dir string, name string, args ...string) ([]byte, error)
}

// ExecRunner runs commands for real; their output goes to Out.
type ExecRunner struct {
	Out io.Writer
	// User, when set, runs the commands as that user (the process must be root).
	User *RunAs
}

// RunAs is the user commands run as, with its home directory.
type RunAs struct {
	Name     string
	UID, GID int
	Home     string
}

func (r ExecRunner) command(ctx context.Context, dir, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if u := r.User; u != nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(u.UID), Gid: uint32(u.GID)}}
		// The user's own HOME: gpg keeps its keyring there, makepkg its configuration.
		env := []string{"HOME=" + u.Home, "USER=" + u.Name, "LOGNAME=" + u.Name}
		for _, e := range os.Environ() {
			if !strings.HasPrefix(e, "HOME=") && !strings.HasPrefix(e, "USER=") && !strings.HasPrefix(e, "LOGNAME=") {
				env = append(env, e)
			}
		}
		cmd.Env = env
	}
	return cmd
}

func (r ExecRunner) Run(ctx context.Context, dir, name string, args ...string) error {
	cmd := r.command(ctx, dir, name, args...)
	cmd.Stdout = r.Out
	cmd.Stderr = r.Out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func (r ExecRunner) Output(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := r.command(ctx, dir, name, args...)
	cmd.Stderr = r.Out
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}
