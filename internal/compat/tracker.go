package compat

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/markusressel/arch-zfs-docker/internal/k8s"
	"github.com/markusressel/arch-zfs-docker/internal/kernel"
)

// dotted returns the kernel version in package file name form, without epoch.
func dotted(v string) string {
	return kernel.Dotted(kernel.StripEpoch(v))
}

// Tracker learns from failed builds and predicts compatibility of the upstream kernel.
type Tracker struct {
	store   *Store
	sources *Sources
	logsDir string
}

// NewTracker creates a tracker keeping its state in repoDir/compat.json and reading
// build logs from repoDir/logs.
func NewTracker(repoDir string) *Tracker {
	return &Tracker{
		store:   NewStore(filepath.Join(repoDir, "compat.json")),
		sources: NewSources(),
		logsDir: filepath.Join(repoDir, "logs"),
	}
}

// Observe classifies failed jobs that were not seen before and remembers kernel incompatibilities.
// The job list is polled constantly, so the store is only written when a job was classified.
func (t *Tracker) Observe(jobs []k8s.JobSummary) {
	changed := false
	for _, j := range jobs {
		if j.Status != k8s.StatusFailed || !t.store.claim(j.Name) {
			continue
		}
		if t.observe(j) {
			changed = true
		} else {
			t.store.release(j.Name) // try again later (log not written yet, AUR down, ...)
		}
	}
	if changed {
		t.store.persist()
	}
}

// observe returns false when the job should be looked at again later.
func (t *Tracker) observe(j k8s.JobSummary) bool {
	b, err := os.ReadFile(filepath.Join(t.logsDir, j.Name+".log"))
	if err != nil {
		return false
	}
	text := string(b)

	verdict := Classify(text)
	if !verdict.Incompatible {
		return true
	}

	kernelVer, zfsVer := ParseVersions(text)
	if kernelVer == "" {
		kernelVer = j.KernelVersion
	}
	if zfsVer == "" {
		zfsVer = j.ZfsVersion
	}
	if kernelVer == "" {
		return true // cannot attribute the failure to a kernel
	}
	if zfsVer == "" {
		if zfsVer, err = t.sources.AurVersion(); err != nil {
			return false
		}
	}

	at := time.Now()
	if j.EndTime != nil {
		at = *j.EndTime
	}
	t.store.record(Entry{Kernel: dotted(kernelVer), ZfsVersion: zfsVer, Reason: verdict.Reason, Evidence: verdict.Evidence, Job: j.Name, At: at})
	log.Printf("[compat] Build %s failed: %s (kernel %s, zfs %s)\n", j.Name, verdict.Reason, kernelVer, zfsVer)
	return true
}

// Assess returns the compatibility of the given kernel version with the ZFS release that
// the next build would package.
func (t *Tracker) Assess(kernelVersion string) Info {
	zfsVer, err := t.sources.AurVersion()
	if err != nil {
		return Info{State: StateUnknown, Reason: "could not determine the packaged ZFS version: " + err.Error()}
	}
	info := Info{State: StateOK, ZfsVersion: zfsVer}

	if e, ok := t.store.lookup(dotted(kernelVersion), zfsVer); ok {
		info.State = StateIncompatible
		info.Reason = e.Reason
		info.Job = e.Job
		since := e.At
		info.Since = &since
	}

	if max, err := t.sources.LinuxMax(zfsVer); err == nil {
		info.LinuxMax = max
		if info.State == StateOK {
			kMaj, kMin, ok1 := majorMinor(kernel.Normalize(kernelVersion))
			mMaj, mMin, ok2 := majorMinor(max)
			if ok1 && ok2 && (kMaj > mMaj || (kMaj == mMaj && kMin > mMin)) {
				info.State = StatePredictedUnsupported
				info.Reason = fmt.Sprintf("OpenZFS %s declares support up to Linux %s", zfsVer, max)
			}
		}
	}

	if info.State == StateIncompatible || info.State == StatePredictedUnsupported {
		if maj, min, ok := majorMinor(kernel.Normalize(kernelVersion)); ok {
			issues, page, _ := t.sources.Issues(fmt.Sprintf("%d.%d", maj, min))
			info.Issues, info.IssuesURL = issues, page
		}
	}
	return info
}
