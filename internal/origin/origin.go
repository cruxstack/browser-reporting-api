// Package origin matches the Origin header of browser requests against the
// allowed origins, which may contain wildcards. The configuration validates
// entries with Validate, and the CORS middleware matches with a Matcher, so
// both apply the same rules.
package origin

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Any is the entry that allows every origin.
const Any = "*"

var wildcardPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*://.*\*.*$`)

// Validate reports whether entry is "*", an origin (scheme://host[:port]),
// or an origin with wildcards (https://*.example.com, http://localhost:*).
func Validate(entry string) error {
	if entry == Any {
		return nil
	}

	if strings.Contains(entry, "*") {
		if !wildcardPattern.MatchString(entry) {
			return fmt.Errorf("invalid wildcard origin pattern: %q", entry)
		}
		return nil
	}

	if !strings.Contains(entry, "://") {
		return fmt.Errorf("invalid origin %q: must include scheme", entry)
	}

	parsed, err := url.Parse(entry)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("invalid origin %q: must be scheme://host[:port]", entry)
	}

	if parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return fmt.Errorf("invalid origin %q: must be scheme://host[:port]", entry)
	}

	return nil
}

// Matcher decides whether an origin is allowed.
type Matcher struct {
	allowAll bool
	exact    map[string]struct{}
	patterns []*regexp.Regexp
}

// NewMatcher returns a Matcher for entries, each of which must pass
// Validate.
func NewMatcher(entries []string) (*Matcher, error) {
	m := &Matcher{exact: make(map[string]struct{})}

	for _, raw := range entries {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		if err := Validate(entry); err != nil {
			return nil, err
		}

		switch {
		case entry == Any:
			m.allowAll = true
		case strings.Contains(entry, "*"):
			re, err := wildcardToRegexp(entry)
			if err != nil {
				return nil, err
			}
			m.patterns = append(m.patterns, re)
		default:
			m.exact[entry] = struct{}{}
		}
	}

	if len(m.exact) == 0 && len(m.patterns) == 0 && !m.allowAll {
		return nil, errors.New("no valid origins provided")
	}

	return m, nil
}

// Allow reports whether origin is allowed, and returns the value of the
// Access-Control-Allow-Origin response header for it.
func (m *Matcher) Allow(origin string) (string, bool) {
	if m.allowAll {
		return Any, true
	}

	origin = strings.TrimSpace(origin)
	if origin == "" {
		return "", false
	}

	if _, ok := m.exact[origin]; ok {
		return origin, true
	}

	for _, re := range m.patterns {
		if re.MatchString(origin) {
			return origin, true
		}
	}

	return "", false
}

func wildcardToRegexp(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for i, ch := range pattern {
		if ch == '*' {
			if isPortWildcard(pattern, i) {
				b.WriteString("[0-9]+")
				continue
			}

			// Match any character except '.' so that a single wildcard
			// segment cannot span subdomain or port boundaries, matching
			// standard CORS same-origin semantics.
			b.WriteString("[^.]*")
			continue
		}
		b.WriteString(regexp.QuoteMeta(string(ch)))
	}
	b.WriteString("$")

	return regexp.Compile(b.String())
}

func isPortWildcard(pattern string, wildcardIndex int) bool {
	if wildcardIndex <= 0 {
		return false
	}

	return pattern[wildcardIndex-1] == ':'
}
