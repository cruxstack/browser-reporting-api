package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cruxstack/browser-reporting-api/v2/internal/origin"
)

func TestRouter_BasePath(t *testing.T) {
	m, err := origin.NewMatcher([]string{"*"})
	if err != nil {
		t.Fatal(err)
	}
	h := NewRouter(Config{BasePath: "/collector", MaxBodyBytes: 1024, Origins: m, Ingester: &fakeIngester{}})

	tests := []struct {
		method, path string
		want         int
	}{
		{http.MethodPost, "/collector/v1/reports", http.StatusOK},
		{http.MethodGet, "/collector/v1/manage/healthz", http.StatusOK},
		{http.MethodGet, "/collector/v1/manage/readyz", http.StatusOK},
		{http.MethodPost, "/v1/reports", http.StatusNotFound},
		{http.MethodGet, "/v1/manage/healthz", http.StatusNotFound},
		{http.MethodGet, "/collector/v1/reports", http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, strings.NewReader("[]"))
		req.Header.Set("Content-Type", "application/reports+json")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != tt.want {
			t.Errorf("%s %s = %d, want %d", tt.method, tt.path, rr.Code, tt.want)
		}
	}
}
