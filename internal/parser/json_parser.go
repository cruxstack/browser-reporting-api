package parser

import (
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
	var reports []domain.IncomingReport
	if err := decoder.Decode(&reports); err != nil {
		return nil, errors.Wrap(err, "decode reports payload")
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
