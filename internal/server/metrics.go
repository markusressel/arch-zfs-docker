package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/markusressel/arch-zfs-docker/internal/compat"
	"github.com/markusressel/arch-zfs-docker/internal/k8s"
)

// Prometheus metrics in the text exposition format. They are computed on every scrape from the
// same sources as the API, so the dashboard and the alerts see what the web UI shows.

// compatStates are exported one-hot, so a state that is not current reads 0 instead of vanishing.
var compatStates = []compat.State{compat.StateOK, compat.StatePredictedUnsupported, compat.StateIncompatible, compat.StateUnknown}

// failureCategories are exported one-hot for the newest finished build of a kernel.
var failureCategories = []compat.Category{compat.CategoryEnvironment, compat.CategoryKernel, compat.CategoryBuild, "unknown"}

func kernelPkgOf(variant string) string {
	if variant == "" {
		return "linux"
	}
	return "linux-" + variant
}

type metricsWriter struct {
	w    io.Writer
	seen map[string]bool
}

// family writes the HELP and TYPE lines once per metric name.
func (m *metricsWriter) family(name, typ, help string) {
	if m.seen[name] {
		return
	}
	m.seen[name] = true
	fmt.Fprintf(m.w, "# HELP %s %s\n# TYPE %s %s\n", name, help, name, typ)
}

func (m *metricsWriter) sample(name string, value float64, labels ...string) {
	var b strings.Builder
	b.WriteString(name)
	if len(labels) > 0 {
		b.WriteByte('{')
		for i := 0; i+1 < len(labels); i += 2 {
			if i > 0 {
				b.WriteByte(',')
			}
			fmt.Fprintf(&b, "%s=%q", labels[i], labels[i+1])
		}
		b.WriteByte('}')
	}
	fmt.Fprintf(m.w, "%s %g\n", b.String(), value)
}

func (m *metricsWriter) gauge(name, help string, value float64, labels ...string) {
	m.family(name, "gauge", help)
	m.sample(name, value, labels...)
}

