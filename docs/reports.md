# Reports

## Sending reports to the service

Browsers send reports to the endpoints a page names in its response
headers. The service accepts both ways of naming one:

```http
Reporting-Endpoints: default="https://reports.example.com/v1/reports"
Content-Security-Policy: script-src 'self'; report-to default; report-uri https://reports.example.com/v1/reports
```

- `Reporting-Endpoints` names an endpoint for the
  [Reporting API](https://developer.mozilla.org/en-US/docs/Web/API/Reporting_API).
  CSP's `report-to` directive, and other features such as
  `Permissions-Policy` and COEP, send their reports there in batches.
- `report-uri` is CSP's legacy directive. Browsers that support
  `report-to` ignore it when both are set, so keep it for older browsers.

Use `Content-Security-Policy-Report-Only` to collect violations without
blocking anything, for example to build an inventory of the scripts a page
loads before enforcing a policy.

If the service runs on a different origin than your pages, which it
usually does, allow those origins with
[`APP_REPORTS_ALLOWED_ORIGINS`](configuration.md#reports).

## The endpoint

`POST /v1/reports`, under `APP_BASE_PATH` when it is set.

| Request              | Accepted                                                                     |
| -------------------- | ---------------------------------------------------------------------------- |
| `Content-Type`       | `application/reports+json` or `application/csp-report`, with any parameters. |
| Body                 | One JSON value, at most `APP_MAX_BODY_BYTES` (1 MiB by default).             |
| `Origin`             | An allowed origin, or none (requests that aren't cross-origin browser requests). |

The body is either a Reporting API batch or a legacy CSP report. The
service decides by the body, not the content type, since either content
type may carry either format.

**A Reporting API batch** is an array of reports:

```json
[
  {
    "age": 1,
    "type": "csp-violation",
    "url": "https://app.example.com/checkout",
    "user_agent": "Mozilla/5.0 ...",
    "body": {"blockedURL": "https://cdn.example.net/x.js", "effectiveDirective": "script-src-elem"}
  }
]
```

**A legacy CSP report** is one violation, in an object with a `csp-report`
member. It becomes one `csp-violation` report, whose `url` is the
violation's `document-uri` and whose body is the violation with its legacy
(kebab-case) field names:

```json
{"csp-report": {"document-uri": "https://app.example.com/checkout", "blocked-uri": "https://cdn.example.net/x.js", "violated-directive": "script-src-elem"}}
```

[`examples/reports`](../examples/reports) has a sample of each. The tests
send both through the service, so they stay valid.

### Vetting

Each report in a batch is vetted on its own. A report is rejected, and the
rest of its batch is still accepted, when:

- it isn't an object, or a field has the wrong JSON type
- `type` is empty or longer than 128 characters
- `url` isn't an absolute URL, or is longer than 4096 characters
- `user_agent` is longer than 1024 characters
- `age` is negative
- `body` is missing or isn't an object

### Responses

| Status | When                                                                     | Body                                            |
| ------ | ------------------------------------------------------------------------ | ----------------------------------------------- |
| 200    | The body was a valid batch or CSP report, even if some reports were rejected. | `{"received": 3, "accepted": 2, "rejected": 1}` |
| 400    | The body isn't a JSON array or an object with `csp-report`, is an empty array, or has data after its JSON value. | `{"error": "..."}` |
| 403    | The `Origin` isn't allowed. Nothing is written.                          | `{"error": "origin not allowed"}`               |
| 413    | The body is larger than `APP_MAX_BODY_BYTES`.                            | `{"error": "payload too large"}`                |
| 415    | The content type isn't one of the two accepted.                          | `{"error": "..."}`                              |
| 500    | A report couldn't be written to stdout. The error is logged.             | `{"error": "failed to ingest reports"}`         |

Browsers don't read the response, but they may send a report again after an
error. A 500 is the only response after which some of a batch may already
have been written.

`OPTIONS /v1/reports` answers CORS preflight requests from allowed origins
with `204`, and allows browsers to cache the answer for two hours.

## The output

Each accepted report is written to stdout as one JSON object per line. The
fields are:

| Field         | Value                                                                   |
| ------------- | ----------------------------------------------------------------------- |
| `time`        | When the line was written.                                              |
| `level`, `msg` | Always `INFO` and `accepted report`.                                   |
| `received_at` | When the service received the batch, RFC 3339 in UTC with milliseconds. |
| `age`         | Milliseconds between the report's creation and its upload, as the browser sent it (0 for legacy CSP reports). |
| `type`        | The report type, such as `csp-violation` or `deprecation`.              |
| `url`         | The URL of the document the report is about.                            |
| `user_agent`  | The browser's user agent, as sent in the report (empty for legacy CSP reports). |
| `body`        | The report's body, as the browser sent it.                              |

A report's time of creation is about `received_at` minus `age`. Every
accepted report is written whatever `APP_LOG_LEVEL` is. The service's own
logs go to stderr, so stdout carries only reports.
