package report

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// The content types the endpoint accepts. Either may carry either body
// format; Parse decides by the body itself.
const (
	// ContentTypeReports is the Reporting API's batch format, sent to
	// endpoints named by the Reporting-Endpoints header and CSP report-to.
	ContentTypeReports = "application/reports+json"
	// ContentTypeCSPReport is the legacy format of the CSP report-uri
	// directive: one violation per request.
	ContentTypeCSPReport = "application/csp-report"
)

// TypeCSPViolation is the Reporting API type of a CSP violation report.
// Legacy report-uri reports are given this type.
const TypeCSPViolation = "csp-violation"

// Parse decodes a request body into its report entries. The body is a
// single JSON value: either a Reporting API batch (an array of reports) or a
// legacy CSP report (an object with a "csp-report" member). An error means
// the body as a whole is malformed.
func Parse(r io.Reader) ([]Incoming, error) {
	decoder := json.NewDecoder(r)

	var payload json.RawMessage
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode reports payload: %w", err)
	}

	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 {
		return nil, errors.New("reports payload must include at least one report")
	}

	reports, err := parsePayload(payload)
	if err != nil {
		return nil, err
	}

	if len(reports) == 0 {
		return nil, errors.New("reports payload must include at least one report")
	}

	// Enforce a single top-level JSON value; trailing junk should be rejected.
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, errors.New("reports payload must contain exactly one JSON value")
	}

	return reports, nil
}

func parsePayload(payload json.RawMessage) ([]Incoming, error) {
	switch payload[0] {
	case '[':
		var reports []Incoming
		if err := json.Unmarshal(payload, &reports); err != nil {
			return nil, fmt.Errorf("decode reports payload: %w", err)
		}
		return reports, nil
	case '{':
		return parseLegacyCSPReport(payload)
	default:
		return nil, errors.New("reports payload must be a JSON array or object")
	}
}

// parseLegacyCSPReport converts a report-uri body into one csp-violation
// entry. The violation keeps its legacy (kebab-case) fields as the body.
func parseLegacyCSPReport(payload json.RawMessage) ([]Incoming, error) {
	var legacy struct {
		CSPReport json.RawMessage `json:"csp-report"`
	}

	if err := json.Unmarshal(payload, &legacy); err != nil {
		return nil, fmt.Errorf("decode reports payload: %w", err)
	}

	trimmed := bytes.TrimSpace(legacy.CSPReport)
	if len(trimmed) == 0 {
		return nil, errors.New("legacy csp payload must include csp-report")
	}
	if trimmed[0] != '{' {
		return nil, errors.New("legacy csp payload must include csp-report object")
	}

	var body struct {
		DocumentURI string `json:"document-uri"`
	}
	if err := json.Unmarshal(trimmed, &body); err != nil {
		return nil, fmt.Errorf("decode legacy csp-report body: %w", err)
	}

	return []Incoming{{
		Type: TypeCSPViolation,
		URL:  body.DocumentURI,
		Body: trimmed,
	}}, nil
}
