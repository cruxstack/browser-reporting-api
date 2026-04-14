package config

import "testing"

func TestValidateAllowedOriginsAcceptsValidExactOrigin(t *testing.T) {
	err := validateAllowedOrigins([]string{"https://app.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAllowedOriginsRejectsOriginWithPath(t *testing.T) {
	err := validateAllowedOrigins([]string{"https://app.example.com/path"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateAllowedOriginsRejectsOriginWithTrailingSlash(t *testing.T) {
	err := validateAllowedOrigins([]string{"https://app.example.com/"})
	if err == nil {
		t.Fatal("expected error")
	}
}
