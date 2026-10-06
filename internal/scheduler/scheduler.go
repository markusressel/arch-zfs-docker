package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/markusressel/arch-zfs-docker/internal/k8s"
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
}

// Scheduler periodically queries Arch Linux upstream and triggers build jobs when kernels change.
type Scheduler struct {
	indexer   *repo.Indexer
	k8sClient k8s.K8sClient
	interval  time.Duration
	client    *http.Client

	mu        sync.RWMutex
	statuses  map[string]*VariantStatus
	lastCheck time.Time
}

// NewScheduler creates a new scheduler.
func NewScheduler(indexer *repo.Indexer, k8sClient k8s.K8sClient, interval time.Duration) *Scheduler {
	return &Scheduler{
		indexer:   indexer,
		k8sClient: k8sClient,
		interval:  interval,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		statuses: make(map[string]*VariantStatus),
	}
}

// Start begins periodic background checks.
func (s *Scheduler) Start(ctx context.Context) {
	if s.interval <= 0 {
		log.Println("[scheduler] Auto-checking disabled (interval <= 0)")
		return
	}

	log.Printf("[scheduler] Starting upstream kernel checker (interval: %v)\n", s.interval)

	// Run initial check in background after 5 seconds to let server start up cleanly
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
			s.CheckAllAndTrigger()
		}
	}()

	ticker := time.NewTicker(s.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.CheckAllAndTrigger()
			}
		}
	}()
}

// GetStatus returns the latest check status for all variants.
func (s *Scheduler) GetStatus() map[string]*VariantStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]*VariantStatus)
	for k, v := range s.statuses {
		cp := *v
		out[k] = &cp
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

	// Normalize kernel version string for comparison (e.g. 7.2.8.arch1-2 -> 7.2.8.arch1.2)
	normalizedUpstream := strings.ReplaceAll(upstreamVer, "-", ".")

	var latestLocalKernel string
	hasMatch := false

	targetPkgPrefix := "zfs-linux"
	if variant != "" {
		targetPkgPrefix = fmt.Sprintf("zfs-linux-%s", variant)
	}

	for _, p := range summary.Packages {
		if strings.HasPrefix(p.PackageName, targetPkgPrefix) && p.KernelVersion != "" {
			if latestLocalKernel == "" {
				latestLocalKernel = p.KernelVersion
			}
			// Match if package kernel version matches upstream or normalized upstream
			if p.KernelVersion == upstreamVer || p.KernelVersion == normalizedUpstream ||
				strings.HasPrefix(normalizedUpstream, p.KernelVersion) || strings.HasPrefix(p.KernelVersion, normalizedUpstream) {
				hasMatch = true
				break
			}
		}
	}

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
	return status, needsBuild, nil
}

// CheckAllAndTrigger checks upstream kernels for all variants and triggers build jobs when needed.
func (s *Scheduler) CheckAllAndTrigger() {
	variants := []string{"", "lts"}

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

		if needsBuild {
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
