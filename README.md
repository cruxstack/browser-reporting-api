# browser-reporting-api

Simple, self-hosted Go service for browser Reporting API ingestion.

## General

### What

`browser-reporting-api` is an HTTP service that receives browser reporting
payloads (`application/reports+json` and legacy `application/csp-report` from
`report-uri`), validates each report entry, and streams accepted entries to
stdout as NDJSON (one JSON object per line).

The payload format and reporting behavior align with the browser Reporting API
documented by
[MDN](https://developer.mozilla.org/en-US/docs/Web/API/Reporting_API) and
[Chrome](https://developer.chrome.com/docs/capabilities/web-apis/reporting-api).

Endpoints:

- `POST /v1/reports` (or `{APP_BASE_PATH}/v1/reports`)
- `GET /v1/manage/healthz`
- `GET /v1/manage/readyz`

### Why

- Simple to run and reason about
- Safe enough for ingestion (size limits, content-type checks, per-entry
  vetting)
- Easy to self-host and observe (accepted reports stream to stdout)
- Collected browser reports (including CSP report traffic) may support
  client-side monitoring and script-governance such as those relevant to PCI DSS
  payment-page security guidance for Requirements 6.4.3 and 11.6.1 from the
  [PCI Security Standards Council](https://blog.pcisecuritystandards.org/new-information-supplement-payment-page-security-and-preventing-e-skimming)

### How

Run locally with Go:

```bash
go run ./cmd/server
```

Run a local demo with Docker Compose:

```bash
make demo
```

This starts the API, sends a sample batched report payload, and prints API logs.
To keep watching streamed report lines:

```bash
make logs
```

The demo sender payloads are stored in `examples/reports/`.

Send a manual sample report:

```bash
make report
```

`make report` sends the same payload files used by the compose demo.

Health check:

```bash
make health
```

Stop containers:

```bash
make down
```

Environment variables:

- `APP_LISTEN_ADDR` (default `:8080`)
- `APP_BASE_PATH` (default `/`)
- `APP_MAX_BODY_BYTES` (default `1048576`)
- `APP_REPORTS_ALLOWED_ORIGINS` (default `*`)
- `APP_LOG_LEVEL` (default `info`)

Allowed origin examples:

- `APP_REPORTS_ALLOWED_ORIGINS=*`
- `APP_REPORTS_ALLOWED_ORIGINS=https://app.example.com,https://admin.example.com`
- `APP_REPORTS_ALLOWED_ORIGINS=https://*.example.com,http://localhost:*`

If `APP_BASE_PATH=/collector`, endpoints become:

- `POST /collector/v1/reports`
- `GET /collector/v1/manage/healthz`
- `GET /collector/v1/manage/readyz`

## Development

Run lint, tests, and the vulnerability check, as CI does:

```bash
make check
```

`make help` lists every target.

Implementation notes:

- Routes are mounted under configurable `APP_BASE_PATH`.
- Reporting ingestion accepts batched arrays and processes entries
  independently.
- Invalid entries are rejected while valid entries in the same batch are still
  accepted.
- Accepted entries are emitted to stdout in NDJSON format.
