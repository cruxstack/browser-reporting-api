package report

import (
	"strings"
	"testing"
	"time"
)

func TestParseBatchAcceptsSingleArrayValue(t *testing.T) {
	reports, err := Parse(strings.NewReader(`[{"age":1,"type":"csp-violation","url":"https://site.example","body":{"x":1}}]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
}

func TestParseBatchRejectsEmptyArray(t *testing.T) {
	_, err := Parse(strings.NewReader(`[]`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseBatchRejectsTrailingTopLevelValue(t *testing.T) {
	_, err := Parse(strings.NewReader(`[{"age":1,"type":"csp-violation","url":"https://site.example","body":{"x":1}}]{"extra":true}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseBatchAcceptsLegacyCSPReportURIFormat(t *testing.T) {
	reports, err := Parse(strings.NewReader(`{"csp-report":{"document-uri":"https://site.example/page","violated-directive":"frame-src"}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}

	if reports[0].Type != TypeCSPViolation {
		t.Fatalf("expected csp-violation type, got %q", reports[0].Type)
	}

	if reports[0].URL != "https://site.example/page" {
		t.Fatalf("unexpected url %q", reports[0].URL)
	}
}

func TestParseBatchRejectsLegacyCSPReportWithoutBody(t *testing.T) {
	_, err := Parse(strings.NewReader(`{"type":"csp-violation"}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

// An entry that doesn't decode is returned for Vet to reject, so the valid
// entries of its batch are kept.
func TestParseKeepsBatchWhenAnEntryDoesNotDecode(t *testing.T) {
	reports, err := Parse(strings.NewReader(`[
		{"age":"1","type":"csp-violation","url":"https://site.example","body":{}},
		"not an object",
		{"age":1,"type":"csp-violation","url":"https://site.example","body":{"x":1}}
	]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reports) != 3 {
		t.Fatalf("expected 3 reports, got %d", len(reports))
	}

	now := time.Now()
	for i, wantErr := range []bool{true, true, false} {
		if _, err := Vet(reports[i], now); (err != nil) != wantErr {
			t.Errorf("Vet(reports[%d]) error = %v, want error %t", i, err, wantErr)
		}
	}
}
