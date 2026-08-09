package httpapi

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestWithFSServesAssetsAndSPA(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>ui</html>")},
		"assets/a.js": &fstest.MapFile{Data: []byte("console.log(1)")},
	}
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	h := WithFS(api, fs.FS(fsys))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if rr.Body.String() != `{"ok":true}` {
		t.Fatalf("api body=%q", rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/assets/a.js", nil))
	if rr.Body.String() != "console.log(1)" {
		t.Fatalf("asset body=%q", rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/containers/abc", nil))
	if rr.Body.String() != "<html>ui</html>" {
		t.Fatalf("spa body=%q", rr.Body.String())
	}
}
