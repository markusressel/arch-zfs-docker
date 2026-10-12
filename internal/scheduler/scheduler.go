package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/markusressel/arch-zfs-docker/internal/compat"
	"github.com/markusressel/arch-zfs-docker/internal/k8s"
	"github.com/markusressel/arch-zfs-docker/internal/kernel"
	"github.com/markusressel/arch-zfs-docker/internal/repo"
)

// ArchPkgResponse models the response from https://archlinux.org/packages/<repo>/x86_64/<pkg>/json/
type ArchPkgResponse struct {
	PkgName string `json:"pkgname"`
	PkgVer  string `json:"pkgver"`
	PkgRel  string `json:"pkgrel"`
	Epoch   int    `json:"epoch"`
}

// FullVersion returns the standard version-release string e.g. 7.2.8.arch1-2
func (r *ArchPkgResponse) FullVersion() string {
	if r.Epoch > 0 {
		return fmt.Sprintf("%d:%s-%s", r.Epoch, r.PkgVer, r.PkgRel)
	}
	return fmt.Sprintf("%s-%s", r.PkgVer, r.PkgRel)
}

// VariantStatus holds the upstream comparison result for a kernel variant.
type VariantStatus struct {
	Variant        string    `json:"variant"` // "" (standard) or "lts"
	KernelPkg      string    `json:"kernelPkg"`
	UpstreamKernel string    `json:"upstreamKernel"`
	LocalKernel    string    `json:"localKernel"`
	UpToDate       bool      `json:"upToDate"`
	CheckedAt      time.Time `json:"checkedAt"`
	// Compat is set while the variant is not up to date: whether the packaged OpenZFS
	// release can be expected to build against the upstream kernel.
	Compat *compat.Info `json:"compat,omitempty"`
}

// Scheduler periodically queries Arch Linux upstream and triggers build jobs when kernels change.
type Scheduler struct {
	indexer      *repo.Indexer
	k8sClient    k8s.K8sClient
	interval     time.Duration
	intervalChan chan time.Duration
	client       *http.Client

	compat    *compat.Tracker
	mu        sync.RWMutex
	statuses  map[string]*VariantStatus
	lastCheck time.Time
}

// NewScheduler creates a new scheduler.
func NewScheduler(indexer *repo.Indexer, k8sClient k8s.K8sClient, interval time.Duration) *Scheduler {
	return &Scheduler{
		indexer:      indexer,
		k8sClient:    k8sClient,
		interval:     interval,
		intervalChan: make(chan time.Duration, 1),
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		statuses: make(map[string]*VariantStatus),
	}
}

// SetCompat enables kernel/ZFS compatibility tracking.
func (s *Scheduler) SetCompat(t *compat.Tracker) { s.compat = t }

// SetInterval dynamically updates the polling interval.
func (s *Scheduler) SetInterval(interval time.Duration) {
	s.mu.Lock()
	s.interval = interval
	s.mu.Unlock()

	select {
	case s.intervalChan <- interval:
	default:
	}
}

// GetInterval returns the current polling interval.
func (s *Scheduler) GetInterval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.interval
}

// Start begins periodic background checks.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.RLock()
	currentInterval := s.interval
	s.mu.RUnlock()

	if currentInterval > 0 {
		log.Printf("[scheduler] Starting upstream kernel checker (interval: %v)\n", currentInterval)
		// Run initial check in background after 5 seconds to let server start up cleanly
		go func() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				s.CheckAllAndTrigger()
			}
		}()
	} else {
		log.Println("[scheduler] Auto-checking initially disabled (interval <= 0)")
	}

	go func() {
		var ticker *time.Ticker
		var tickerC <-chan time.Time

		if currentInterval > 0 {
			ticker = time.NewTicker(currentInterval)
			tickerC = ticker.C
		}

		defer func() {
			if ticker != nil {
				ticker.Stop()
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case newInterval := <-s.intervalChan:
				if ticker != nil {
					ticker.Stop()
					ticker = nil
					tickerC = nil
				}
				if newInterval > 0 {
					ticker = time.NewTicker(newInterval)
					tickerC = ticker.C
					log.Printf("[scheduler] Auto-check interval updated to %v\n", newInterval)
				} else {
					log.Println("[scheduler] Auto-check disabled (interval <= 0)")
				}
			case <-tickerC:
				s.CheckAllAndTrigger()
			}
		}
	}()
}

// GetStatus returns the latest check status for all variants.
// The local repository state and the compatibility verdict are re-evaluated on every call, so a
// finished (or failed) build shows up without waiting for the next upstream poll.
func (s *Scheduler) GetStatus() map[string]*VariantStatus {
	s.mu.RLock()
	out := make(map[string]*VariantStatus, len(s.statuses))
	for k, v := range s.statuses {
		cp := *v
		out[k] = &cp
	}
	s.mu.RUnlock()

	var pkgs []repo.PackageInfo
	haveRepo := false
	if len(out) > 0 {
		if summary, err := s.indexer.GetSummary("x86_64"); err == nil {
			pkgs, haveRepo = summary.Packages, true
		}
	}

	for _, st := range out {
		if haveRepo {
			st.LocalKernel, st.UpToDate = localKernelState(pkgs, st.KernelPkg, st.UpstreamKernel)
		}
		st.Compat = nil
		if !st.UpToDate && s.compat != nil {
			info := s.compat.Assess(st.UpstreamKernel)
			st.Compat = &info
		}
	}
	return out
}

