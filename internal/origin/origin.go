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

// label matches one DNS label, which is what a host wildcard stands for. It
// can't match a dot or a colon, so a wildcard can't span labels or reach
// into the port.
const label = `[a-z0-9-]+`

// Validate reports whether entry is "*", an origin (scheme://host[:port]),
// or an origin in which whole host labels or the whole port are "*", such as
// https://*.example.com or http://localhost:*. A wildcard label matches
// exactly one label. Entries are case-insensitive.
func Validate(entry string) error {
	_, err := parse(entry)
	return err
}

// rule is one parsed entry.
type rule struct {
	any     bool
	exact   string
	pattern *regexp.Regexp
}

func parse(entry string) (rule, error) {
	if entry == Any {
		return rule{any: true}, nil
	}

	e := strings.ToLower(entry)
	wildPort := strings.HasSuffix(e, ":*")
	base := strings.TrimSuffix(e, ":*")

	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" || strings.HasSuffix(u.Host, ":") ||
		u.Scheme+"://"+u.Host != base || (wildPort && u.Port() != "") {
		return rule{}, fmt.Errorf("%q must be an origin, scheme://host[:port], such as https://app.example.com", entry)
	}

	labels := strings.Split(u.Hostname(), ".")
	wild := wildPort
	for _, l := range labels {
		switch {
		case l == "*":
			wild = true
		case strings.Contains(l, "*"):
			return rule{}, fmt.Errorf("%q: a wildcard must be a whole host label, as in https://*.example.com", entry)
		}
	}
	if !wild {
		return rule{exact: e}, nil
	}

	var b strings.Builder
	b.WriteString("^" + regexp.QuoteMeta(u.Scheme) + "://")
	for i, l := range labels {
		if i > 0 {
			b.WriteString(`\.`)
		}
		if l == "*" {
			b.WriteString(label)
		} else {
			b.WriteString(regexp.QuoteMeta(l))
		}
	}
	if port := u.Port(); port != "" {
		b.WriteString(":" + port)
	}
	if wildPort {
		b.WriteString(`:[0-9]+`)
	}
	b.WriteString("$")
	return rule{pattern: regexp.MustCompile(b.String())}, nil
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
		r, err := parse(entry)
		if err != nil {
			return nil, err
		}

		switch {
		case r.any:
			m.allowAll = true
		case r.pattern != nil:
			m.patterns = append(m.patterns, r.pattern)
		default:
			m.exact[r.exact] = struct{}{}
		}
	}

	if len(m.exact) == 0 && len(m.patterns) == 0 && !m.allowAll {
		return nil, errors.New("no valid origins provided")
	}

	return m, nil
}

// Allow reports whether origin is allowed, and returns the value of the
// Access-Control-Allow-Origin response header for it: "*" when every origin
// is allowed, or else origin as the browser sent it.
func (m *Matcher) Allow(origin string) (string, bool) {
	if m.allowAll {
		return Any, true
	}

	origin = strings.TrimSpace(origin)
	if origin == "" {
		return "", false
	}

	key := strings.ToLower(origin)
	if _, ok := m.exact[key]; ok {
		return origin, true
	}

	for _, re := range m.patterns {
		if re.MatchString(key) {
			return origin, true
		}
	}

	return "", false
}
