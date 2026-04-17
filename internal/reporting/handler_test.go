package reporting

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cruxstack/browser-reporting-api/internal/domain"
	"github.com/cruxstack/browser-reporting-api/internal/httpx"
	"github.com/cruxstack/browser-reporting-api/internal/parser"
)

type captureSink struct {
	reports []domain.AcceptedReport
	err     error
}

func (s *captureSink) WriteReport(_ context.Context, report domain.AcceptedReport) error {
	if s.err != nil {
		return s.err
	}
	s.reports = append(s.reports, report)
	return nil
}

func newTestHandler(t *testing.T, origins []string) http.Handler {
	t.Helper()

	match, err := httpx.NewOriginMatcher(origins)
	if err != nil {
		t.Fatalf("new matcher: %v", err)
	}

	sink := &captureSink{}
	service := NewService(parser.NewJSONBatchParser(), sink)
	return NewHandler(service, 1024*1024, match).Routes()
}

func TestIngestAcceptsBatch(t *testing.T) {
	h := newTestHandler(t, []string{"*"})

	body := `[
		{"age":1,"type":"csp-violation","url":"https://site.example","user_agent":"ua","body":{"foo":"bar"}},
		{"age":2,"type":"coep","url":"https://site.example/page","body":{"blockedURL":"https://other.example"}}
	]`

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/reports+json")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	var response map[string]int
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response["received"] != 2 || response["accepted"] != 2 || response["rejected"] != 0 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestIngestRejectsDisallowedOrigin(t *testing.T) {
	h := newTestHandler(t, []string{"https://*.example.com"})

	body := `[{"age":1,"type":"csp-violation","url":"https://site.example","body":{"foo":"bar"}}]`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Origin", "https://attacker.test")
	req.Header.Set("Content-Type", "application/reports+json")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestIngestAcceptsLegacyCSPReportURIFormat(t *testing.T) {
	h := newTestHandler(t, []string{"*"})

	body := `{"csp-report":{"document-uri":"https://site.example/page","violated-directive":"frame-src"}}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/csp-report")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	var response map[string]int
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response["received"] != 1 || response["accepted"] != 1 || response["rejected"] != 0 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestPreflightAllowsWildcardPatternOrigin(t *testing.T) {
	h := newTestHandler(t, []string{"https://*.example.com"})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, rr.Code)
	}

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("unexpected allow origin %q", got)
	}
}
