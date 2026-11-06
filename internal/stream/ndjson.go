// Package stream writes accepted reports to the report stream: stdout, one
// JSON object per line (NDJSON).
package stream

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/cruxstack/browser-reporting-api/v2/internal/report"
)

// receivedAtLayout is RFC 3339 with milliseconds.
const receivedAtLayout = "2006-01-02T15:04:05.000Z07:00"

// NDJSONWriter is an ingest.Sink that writes each report as one JSON line.
// It is safe for concurrent use.
type NDJSONWriter struct {
	// handler is used directly, not through a slog.Logger, because a Logger
	// drops the handler's errors, and a report that can't be written must
	// fail its request.
	handler slog.Handler
}

// NewNDJSONWriter returns a writer to w. Its lines don't depend on the
// service's log level: every accepted report is written.
func NewNDJSONWriter(w io.Writer) *NDJSONWriter {
	return &NDJSONWriter{handler: slog.NewJSONHandler(w, nil)}
}

func (w *NDJSONWriter) WriteReport(ctx context.Context, r report.Accepted) error {
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "accepted report", 0)
	rec.AddAttrs(
		slog.String("received_at", r.ReceivedAt.Format(receivedAtLayout)),
		slog.Int64("age", r.Age),
		slog.String("type", r.Type),
		slog.String("url", r.URL),
		slog.String("user_agent", r.UserAgent),
		// the body is valid JSON; the encoder compacts it onto one line
		slog.Any("body", r.Body),
	)
	if err := w.handler.Handle(ctx, rec); err != nil {
		return fmt.Errorf("write report to stream: %w", err)
	}
	return nil
}
