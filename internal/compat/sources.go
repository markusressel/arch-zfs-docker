package compat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultMetaURL   = "https://raw.githubusercontent.com/openzfs/zfs/zfs-%s/META"
	defaultAurURL    = "https://aur.archlinux.org/rpc/v5/info?arg[]=zfs-utils"
	defaultSearchURL = "https://api.github.com/search/issues"
	issuesPageURL    = "https://github.com/openzfs/zfs/issues"

	metaTTL   = 24 * time.Hour // a release's META never changes
	aurTTL    = 15 * time.Minute
	issuesTTL = 6 * time.Hour
	// Failed lookups are remembered briefly so a down service is not hammered on every poll.
	errorTTL = 5 * time.Minute
)

var linuxMaxRe = regexp.MustCompile(`(?m)^Linux-Maximum:\s*(\d+\.\d+)\s*$`)

type cacheEntry struct {
	value any
	err   error
	at    time.Time
	ttl   time.Duration
}

// Sources queries the OpenZFS repository (META, issues) and the AUR, with caching.
type Sources struct {
	client    *http.Client
	metaURL   string // contains one %s for the ZFS version
	aurURL    string
	searchURL string

	mu    sync.Mutex
	cache map[string]cacheEntry
}

// NewSources creates Sources talking to the public services.
func NewSources() *Sources {
	return &Sources{
		client:    &http.Client{Timeout: 15 * time.Second},
		metaURL:   defaultMetaURL,
		aurURL:    defaultAurURL,
		searchURL: defaultSearchURL,
		cache:     make(map[string]cacheEntry),
	}
}

func (s *Sources) memo(key string, ttl time.Duration, fn func() (any, error)) (any, error) {
	s.mu.Lock()
	if e, ok := s.cache[key]; ok && time.Since(e.at) < e.ttl {
		s.mu.Unlock()
		return e.value, e.err
	}
	s.mu.Unlock()

	v, err := fn()
	entryTTL := ttl
	if err != nil {
		entryTTL = errorTTL
	}
	s.mu.Lock()
	s.cache[key] = cacheEntry{value: v, err: err, at: time.Now(), ttl: entryTTL}
	s.mu.Unlock()
	return v, err
}

func (s *Sources) get(u string, header map[string]string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "arch-zfs-docker/1.0")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %d", req.URL.Host, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

// LinuxMax returns the Linux-Maximum (major.minor) declared by the given OpenZFS release.
func (s *Sources) LinuxMax(zfsVersion string) (string, error) {
	v, err := s.memo("meta:"+zfsVersion, metaTTL, func() (any, error) {
		body, err := s.get(fmt.Sprintf(s.metaURL, url.PathEscape(zfsVersion)), nil)
		if err != nil {
			return "", fmt.Errorf("fetch META of zfs %s: %w", zfsVersion, err)
		}
		m := linuxMaxRe.FindSubmatch(body)
		if m == nil {
			return "", fmt.Errorf("META of zfs %s has no Linux-Maximum", zfsVersion)
		}
		return string(m[1]), nil
	})
	max, _ := v.(string)
	return max, err
}

// AurVersion returns the OpenZFS version (without pkgrel) currently packaged in the AUR.
func (s *Sources) AurVersion() (string, error) {
	v, err := s.memo("aur", aurTTL, func() (any, error) {
		body, err := s.get(s.aurURL, nil)
		if err != nil {
			return "", fmt.Errorf("query AUR: %w", err)
		}
		var resp struct {
			Results []struct {
				Version string `json:"Version"`
			} `json:"results"`
		}
		if err := json.Unmarshal(body, &resp); err != nil || len(resp.Results) == 0 {
			return "", fmt.Errorf("AUR returned no zfs-utils package")
		}
		ver := resp.Results[0].Version
		if i := strings.Index(ver, ":"); i >= 0 {
			ver = ver[i+1:]
		}
		if i := strings.LastIndex(ver, "-"); i >= 0 {
			ver = ver[:i]
		}
		return ver, nil
	})
	ver, _ := v.(string)
	return ver, err
}

type issuesResult struct {
	issues []Issue
	page   string
}

var relevantTitleRe = regexp.MustCompile(`(?i)linux|kernel|compat|build|support`)

// Issues searches OpenZFS issues that mention the given kernel major.minor, newest first.
// The page URL is returned even when the search itself fails.
func (s *Sources) Issues(majorMinor string) ([]Issue, string, error) {
	query := fmt.Sprintf("repo:openzfs/zfs is:issue in:title %s", majorMinor)
	page := issuesPageURL + "?q=" + url.QueryEscape("is:issue "+majorMinor)

	v, err := s.memo("issues:"+majorMinor, issuesTTL, func() (any, error) {
		q := url.Values{"q": {query}, "sort": {"created"}, "order": {"desc"}, "per_page": {"15"}}
		body, err := s.get(s.searchURL+"?"+q.Encode(), map[string]string{"Accept": "application/vnd.github+json"})
		if err != nil {
			return issuesResult{page: page}, fmt.Errorf("search issues: %w", err)
		}
		var resp struct {
			Items []struct {
				Number  int    `json:"number"`
				Title   string `json:"title"`
				HTMLURL string `json:"html_url"`
				State   string `json:"state"`
			} `json:"items"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return issuesResult{page: page}, fmt.Errorf("decode issue search: %w", err)
		}
		res := issuesResult{page: page}
		for _, it := range resp.Items {
			if len(res.issues) == 5 {
				break
			}
			if relevantTitleRe.MatchString(it.Title) {
				res.issues = append(res.issues, Issue{Number: it.Number, Title: it.Title, URL: it.HTMLURL, State: it.State})
			}
		}
		return res, nil
	})
	res, _ := v.(issuesResult)
	if res.page == "" {
		res.page = page
	}
	return res.issues, res.page, err
}

// majorMinor reduces a kernel version such as "7.2.9.arch1-1" to "7.2".
func majorMinor(kernelVersion string) (major, minor int, ok bool) {
	parts := strings.SplitN(kernelVersion, ".", 3)
	if len(parts) < 2 {
		return 0, 0, false
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(strings.SplitN(parts[1], "-", 2)[0])
	return major, minor, err1 == nil && err2 == nil
}
