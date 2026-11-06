package origin

import "testing"

func TestValidate(t *testing.T) {
	valid := []string{
		"*",
		"https://app.example.com",
		"HTTPS://App.Example.com",
		"http://localhost:3000",
		"http://127.0.0.1:8080",
		"http://[::1]:8080",
		"https://*.example.com",
		"https://*.*.example.com",
		"https://*.example.com:8443",
		"http://localhost:*",
		"https://*.example.com:*",
	}
	for _, entry := range valid {
		if err := Validate(entry); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", entry, err)
		}
	}

	invalid := []string{
		"",
		"app.example.com",
		"*.example.com",
		"*://example.com",
		"https://",
		"https://app.example.com/",
		"https://app.example.com/path",
		"https://app.example.com/*",
		"https://app.example.com?q=1",
		"https://app.example.com#x",
		"https://user@app.example.com",
		"https://app.example.com:",
		"https://app.example.com:80:*",
		"https://app.example.com:8*",
		"https://*example.com",
		"https://app*.example.com",
		"https://ex%41mple.com",
	}
	for _, entry := range invalid {
		if err := Validate(entry); err == nil {
			t.Errorf("Validate(%q) = nil, want an error", entry)
		}
	}
}

func TestMatcher_AllowAll(t *testing.T) {
	m, err := NewMatcher([]string{"*"})
	if err != nil {
		t.Fatal(err)
	}

	allow, ok := m.Allow("https://any.example")
	if !ok || allow != "*" {
		t.Fatalf("Allow() = %q, %t, want *, true", allow, ok)
	}
}

func TestMatcher_ExactAndWildcard(t *testing.T) {
	m, err := NewMatcher([]string{"https://app.example.com", "https://*.example.org", "http://localhost:*", "https://*.example.net:8443"})
	if err != nil {
		t.Fatal(err)
	}

	for origin, want := range map[string]bool{
		"https://app.example.com":       true,
		"http://app.example.com":        false,
		"https://app.example.com:443":   false,
		"https://www.example.com":       false,
		"https://a.example.org":         true,
		"https://a-1.example.org":       true,
		"https://a.b.example.org":       false,
		"https://.example.org":          false,
		"https://example.org":           false,
		"https://evil.com/.example.org": false,
		"http://localhost:3000":         true,
		"http://localhost":              false,
		"http://localhost:abc":          false,
		"https://a.example.net:8443":    true,
		"https://a.example.net":         false,
		"https://attacker.example":      false,
		"":                              false,
	} {
		allow, ok := m.Allow(origin)
		if ok != want {
			t.Errorf("Allow(%q) ok = %t, want %t", origin, ok, want)
		}
		if ok && allow != origin {
			t.Errorf("Allow(%q) = %q, want the origin echoed", origin, allow)
		}
	}
}

// A wildcard stands for one whole label. Before, it matched any run of
// characters other than a dot, so a pattern missing its dot allowed
// look-alike domains anyone can register, and a trailing wildcard reached
// into the port.
func TestMatcher_WildcardIsAWholeLabel(t *testing.T) {
	if _, err := NewMatcher([]string{"https://*example.com"}); err == nil {
		t.Error("NewMatcher(https://*example.com) = nil error, want an error instead of matching https://attackerexample.com")
	}

	m, err := NewMatcher([]string{"https://app.*"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Allow("https://app.evil:8443"); ok {
		t.Error("https://app.* allows https://app.evil:8443, want the wildcard to stop at the port")
	}
	if _, ok := m.Allow("https://app.example"); !ok {
		t.Error("https://app.* rejects https://app.example")
	}
}

// Scheme and host are case-insensitive: browsers send them lowercase, and an
// entry written with capitals still matches. The allowed origin is echoed
// as sent, since browsers compare it exactly.
func TestMatcher_CaseInsensitive(t *testing.T) {
	m, err := NewMatcher([]string{"https://App.Example.com", "https://*.Example.ORG"})
	if err != nil {
		t.Fatal(err)
	}

	for _, origin := range []string{"https://app.example.com", "https://APP.example.com", "https://shop.example.org"} {
		allow, ok := m.Allow(origin)
		if !ok || allow != origin {
			t.Errorf("Allow(%q) = %q, %t, want it allowed and echoed", origin, allow, ok)
		}
	}
}

func TestNewMatcher_RejectsInvalidEntry(t *testing.T) {
	if _, err := NewMatcher([]string{"app.example.com"}); err == nil {
		t.Error("NewMatcher() = nil error, want an error")
	}
}
