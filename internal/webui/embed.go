// Package webui embeds the built React frontend into the binary so the server
// is a single self-contained, self-updatable artifact.
//
// The real frontend is built into internal/webui/dist before a production build
// (see the release workflow / README). For backend-only builds, only a .gitkeep
// is present and Available() returns false, in which case the API still runs but
// the UI is not served (developers use the Vite dev server instead).
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// FS returns the embedded frontend rooted at the dist directory.
func FS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return distFS
	}
	return sub
}

// Available reports whether a real frontend build is embedded (index.html present).
func Available() bool {
	if _, err := fs.Stat(FS(), "index.html"); err == nil {
		return true
	}
	return false
}
