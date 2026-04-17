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

- `POST /v1/reports` (or `{BASE_PATH}/v1/reports`)
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

The demo sender payload is stored at `demo/reports.json`.

Send a manual sample report:

```bash
make report
```

`make report` sends the same payload file used by the compose demo:
`demo/reports.json`.

Health check:

```bash
make health
```

Stop containers:

```bash
make down
```

Environment variables:

- `LISTEN_ADDR` (default `:8080`)
- `BASE_PATH` (default `/`)
- `MAX_BODY_BYTES` (default `1048576`)
- `REPORTS_ALLOWED_ORIGINS` (default `*`)

Allowed origin examples:

- `REPORTS_ALLOWED_ORIGINS=*`
- `REPORTS_ALLOWED_ORIGINS=https://app.example.com,https://admin.example.com`
- `REPORTS_ALLOWED_ORIGINS=https://*.example.com,http://localhost:*`

If `BASE_PATH=/collector`, endpoints become:

- `POST /collector/v1/reports`
- `GET /collector/v1/manage/healthz`
- `GET /collector/v1/manage/readyz`

## Development

Run tests:

```bash
go test ./...
```

Make targets used during development:

```bash
make test
make run
make up
make demo-send
make demo
make logs
make report
make health
make down
```

Implementation notes:

- Routes are mounted under configurable `BASE_PATH`.
- Reporting ingestion accepts batched arrays and processes entries
  independently.
- Invalid entries are rejected while valid entries in the same batch are still
  accepted.
- Accepted entries are emitted to stdout in NDJSON format.
