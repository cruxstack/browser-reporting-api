package management

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestManagementHealthz(t *testing.T) {
	h := NewHandler().Routes()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	if body := rr.Body.String(); body != "{\"status\":\"ok\"}\n" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestManagementReadyz(t *testing.T) {
	h := NewHandler().Routes()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	if body := rr.Body.String(); body != "{\"status\":\"ready\"}\n" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestManagementMethodNotAllowed(t *testing.T) {
	h := NewHandler().Routes()

	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}
