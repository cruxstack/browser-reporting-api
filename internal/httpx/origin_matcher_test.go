package httpx

import "testing"

func TestOriginMatcherAllowAll(t *testing.T) {
	m, err := NewOriginMatcher([]string{"*"})
	if err != nil {
		t.Fatalf("new matcher: %v", err)
	}

	allow, ok := m.AllowHeaderValue("https://any.example")
	if !ok || allow != "*" {
		t.Fatalf("expected allow all, got allow=%q ok=%t", allow, ok)
	}
}

func TestOriginMatcherWildcardPattern(t *testing.T) {
	m, err := NewOriginMatcher([]string{"https://*.example.com", "http://localhost:*"})
	if err != nil {
		t.Fatalf("new matcher: %v", err)
	}

	if _, ok := m.AllowHeaderValue("https://app.example.com"); !ok {
		t.Fatal("expected wildcard host to match")
	}

	if _, ok := m.AllowHeaderValue("http://localhost:3000"); !ok {
		t.Fatal("expected wildcard port to match")
	}

	if _, ok := m.AllowHeaderValue("http://localhost:abc"); ok {
		t.Fatal("expected wildcard port to reject non-numeric value")
	}

	if _, ok := m.AllowHeaderValue("https://evil.test"); ok {
		t.Fatal("expected disallowed origin")
	}
}