func boolValue(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func unixSeconds(t *time.Time) float64 {
	if t == nil || t.IsZero() {
		return 0
	}
	return float64(t.Unix())
}

// variantBuilds summarizes the build jobs of one kernel variant, jobs newest first.
type variantBuilds struct {
	running       int
	byStatus      map[k8s.JobStatus]int
	failStreak    int
	lastFinished  *k8s.JobSummary
	lastSuccess   *k8s.JobSummary
	streakCounted bool
}

func summarizeBuilds(jobs []k8s.JobSummary) map[string]*variantBuilds {
	out := map[string]*variantBuilds{}
	for i := range jobs {
		j := &jobs[i]
		if !j.IsBuild() {
			continue
		}
		pkg := kernelPkgOf(j.Variant)
		v := out[pkg]
		if v == nil {
			v = &variantBuilds{byStatus: map[k8s.JobStatus]int{}}
			out[pkg] = v
		}
		v.byStatus[j.Status]++
		if j.IsActive() {
			v.running++
			continue
		}
		if !j.IsFinished() {
			continue
		}
		if v.lastFinished == nil {
			v.lastFinished = j
		}
		if j.Status == k8s.StatusSucceeded {
			if v.lastSuccess == nil {
				v.lastSuccess = j
			}
			v.streakCounted = true
		} else if !v.streakCounted {
			v.failStreak++
		}
	}
	return out
}

func endOf(j *k8s.JobSummary) *time.Time {
	if j == nil {
		return nil
	}
	if j.EndTime != nil {
		return j.EndTime
	}
	return j.StartTime
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	m := &metricsWriter{w: w, seen: map[string]bool{}}

	// Kernels: upstream vs. repository, and whether OpenZFS supports the upstream kernel.
	statuses := s.scheduler.GetStatus()
	pkgs := make([]string, 0, len(statuses))
	for pkg := range statuses {
		pkgs = append(pkgs, pkg)
	}
	sort.Strings(pkgs)
	for _, pkg := range pkgs {
		st := statuses[pkg]
		m.gauge("arch_repo_kernel_up_to_date", "1 when the repository has packages for the newest upstream kernel.", boolValue(st.UpToDate), "kernel_pkg", pkg)
	}
	for _, pkg := range pkgs {
		st := statuses[pkg]
		m.gauge("arch_repo_kernel_info", "Upstream kernel and the newest kernel packaged in the repository.", 1,
			"kernel_pkg", pkg, "upstream", st.UpstreamKernel, "local", st.LocalKernel)
	}
	for _, pkg := range pkgs {
		st := statuses[pkg]
		m.gauge("arch_repo_upstream_checked_timestamp_seconds", "Time of the last upstream kernel check.", unixSeconds(&st.CheckedAt), "kernel_pkg", pkg)
	}
	for _, pkg := range pkgs {
		st := statuses[pkg]
		current := compat.StateOK // compatibility is only assessed while a kernel is not up to date
		zfs := ""
		if st.Compat != nil {
			current, zfs = st.Compat.State, st.Compat.ZfsVersion
		}
		for _, state := range compatStates {
			m.gauge("arch_repo_kernel_compat", "Compatibility of the upstream kernel with the packaged OpenZFS release (one-hot over state).",
				boolValue(state == current), "kernel_pkg", pkg, "state", string(state), "zfs_version", zfs)
		}
	}

	// Builds: known jobs of each kernel, newest first.
	jobs, err := s.k8sClient.ListJobs()
	m.gauge("arch_repo_jobs_list_ok", "1 when the build jobs could be listed; the build metrics are missing otherwise.", boolValue(err == nil))
	if err != nil {
		log.Printf("[metrics] Listing jobs failed: %v\n", err)
	} else {
		// Prometheus scrapes every minute, so a failed build is classified soon after it ends and
		// arch_repo_kernel_compat does not read "ok" or "predicted_unsupported" for hours.
		s.observeInBackground(jobs)
		s.compat.Annotate(jobs)
		builds := summarizeBuilds(jobs)
		names := make([]string, 0, len(builds))
		for pkg := range builds {
			names = append(names, pkg)
		}
		sort.Strings(names)
		for _, pkg := range names {
			for _, st := range []k8s.JobStatus{k8s.StatusPending, k8s.StatusRunning, k8s.StatusSucceeded, k8s.StatusFailed} {
				m.gauge("arch_repo_build_jobs", "Build jobs still known to Kubernetes, by status.", float64(builds[pkg].byStatus[st]), "kernel_pkg", pkg, "status", string(st))
			}
		}
		for _, pkg := range names {
			m.gauge("arch_repo_build_fail_streak", "Failed builds since the last successful one.", float64(builds[pkg].failStreak), "kernel_pkg", pkg)
		}
		for _, pkg := range names {
			m.gauge("arch_repo_build_last_success_timestamp_seconds", "End of the newest successful build, 0 if none is known.", unixSeconds(endOf(builds[pkg].lastSuccess)), "kernel_pkg", pkg)
		}
		for _, pkg := range names {
			m.gauge("arch_repo_build_last_finished_timestamp_seconds", "End of the newest finished build, 0 if none is known.", unixSeconds(endOf(builds[pkg].lastFinished)), "kernel_pkg", pkg)
		}
		for _, pkg := range names {
			var d float64
			if j := builds[pkg].lastFinished; j != nil {
				d = float64(j.DurationSec)
			}
			m.gauge("arch_repo_build_last_duration_seconds", "Duration of the newest finished build.", d, "kernel_pkg", pkg)
		}
		for _, pkg := range names {
			last := builds[pkg].lastFinished
			category := compat.Category("")
			if last != nil && last.Status == k8s.StatusFailed {
				category = "unknown"
				if last.Failure != nil {
					category = compat.Category(last.Failure.Category)
				}
			}
			for _, c := range failureCategories {
				m.gauge("arch_repo_build_last_failed", "1 when the newest finished build failed, labelled with the classified cause (one-hot over category).",
					boolValue(c == category), "kernel_pkg", pkg, "category", string(c))
			}
		}
	}

	// Repository contents.
	summary, err := s.indexer.GetSummary(arch)
	m.gauge("arch_repo_index_ok", "1 when the repository directory could be read.", boolValue(err == nil))
	if err == nil {
		m.gauge("arch_repo_packages", "Package files in the repository.", float64(summary.PackageCount))
		m.gauge("arch_repo_size_bytes", "Total size of the package files.", float64(summary.TotalSizeBytes))
		m.gauge("arch_repo_db_last_modified_timestamp_seconds", "Last change of the pacman database.", unixSeconds(summary.DbLastModified))
	}
}
