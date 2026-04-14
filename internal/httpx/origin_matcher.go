package httpx

import (
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"
)

type OriginMatcher struct {
	allowAll bool
	exact    map[string]struct{}
	patterns []*regexp.Regexp
}

func NewOriginMatcher(origins []string) (*OriginMatcher, error) {
	m := &OriginMatcher{exact: make(map[string]struct{})}

	for _, raw := range origins {
		origin := strings.TrimSpace(raw)
		if origin == "" {
			continue
		}

		if origin == "*" {
			m.allowAll = true
			continue
		}

		if strings.Contains(origin, "*") {
			re, err := wildcardPatternToRegex(origin)
			if err != nil {
				return nil, err
			}
			m.patterns = append(m.patterns, re)
			continue
		}

		m.exact[origin] = struct{}{}
	}

	if len(m.exact) == 0 && len(m.patterns) == 0 && !m.allowAll {
		return nil, errors.New("no valid origins provided")
	}

	return m, nil
}

func (m *OriginMatcher) AllowHeaderValue(origin string) (string, bool) {
	if m.allowAll {
		return "*", true
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

func wildcardPatternToRegex(pattern string) (*regexp.Regexp, error) {
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
