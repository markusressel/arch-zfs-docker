// Package kernel handles Arch Linux kernel version strings and discovery of
// installable kernel versions from the Arch Linux Archive.
package kernel

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	defaultArchiveURL = "https://archive.archlinux.org/packages/l"
	cacheTTL          = time.Hour
	maxVersions       = 150
)

var archiveLinkRe = regexp.MustCompile(`href="([^"/]+)-x86_64\.pkg\.tar\.(?:zst|xz)"`)

// PackageName returns the kernel package name for a variant ("" -> linux, "lts" -> linux-lts).
func PackageName(variant string) string {
	if variant == "" {
		return "linux"
	}
	return "linux-" + variant
}

// Normalize converts user input such as "7.1.8-arch1-3" into the pacman
// package version "7.1.8.arch1-3" (the form used by Arch repos and archive).
func Normalize(v string) string {
	v = strings.TrimSpace(v)
	return strings.Replace(v, "-arch", ".arch", 1)
}

// Dotted returns the version as it appears inside zfs-linux package file names
// (all separators are dots, e.g. 7.1.8.arch1.3).
func Dotted(v string) string {
	return strings.ReplaceAll(Normalize(v), "-", ".")
}

// Matches reports whether a kernel version recorded in a local package
// (dotted form) corresponds to the given upstream/target version.
func Matches(local, target string) bool {
	return local != "" && target != "" && Dotted(local) == Dotted(target)
}

// Lister fetches and caches the available kernel versions.
type Lister struct {
	client     *http.Client
	archiveURL string

	mu    sync.Mutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	versions []string
	fetched  time.Time
}

// NewLister creates a Lister querying the official Arch Linux Archive.
func NewLister() *Lister {
	return &Lister{
		client:     &http.Client{Timeout: 20 * time.Second},
		archiveURL: defaultArchiveURL,
		cache:      make(map[string]cacheEntry),
	}
}

// Versions returns available versions for a variant, newest first.
func (l *Lister) Versions(variant string) ([]string, error) {
	pkg := PackageName(variant)

	l.mu.Lock()
	defer l.mu.Unlock()
	if e, ok := l.cache[pkg]; ok && time.Since(e.fetched) < cacheTTL {
		return e.versions, nil
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s/", l.archiveURL, pkg), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "arch-zfs-docker/1.0")
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch kernel list: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kernel list: archive returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, err
	}

	versions := ParseListing(string(body), pkg)
	l.cache[pkg] = cacheEntry{versions: versions, fetched: time.Now()}
	return versions, nil
}

// ParseListing extracts versions of pkg from an archive directory index, newest first.
func ParseListing(html, pkg string) []string {
	prefix := pkg + "-"
	seen := map[string]bool{}
	var out []string
	for _, m := range archiveLinkRe.FindAllStringSubmatch(html, -1) {
		name := m[1]
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		v := strings.TrimPrefix(name, prefix)
		// Skip sibling packages such as linux-lts-headers listed in the same dir.
		if v == "" || !unicode.IsDigit(rune(v[0])) || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return Compare(out[i], out[j]) > 0 })
	if len(out) > maxVersions {
		out = out[:maxVersions]
	}
	return out
}

// Compare orders two version strings segment-wise (numbers numerically).
func Compare(a, b string) int {
	sa, sb := segments(a), segments(b)
	for i := 0; i < len(sa) && i < len(sb); i++ {
		x, y := sa[i], sb[i]
		if x == y {
			continue
		}
		xn, yn := isNum(x), isNum(y)
		switch {
		case xn && yn:
			if len(x) != len(y) { // strip-free numeric compare via length
				if len(x) < len(y) {
					return -1
				}
				return 1
			}
		case xn != yn:
			if xn {
				return 1
			}
			return -1
		}
		if x < y {
			return -1
		}
		return 1
	}
	return len(sa) - len(sb)
}

func isNum(s string) bool { return s != "" && unicode.IsDigit(rune(s[0])) }

func segments(v string) []string {
	var out []string
	var cur []rune
	var curDigit bool
	flush := func() {
		if len(cur) > 0 {
			out = append(out, string(cur))
			cur = nil
		}
	}
	for _, r := range v {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		d := unicode.IsDigit(r)
		if len(cur) > 0 && d != curDigit {
			flush()
		}
		curDigit = d
		cur = append(cur, r)
	}
	flush()
	return out
}
