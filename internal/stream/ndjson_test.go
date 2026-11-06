package stream

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cruxstack/browser-reporting-api/v2/internal/report"
)

func TestNDJSONWriter_WriteReport(t *testing.T) {
	var buf bytes.Buffer
	w := NewNDJSONWriter(&buf)

	r := report.Accepted{
		ReceivedAt: time.Date(2026, time.April, 13, 10, 30, 45, 0, time.UTC),
		Age:        2,
		Type:       "csp-violation",
		URL:        "https://site.example/path?a=1&b=2",
		UserAgent:  "demo/1.0",
		Body:       json.RawMessage(`{"effectiveDirective":"script-src"}`),
	}

	if err := w.WriteReport(context.Background(), r); err != nil {
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

	if got["type"] != r.Type || got["url"] != r.URL || got["user_agent"] != r.UserAgent {
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

// A body sent pretty-printed is written on one line, as sent otherwise: its
// keys keep their order and large numbers keep their digits.
func TestNDJSONWriter_WritesBodyAsSentOnOneLine(t *testing.T) {
	var buf bytes.Buffer
	w := NewNDJSONWriter(&buf)

	body := "{\n  \"z\": 1,\n  \"a\": 9007199254740993\n}"
	err := w.WriteReport(context.Background(), report.Accepted{Type: "csp-violation", URL: "https://site.example", Body: json.RawMessage(body)})
	if err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if strings.Count(out, "\n") != 1 {
		t.Fatalf("wrote %q, want one line", out)
	}
	if !strings.Contains(out, `"body":{"z":1,"a":9007199254740993}`) {
		t.Errorf("wrote %q, want the body compacted with its keys and digits unchanged", out)
	}
}

func TestNDJSONWriter_ConcurrentWrites(t *testing.T) {
	var buf bytes.Buffer
	w := NewNDJSONWriter(&buf)

	const total = 20
	var wg sync.WaitGroup
	for i := range total {
		wg.Go(func() {
			r := report.Accepted{
				ReceivedAt: time.Date(2026, time.April, 13, 12, 0, i, 0, time.UTC),
				Age:        int64(i),
				Type:       "deprecation",
				URL:        "https://site.example/page",
				Body:       json.RawMessage(`{"id":"FeatureUse"}`),
			}
			if err := w.WriteReport(context.Background(), r); err != nil {
				t.Errorf("write report %d: %v", i, err)
			}
		})
	}
	wg.Wait()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != total {
		t.Fatalf("expected %d lines, got %d", total, len(lines))
	}
}
