// Package ingest runs each batch of reports through the pipeline: parse the
// request body, vet every entry on its own, and write the accepted entries to
// the sink.
package ingest

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/cruxstack/browser-reporting-api/v2/internal/report"
)

// Sink receives each accepted report. Requests are served concurrently, so a
// Sink must be safe for concurrent use.
type Sink interface {
	WriteReport(ctx context.Context, r report.Accepted) error
}

// Result summarizes one batch. It is the reports endpoint's response body.
type Result struct {
	Received int `json:"received"`
	Accepted int `json:"accepted"`
	Rejected int `json:"rejected"`
}

// BadPayloadError reports a request body that isn't a valid batch. It is
// the client's error, unlike an error from the sink.
type BadPayloadError struct {
	Err error
}

func (e *BadPayloadError) Error() string { return e.Err.Error() }
func (e *BadPayloadError) Unwrap() error { return e.Err }

// Service ingests batches of reports.
type Service struct {
	Sink Sink
	// Now stamps accepted reports. It defaults to time.Now.
	Now func() time.Time
}

// Ingest reads one batch from body. A body that can't be parsed is a
// *BadPayloadError. Entries that fail vetting are counted as rejected, and
// the others are still written. An error from the sink stops the batch.
func (s *Service) Ingest(ctx context.Context, body io.Reader) (Result, error) {
	reports, err := report.Parse(body)
	if err != nil {
		return Result{}, &BadPayloadError{Err: err}
	}

	now := s.Now
	if now == nil {
		now = time.Now
	}
	receivedAt := now()

	result := Result{Received: len(reports)}
	for i, in := range reports {
		accepted, err := report.Vet(in, receivedAt)
		if err != nil {
			slog.DebugContext(ctx, "rejected report", "index", i, "type", in.Type, "error", err)
			continue
		}

		if err := s.Sink.WriteReport(ctx, accepted); err != nil {
			return Result{}, fmt.Errorf("write report: %w", err)
		}
		result.Accepted++
	}

	result.Rejected = result.Received - result.Accepted
	return result, nil
}
