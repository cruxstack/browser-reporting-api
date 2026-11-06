# Configuration

The service is configured with environment variables. They are validated
when it starts: an invalid value or a variable removed in v2 stops it from
starting, and the error lists every problem at once.

Lists are comma-separated; spaces around items are ignored.

## Server

| Variable          | Description                                                                                       | Default |
| ----------------- | ------------------------------------------------------------------------------------------------- | ------- |
| `APP_LISTEN_ADDR` | TCP address to listen on, `host:port` or `:port`.                                                 | `:8080` |
| `APP_BASE_PATH`   | Path every route is mounted under, such as `/collector`. Letters, digits, and `.` `_` `~` `-` in each segment; leading and trailing slashes are optional. | `/` |
| `APP_LOG_LEVEL`   | `debug`, `info`, `warn`, or `error`. `debug` also logs each batch's counts, each rejected report and why, and bad requests. | `info` |

With `APP_BASE_PATH=/collector` the routes are:

- `POST /collector/v1/reports`
- `GET /collector/v1/manage/healthz`
- `GET /collector/v1/manage/readyz`

## Reports

| Variable                      | Description                                                                 | Default   |
| ----------------------------- | --------------------------------------------------------------------------- | --------- |
| `APP_MAX_BODY_BYTES`          | Largest request body, in bytes. Larger requests get a 413.                  | `1048576` |
| `APP_REPORTS_ALLOWED_ORIGINS` | Origins browsers may send reports from, or `*` for any. See below.          | `*`       |

Each entry of `APP_REPORTS_ALLOWED_ORIGINS` is one of:

- `*`, which allows every origin and can't be combined with other entries
- an origin, `scheme://host[:port]`, such as `https://app.example.com`,
  with no path or trailing slash
- an origin with wildcards: a `*` that is a whole host label matches
  exactly one label (`https://*.example.com` matches
  `https://shop.example.com`, but not `https://example.com` or
  `https://a.b.example.com`), and a `*` port matches any port
  (`http://localhost:*`)

Entries are case-insensitive. A `*` inside a label (`https://*example.com`)
is an error, since it would match look-alike domains such as
`https://attackerexample.com`.

```sh
APP_REPORTS_ALLOWED_ORIGINS=https://app.example.com,https://admin.example.com
APP_REPORTS_ALLOWED_ORIGINS=https://*.example.com,http://localhost:*
```

The allowed origins are a CORS policy, not authentication. They stop
browsers on other sites from sending reports to the service, but any
client that isn't a browser can send a request without an `Origin`
header, or with any `Origin` it likes. Treat every report as untrusted
input.

## Logs and output

- **stdout** carries only accepted reports, one JSON object per line. See
  [the output](reports.md#the-output).
- **stderr** carries the service's own logs as JSON: startup (with the
  version, revision, and settings), shutdown, errors, and at `debug` level
  each request's outcome. Records about a request carry its `request_id`,
  which is taken from the `X-Request-Id` header when the client sends one.

## Health checks

| Endpoint                 | Answers                 |
| ------------------------ | ----------------------- |
| `GET /v1/manage/healthz` | `200 {"status":"ok"}`    |
| `GET /v1/manage/readyz`  | `200 {"status":"ready"}` |

Both are under `APP_BASE_PATH`, and both answer as long as the process
serves requests: the service has no dependencies to check yet.

## Running in production

- **Image:** the [Dockerfile](../Dockerfile) builds a native image for
  `linux/amd64` and `linux/arm64` that runs as the unprivileged user
  `65532`. The service writes no files, so the root filesystem can be read
  only.
- **TLS:** the service serves plain HTTP. Put it behind a load balancer or
  reverse proxy that terminates TLS: the Reporting API only sends reports
  to HTTPS endpoints.
- **Shutdown:** on `SIGTERM` or `SIGINT` the service stops accepting
  connections and waits up to 10 seconds for requests in flight. Give the
  platform's stop timeout more than that.
- **Timeouts:** a request's headers must arrive within 5 seconds, and the
  whole request within 10.
- **Client addresses** aren't used or logged. The service doesn't read
  `X-Forwarded-For` or similar headers, which any client can set.
