// Package stream writes accepted reports to the report stream: stdout, one
// JSON object per line (NDJSON).
package stream

import (
	"context"
	"io"
	"log/slog"

	"github.com/cruxstack/browser-reporting-api/v2/internal/report"
)

// receivedAtLayout is RFC 3339 with milliseconds.
const receivedAtLayout = "2006-01-02T15:04:05.000Z07:00"

// NDJSONWriter is an ingest.Sink that writes each report as one JSON line.
// It is safe for concurrent use.
type NDJSONWriter struct {
	logger *slog.Logger
}

// NewNDJSONWriter returns a writer to w. Its lines don't depend on the
// service's log level: every accepted report is written.
func NewNDJSONWriter(w io.Writer) *NDJSONWriter {
	return &NDJSONWriter{
		logger: slog.New(slog.NewJSONHandler(w, nil)),
	}
}

func (w *NDJSONWriter) WriteReport(ctx context.Context, r report.Accepted) error {
	w.logger.InfoContext(
		ctx,
		"accepted report",
		"received_at", r.ReceivedAt.Format(receivedAtLayout),
		"age", r.Age,
		"type", r.Type,
		"url", r.URL,
		"user_agent", r.UserAgent,
		// the body is valid JSON; the encoder compacts it onto one line
		"body", r.Body,
	)
	return nil
}
