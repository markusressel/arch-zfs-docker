package builder

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// Command is the argument that switches the server binary into builder mode, e.g.
// `arch-repo builder update-repo-db <dir> <repo-name>`. Build jobs run the server's own binary, so the
// builder always matches the server that started the job.
const Command = "builder"

const usage = `usage: arch-repo builder <command>

commands:
  build                             build the packages for $VARIANT ($KERNEL_VERSION, $ZFS_VERSION, ...) and
                                    publish them to repository $REPO_NAME; logs to $REPO_ROOT/logs/$JOB_NAME.log
  update-repo-db <dir> <repo-name>  rebuild the pacman database of a repository directory
  update-pkgbuild [flags]           adapt the AUR zfs-linux PKGBUILD in the current directory to the installed
                                    kernel: -zfs-version X.Y.Z [-variant lts] [-allow-unsupported-kernel]
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
	case "build":
		err = buildJob(ctx)
	case "update-pkgbuild":
		err = updatePkgbuildCommand(ctx, args[1:])
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

func updatePkgbuildCommand(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("update-pkgbuild", flag.ContinueOnError)
	var t PkgbuildTarget
	fs.StringVar(&t.ZfsVersion, "zfs-version", "", "OpenZFS release to build (X.Y.Z)")
	fs.StringVar(&t.Variant, "variant", "", `kernel variant: "" for linux, "lts" for linux-lts`)
	fs.BoolVar(&t.AllowUnsupportedKernel, "allow-unsupported-kernel", false, "build even if OpenZFS does not support the kernel")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if t.ZfsVersion == "" {
		return fmt.Errorf("-zfs-version is required")
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	return UpdatePkgbuild(ctx, ExecRunner{Out: os.Stdout}, os.Stdout, dir, t)
}

// buildJob is the entrypoint of a build job.
func buildJob(ctx context.Context) error {
	job := BuildJob{
		JobName:                envOr("JOB_NAME", fmt.Sprintf("build-%d", time.Now().Unix())),
		RepoName:               envOr("REPO_NAME", "zfslocal"),
		Variant:                os.Getenv("VARIANT"),
		KernelVersion:          os.Getenv("KERNEL_VERSION"),
		ZfsVersion:             os.Getenv("ZFS_VERSION"),
		ForceBuild:             os.Getenv("FORCE_BUILD") == "true",
		ForceRebuildUtils:      os.Getenv("FORCE_REBUILD_UTILS") == "true",
		AllowUnsupportedKernel: os.Getenv("ALLOW_UNSUPPORTED_KERNEL") == "true",
	}
	root := envOr("REPO_ROOT", "/repo")
	out, closeLog, err := jobLog(filepath.Join(root, "logs"), job.JobName)
	if err != nil {
		return err
	}
	defer closeLog()

	home := "/home/" + buildUser
	sys := &BuildSystem{
		Root: ExecRunner{Out: out},
		BuildUser: func(uid, gid int) Runner {
			return ExecRunner{Out: out, User: &RunAs{Name: buildUser, UID: uid, GID: gid, Home: home}}
		},
		Out:         out,
		Head:        httpHead,
		LookupUser:  lookupUser,
		RepoRoot:    root,
		PacmanConf:  "/etc/pacman.conf",
		MakepkgConf: "/etc/makepkg.conf",
		SudoersFile: "/etc/sudoers.d/" + buildUser,
		BuildHome:   home,
	}
	if err := job.Run(ctx, sys); err != nil {
		// Into the job log too, where the UI shows it.
		fmt.Fprintf(out, "ERROR: %v\n", err)
		return err
	}
	return nil
}

func lookupUser(name string) (int, int, error) {
	u, err := user.Lookup(name)
	if err != nil {
		return 0, 0, err
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return 0, 0, err
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return 0, 0, err
	}
	return uid, gid, nil
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
