package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/cruxstack/browser-reporting-api/v2/internal/ingest"
)

// The example Reporting API batch is accepted in full, and each report is
// written to the stream as one line with the fields downstream consumers
// read.
func TestReports_ExampleBatch(t *testing.T) {
	e := newEnv(t, nil)
	body := readExample(t, "reporting-api.json")
	var sent []map[string]any
	if err := json.Unmarshal([]byte(body), &sent); err != nil {
		t.Fatal(err)
	}

	resp := e.postReports("application/reports+json", body)
	if resp.status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", resp.status, resp.body)
	}
	if got, want := resp.result(t), (ingest.Result{Received: len(sent), Accepted: len(sent)}); got != want {
		t.Errorf("result = %+v, want %+v", got, want)
	}

	lines := e.reports()
	if len(lines) != len(sent) {
		t.Fatalf("stream has %d lines, want %d", len(lines), len(sent))
	}
	for i, line := range lines {
		for _, key := range []string{"type", "url", "user_agent"} {
			if line[key] != sent[i][key] {
				t.Errorf("line %d: %s = %v, want %v", i, key, line[key], sent[i][key])
			}
		}
		if line["received_at"] != "2026-11-06T12:30:00.000Z" {
			t.Errorf("line %d: received_at = %v", i, line["received_at"])
		}
		if _, ok := line["body"].(map[string]any); !ok {
			t.Errorf("line %d: body = %v, want an object", i, line["body"])
		}
	}
}

// A legacy report-uri report becomes one csp-violation report for its
// document, with the legacy fields as its body.
func TestReports_ExampleLegacyCSPReport(t *testing.T) {
	e := newEnv(t, nil)

	resp := e.postReports("application/csp-report", readExample(t, "csp-report.json"))
	if resp.status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", resp.status, resp.body)
	}
	if got, want := resp.result(t), (ingest.Result{Received: 1, Accepted: 1}); got != want {
		t.Errorf("result = %+v, want %+v", got, want)
	}

	lines := e.reports()
	if len(lines) != 1 {
		t.Fatalf("stream has %d lines, want 1", len(lines))
	}
	line := lines[0]
	if line["type"] != "csp-violation" || line["url"] != "https://site.example/checkout" {
		t.Errorf("line = %v, want a csp-violation for https://site.example/checkout", line)
	}
	body, _ := line["body"].(map[string]any)
	if body["blocked-uri"] != "https://cdn.bad-example.test/skimmer.js" {
		t.Errorf("body = %v, want the legacy fields", body)
	}
}

// Entries are vetted one by one: an invalid entry is counted as rejected,
// and the rest of its batch is still written.
func TestReports_InvalidEntriesAreRejectedOneByOne(t *testing.T) {
	e := newEnv(t, nil)

	resp := e.postReports("application/reports+json", `[
		{"age":1,"type":"csp-violation","url":"https://site.example","body":{"a":1}},
		{"age":1,"type":"","url":"https://site.example","body":{"a":1}},
		{"age":1,"type":"csp-violation","url":"/relative","body":{"a":1}},
		{"age":-1,"type":"csp-violation","url":"https://site.example","body":{"a":1}},
		{"age":1,"type":"csp-violation","url":"https://site.example","body":[1]},
		{"age":1,"type":"csp-violation","url":"https://site.example"},
		{"age":"1","type":"csp-violation","url":"https://site.example","body":{"a":1}},
		"not an object",
		null,
		{"age":1,"type":"deprecation","url":"https://site.example/page","body":{"id":"x"}}
	]`)
	if resp.status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", resp.status, resp.body)
	}
	if got, want := resp.result(t), (ingest.Result{Received: 10, Accepted: 2, Rejected: 8}); got != want {
		t.Errorf("result = %+v, want %+v", got, want)
	}
	if got := len(e.reports()); got != 2 {
		t.Errorf("stream has %d lines, want 2", got)
	}
}

func TestReports_BadRequests(t *testing.T) {
	valid := `[{"age":1,"type":"csp-violation","url":"https://site.example","body":{"a":1}}]`
	tests := map[string]struct {
		contentType string
		body        string
		want        int
	}{
		"json content type":         {"application/json", valid, http.StatusUnsupportedMediaType},
		"no content type":           {"", valid, http.StatusUnsupportedMediaType},
		"too large":                 {"application/reports+json", `[{"body":{"a":"` + strings.Repeat("x", 128) + `"}}]`, http.StatusRequestEntityTooLarge},
		"not json":                  {"application/reports+json", `reports`, http.StatusBadRequest},
		"empty batch":               {"application/reports+json", `[]`, http.StatusBadRequest},
		"trailing value":            {"application/reports+json", valid + `{}`, http.StatusBadRequest},
		"scalar":                    {"application/reports+json", `42`, http.StatusBadRequest},
		"object without csp-report": {"application/csp-report", `{"type":"csp-violation"}`, http.StatusBadRequest},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t, map[string]string{"APP_MAX_BODY_BYTES": "128"})

			resp := e.postReports(tt.contentType, tt.body)
			if resp.status != tt.want {
				t.Errorf("status = %d, want %d (body %s)", resp.status, tt.want, resp.body)
			}
			if got := len(e.reports()); got != 0 {
				t.Errorf("stream has %d lines, want 0", got)
			}
		})
	}
}

// With a base path, every route moves under it.
func TestReports_BasePath(t *testing.T) {
	e := newEnv(t, map[string]string{"APP_BASE_PATH": "/collector/"})

	if resp := e.postReports("application/csp-report", readExample(t, "csp-report.json")); resp.status != http.StatusOK {
		t.Errorf("POST /collector/v1/reports = %d, want 200", resp.status)
	}
	if resp := e.do(http.MethodGet, "/v1/manage/healthz", nil, ""); resp.status != http.StatusOK {
		t.Errorf("GET /collector/v1/manage/healthz = %d, want 200", resp.status)
	}

	root := strings.TrimSuffix(e.url, "/collector")
	resp, err := http.Post(root+"/v1/reports", "application/csp-report", strings.NewReader(readExample(t, "csp-report.json")))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("POST /v1/reports = %d, want 404 outside the base path", resp.StatusCode)
	}
}
