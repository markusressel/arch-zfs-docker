// Package zfs handles OpenZFS release versions: normalization of user input and discovery
// of released versions from the OpenZFS GitHub repository.
package zfs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/markusressel/arch-zfs-docker/internal/kernel"
)

const (
	defaultReleasesURL = "https://api.github.com/repos/openzfs/zfs/releases?per_page=100"
	cacheTTL           = time.Hour
)

var versionRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// Normalize converts input such as "zfs-2.4.4" or "v2.4.4" into "2.4.4".
func Normalize(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "zfs-")
	return strings.TrimPrefix(v, "v")
}

// Valid reports whether v is empty (meaning "latest packaged") or a plain X.Y.Z release version.
// The value ends up in shell scripts and PKGBUILDs, so anything else is rejected.
func Valid(v string) bool {
	return v == "" || versionRe.MatchString(v)
}

// Lister fetches and caches released OpenZFS versions.
type Lister struct {
	client      *http.Client
	releasesURL string

	mu       sync.Mutex
	versions []string
	fetched  time.Time
}

// NewLister creates a Lister querying the GitHub releases API.
func NewLister() *Lister {
	return &Lister{client: &http.Client{Timeout: 20 * time.Second}, releasesURL: defaultReleasesURL}
}

// Versions returns stable released versions, newest first.
func (l *Lister) Versions() ([]string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.versions != nil && time.Since(l.fetched) < cacheTTL {
		return l.versions, nil
	}

	req, err := http.NewRequest("GET", l.releasesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "arch-zfs-docker/1.0")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch zfs releases: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("zfs releases: GitHub returned %d", resp.StatusCode)
	}

	var releases []struct {
		Tag        string `json:"tag_name"`
		Prerelease bool   `json:"prerelease"`
		Draft      bool   `json:"draft"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode zfs releases: %w", err)
	}

	stable := make([]string, 0, len(releases))
	for _, r := range releases {
		if r.Draft || r.Prerelease {
			continue
		}
		if v := Normalize(r.Tag); versionRe.MatchString(v) {
			stable = append(stable, v)
		}
	}
	sort.Slice(stable, func(i, j int) bool { return kernel.Compare(stable[i], stable[j]) > 0 })

	l.versions, l.fetched = stable, time.Now()
	return stable, nil
}
