package httpapi

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/cruxstack/browser-reporting-api/v2/internal/logging"
)

// logContext adds the request ID (from middleware.RequestID) to every log
// record for the request.
func logContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := logging.WithAttrs(r.Context(), slog.String("request_id", middleware.GetReqID(r.Context())))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// recoverPanic turns a panic into a 500 response and an error log record,
// in place of chi's Recoverer, which prints a plain-text stack to stderr
// between the JSON log lines.
func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			// the server uses this panic to abort a response; let it
			if err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(v)
			}
			slog.ErrorContext(r.Context(), "panic serving request",
				"method", r.Method,
				"path", r.URL.Path,
				"panic", fmt.Sprint(v),
				"stack", string(debug.Stack()),
			)
			writeError(w, http.StatusInternalServerError, "internal error")
		}()
		next.ServeHTTP(w, r)
	})
}
