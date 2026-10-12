package builder

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

// Command is the argument that switches the server binary into builder mode, e.g.
// `arch-repo builder update-repo-db <dir> <repo-name>`. Build jobs run the server's own binary, so the
// builder always matches the server that started the job.
const Command = "builder"

const usage = `usage: arch-repo builder <command>

commands:
  update-repo-db <dir> <repo-name>  rebuild the pacman database of a repository directory
  remove-package                    remove $PACKAGE_FILE from repository $REPO_NAME (in $REPO_ROOT, default /repo)
                                    and rebuild its database; logs to $REPO_ROOT/logs/$JOB_NAME.log
`

// Main runs a builder command and returns the process exit code.
func Main(args []string) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	var err error
	switch args[0] {
	case "update-repo-db":
		if len(args) != 3 {
			fmt.Fprint(os.Stderr, usage)
			return 2
		}
		err = UpdateRepoDB(ctx, ExecRunner{Out: os.Stdout}, os.Stdout, args[1], args[2])
	case "remove-package":
		err = removePackageJob(ctx)
	default:
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	return 0
}

// removePackageJob is the entrypoint of a remove-package job, configured like the build jobs via environment.
func removePackageJob(ctx context.Context) error {
	repoName, file := os.Getenv("REPO_NAME"), os.Getenv("PACKAGE_FILE")
	if repoName == "" || file == "" {
		return fmt.Errorf("REPO_NAME and PACKAGE_FILE are required")
	}
	root := envOr("REPO_ROOT", "/repo")
	jobName := envOr("JOB_NAME", fmt.Sprintf("remove-%d", time.Now().Unix()))

	out, closeLog, err := jobLog(filepath.Join(root, "logs"), jobName)
	if err != nil {
		return err
	}
	defer closeLog()
	err = RemovePackage(ctx, ExecRunner{Out: out}, out, filepath.Join(root, repoName, "x86_64"), repoName, file)
	if err != nil {
		// Into the job log too, where the UI shows it.
		fmt.Fprintf(out, "ERROR: %v\n", err)
	}
	return err
}

// jobLog writes to stdout and appends to the job's log file, which the server keeps after the job is gone.
func jobLog(dir, jobName string) (io.Writer, func(), error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, jobName+".log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}
	return io.MultiWriter(os.Stdout, f), func() { f.Close() }, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
