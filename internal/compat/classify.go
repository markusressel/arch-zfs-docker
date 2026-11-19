package compat

import (
	"regexp"
	"strings"
)

// buildPhaseMarker is printed by check_and_build.sh right before the kernel module is built.
const buildPhaseMarker = "==> 2/2: Building zfs-linux"

var (
	// ZFS' configure step refuses kernels outside of META's Linux-Minimum/Linux-Maximum.
	configureRe = regexp.MustCompile(`(?im)^.*(maximum supported kernel|linux-maximum|kernel version[^\n]{0,60}(not supported|unsupported)|(not supported|unsupported)[^\n]{0,40}kernel version|cannot build against).*$`)
	// gcc/clang diagnostics from compiling the module against kernel headers.
	compileRe = regexp.MustCompile(`(?m)^\S+\.[ch]:\d+(:\d+)?: (fatal )?error:.*$`)

	// The error a failed command left in the log: the builder's own "ERROR: ..." lines, pacman's
	// "error: ..." and makepkg's "==> ERROR: ...".
	errorLineRe = regexp.MustCompile(`(?m)^(==> )?(ERROR|error):.*$`)

	requestedKernelRe = regexp.MustCompile(`(?m)^==> Requested \S+ version: (\S+)\s*$`)
	latestKernelRe    = regexp.MustCompile(`(?m)^==> Latest available \S+ in official repos: (\S+)\s*$`)
	zfsVersionRe      = regexp.MustCompile(`(?m)^==> (?:Using|Requested) OpenZFS version: (\d+\.\d+\.\d+)`)
)

// Category says which part of a build failed.
type Category string

const (
	// CategoryEnvironment: the build failed before the kernel module build started (pacman,
	// keyring, network, the zfs-utils package). The kernel version cannot be the reason.
	CategoryEnvironment Category = "environment"
	// CategoryKernel: OpenZFS does not support or does not compile against the kernel.
	CategoryKernel Category = "kernel"
	// CategoryBuild: the kernel module build failed for another reason (downloads, signatures, ...).
	CategoryBuild Category = "build"
)

// Verdict is the result of classifying a failed build log.
type Verdict struct {
	Category Category
	// Incompatible is true when the failure was caused by the kernel version, not by
	// infrastructure (network, pacman, ...).
	Incompatible bool
	Reason       string
	// Evidence is the log line the verdict is based on.
	Evidence string
}

// Classify decides whether a failed build log shows a kernel incompatibility. Only failures
// that happen after the kernel module build started are considered.
func Classify(log string) Verdict {
	i := strings.Index(log, buildPhaseMarker)
	if i < 0 {
		return Verdict{Category: CategoryEnvironment, Reason: "the build environment could not be prepared, before the kernel module build", Evidence: firstError(log)}
	}
	phase := log[i:]

	if m := configureRe.FindString(phase); m != "" {
		return Verdict{Category: CategoryKernel, Incompatible: true, Reason: "OpenZFS does not support this kernel version", Evidence: trimLine(m)}
	}
	if m := compileRe.FindString(phase); m != "" {
		return Verdict{Category: CategoryKernel, Incompatible: true, Reason: "the ZFS kernel module does not compile against this kernel", Evidence: trimLine(m)}
	}
	return Verdict{Category: CategoryBuild, Reason: "the kernel module build failed for a reason unrelated to the kernel version", Evidence: firstError(phase)}
}

// firstError returns the first error line of a log, "" when there is none. The first one is the
// cause: a job retries in the same log, and later attempts often fail differently (killed by the
// job's backoff limit, for example).
func firstError(log string) string {
	return trimLine(errorLineRe.FindString(log))
}

// ParseVersions extracts the kernel and ZFS version a build used from its log ("" when not printed).
func ParseVersions(log string) (kernelVersion, zfsVersion string) {
	if m := requestedKernelRe.FindStringSubmatch(log); m != nil {
		kernelVersion = m[1]
	} else if m := latestKernelRe.FindStringSubmatch(log); m != nil {
		kernelVersion = m[1]
	}
	if m := zfsVersionRe.FindStringSubmatch(log); m != nil {
		zfsVersion = m[1]
	}
	return kernelVersion, zfsVersion
}

func trimLine(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 300 {
		s = s[:300] + "…"
	}
	return s
}
