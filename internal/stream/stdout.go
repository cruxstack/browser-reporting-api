package stream

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"

	"github.com/cockroachdb/errors"

	"github.com/cruxstack/browser-reporting-api/internal/domain"
)

// NDJSONWriter implements domain.Sink, emitting one JSON object per line to
// the provided writer (NDJSON / JSON Lines format).
type NDJSONWriter struct {
	logger *slog.Logger
}

func NewNDJSONWriter(w io.Writer) *NDJSONWriter {
	return &NDJSONWriter{
		logger: slog.New(slog.NewJSONHandler(w, nil)),
	}
}

func (w *NDJSONWriter) WriteReport(_ context.Context, report domain.AcceptedReport) error {
	var body map[string]any
	if err := json.Unmarshal(report.Body, &body); err != nil {
		return errors.Wrap(err, "decode report body")
	}

	w.logger.Info(
		"accepted report",
		"received_at", report.ReceivedAt.Format("2006-01-02T15:04:05.000Z07:00"),
		"age", report.Age,
		"type", report.Type,
		"url", report.URL,
		"user_agent", report.UserAgent,
		"body", body,
	)

	return nil
}
