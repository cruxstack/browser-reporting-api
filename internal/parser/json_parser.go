package parser

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/cockroachdb/errors"
	"github.com/cruxstack/browser-reporting-api/internal/domain"
)

// JSONBatchParser implements domain.Parser for the application/reports+json
// wire format: a single JSON array of report objects.
type JSONBatchParser struct{}

func NewJSONBatchParser() *JSONBatchParser {
	return &JSONBatchParser{}
}

func (p *JSONBatchParser) ParseBatch(r io.Reader) ([]domain.IncomingReport, error) {
	decoder := json.NewDecoder(r)

	var payload json.RawMessage
	if err := decoder.Decode(&payload); err != nil {
		return nil, errors.Wrap(err, "decode reports payload")
	}

	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 {
		return nil, errors.New("reports payload must include at least one report")
	}

	reports, err := parseReportsPayload(payload)
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

func parseReportsPayload(payload json.RawMessage) ([]domain.IncomingReport, error) {
	if payload[0] == '[' {
		var reports []domain.IncomingReport
		if err := json.Unmarshal(payload, &reports); err != nil {
			return nil, errors.Wrap(err, "decode reports payload")
		}
		return reports, nil
	}

	if payload[0] == '{' {
		return parseLegacyCSPReportPayload(payload)
	}

	return nil, errors.New("reports payload must be a JSON array or object")
}

func parseLegacyCSPReportPayload(payload json.RawMessage) ([]domain.IncomingReport, error) {
	var legacy struct {
		CSPReport json.RawMessage `json:"csp-report"`
	}

	if err := json.Unmarshal(payload, &legacy); err != nil {
		return nil, errors.Wrap(err, "decode reports payload")
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
		return nil, errors.Wrap(err, "decode legacy csp-report body")
	}

	return []domain.IncomingReport{{
		Type: "csp-violation",
		URL:  body.DocumentURI,
		Body: trimmed,
	}}, nil
}
