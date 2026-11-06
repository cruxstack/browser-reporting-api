package ingest

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cruxstack/browser-reporting-api/v2/internal/report"
)

type captureSink struct {
	reports []report.Accepted
	err     error
}

func (s *captureSink) WriteReport(_ context.Context, r report.Accepted) error {
	if s.err != nil {
		return s.err
	}
	s.reports = append(s.reports, r)
	return nil
}

func TestIngest_AcceptsValidAndCountsRejected(t *testing.T) {
	receivedAt := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	sink := &captureSink{}
	s := &Service{Sink: sink, Now: func() time.Time { return receivedAt }}

	body := `[
		{"age":1,"type":"csp-violation","url":"https://site.example","body":{"a":1}},
		{"age":2,"type":"","url":"https://site.example","body":{"a":1}},
		{"age":3,"type":"deprecation","url":"https://site.example/page","body":{"id":"x"}}
	]`
	got, err := s.Ingest(context.Background(), strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if want := (Result{Received: 3, Accepted: 2, Rejected: 1}); got != want {
		t.Errorf("Ingest() = %+v, want %+v", got, want)
	}
	if len(sink.reports) != 2 {
		t.Fatalf("sink got %d reports, want 2", len(sink.reports))
	}
	for _, r := range sink.reports {
		if !r.ReceivedAt.Equal(receivedAt) {
			t.Errorf("ReceivedAt = %v, want %v", r.ReceivedAt, receivedAt)
		}
	}
}

func TestIngest_MalformedBodyIsBadPayload(t *testing.T) {
	s := &Service{Sink: &captureSink{}}

	_, err := s.Ingest(context.Background(), strings.NewReader(`not json`))
	var bad *BadPayloadError
	if !errors.As(err, &bad) {
		t.Errorf("err = %v, want a *BadPayloadError", err)
	}
}

func TestIngest_SinkErrorIsNotBadPayload(t *testing.T) {
	s := &Service{Sink: &captureSink{err: errors.New("disk full")}}

	_, err := s.Ingest(context.Background(), strings.NewReader(`[{"type":"csp-violation","url":"https://site.example","body":{}}]`))
	var bad *BadPayloadError
	if err == nil || errors.As(err, &bad) {
		t.Errorf("err = %v, want a sink error", err)
	}
}
