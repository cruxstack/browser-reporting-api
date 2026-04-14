package stream

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cruxstack/browser-reporting-api/internal/domain"
)

func TestNDJSONWriterWriteReport(t *testing.T) {
	var buf bytes.Buffer
	w := NewNDJSONWriter(&buf)

	report := domain.AcceptedReport{
		ReceivedAt: time.Date(2026, time.April, 13, 10, 30, 45, 0, time.UTC),
		Age:        2,
		Type:       "csp-violation",
		URL:        "https://site.example/path?a=1&b=2",
		UserAgent:  "demo/1.0",
		Body:       json.RawMessage(`{"effectiveDirective":"script-src"}`),
	}

	if err := w.WriteReport(context.Background(), report); err != nil {
		t.Fatalf("write report: %v", err)
	}

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatal("expected NDJSON output line")
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatalf("decode output: %v", err)
	}

	if got["level"] != "INFO" {
		t.Fatalf("unexpected level: %v", got["level"])
	}

	if got["msg"] != "accepted report" {
		t.Fatalf("unexpected msg: %v", got["msg"])
	}

	if got["received_at"] != "2026-04-13T10:30:45.000Z" {
		t.Fatalf("unexpected received_at: %v", got["received_at"])
	}

	if got["type"] != report.Type || got["url"] != report.URL || got["user_agent"] != report.UserAgent {
		t.Fatalf("unexpected report output fields: %+v", got)
	}

	body, ok := got["body"].(map[string]any)
	if !ok {
		t.Fatalf("expected body object, got %T", got["body"])
	}

	if body["effectiveDirective"] != "script-src" {
		t.Fatalf("unexpected body field: %+v", body)
	}
}

func TestNDJSONWriterConcurrentWrites(t *testing.T) {
	var buf bytes.Buffer
	w := NewNDJSONWriter(&buf)

	const total = 20
	var wg sync.WaitGroup
	wg.Add(total)

	for i := range total {
		go func(i int) {
			defer wg.Done()
			report := domain.AcceptedReport{
				ReceivedAt: time.Date(2026, time.April, 13, 12, 0, i, 0, time.UTC),
				Age:        int64(i),
				Type:       "deprecation",
				URL:        "https://site.example/page",
				Body:       json.RawMessage(`{"id":"FeatureUse"}`),
			}
			if err := w.WriteReport(context.Background(), report); err != nil {
				t.Errorf("write report %d: %v", i, err)
			}
		}(i)
	}

	wg.Wait()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != total {
		t.Fatalf("expected %d lines, got %d", total, len(lines))
	}
}
