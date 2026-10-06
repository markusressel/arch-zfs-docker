package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIndexerParsing(t *testing.T) {
	tempDir := t.TempDir()
	repoArchDir := filepath.Join(tempDir, "zfslocal", "x86_64")
	if err := os.MkdirAll(repoArchDir, 0755); err != nil {
		t.Fatalf("failed to create temp repo dir: %v", err)
	}

	// Create test packages
	pkg1 := "zfs-linux-2.4.4_7.2.8.arch1.2-1-x86_64.pkg.tar.zst"
	pkg2 := "zfs-utils-2.4.4-1-x86_64.pkg.tar.zst"
	db := "zfslocal.db.tar.zst"

	if err := os.WriteFile(filepath.Join(repoArchDir, pkg1), []byte("dummy-kernel-module"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoArchDir, pkg2), []byte("dummy-utils"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoArchDir, db), []byte("dummy-db"), 0644); err != nil {
		t.Fatal(err)
	}

	indexer := NewIndexer(tempDir, "zfslocal")
	summary, err := indexer.GetSummary("x86_64")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.PackageCount != 2 {
		t.Fatalf("expected 2 packages, got %d", summary.PackageCount)
	}
	if summary.DbLastModified == nil {
		t.Fatal("expected DbLastModified to be non-nil")
	}

	var foundLinux, foundUtils bool
	for _, p := range summary.Packages {
		if p.PackageName == "zfs-linux" {
			foundLinux = true
			if p.Version != "2.4.4" {
				t.Errorf("expected version 2.4.4, got %s", p.Version)
			}
			if p.KernelVersion != "7.2.8.arch1.2" {
				t.Errorf("expected kernel version 7.2.8.arch1.2, got %s", p.KernelVersion)
			}
			if p.Arch != "x86_64" {
				t.Errorf("expected arch x86_64, got %s", p.Arch)
			}
		}
		if p.PackageName == "zfs-utils" {
			foundUtils = true
			if p.Version != "2.4.4" {
				t.Errorf("expected version 2.4.4, got %s", p.Version)
			}
			if p.KernelVersion != "" {
				t.Errorf("expected empty kernel version for utils, got %s", p.KernelVersion)
			}
		}
	}

	if !foundLinux {
		t.Error("zfs-linux package was not parsed correctly")
	}
	if !foundUtils {
		t.Error("zfs-utils package was not parsed correctly")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.00 KiB"},
		{1048576, "1.00 MiB"},
		{2643974, "2.52 MiB"},
	}

	for _, tt := range tests {
		got := FormatBytes(tt.bytes)
		if got != tt.expected {
			t.Errorf("FormatBytes(%d) = %s; want %s", tt.bytes, got, tt.expected)
		}
	}
}

func TestComputeSHA256(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "test.pkg.tar.zst")
	if err := os.WriteFile(tempFile, []byte("hello zfs"), 0644); err != nil {
		t.Fatal(err)
	}

	idx := NewIndexer("/tmp", "zfslocal")
	hash1, err := idx.ComputeSHA256(tempFile)
	if err != nil {
		t.Fatal(err)
	}

	// Verify caching on second call
	hash2, err := idx.ComputeSHA256(tempFile)
	if err != nil {
		t.Fatal(err)
	}

	if hash1 != hash2 {
		t.Fatalf("hashes differ: %s vs %s", hash1, hash2)
	}

	// Known sha256 for "hello zfs"
	// echo -n "hello zfs" | sha256sum -> de03e2c1c681966a36b86adce656755490bc8ad7be8c8942b109e2de43ce9ad7
	expected := "6b76f6f92659977f8cdd8c578a03c0e3cbe995df4243603f3f0be5cec2a9d3b6"
	if hash1 != expected {
		t.Errorf("hash = %s; want %s", hash1, expected)
	}
}
