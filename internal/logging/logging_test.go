package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestContextHandler(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(NewJSONHandler(&buf, slog.LevelInfo))

	ctx := WithAttrs(context.Background(), slog.String("request_id", "req-1"))
	ctx = WithAttrs(ctx, slog.String("origin", "https://app.example.com"))
	log.InfoContext(ctx, "hello", "n", 1)

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]any{"msg": "hello", "request_id": "req-1", "origin": "https://app.example.com", "n": 1.0} {
		if rec[k] != want {
			t.Errorf("%s = %v, want %v", k, rec[k], want)
		}
	}
}

func TestContextHandler_RespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(NewJSONHandler(&buf, slog.LevelInfo))

	log.DebugContext(WithAttrs(context.Background(), slog.String("a", "1")), "hidden")
	if buf.Len() != 0 {
		t.Errorf("logged %q below the level", buf.String())
	}
}

func TestWithAttrs_DoesNotShareBackingArray(t *testing.T) {
	base := WithAttrs(context.Background(), slog.String("a", "1"))
	one := WithAttrs(base, slog.String("b", "1"))
	two := WithAttrs(base, slog.String("b", "2"))

	got1, _ := one.Value(ctxKey{}).([]slog.Attr)
	got2, _ := two.Value(ctxKey{}).([]slog.Attr)
	if got1[1].Value.String() != "1" || got2[1].Value.String() != "2" {
		t.Errorf("attrs leaked between contexts: %v %v", got1, got2)
	}
}