// FetchUpstreamKernel queries the Arch Linux official package API.
func (s *Scheduler) FetchUpstreamKernel(kernelPkg string) (*ArchPkgResponse, error) {
	url := fmt.Sprintf("https://archlinux.org/packages/core/x86_64/%s/json/", kernelPkg)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "arch-zfs-docker-scheduler/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream API error (%d) for %s", resp.StatusCode, kernelPkg)
	}

	var pkgResp ArchPkgResponse
	if err := json.NewDecoder(resp.Body).Decode(&pkgResp); err != nil {
		return nil, fmt.Errorf("decode upstream json: %w", err)
	}

	return &pkgResp, nil
}

// CheckVariant checks if the local repository has packages for the latest upstream kernel.
func (s *Scheduler) CheckVariant(variant string) (*VariantStatus, bool, error) {
	kernelPkg := "linux"
	if variant != "" {
		kernelPkg = fmt.Sprintf("linux-%s", variant)
	}

	upstream, err := s.FetchUpstreamKernel(kernelPkg)
	if err != nil {
		return nil, false, err
	}

	upstreamVer := upstream.FullVersion()

	// Get local packages
	summary, err := s.indexer.GetSummary("x86_64")
	if err != nil {
		return nil, false, fmt.Errorf("get local repo summary: %w", err)
	}

	latestLocalKernel, hasMatch := localKernelState(summary.Packages, kernelPkg, upstreamVer)

	status := &VariantStatus{
		Variant:        variant,
		KernelPkg:      kernelPkg,
		UpstreamKernel: upstreamVer,
		LocalKernel:    latestLocalKernel,
		UpToDate:       hasMatch,
		CheckedAt:      time.Now(),
	}

	s.mu.Lock()
	s.statuses[kernelPkg] = status
	s.mu.Unlock()

	needsBuild := !hasMatch
	if needsBuild && s.compat != nil {
		info := s.compat.Assess(upstreamVer)
		status.Compat = &info
	}
	return status, needsBuild, nil
}

// localKernelState returns the newest kernel version built for kernelPkg and
// whether any build matches upstreamVer. Only this variant's main package counts
// ("zfs-linux" must not pick up zfs-linux-lts), as that is what the build script
// checks before skipping a build.
func localKernelState(pkgs []repo.PackageInfo, kernelPkg, upstreamVer string) (latest string, match bool) {
	targetPkg := "zfs-" + kernelPkg
	for _, p := range pkgs {
		if p.KernelVersion == "" || p.PackageName != targetPkg {
			continue
		}
		if latest == "" || kernel.Compare(kernel.Dotted(p.KernelVersion), kernel.Dotted(latest)) > 0 {
			latest = p.KernelVersion
		}
		if kernel.Matches(p.KernelVersion, upstreamVer) {
			match = true
		}
	}
	return latest, match
}

// hasActiveJob reports whether a build for the kernel variant is pending or running
// (maintenance tasks such as package removal do not count).
func hasActiveJob(jobs []k8s.JobSummary, variant string) bool {
	for _, j := range jobs {
		if j.IsBuild() && j.IsActive() && j.Variant == variant {
			return true
		}
	}
	return false
}

// CheckAllAndTrigger checks upstream kernels for all variants and triggers build jobs when needed.
func (s *Scheduler) CheckAllAndTrigger() {
	variants := []string{"", "lts"}

	// If the job list is unavailable, building anyway beats silently never building.
	activeJobs, err := s.k8sClient.ListJobs()
	if err != nil {
		log.Printf("[scheduler] Could not list build jobs, duplicate builds possible: %v\n", err)
	}

	if s.compat != nil {
		s.compat.Observe(activeJobs)
	}

	for _, v := range variants {
		vName := v
		if vName == "" {
			vName = "default (standard linux)"
		}

		status, needsBuild, err := s.CheckVariant(v)
		if err != nil {
			log.Printf("[scheduler] Error checking variant %s: %v\n", vName, err)
			continue
		}

		if needsBuild && status.Compat != nil && status.Compat.State == compat.StateIncompatible {
			log.Printf("[scheduler] Not building %s %s: %s (zfs %s, build %s). Retrying once a new OpenZFS release is packaged.\n",
				status.KernelPkg, status.UpstreamKernel, status.Compat.Reason, status.Compat.ZfsVersion, status.Compat.Job)
		} else if needsBuild && hasActiveJob(activeJobs, v) {
			log.Printf("[scheduler] Build for %s is already running, not starting another one (upstream: %s)\n", status.KernelPkg, status.UpstreamKernel)
		} else if needsBuild {
			log.Printf("[scheduler] NEW KERNEL DETECTED for %s! Upstream: %s (local: %s). Triggering build job...\n",
				status.KernelPkg, status.UpstreamKernel, status.LocalKernel)

			job, err := s.k8sClient.TriggerBuild(k8s.BuildRequest{
				Variant:    v,
				ForceBuild: false,
			})
			if err != nil {
				log.Printf("[scheduler] Failed to trigger build job for %s: %v\n", status.KernelPkg, err)
			} else {
				log.Printf("[scheduler] Successfully triggered build job %s for %s\n", job.Name, status.KernelPkg)
			}
		} else {
			log.Printf("[scheduler] Variant %s is up-to-date (kernel: %s)\n", status.KernelPkg, status.UpstreamKernel)
		}
	}
}
