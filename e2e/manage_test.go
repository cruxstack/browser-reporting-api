package e2e

import (
	"net/http"
	"testing"
)

func TestManage_HealthEndpoints(t *testing.T) {
	e := newEnv(t, nil)

	for path, want := range map[string]string{
		"/v1/manage/healthz": "{\"status\":\"ok\"}\n",
		"/v1/manage/readyz":  "{\"status\":\"ready\"}\n",
	} {
		resp := e.do(http.MethodGet, path, nil, "")
		if resp.status != http.StatusOK || string(resp.body) != want {
			t.Errorf("GET %s = %d %q, want 200 %q", path, resp.status, resp.body, want)
		}
		if got := resp.header.Get("Content-Type"); got != "application/json" {
			t.Errorf("GET %s Content-Type = %q, want application/json", path, got)
		}
	}
}
