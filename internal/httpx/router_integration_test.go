package httpx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cruxstack/browser-reporting-api/internal/domain"
	"github.com/cruxstack/browser-reporting-api/internal/httpx"
	"github.com/cruxstack/browser-reporting-api/internal/management"
	"github.com/cruxstack/browser-reporting-api/internal/parser"
	"github.com/cruxstack/browser-reporting-api/internal/reporting"
)

type captureSink struct {
	reports []domain.AcceptedReport
}

func (s *captureSink) WriteReport(_ context.Context, report domain.AcceptedReport) error {
	s.reports = append(s.reports, report)
	return nil
}

func TestIntegrationRouterHandlesReportingAndManagement(t *testing.T) {
	sink := &captureSink{}
	reportsParser := parser.NewJSONBatchParser()
	reportsService := reporting.NewService(reportsParser, sink)
	origins, err := httpx.NewOriginMatcher([]string{"https://*.example.com"})
	if err != nil {
		t.Fatalf("new matcher: %v", err)
	}

	reportsHandler := reporting.NewHandler(reportsService, 1024*1024, origins)
	managementHandler := management.NewHandler()
	router := httpx.NewRouter("/collector", reportsHandler.Routes(), managementHandler.Routes())
	server := httptest.NewServer(router)
	defer server.Close()

	payload := `[
		{"age":1,"type":"csp-violation","url":"https://site.example","user_agent":"demo/1.0","body":{"effectiveDirective":"script-src"}},
		{"age":4,"type":"deprecation","url":"https://site.example/page","body":{"id":"PrefixedStorageInfo"}},
		{"age":3,"type":"coep","url":"https://site.example/embed","body":{"blockedURL":"https://cdn.example/script.js"}},
		{"age":7,"type":"intervention","url":"https://site.example/feature","body":{"message":"Feature policy blocked"}},
		{"age":2,"type":"","url":"https://site.example/invalid","body":{"bad":true}}
	]`

	req, err := http.NewRequest(http.MethodPost, server.URL+"/collector/v1/reports", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Content-Type", "application/reports+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result domain.IngestResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if result.Received != 5 || result.Accepted != 4 || result.Rejected != 1 {
		t.Fatalf("unexpected ingest result: %+v", result)
	}

	if len(sink.reports) != 4 {
		t.Fatalf("expected 4 accepted reports written to sink, got %d", len(sink.reports))
	}

	healthResp, err := http.Get(server.URL + "/collector/v1/manage/healthz")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	defer healthResp.Body.Close()

	if healthResp.StatusCode != http.StatusOK {
		t.Fatalf("expected health status %d, got %d", http.StatusOK, healthResp.StatusCode)
	}

	readyResp, err := http.Get(server.URL + "/collector/v1/manage/readyz")
	if err != nil {
		t.Fatalf("ready request: %v", err)
	}
	defer readyResp.Body.Close()

	if readyResp.StatusCode != http.StatusOK {
		t.Fatalf("expected ready status %d, got %d", http.StatusOK, readyResp.StatusCode)
	}
}

func TestIntegrationRouterRejectsUnsupportedContentType(t *testing.T) {
	sink := &captureSink{}
	reportsParser := parser.NewJSONBatchParser()
	reportsService := reporting.NewService(reportsParser, sink)
	origins, err := httpx.NewOriginMatcher([]string{"*"})
	if err != nil {
		t.Fatalf("new matcher: %v", err)
	}

	reportsHandler := reporting.NewHandler(reportsService, 1024*1024, origins)
	router := httpx.NewRouter("/", reportsHandler.Routes(), management.NewHandler().Routes())
	server := httptest.NewServer(router)
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/reports", bytes.NewBufferString(`[]`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("expected %d, got %d", http.StatusUnsupportedMediaType, resp.StatusCode)
	}
}

func TestIntegrationRouterRejectsOversizeBody(t *testing.T) {
	sink := &captureSink{}
	reportsParser := parser.NewJSONBatchParser()
	reportsService := reporting.NewService(reportsParser, sink)
	origins, err := httpx.NewOriginMatcher([]string{"*"})
	if err != nil {
		t.Fatalf("new matcher: %v", err)
	}

	reportsHandler := reporting.NewHandler(reportsService, 32, origins)
	router := httpx.NewRouter("/", reportsHandler.Routes(), management.NewHandler().Routes())
	server := httptest.NewServer(router)
	defer server.Close()

	payload := `[{"age":1,"type":"csp-violation","url":"https://site.example","body":{"message":"` + strings.Repeat("x", 256) + `"}}]`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/reports", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/reports+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected %d, got %d body=%s", http.StatusRequestEntityTooLarge, resp.StatusCode, string(body))
	}
}
