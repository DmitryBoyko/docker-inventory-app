package uiembed

import (
	"embed"
	"io/fs"
)

// Dist contains the Vite production build (synced by `make ui`).
// A stub index.html is committed so `go build` works before the first UI sync.
//
//go:embed all:dist
var dist embed.FS

// FS returns the embedded SPA root (contents of dist/).
func FS() (fs.FS, error) {
	return fs.Sub(dist, "dist")
}

// Available reports whether a usable index.html is embedded.
func Available() bool {
	root, err := FS()
	if err != nil {
		return false
	}
	f, err := root.Open("index.html")
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}
