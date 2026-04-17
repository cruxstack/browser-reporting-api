package reporting

import (
	"context"
	"io"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/cruxstack/browser-reporting-api/internal/domain"
)

// ServiceOption configures a Service.
type ServiceOption func(*Service)

// WithClock overrides the time source used to stamp accepted reports.
// This is primarily useful for testing.
func WithClock(fn func() time.Time) ServiceOption {
	return func(s *Service) {
		s.now = fn
	}
}

// Service orchestrates report ingestion: parsing, per-entry validation,
// and writing accepted entries to the configured sink.
type Service struct {
	parser domain.Parser
	sink   domain.Sink
	now    func() time.Time
}

func NewService(parser domain.Parser, sink domain.Sink, opts ...ServiceOption) *Service {
	s := &Service{
		parser: parser,
		sink:   sink,
		now:    time.Now,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Service) Ingest(ctx context.Context, body io.Reader) (domain.IngestResult, error) {
	reports, err := s.parser.ParseBatch(body)
	if err != nil {
		return domain.IngestResult{}, &domain.BadPayloadError{Err: err}
	}

	result := domain.IngestResult{Received: len(reports)}
	now := s.now()

	for _, rep := range reports {
		accepted, vetErr := domain.Vet(rep, now)
		if vetErr != nil {
			continue
		}

		if err := s.sink.WriteReport(ctx, accepted); err != nil {
			return domain.IngestResult{}, errors.Wrap(err, "stream accepted report")
		}

		result.Accepted++
	}

	result.Rejected = result.Received - result.Accepted
	return result, nil
}
