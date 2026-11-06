# Upgrading to v2

v2 renames the environment variables and checks them more strictly. The
endpoint, the accepted formats, and the output lines are unchanged.

Upgrade checklist:

1. Rename the variables below wherever the service is deployed.
2. Check `APP_REPORTS_ALLOWED_ORIGINS` and `APP_BASE_PATH` against the
   [stricter checks](#stricter-checks).
3. Start the service and confirm it logs `starting`. A configuration
   problem stops it with an error that names every variable to fix.
4. Pin the version you deploy. Building from a branch picks up breaking
   changes without warning.

## Configuration

Every variable now has the `APP_` prefix. v2 refuses to start when a v1
variable is set to a non-empty value, and names the replacement in the
error, so an old configuration can't silently fall back to a default, such
as allowing every origin.

| v1                        | v2                            |
| ------------------------- | ----------------------------- |
| `LISTEN_ADDR`             | `APP_LISTEN_ADDR`             |
| `BASE_PATH`               | `APP_BASE_PATH`               |
| `MAX_BODY_BYTES`          | `APP_MAX_BODY_BYTES`          |
| `REPORTS_ALLOWED_ORIGINS` | `APP_REPORTS_ALLOWED_ORIGINS` |
| `REPORTS_ALLOWED_ORIGIN`  | `APP_REPORTS_ALLOWED_ORIGINS` |

`APP_LOG_LEVEL` is new (`info` by default). The defaults are unchanged.

## Stricter checks

These settings started in v1 but stop v2 at startup:

- **A wildcard inside a host label,** such as `https://*example.com`. In v1
  it matched any characters, including look-alike domains such as
  `https://attackerexample.com`. Use `https://*.example.com`, which matches
  one whole label.
- **A wildcard anywhere but a whole host label or the whole port,** such
  as `https://example.com/*`, which never matched, or
  `https://example.com:8*`, which matched every port starting with 8. Use
  `:*` for any port.
- **`*` with other origins,** such as `*,https://app.example.com`. `*`
  already allows every origin; use it alone.
- **An `APP_BASE_PATH` with characters other than letters, digits, and
  `. _ ~ -` in its segments,** or with empty, `.`, or `..` segments.

## Behavior changes

- **Allowed origins are case-insensitive.** In v1 an entry with capitals,
  such as `https://App.Example.com`, never matched, because browsers send
  origins in lowercase.
- **A wildcard label matches exactly one label,** and never an empty one or
  part of the port. `https://app.*` no longer matches
  `https://app.evil:8443`.
- **One malformed report no longer fails its batch.** A report with a field
  of the wrong type, or one that isn't an object, is counted as rejected
  and the rest of the batch is written. v1 answered 400 and wrote nothing.
- **A failed write to stdout is an error.** v1 answered 200 and counted the
  report as accepted. v2 answers 500 and logs the failure.
- **Preflight responses can be cached** for two hours
  (`Access-Control-Max-Age: 7200`), so browsers send fewer preflights.
- **Report bodies are written as sent.** v1 decoded and re-encoded them,
  which sorted their keys and could round large numbers. The fields are the
  same.
- **Logs:** the service's logs carry a `request_id`, panics and server
  errors are logged as JSON, and `APP_LOG_LEVEL=debug` logs each batch's
  counts and each rejected report.

## Building and running

- **Go 1.27.3 or later** is required to build. The module path is now
  `github.com/cruxstack/browser-reporting-api/v2`.
- **The image is native for its platform.** v1 always built an amd64
  binary, which failed on arm64 hosts without emulation. v2 images run as
  the numeric user `65532` instead of the named user `appuser`.
- **Releases attach binaries:** `browser-reporting-api_linux_amd64.tar.gz`,
  `browser-reporting-api_linux_arm64.tar.gz`, and `checksums.txt`.
