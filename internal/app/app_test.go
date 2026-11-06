package app

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/cruxstack/browser-reporting-api/v2/internal/config"
)

func newApp(t *testing.T) *App {
	t.Helper()
	cfg, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	a, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestServe_StopsWhenContextIsDone(t *testing.T) {
	a := newApp(t)
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- a.Serve(ctx, ln) }()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+ln.Addr().String()+"/v1/manage/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz = %d, want 200", resp.StatusCode)
	}

	cancel()
	select {
	case err := <-served:
		if err != nil {
			t.Errorf("Serve() = %v, want nil after shutdown", err)
		}
	case <-time.After(ShutdownTimeout):
		t.Fatal("Serve() did not return after its context was done")
	}
}

func TestRun_ReturnsListenError(t *testing.T) {
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	a := newApp(t)
	a.cfg.Server.ListenAddr = ln.Addr().String() // already in use
	if err := a.Run(context.Background()); err == nil {
		t.Error("Run() = nil, want an error for an address in use")
	}
}
