// Package e2e tests the service end to end: configuration is read from
// environment variables and wired by app.New exactly as in production, and
// requests go through a real HTTP server. Only the report stream (stdout) is
// replaced, with a buffer the tests read.
package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cruxstack/browser-reporting-api/v2/internal/app"
	"github.com/cruxstack/browser-reporting-api/v2/internal/config"
	"github.com/cruxstack/browser-reporting-api/v2/internal/ingest"
)

const examples = "../examples/reports/"

// receivedAt is the time every accepted report is stamped with.
var receivedAt = time.Date(2026, 11, 6, 12, 30, 0, 0, time.UTC)

// env is one configured instance of the service.
type env struct {
	t *testing.T
	// url is the server's URL, including the base path.
	url    string
	stream *syncBuffer
}

// newEnv builds the service from vars, which are the environment variables
// the server would be started with.
func newEnv(t *testing.T, vars map[string]string) *env {
	t.Helper()
	stream := &syncBuffer{}
	e := newEnvWithStream(t, vars, stream)
	e.stream = stream
	return e
}

// newEnvWithStream builds the service with w as its report stream. The
// env's reports method can't read w.
func newEnvWithStream(t *testing.T, vars map[string]string, w io.Writer) *env {
	t.Helper()
	cfg, err := config.Load(func(k string) (string, bool) { v, ok := vars[k]; return v, ok })
	if err != nil {
		t.Fatal(err)
	}

	a, err := app.New(cfg,
		app.WithReportWriter(w),
		app.WithClock(func() time.Time { return receivedAt }),
	)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(a.Handler())
	t.Cleanup(srv.Close)
	return &env{t: t, url: srv.URL + strings.TrimSuffix(cfg.Server.BasePath, "/")}
}

// response is a finished HTTP response.
type response struct {
	status int
	header http.Header
	body   []byte
}

// result decodes the response of the reports endpoint.
func (r response) result(t *testing.T) ingest.Result {
	t.Helper()
	var res ingest.Result
	if err := json.Unmarshal(r.body, &res); err != nil {
		t.Fatalf("decode response %q: %v", r.body, err)
	}
	return res
}

// do sends a request to path, relative to the base path.
func (e *env) do(method, path string, header map[string]string, body string) response {
	e.t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, e.url+path, strings.NewReader(body))
	if err != nil {
		e.t.Fatal(err)
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		e.t.Fatal(err)
	}
	return response{status: resp.StatusCode, header: resp.Header, body: b}
}

// postReports sends body to the reports endpoint as contentType.
func (e *env) postReports(contentType, body string) response {
	e.t.Helper()
	return e.do(http.MethodPost, "/v1/reports", map[string]string{"Content-Type": contentType}, body)
}

// reports returns the lines written to the report stream, decoded.
func (e *env) reports() []map[string]any {
	e.t.Helper()
	var out []map[string]any
	sc := bufio.NewScanner(bytes.NewReader(e.stream.Bytes()))
	for sc.Scan() {
		var line map[string]any
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			e.t.Fatalf("report stream line %q is not JSON: %v", sc.Text(), err)
		}
		out = append(out, line)
	}
	return out
}

func readExample(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(examples + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// syncBuffer is a bytes.Buffer safe for the server's goroutines to write
// while a test reads it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return bytes.Clone(b.buf.Bytes())
}
