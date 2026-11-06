# Architecture

One HTTP server receives report batches from browsers. Each request goes
through the router's middleware, the reports handler checks it, the ingest
service parses and vets its reports, and each accepted report is written to
the report stream on stdout.

```text
browser ─▶ httpapi ───────────────▶ ingest.Service ─────────────────────▶ stream
           request ID, CORS         report.Parse ─▶ report.Vet            stdout, one
           (origin.Matcher),        (batch or       (each entry           JSON line
           content type, size       legacy CSP)      on its own)          per report
```

## Packages

| Package            | Responsibility                                                                                         |
| ------------------ | ------------------------------------------------------------------------------------------------------ |
| `cmd/server`       | Entry point: logging, configuration, `app.New`, and `App.Run` until `SIGTERM`.                         |
| `internal/app`     | Composition root. Builds every dependency from the configuration and runs the server; options replace the report stream and the clock in tests. |
| `internal/config`  | Reads and validates environment variables. No other package reads the environment.                   |
| `internal/httpapi` | The router and middleware, the reports endpoint (CORS, content type, size limit, status codes), and the health endpoints. |
| `internal/ingest`  | The pipeline for one batch: parse, vet each entry, write accepted reports to a `Sink`.                 |
| `internal/report`  | The report types, `Parse` for both wire formats, and `Vet`.                                            |
| `internal/origin`  | Allowed-origin entries and wildcards, shared by config validation and CORS.                            |
| `internal/stream`  | The NDJSON `Sink` that writes reports to stdout.                                                        |
| `internal/logging` | The JSON log handler, which adds each request's attributes (`request_id`) to its records.              |
| `e2e`              | End-to-end tests: configuration from variables, `app.New`, and a real HTTP server.                     |

Dependencies point one way: `app` knows every package, and the others
depend on small interfaces (`httpapi.Ingester`, `ingest.Sink`) rather than
on configuration, which keeps them testable with fakes.

## Failure handling

| Failure                              | Result                                                                    |
| ------------------------------------ | ------------------------------------------------------------------------- |
| Invalid configuration                | The server doesn't start, and the error lists every problem.             |
| Origin not allowed                   | 403. Nothing is written.                                                  |
| Wrong content type, or body too large | 415 or 413. Nothing is written.                                          |
| Malformed body                       | 400. Nothing is written.                                                  |
| Invalid report in a batch            | Counted as rejected; the rest of the batch is written.                    |
| A report can't be written to stdout  | 500, logged. Reports earlier in the batch may already be written; browsers may send the batch again. |
| Panic in a handler                   | 500, logged as JSON with the stack.                                       |

## Design notes

- **Configuration fails fast.** Invalid values and removed variables stop
  the server at startup, where a deployment check notices, instead of
  silently changing behavior, such as falling back to allowing every
  origin.
- **Everything is built at startup,** in `app.New`, so a problem with a
  dependency surfaces before the first request.
- **stdout is data, stderr is logs.** The report stream never mixes with
  the service's logs, so a log pipeline can route them apart, and the
  stream doesn't depend on `APP_LOG_LEVEL`.
- **Reports are untrusted input.** Anyone can send any report; CORS only
  stops browsers on other sites. Every URL, user agent, and body in a
  report may be crafted. Code that acts on report contents must not fetch
  a URL from a report without guarding against requests to internal
  addresses (resolve the host and refuse loopback, private, and link-local
  addresses, and limit redirects, size, and time), and must not pass report
  contents to another system as instructions.
- **Client addresses aren't trusted.** The service doesn't read
  `X-Forwarded-For` or similar headers. Anything that needs the client's
  address, such as rate limiting, must take it only from a proxy it
  trusts.

## Extending

### A new consumer of reports

Something that acts on accepted reports is an `ingest.Sink`:

1. Add `internal/<name>` with a type whose `WriteReport` handles one
   `report.Accepted`. It runs inside the request, so keep it fast and hand
   slow work, such as network calls, to background work.
2. Wire it in `app.New`. `ingest.Service` writes to one sink, so add a
   sink that calls several in order, and decide what a failure means for
   each: an error fails the request, and the browser may send the whole
   batch again, which repeats what the other sinks already did.
3. Add an `app.With...` option to replace its external dependency in tests,
   and an end-to-end test through `app.New`.

### Background work

Work that runs outside requests starts and stops with the server in
`App.Serve`: start it before serving, stop it after the HTTP server has
shut down (so requests in flight can still hand it work), and give it the
rest of `ShutdownTimeout` to finish. Check its dependencies in `readyz`.

### A new setting

Add it to `internal/config` with the `APP_` prefix, in the section of
`Config` it belongs to (or a new one), validate it in `validate`, and
document it in [configuration](configuration.md). Renaming or removing a
variable is a breaking change: add the old name to `removed` so setting it
is an error, and describe the change in an upgrade guide.
