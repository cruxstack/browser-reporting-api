package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestManage(t *testing.T) {
	h := manageRoutes()

	tests := []struct {
		method, path string
		want         int
		body         string
	}{
		{http.MethodGet, "/healthz", http.StatusOK, "{\"status\":\"ok\"}\n"},
		{http.MethodGet, "/readyz", http.StatusOK, "{\"status\":\"ready\"}\n"},
		{http.MethodPost, "/healthz", http.StatusMethodNotAllowed, ""},
	}
	for _, tt := range tests {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(tt.method, tt.path, nil))
		if rr.Code != tt.want {
			t.Errorf("%s %s = %d, want %d", tt.method, tt.path, rr.Code, tt.want)
		}
		if tt.body != "" && rr.Body.String() != tt.body {
			t.Errorf("%s %s body = %q, want %q", tt.method, tt.path, rr.Body, tt.body)
		}
	}
}
