// Package report defines the report entries browsers send, decodes them from
// the wire formats the endpoint accepts, and vets each entry before it is
// written.
package report

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"
)

const (
	maxTypeLen      = 128
	maxURLLen       = 4096
	maxUserAgentLen = 1024
)

// Incoming is one report entry as the browser sent it, before vetting.
type Incoming struct {
	Age       int64           `json:"age"`
	Type      string          `json:"type"`
	URL       string          `json:"url"`
	UserAgent string          `json:"user_agent"`
	Body      json.RawMessage `json:"body"`
}

// Accepted is a vetted report entry, stamped with the time it was received.
type Accepted struct {
	ReceivedAt time.Time
	Age        int64
	Type       string
	URL        string
	UserAgent  string
	Body       json.RawMessage
}

// Vet validates one entry and returns it as an Accepted report stamped with
// now. It returns an error, which says which field is invalid, if the entry
// is rejected.
func Vet(in Incoming, now time.Time) (Accepted, error) {
	reportType := strings.TrimSpace(in.Type)
	if reportType == "" || len(reportType) > maxTypeLen {
		return Accepted{}, errors.New("invalid type")
	}

	reportURL := strings.TrimSpace(in.URL)
	if reportURL == "" || len(reportURL) > maxURLLen {
		return Accepted{}, errors.New("invalid url")
	}

	parsedURL, err := url.ParseRequestURI(reportURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return Accepted{}, errors.New("invalid url")
	}

	userAgent := strings.TrimSpace(in.UserAgent)
	if len(userAgent) > maxUserAgentLen {
		return Accepted{}, errors.New("invalid user_agent")
	}

	if in.Age < 0 {
		return Accepted{}, errors.New("invalid age")
	}

	if len(in.Body) == 0 {
		return Accepted{}, errors.New("missing body")
	}

	// The decoder guarantees Body is valid JSON, so only its type (the first
	// non-whitespace byte) needs checking.
	if trimmed := bytes.TrimSpace(in.Body); len(trimmed) == 0 || trimmed[0] != '{' {
		return Accepted{}, errors.New("invalid body: must be a JSON object")
	}

	return Accepted{
		ReceivedAt: now.UTC(),
		Age:        in.Age,
		Type:       reportType,
		URL:        reportURL,
		UserAgent:  userAgent,
		Body:       in.Body,
	}, nil
}
