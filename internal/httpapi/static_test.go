package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestWithStaticServesIndexAndKeepsAPI(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>ui</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte("console.log(1)"), 0o644); err != nil {
		t.Fatal(err)
	}

	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	h := WithStatic(api, dir)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if rr.Code != http.StatusOK || rr.Body.String() != `{"ok":true}` {
		t.Fatalf("api: status=%d body=%q", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rr.Code != http.StatusOK || rr.Body.String() != "console.log(1)" {
		t.Fatalf("asset: status=%d body=%q", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/containers", nil))
	if rr.Code != http.StatusOK || rr.Body.String() != "<html>ui</html>" {
		t.Fatalf("spa: status=%d body=%q", rr.Code, rr.Body.String())
	}
}

func TestStaticDirExists(t *testing.T) {
	dir := t.TempDir()
	if StaticDirExists(dir) {
		t.Fatal("expected false without index")
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !StaticDirExists(dir) {
		t.Fatal("expected true")
	}
}
