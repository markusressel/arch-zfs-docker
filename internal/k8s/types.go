package k8s

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// BuildRequest defines parameters for triggering a new build.
type BuildRequest struct {
	KernelVersion     string `json:"kernelVersion"`        // specific version, or empty for latest
	Variant           string `json:"variant"`              // "" for normal, "lts" for LTS
	ZfsVersion        string `json:"zfsVersion,omitempty"` // OpenZFS release (X.Y.Z), or empty for the AUR default
	ForceBuild        bool   `json:"forceBuild"`
	ForceRebuildUtils bool   `json:"forceRebuildUtils,omitempty"`
}

// JobSummary represents a build job status in Kubernetes.
type JobSummary struct {
	Name          string     `json:"name"`
	Namespace     string     `json:"namespace"`
	Status        string     `json:"status"` // "Pending", "Running", "Succeeded", "Failed"
	StartTime     *time.Time `json:"startTime,omitempty"`
	EndTime       *time.Time `json:"endTime,omitempty"`
	DurationSec   int64      `json:"durationSec"`
	PodName       string     `json:"podName,omitempty"`
	Variant       string     `json:"variant"`
	KernelVersion string     `json:"kernelVersion,omitempty"` // requested version, empty for latest
	ZfsVersion    string     `json:"zfsVersion,omitempty"`
}

// K8sClient interface for interacting with cluster builds.
type K8sClient interface {
	ListJobs() ([]JobSummary, error)
	TriggerBuild(req BuildRequest) (*JobSummary, error)
	DeleteJob(name string) error
	StreamLogs(ctx context.Context, jobName string) (<-chan string, error)
	SetBuildNode(node string)
}

// ErrJobNotFound is returned when a job to delete does not exist.
var ErrJobNotFound = errors.New("job not found")

var jobNameRe = regexp.MustCompile(`^[a-z0-9]([-a-z0-9.]*[a-z0-9])?$`)

func validJobName(name string) bool {
	return len(name) <= 253 && jobNameRe.MatchString(name)
}

// newJobName returns a unique job name. The random suffix keeps jobs created within
// the same second (e.g. by the scheduler for several variants) from colliding.
func newJobName() string {
	suffix := make([]byte, 3)
	_, _ = rand.Read(suffix)
	return fmt.Sprintf("zfs-build-%d-%s", time.Now().Unix(), hex.EncodeToString(suffix))
}
