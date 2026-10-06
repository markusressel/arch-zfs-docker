package repo

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Indexer scans and indexes package files on disk.
type Indexer struct {
	repoDir  string
	repoName string
	shaCache sync.Map // filename+modtime -> sha256 hex
}

// NewIndexer creates a new repository indexer.
func NewIndexer(repoDir, repoName string) *Indexer {
	return &Indexer{
		repoDir:  repoDir,
		repoName: repoName,
	}
}

// GetSummary scans the repository directory for the specified architecture.
func (idx *Indexer) GetSummary(arch string) (*RepoSummary, error) {
	if arch == "" {
		arch = "x86_64"
	}

	// Support paths like /repo/zfslocal/x86_64 or /repo/x86_64 or /repo
	candidates := []string{
		filepath.Join(idx.repoDir, idx.repoName, arch),
		filepath.Join(idx.repoDir, arch),
		idx.repoDir,
	}

	var targetDir string
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			targetDir = c
			break
		}
	}

	summary := &RepoSummary{
		RepoName: idx.repoName,
		Arch:     arch,
		Packages: []PackageInfo{},
	}

	if targetDir == "" {
		return summary, nil
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return nil, fmt.Errorf("read repo dir: %w", err)
	}

	// Check database timestamp
	dbPath := filepath.Join(targetDir, idx.repoName+".db.tar.zst")
	if fi, err := os.Stat(dbPath); err == nil {
		t := fi.ModTime()
		summary.DbLastModified = &t
	}

	var totalBytes int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !isPackageFile(name) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		pkg := idx.parsePackage(name, info, arch)
		totalBytes += info.Size()
		summary.Packages = append(summary.Packages, pkg)
	}

	// Sort packages newest first
	sort.Slice(summary.Packages, func(i, j int) bool {
		return summary.Packages[i].ModTime.After(summary.Packages[j].ModTime)
	})

	summary.PackageCount = len(summary.Packages)
	summary.TotalSizeBytes = totalBytes
	summary.TotalSizeHuman = FormatBytes(totalBytes)

	return summary, nil
}

func isPackageFile(name string) bool {
	if strings.HasSuffix(name, ".sig") || strings.HasSuffix(name, ".old") {
		return false
	}
	return strings.Contains(name, ".pkg.tar")
}

func (idx *Indexer) parsePackage(filename string, info os.FileInfo, arch string) PackageInfo {
	pkg := PackageInfo{
		Filename:    filename,
		SizeBytes:   info.Size(),
		SizeHuman:   FormatBytes(info.Size()),
		ModTime:     info.ModTime(),
		Arch:        arch,
		DownloadURL: fmt.Sprintf("/%s/%s/%s", idx.repoName, arch, filename),
	}

	// Filename format typically:
	// <pkgname>-<pkgver>-<pkgrel>-<arch>.pkg.tar.<ext>
	// e.g. zfs-linux-2.4.4_7.2.8.arch1.2-1-x86_64.pkg.tar.zst
	// or zfs-utils-2.4.4-1-x86_64.pkg.tar.zst
	base := filename
	if idxPkg := strings.Index(base, ".pkg.tar"); idxPkg != -1 {
		base = base[:idxPkg]
	}

	parts := strings.Split(base, "-")
	if len(parts) >= 4 {
		// last part is arch, second to last is pkgrel
		pkg.Arch = parts[len(parts)-1]
		fullVer := parts[len(parts)-3]
		pkgName := strings.Join(parts[:len(parts)-3], "-")
		pkg.PackageName = pkgName

		if strings.Contains(fullVer, "_") {
			verParts := strings.SplitN(fullVer, "_", 2)
			pkg.Version = verParts[0]
			pkg.KernelVersion = verParts[1]
		} else {
			pkg.Version = fullVer
		}
	} else {
		pkg.PackageName = base
	}

	return pkg
}

// ComputeSHA256 returns cached or newly calculated SHA256 hex string for a package file.
func (idx *Indexer) ComputeSHA256(filePath string) (string, error) {
	fi, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	cacheKey := fmt.Sprintf("%s:%d:%d", filePath, fi.Size(), fi.ModTime().UnixNano())
	if cached, ok := idx.shaCache.Load(cacheKey); ok {
		return cached.(string), nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}

	hash := hex.EncodeToString(hasher.Sum(nil))
	idx.shaCache.Store(cacheKey, hash)
	return hash, nil
}

// FormatBytes formats byte count into human-readable string (KiB, MiB, GiB).
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
