// Package logging configures the service's structured logs, which carry the
// attributes of the request they were logged for.
package logging

import (
	"context"
	"io"
	"log/slog"
	"slices"
)

// NewJSONHandler returns a JSON handler wrapped with ContextHandler.
func NewJSONHandler(w io.Writer, level slog.Leveler) slog.Handler {
	return ContextHandler(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level}))
}

// ContextHandler wraps h so each record logged with a context carries the
// attributes added to that context with WithAttrs.
func ContextHandler(h slog.Handler) slog.Handler {
	return contextHandler{h}
}

type ctxKey struct{}

// WithAttrs returns a context whose log records carry attrs, in addition to
// any attributes already on ctx.
func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	prev, _ := ctx.Value(ctxKey{}).([]slog.Attr)
	return context.WithValue(ctx, ctxKey{}, append(slices.Clip(prev), attrs...))
}

type contextHandler struct {
	slog.Handler
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, ok := ctx.Value(ctxKey{}).([]slog.Attr); ok {
		r.AddAttrs(attrs...)
	}
	return h.Handler.Handle(ctx, r)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}
