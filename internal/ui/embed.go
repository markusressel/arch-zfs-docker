package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

// dist holds the production build of the Vue app in /web (`just build-ui`).
// The directory always exists (tracked .gitkeep); `all:` keeps that placeholder embeddable.
//
//go:embed all:dist
var distFS embed.FS

// GetFileSystem returns an http.FileSystem serving the embedded UI assets.
func GetFileSystem() http.FileSystem {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	return http.FS(sub)
}
