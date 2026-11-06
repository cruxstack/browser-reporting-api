package origin

import "testing"

func TestValidate(t *testing.T) {
	valid := []string{
		"*",
		"https://app.example.com",
		"http://localhost:3000",
		"https://*.example.com",
		"http://localhost:*",
	}
	for _, entry := range valid {
		if err := Validate(entry); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", entry, err)
		}
	}

	invalid := []string{
		"app.example.com",
		"*.example.com",
		"https://app.example.com/",
		"https://app.example.com/path",
		"https://app.example.com?q=1",
		"https://user@app.example.com",
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
	m, err := NewMatcher([]string{"https://app.example.com", "https://*.example.org", "http://localhost:*"})
	if err != nil {
		t.Fatal(err)
	}

	for origin, want := range map[string]bool{
		"https://app.example.com":  true,
		"https://www.example.com":  false,
		"https://a.example.org":    true,
		"https://a.b.example.org":  false,
		"http://localhost:3000":    true,
		"http://localhost:abc":     false,
		"https://attacker.example": false,
		"":                         false,
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

func TestNewMatcher_RejectsInvalidEntry(t *testing.T) {
	if _, err := NewMatcher([]string{"app.example.com"}); err == nil {
		t.Error("NewMatcher() = nil error, want an error")
	}
}
