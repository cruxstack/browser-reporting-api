package domain

import (
	"context"
	"io"
)

// Parser parses a batch of incoming reports from a reader.
type Parser interface {
	ParseBatch(r io.Reader) ([]IncomingReport, error)
}

// Sink writes an accepted report to an output destination.
type Sink interface {
	WriteReport(ctx context.Context, report AcceptedReport) error
}

// IngestResult summarises the outcome of a single ingest call.
type IngestResult struct {
	Received int `json:"received"`
	Accepted int `json:"accepted"`
	Rejected int `json:"rejected"`
}

// BadPayloadError wraps errors caused by a malformed or invalid client
// payload. Callers use errors.As to distinguish 4xx from 5xx conditions.
type BadPayloadError struct {
	Err error
}

func (e *BadPayloadError) Error() string { return e.Err.Error() }
func (e *BadPayloadError) Unwrap() error { return e.Err }
