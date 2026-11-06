package config

import (
	"log/slog"
	"reflect"
	"strings"
	"testing"
)

func load(vars map[string]string) (*Config, error) {
	return Load(func(k string) (string, bool) {
		v, ok := vars[k]
		return v, ok
	})
}

func TestLoad_Defaults(t *testing.T) {
	cfg, err := load(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := &Config{
		LogLevel: slog.LevelInfo,
		Server:   Server{ListenAddr: ":8080", BasePath: "/"},
		Reports:  Reports{MaxBodyBytes: 1 << 20, AllowedOrigins: []string{"*"}},
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("Load() = %+v, want %+v", cfg, want)
	}
}

func TestLoad_Values(t *testing.T) {
	cfg, err := load(map[string]string{
		"APP_LOG_LEVEL":               "DEBUG",
		"APP_LISTEN_ADDR":             "127.0.0.1:9000",
		"APP_BASE_PATH":               "collector/",
		"APP_MAX_BODY_BYTES":          "65536",
		"APP_REPORTS_ALLOWED_ORIGINS": " https://app.example.com, ,https://*.example.org ",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := &Config{
		LogLevel: slog.LevelDebug,
		Server:   Server{ListenAddr: "127.0.0.1:9000", BasePath: "/collector"},
		Reports:  Reports{MaxBodyBytes: 65536, AllowedOrigins: []string{"https://app.example.com", "https://*.example.org"}},
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("Load() = %+v, want %+v", cfg, want)
	}
}

func TestLoad_BasePath(t *testing.T) {
	for in, want := range map[string]string{
		"":                 "/",
		"/":                "/",
		"//":               "/",
		"collector":        "/collector",
		"/collector/":      "/collector",
		"/api/v1.reports~": "/api/v1.reports~",
	} {
		cfg, err := load(map[string]string{"APP_BASE_PATH": in})
		if err != nil {
			t.Errorf("APP_BASE_PATH=%q: %v", in, err)
			continue
		}
		if cfg.Server.BasePath != want {
			t.Errorf("APP_BASE_PATH=%q: BasePath = %q, want %q", in, cfg.Server.BasePath, want)
		}
	}
}

func TestLoad_Invalid(t *testing.T) {
	tests := map[string]struct {
		vars map[string]string
		want string
	}{
		"log level":               {map[string]string{"APP_LOG_LEVEL": "loud"}, "APP_LOG_LEVEL"},
		"listen addr":             {map[string]string{"APP_LISTEN_ADDR": "8080"}, "APP_LISTEN_ADDR"},
		"max body text":           {map[string]string{"APP_MAX_BODY_BYTES": "1MB"}, "APP_MAX_BODY_BYTES"},
		"max body zero":           {map[string]string{"APP_MAX_BODY_BYTES": "0"}, "APP_MAX_BODY_BYTES"},
		"base path param":         {map[string]string{"APP_BASE_PATH": "/{id}"}, "APP_BASE_PATH"},
		"base path empty":         {map[string]string{"APP_BASE_PATH": "/a//b"}, "APP_BASE_PATH"},
		"base path dots":          {map[string]string{"APP_BASE_PATH": "/a/../b"}, "APP_BASE_PATH"},
		"base path query":         {map[string]string{"APP_BASE_PATH": "/a?b"}, "APP_BASE_PATH"},
		"origin path":             {map[string]string{"APP_REPORTS_ALLOWED_ORIGINS": "https://app.example.com/path"}, "APP_REPORTS_ALLOWED_ORIGINS"},
		"origin slash":            {map[string]string{"APP_REPORTS_ALLOWED_ORIGINS": "https://app.example.com/"}, "APP_REPORTS_ALLOWED_ORIGINS"},
		"origin scheme":           {map[string]string{"APP_REPORTS_ALLOWED_ORIGINS": "app.example.com"}, "APP_REPORTS_ALLOWED_ORIGINS"},
		"origin partial wildcard": {map[string]string{"APP_REPORTS_ALLOWED_ORIGINS": "https://*example.com"}, "whole host label"},
		"any with others":         {map[string]string{"APP_REPORTS_ALLOWED_ORIGINS": "*,https://app.example.com"}, "can't be combined"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := load(tt.vars)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Load() = %v, want an error mentioning %q", err, tt.want)
			}
		})
	}
}

func TestLoad_RemovedVariables(t *testing.T) {
	_, err := load(map[string]string{
		"LISTEN_ADDR":             ":8080",
		"BASE_PATH":               "/",
		"MAX_BODY_BYTES":          "1024",
		"REPORTS_ALLOWED_ORIGINS": "https://app.example.com",
		"REPORTS_ALLOWED_ORIGIN":  "https://app.example.com",
	})
	if err == nil {
		t.Fatal("Load() = nil error, want an error for each removed variable")
	}
	for _, want := range []string{
		"LISTEN_ADDR was removed in v2: use APP_LISTEN_ADDR",
		"BASE_PATH was removed in v2: use APP_BASE_PATH",
		"MAX_BODY_BYTES was removed in v2: use APP_MAX_BODY_BYTES",
		"REPORTS_ALLOWED_ORIGINS was removed in v2: use APP_REPORTS_ALLOWED_ORIGINS",
		"REPORTS_ALLOWED_ORIGIN was removed in v2: use APP_REPORTS_ALLOWED_ORIGINS",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
	}
}

func TestLoad_ReportsEveryProblem(t *testing.T) {
	_, err := load(map[string]string{"APP_LOG_LEVEL": "loud", "APP_MAX_BODY_BYTES": "-1"})
	if err == nil || !strings.Contains(err.Error(), "APP_LOG_LEVEL") || !strings.Contains(err.Error(), "APP_MAX_BODY_BYTES") {
		t.Errorf("Load() = %v, want both problems", err)
	}
}
