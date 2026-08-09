package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/epm-games/docker-visualizer/internal/store"
)

func TestWithAuth_DisabledPassthrough(t *testing.T) {
	s := testServer(t, store.New())
	h := Chain(s.Handler(), WithAuth(""))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/containers", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestWithAuth_RequiresToken(t *testing.T) {
	s := testServer(t, store.New())
	h := Chain(s.Handler(), WithAuth("s3cret"))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/containers", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}

	rr2 := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/containers", nil)
	req.Header.Set("Authorization", "Bearer s3cret")
	h.ServeHTTP(rr2, req)
	if rr2.Code != http.StatusOK {
		t.Fatalf("bearer status=%d body=%s", rr2.Code, rr2.Body.String())
	}

	rr3 := httptest.NewRecorder()
	h.ServeHTTP(rr3, httptest.NewRequest(http.MethodGet, "/api/v1/containers?access_token=s3cret", nil))
	if rr3.Code != http.StatusOK {
		t.Fatalf("query status=%d", rr3.Code)
	}
}

func TestWithAuth_HealthExempt(t *testing.T) {
	s := testServer(t, store.New())
	h := Chain(s.Handler(), WithAuth("s3cret"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestWithAuth_ReadyProtected(t *testing.T) {
	s := testServer(t, store.New())
	h := Chain(s.Handler(), WithAuth("s3cret"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/ready", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
}
