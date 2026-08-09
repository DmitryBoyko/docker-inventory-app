package httpapi

import (
	"net/http"
	"os"
	"path"
	"strings"
)

// WithStatic wraps the API handler to also serve a Vite SPA from dir (e.g. web/dist).
// API routes under /api and /health|/ready keep precedence via the inner mux.
func WithStatic(api http.Handler, dir string) http.Handler {
	if dir == "" {
		return api
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return api
	}
	fileServer := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.HasPrefix(p, "/api/") || p == "/health" || p == "/ready" {
			api.ServeHTTP(w, r)
			return
		}
		// Try exact file; otherwise SPA fallback to index.html.
		clean := path.Clean("/" + strings.TrimPrefix(p, "/"))
		fsPath := path.Join(dir, clean)
		if st, err := os.Stat(fsPath); err == nil && !st.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}
		if clean != "/" {
			if st, err := os.Stat(path.Join(dir, clean)); err == nil && st.IsDir() {
				index := path.Join(dir, clean, "index.html")
				if _, err := os.Stat(index); err == nil {
					fileServer.ServeHTTP(w, r)
					return
				}
			}
		}
		http.ServeFile(w, r, path.Join(dir, "index.html"))
	})
}

// StaticDirExists reports whether dir looks like a built SPA root.
func StaticDirExists(dir string) bool {
	if dir == "" {
		return false
	}
	st, err := os.Stat(path.Join(dir, "index.html"))
	return err == nil && !st.IsDir()
}
