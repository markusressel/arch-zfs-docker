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

	requestedKernelRe = regexp.MustCompile(`(?m)^==> Requested \S+ version: (\S+)\s*$`)
	latestKernelRe    = regexp.MustCompile(`(?m)^==> Latest available \S+ in official repos: (\S+)\s*$`)
	zfsVersionRe      = regexp.MustCompile(`(?m)^==> (?:Using|Requested) OpenZFS version: (\d+\.\d+\.\d+)`)
)

// Verdict is the result of classifying a failed build log.
type Verdict struct {
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
		return Verdict{Reason: "build failed before the kernel module build"}
	}
	phase := log[i:]

	if m := configureRe.FindString(phase); m != "" {
		return Verdict{Incompatible: true, Reason: "OpenZFS does not support this kernel version", Evidence: trimLine(m)}
	}
	if m := compileRe.FindString(phase); m != "" {
		return Verdict{Incompatible: true, Reason: "the ZFS kernel module does not compile against this kernel", Evidence: trimLine(m)}
	}
	return Verdict{Reason: "build failed for a reason unrelated to the kernel version"}
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
