package k8s

import (
	"context"
	"time"
)

// BuildRequest defines parameters for triggering a new build.
type BuildRequest struct {
	KernelVersion string `json:"kernelVersion"` // specific version, or empty for latest
	Variant       string `json:"variant"`       // "" for normal, "lts" for LTS
	ForceBuild        bool   `json:"forceBuild"`
	ForceRebuildUtils bool   `json:"forceRebuildUtils,omitempty"`
}

// JobSummary represents a build job status in Kubernetes.
type JobSummary struct {
	Name        string     `json:"name"`
	Namespace   string     `json:"namespace"`
	Status      string     `json:"status"` // "Pending", "Running", "Succeeded", "Failed"
	StartTime   *time.Time `json:"startTime,omitempty"`
	EndTime     *time.Time `json:"endTime,omitempty"`
	DurationSec int64      `json:"durationSec"`
	PodName     string     `json:"podName,omitempty"`
}

// K8sClient interface for interacting with cluster builds.
type K8sClient interface {
	ListJobs() ([]JobSummary, error)
	TriggerBuild(req BuildRequest) (*JobSummary, error)
	StreamLogs(ctx context.Context, jobName string) (<-chan string, error)
	SetBuildNode(node string)
}
