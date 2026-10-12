// Package compat tracks whether the OpenZFS release that gets packaged supports a given
// Linux kernel: predicted from the release's META file and learned from failed builds.
package compat

import "time"

// State is the compatibility verdict for a (kernel, ZFS version) pair.
type State string

const (
	// StateOK: the kernel is within the Linux-Maximum of the packaged ZFS release.
	StateOK State = "ok"
	// StatePredictedUnsupported: the kernel is newer than the release's Linux-Maximum. A build is
	// still attempted, the maximum is only "tested up to".
	StatePredictedUnsupported State = "predicted_unsupported"
	// StateIncompatible: a build of this exact pair failed because of the kernel.
	StateIncompatible State = "incompatible"
	// StateUnknown: the sources could not be queried.
	StateUnknown State = "unknown"
)

// Issue is a related OpenZFS GitHub issue.
type Issue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	State  string `json:"state"` // "open" | "closed"
}

// Info describes the compatibility of the upstream kernel with the packaged ZFS release.
type Info struct {
	State      State  `json:"state"`
	ZfsVersion string `json:"zfsVersion,omitempty"`
	// LinuxMax is the release's Linux-Maximum (major.minor), when known.
	LinuxMax string `json:"linuxMax,omitempty"`
	Reason   string `json:"reason,omitempty"`
	// Job is the failed build job the verdict was learned from.
	Job   string     `json:"job,omitempty"`
	Since *time.Time `json:"since,omitempty"`
	// Issues and IssuesURL are hints for non-OK states.
	Issues    []Issue `json:"issues,omitempty"`
	IssuesURL string  `json:"issuesUrl,omitempty"`
}
