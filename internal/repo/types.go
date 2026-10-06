package repo

import "time"

// PackageInfo represents metadata about an Arch package archive on disk.
type PackageInfo struct {
	Filename      string    `json:"filename"`
	PackageName   string    `json:"packageName"`
	Version       string    `json:"version"`
	KernelVersion string    `json:"kernelVersion,omitempty"`
	Arch          string    `json:"arch"`
	SizeBytes     int64     `json:"sizeBytes"`
	SizeHuman     string    `json:"sizeHuman"`
	ModTime       time.Time `json:"modTime"`
	SHA256        string    `json:"sha256,omitempty"`
	DownloadURL   string    `json:"downloadUrl"`
}

// RepoSummary represents overall repository statistics and contents.
type RepoSummary struct {
	RepoName       string        `json:"repoName"`
	Arch           string        `json:"arch"`
	PackageCount   int           `json:"packageCount"`
	TotalSizeBytes int64         `json:"totalSizeBytes"`
	TotalSizeHuman string        `json:"totalSizeHuman"`
	DbLastModified *time.Time    `json:"dbLastModified,omitempty"`
	Packages       []PackageInfo `json:"packages"`
}
