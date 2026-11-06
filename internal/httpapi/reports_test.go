package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cruxstack/browser-reporting-api/v2/internal/ingest"
	"github.com/cruxstack/browser-reporting-api/v2/internal/origin"
)

// fakeIngester reads the body and returns result and err.
type fakeIngester struct {
	result ingest.Result
	err    error
	calls  int
}

func (f *fakeIngester) Ingest(_ context.Context, body io.Reader) (ingest.Result, error) {
	f.calls++
	if _, err := io.ReadAll(body); err != nil {
		return ingest.Result{}, &ingest.BadPayloadError{Err: err}
	}
	return f.result, f.err
}

func newTestRouter(t *testing.T, origins []string, ing Ingester) http.Handler {
	t.Helper()
	m, err := origin.NewMatcher(origins)
	if err != nil {
		t.Fatal(err)
	}
	return NewRouter(Config{BasePath: "/", MaxBodyBytes: 64, Origins: m, Ingester: ing})
}

func post(h http.Handler, contentType, requestOrigin, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/reports", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if requestOrigin != "" {
		req.Header.Set("Origin", requestOrigin)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestReports_StatusCodes(t *testing.T) {
	tests := map[string]struct {
		contentType string
		body        string
		err         error
		want        int
	}{
		"accepted":             {contentType: "application/reports+json", body: "[]", want: http.StatusOK},
		"legacy content type":  {contentType: "application/csp-report", body: "{}", want: http.StatusOK},
		"content type params":  {contentType: "application/reports+json; charset=utf-8", body: "[]", want: http.StatusOK},
		"missing content type": {body: "[]", want: http.StatusUnsupportedMediaType},
		"wrong content type":   {contentType: "application/json", body: "[]", want: http.StatusUnsupportedMediaType},
		"invalid content type": {contentType: "application/", body: "[]", want: http.StatusUnsupportedMediaType},
		"too large":            {contentType: "application/reports+json", body: strings.Repeat("x", 65), want: http.StatusRequestEntityTooLarge},
		"bad payload":          {contentType: "application/reports+json", body: "[]", err: &ingest.BadPayloadError{Err: errors.New("bad")}, want: http.StatusBadRequest},
		"sink error is hidden": {contentType: "application/reports+json", body: "[]", err: errors.New("disk full"), want: http.StatusInternalServerError},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ing := &fakeIngester{result: ingest.Result{Received: 1, Accepted: 1}, err: tt.err}
			rr := post(newTestRouter(t, []string{"*"}, ing), tt.contentType, "", tt.body)
			if rr.Code != tt.want {
				t.Fatalf("status = %d, want %d (body %s)", rr.Code, tt.want, rr.Body)
			}
			if strings.Contains(rr.Body.String(), "disk full") {
				t.Errorf("body %s leaks the internal error", rr.Body)
			}
		})
	}
}

func TestReports_ResponseBodyIsTheResult(t *testing.T) {
	ing := &fakeIngester{result: ingest.Result{Received: 3, Accepted: 2, Rejected: 1}}
	rr := post(newTestRouter(t, []string{"*"}, ing), "application/reports+json", "", "[]")

	var got ingest.Result
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got != ing.result {
		t.Errorf("response = %+v, want %+v", got, ing.result)
	}
}

func TestReports_CORS(t *testing.T) {
	origins := []string{"https://*.example.com"}

	t.Run("allowed origin", func(t *testing.T) {
		rr := post(newTestRouter(t, origins, &fakeIngester{}), "application/reports+json", "https://app.example.com", "[]")
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
			t.Errorf("Access-Control-Allow-Origin = %q", got)
		}
		if got := rr.Header().Get("Vary"); got != "Origin" {
			t.Errorf("Vary = %q, want Origin", got)
		}
	})

	t.Run("disallowed origin is not ingested", func(t *testing.T) {
		ing := &fakeIngester{}
		rr := post(newTestRouter(t, origins, ing), "application/reports+json", "https://attacker.test", "[]")
		if rr.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rr.Code)
		}
		if ing.calls != 0 {
			t.Errorf("ingested %d times, want 0", ing.calls)
		}
	})

	t.Run("preflight", func(t *testing.T) {
		h := newTestRouter(t, origins, &fakeIngester{})
		req := httptest.NewRequest(http.MethodOptions, "/v1/reports", nil)
		req.Header.Set("Origin", "https://app.example.com")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", rr.Code)
		}
		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
			t.Errorf("Access-Control-Allow-Origin = %q", got)
		}
		if got := rr.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type" {
			t.Errorf("Access-Control-Allow-Headers = %q", got)
		}
	})

	t.Run("preflight from disallowed origin", func(t *testing.T) {
		h := newTestRouter(t, origins, &fakeIngester{})
		req := httptest.NewRequest(http.MethodOptions, "/v1/reports", nil)
		req.Header.Set("Origin", "https://attacker.test")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rr.Code)
		}
		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("Access-Control-Allow-Origin = %q, want none", got)
		}
	})
}
