package domain

import (
	"strings"
	"testing"
	"time"
)

func TestVetAcceptsValidReport(t *testing.T) {
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	report := IncomingReport{
		Age:       3,
		Type:      "csp-violation",
		URL:       "https://site.example/path",
		UserAgent: "Mozilla/5.0",
		Body:      []byte(`{"directive":"script-src"}`),
	}

	accepted, err := Vet(report, now)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if accepted.Type != report.Type {
		t.Fatalf("expected type %q, got %q", report.Type, accepted.Type)
	}
}

func TestVetRejectsInvalidURL(t *testing.T) {
	_, err := Vet(IncomingReport{
		Type: "csp-violation",
		URL:  "/relative",
		Body: []byte(`{"x":1}`),
	}, time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVetRejectsOversizedUserAgent(t *testing.T) {
	_, err := Vet(IncomingReport{
		Type:      "csp-violation",
		URL:       "https://site.example",
		UserAgent: strings.Repeat("a", maxUserAgentLen+1),
		Body:      []byte(`{"x":1}`),
	}, time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
}
