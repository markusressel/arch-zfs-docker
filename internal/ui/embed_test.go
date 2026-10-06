package ui

import "testing"

func TestGetFileSystem(t *testing.T) {
	fs := GetFileSystem()
	if fs == nil {
		t.Fatal("expected non-nil filesystem")
	}

	f, err := fs.Open("index.html")
	if err != nil {
		t.Skipf("web UI not built (run `just build-ui`): %v", err)
	}
	defer f.Close()
}
