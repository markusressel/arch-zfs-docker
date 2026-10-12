package k8s

import "testing"

func TestJobSummaryPredicates(t *testing.T) {
	cases := []struct {
		job                     JobSummary
		active, finished, build bool
	}{
		{JobSummary{Status: StatusPending}, true, false, true},
		{JobSummary{Status: StatusRunning}, true, false, true},
		{JobSummary{Status: StatusSucceeded}, false, true, true},
		{JobSummary{Status: StatusFailed}, false, true, true},
		{JobSummary{Status: StatusRunning, Task: TaskRemovePackage}, true, false, false},
	}
	for _, c := range cases {
		if got := c.job.IsActive(); got != c.active {
			t.Errorf("%+v IsActive() = %v, want %v", c.job, got, c.active)
		}
		if got := c.job.IsFinished(); got != c.finished {
			t.Errorf("%+v IsFinished() = %v, want %v", c.job, got, c.finished)
		}
		if got := c.job.IsBuild(); got != c.build {
			t.Errorf("%+v IsBuild() = %v, want %v", c.job, got, c.build)
		}
	}
}

func TestKubernetesJobStatus(t *testing.T) {
	cases := []struct {
		active, succeeded, failed int
		want                      JobStatus
	}{
		{0, 0, 0, StatusPending},
		{1, 0, 0, StatusRunning},
		// A retry after a failed attempt is still running.
		{1, 0, 1, StatusRunning},
		{0, 1, 1, StatusSucceeded},
		{0, 0, 3, StatusFailed},
	}
	for _, c := range cases {
		if got := kubernetesJobStatus(c.active, c.succeeded, c.failed); got != c.want {
			t.Errorf("kubernetesJobStatus(%d, %d, %d) = %s, want %s", c.active, c.succeeded, c.failed, got, c.want)
		}
	}
}
