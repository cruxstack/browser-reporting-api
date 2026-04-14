package domain

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
)

const (
	maxTypeLen      = 128
	maxURLLen       = 4096
	maxUserAgentLen = 1024
)

// IncomingReport is the raw, unvalidated report entry received from the browser.
type IncomingReport struct {
	Age       int64           `json:"age"`
	Type      string          `json:"type"`
	URL       string          `json:"url"`
	UserAgent string          `json:"user_agent"`
	Body      json.RawMessage `json:"body"`
}

// AcceptedReport is a validated, timestamped report ready for output.
type AcceptedReport struct {
	ReceivedAt time.Time
	Age        int64
	Type       string
	URL        string
	UserAgent  string
	Body       json.RawMessage
}

// Vet validates a single IncomingReport and returns an AcceptedReport stamped
// with the provided time. It returns an error if any field fails validation.
func Vet(report IncomingReport, now time.Time) (AcceptedReport, error) {
	reportType := strings.TrimSpace(report.Type)
	if reportType == "" || len(reportType) > maxTypeLen {
		return AcceptedReport{}, errors.New("invalid type")
	}

	reportURL := strings.TrimSpace(report.URL)
	if reportURL == "" || len(reportURL) > maxURLLen {
		return AcceptedReport{}, errors.New("invalid url")
	}

	parsedURL, err := url.ParseRequestURI(reportURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return AcceptedReport{}, errors.New("invalid url")
	}

	userAgent := strings.TrimSpace(report.UserAgent)
	if len(userAgent) > maxUserAgentLen {
		return AcceptedReport{}, errors.New("invalid user_agent")
	}

	if report.Age < 0 {
		return AcceptedReport{}, errors.New("invalid age")
	}

	if len(report.Body) == 0 {
		return AcceptedReport{}, errors.New("missing body")
	}

	// Validate that body is a JSON object without allocating a full map.
	// RawMessage is guaranteed valid JSON by the decoder, so only the type
	// (first non-whitespace byte) needs checking.
	if trimmed := bytes.TrimSpace(report.Body); len(trimmed) == 0 || trimmed[0] != '{' {
		return AcceptedReport{}, errors.New("invalid body: must be a JSON object")
	}

	return AcceptedReport{
		ReceivedAt: now.UTC(),
		Age:        report.Age,
		Type:       reportType,
		URL:        reportURL,
		UserAgent:  userAgent,
		Body:       report.Body,
	}, nil
}
