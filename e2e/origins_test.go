package e2e

import (
	"net/http"
	"testing"
)

func postFrom(e *env, requestOrigin string) response {
	e.t.Helper()
	return e.do(http.MethodPost, "/v1/reports", map[string]string{
		"Content-Type": "application/csp-report",
		"Origin":       requestOrigin,
	}, readExample(e.t, "csp-report.json"))
}

// By default browsers may send reports from any origin.
func TestOrigins_DefaultAllowsAny(t *testing.T) {
	e := newEnv(t, nil)

	resp := postFrom(e, "https://anywhere.example")
	if resp.status != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.status)
	}
	if got := resp.header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
	}
}

// With allowed origins, reports from other origins are refused and never
// written. Requests without an Origin header (not cross-origin browser
// requests) are still accepted: CORS is not authentication.
func TestOrigins_AllowedList(t *testing.T) {
	e := newEnv(t, map[string]string{"APP_REPORTS_ALLOWED_ORIGINS": "https://app.example.com,https://*.example.org,http://localhost:*"})

	tests := map[string]int{
		"https://app.example.com":  http.StatusOK,
		"https://shop.example.org": http.StatusOK,
		"http://localhost:5173":    http.StatusOK,
		"https://www.example.com":  http.StatusForbidden,
		"https://attacker.test":    http.StatusForbidden,
	}
	accepted := 0
	for requestOrigin, want := range tests {
		resp := postFrom(e, requestOrigin)
		if resp.status != want {
			t.Errorf("Origin %s: status = %d, want %d", requestOrigin, resp.status, want)
		}
		allow := resp.header.Get("Access-Control-Allow-Origin")
		switch {
		case want == http.StatusOK && allow != requestOrigin:
			t.Errorf("Origin %s: Access-Control-Allow-Origin = %q, want the origin", requestOrigin, allow)
		case want != http.StatusOK && allow != "":
			t.Errorf("Origin %s: Access-Control-Allow-Origin = %q, want none", requestOrigin, allow)
		}
		if want == http.StatusOK {
			accepted++
		}
	}

	if resp := e.postReports("application/csp-report", readExample(t, "csp-report.json")); resp.status != http.StatusOK {
		t.Errorf("no Origin: status = %d, want 200", resp.status)
	}
	accepted++

	if got := len(e.reports()); got != accepted {
		t.Errorf("stream has %d lines, want %d", got, accepted)
	}
}

// Browsers send a preflight before a cross-origin application/reports+json
// request.
func TestOrigins_Preflight(t *testing.T) {
	e := newEnv(t, map[string]string{"APP_REPORTS_ALLOWED_ORIGINS": "https://app.example.com"})

	preflight := func(requestOrigin string) response {
		return e.do(http.MethodOptions, "/v1/reports", map[string]string{
			"Origin":                         requestOrigin,
			"Access-Control-Request-Method":  "POST",
			"Access-Control-Request-Headers": "content-type",
		}, "")
	}

	resp := preflight("https://app.example.com")
	if resp.status != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.status)
	}
	for header, want := range map[string]string{
		"Access-Control-Allow-Origin":  "https://app.example.com",
		"Access-Control-Allow-Methods": "POST, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type",
		"Access-Control-Max-Age":       "7200",
		"Vary":                         "Origin",
	} {
		if got := resp.header.Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}

	if resp := preflight("https://attacker.test"); resp.status != http.StatusForbidden {
		t.Errorf("disallowed preflight: status = %d, want 403", resp.status)
	}
}
