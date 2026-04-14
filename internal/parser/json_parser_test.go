package parser

import (
	"strings"
	"testing"
)

func TestParseBatchAcceptsSingleArrayValue(t *testing.T) {
	reports, err := NewJSONBatchParser().ParseBatch(strings.NewReader(`[{"age":1,"type":"csp-violation","url":"https://site.example","body":{"x":1}}]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
}

func TestParseBatchRejectsEmptyArray(t *testing.T) {
	_, err := NewJSONBatchParser().ParseBatch(strings.NewReader(`[]`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseBatchRejectsTrailingTopLevelValue(t *testing.T) {
	_, err := NewJSONBatchParser().ParseBatch(strings.NewReader(`[{"age":1,"type":"csp-violation","url":"https://site.example","body":{"x":1}}]{"extra":true}`))
	if err == nil {
		t.Fatal("expected error")
	}
}
