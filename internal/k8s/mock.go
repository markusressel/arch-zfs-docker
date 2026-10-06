package k8s

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockClient implements K8sClient for local development or testing.
type MockClient struct {
	mu   sync.Mutex
	jobs []JobSummary
}

// NewMockClient creates a mock K8s client with sample jobs.
func NewMockClient() *MockClient {
	now := time.Now()
	past := now.Add(-10 * time.Minute)
	finished := past.Add(4 * time.Minute)

	return &MockClient{
		jobs: []JobSummary{
			{
				Name:        "zfs-repo-builder-scheduled-demo",
				Namespace:   "arch-repo",
				Status:      "Succeeded",
				StartTime:   &past,
				EndTime:     &finished,
				DurationSec: 240,
			},
		},
	}
}

func (m *MockClient) ListJobs() ([]JobSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([]JobSummary, len(m.jobs))
	copy(copied, m.jobs)
	return copied, nil
}

func (m *MockClient) TriggerBuild(req BuildRequest) (*JobSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	name := fmt.Sprintf("zfs-build-mock-%d", now.Unix())
	job := JobSummary{
		Name:          name,
		Namespace:     "arch-repo",
		Status:        "Running",
		StartTime:     &now,
		Variant:       req.Variant,
		KernelVersion: req.KernelVersion,
	}

	m.jobs = append([]JobSummary{job}, m.jobs...)
	return &job, nil
}

func (m *MockClient) DeleteJob(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, j := range m.jobs {
		if j.Name == name {
			m.jobs = append(m.jobs[:i], m.jobs[i+1:]...)
			return nil
		}
	}
	return ErrJobNotFound
}

func (m *MockClient) StreamLogs(ctx context.Context, jobName string) (<-chan string, error) {
	lines := make(chan string, 10)
	go func() {
		defer close(lines)
		logs := []string{
			"==> Initializing mock build environment...",
			"==> Checking latest Arch kernel...",
			"==> Target kernel: 7.2.8.arch1-2",
			"==> Cloning zfs-utils from AUR...",
			"==> Building zfs-utils with makepkg...",
			"==> Cloning zfs-linux from AUR...",
			"==> Compiling zfs.ko module...",
			"==> Updating pacman repository database: zfslocal.db.tar.zst...",
			"==> Repository updated successfully!",
		}
		for _, l := range logs {
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
				lines <- l
			}
		}
	}()
	return lines, nil
}

func (m *MockClient) SetBuildNode(node string) {
	// mock client no-op
}
