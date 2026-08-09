package httpapi

import (
	"bytes"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// WithFS serves a Vite SPA from an fs.FS (typically go:embed), with SPA fallback.
// API routes under /api and /health|/ready keep precedence via the inner mux.
func WithFS(api http.Handler, fsys fs.FS) http.Handler {
	if fsys == nil {
		return api
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.HasPrefix(p, "/api/") || p == "/health" || p == "/ready" {
			api.ServeHTTP(w, r)
			return
		}

		clean := strings.TrimPrefix(path.Clean("/"+strings.TrimPrefix(p, "/")), "/")
		if clean == "." {
			clean = ""
		}
		if clean == "" {
			serveFSFile(w, r, fsys, "index.html")
			return
		}
		if st, err := fs.Stat(fsys, clean); err == nil && !st.IsDir() {
			serveFSFile(w, r, fsys, clean)
			return
		}
		serveFSFile(w, r, fsys, "index.html")
	})
}

func serveFSFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, name string) {
	f, err := fsys.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	var mod time.Time
	if st, err := f.Stat(); err == nil {
		mod = st.ModTime()
	}
	rs, ok := f.(io.ReadSeeker)
	if !ok {
		// embed.FS files are ReadSeekers; fallback for odd FS impls.
		b, err := io.ReadAll(f)
		if err != nil {
			http.Error(w, "read error", http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, r, path.Base(name), mod, bytes.NewReader(b))
		return
	}
	http.ServeContent(w, r, path.Base(name), mod, rs)
}
