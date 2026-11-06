# browser-reporting-api

A small, self-hosted HTTP service that collects the reports browsers send
with the [Reporting API](https://developer.mozilla.org/en-US/docs/Web/API/Reporting_API),
including Content Security Policy violations, and writes each accepted
report to stdout as one JSON line (NDJSON) for your log pipeline.

## Features

- **Both CSP report formats.** Reporting API batches
  (`application/reports+json`, from `Reporting-Endpoints` and `report-to`)
  and legacy `report-uri` reports (`application/csp-report`) are accepted
  at one endpoint.
- **Every report type.** CSP, deprecation, intervention, COEP, network
  error, permissions policy, and any other type a browser sends.
- **Safe to expose.** Size limits, content-type checks, and per-entry
  vetting, so one bad entry doesn't cost the rest of its batch. CORS is
  limited to the origins you allow, with wildcards.
- **Easy to run and observe.** One static binary or container, configured
  with environment variables that are checked at startup. Reports go to
  stdout and the service's own logs to stderr, both as JSON.

Collected CSP reports help with the client-side script monitoring that
PCI DSS requirements 6.4.3 and 11.6.1 ask of payment pages; see the
[PCI Security Standards Council's guidance](https://blog.pcisecuritystandards.org/new-information-supplement-payment-page-security-and-preventing-e-skimming).

## How it works

Point a page's reporting headers at the service:

```http
Reporting-Endpoints: default="https://reports.example.com/v1/reports"
Content-Security-Policy: script-src 'self'; report-to default; report-uri https://reports.example.com/v1/reports
```

Browsers `POST` their reports to `/v1/reports`, and each accepted report is
written to stdout as one line:

```json
{"time":"2026-11-06T12:30:00.123456789Z","level":"INFO","msg":"accepted report","received_at":"2026-11-06T12:30:00.123Z","age":1,"type":"csp-violation","url":"https://app.example.com/checkout","user_agent":"Mozilla/5.0 ...","body":{"effectiveDirective":"script-src-elem","blockedURL":"https://cdn.example.net/x.js","disposition":"enforce"}}
```

See [reports](docs/reports.md) for the endpoint, the formats, and the
output.

## Running

Build and run the container image:

```sh
docker build -t browser-reporting-api .
docker run --rm -p 8080:8080 \
  -e APP_REPORTS_ALLOWED_ORIGINS=https://app.example.com \
  browser-reporting-api
```

Or download `browser-reporting-api_linux_amd64.tar.gz` (or `arm64`) from
the [latest release](https://github.com/cruxstack/browser-reporting-api/releases/latest)
and run the binary. Every setting is an environment variable; see
[configuration](docs/configuration.md).

## Local development

Start the service in Docker Compose, send the example reports in both
formats, and print what it wrote:

```sh
make demo
make down
```

`make help` lists the other targets; [CONTRIBUTING](CONTRIBUTING.md) covers
tests and conventions.

## Documentation

| Document                                   | Covers                                                             |
| ------------------------------------------ | ------------------------------------------------------------------ |
| [Reports](docs/reports.md)                 | Browser setup, the endpoint, the accepted formats, the output.     |
| [Configuration](docs/configuration.md)     | Environment variables, logs, health checks, running in production. |
| [Architecture](docs/architecture.md)       | Packages, the request flow, failure handling, extending it.        |
| [Upgrading to v2](docs/upgrading-to-v2.md) | Breaking changes from v1.                                          |

## License

[MIT](LICENSE)
