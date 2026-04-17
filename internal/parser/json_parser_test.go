package parser

import (
	"strings"
	"testing"
)

func TestParseBatchAcceptsSingleArrayValue(t *testing.T) {
	reports, err := NewJSONBatchParser().ParseBatch(strings.NewReader(`[{"age":1,"type":"csp-violation","url":"https://site.example","body":{"x":1}}]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
}

func TestParseBatchRejectsEmptyArray(t *testing.T) {
	_, err := NewJSONBatchParser().ParseBatch(strings.NewReader(`[]`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseBatchRejectsTrailingTopLevelValue(t *testing.T) {
	_, err := NewJSONBatchParser().ParseBatch(strings.NewReader(`[{"age":1,"type":"csp-violation","url":"https://site.example","body":{"x":1}}]{"extra":true}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseBatchAcceptsLegacyCSPReportURIFormat(t *testing.T) {
	reports, err := NewJSONBatchParser().ParseBatch(strings.NewReader(`{"csp-report":{"document-uri":"https://site.example/page","violated-directive":"frame-src"}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}

	if reports[0].Type != "csp-violation" {
		t.Fatalf("expected csp-violation type, got %q", reports[0].Type)
	}

	if reports[0].URL != "https://site.example/page" {
		t.Fatalf("unexpected url %q", reports[0].URL)
	}
}

func TestParseBatchRejectsLegacyCSPReportWithoutBody(t *testing.T) {
	_, err := NewJSONBatchParser().ParseBatch(strings.NewReader(`{"type":"csp-violation"}`))
	if err == nil {
		t.Fatal("expected error")
	}
}
