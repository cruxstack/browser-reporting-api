package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRouterUsesBasePath(t *testing.T) {
	reports := chi.NewRouter()
	reports.Post("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	manage := chi.NewRouter()
	manage.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := NewRouter("/collector", reports, manage)

	req := httptest.NewRequest(http.MethodPost, "/collector/v1/reports", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
}
