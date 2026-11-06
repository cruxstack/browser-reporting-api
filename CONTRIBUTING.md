# Contributing

## Setup

You need Go (the version in `go.mod`) and `make`, plus Docker for the
compose demo. Linters and the vulnerability checker run with `go run` at
pinned versions, and every test runs offline. A
[dev container](.devcontainer/devcontainer.json) is included if you prefer
one.

```sh
make help     # list targets
make check    # lint, test, and vulnerability check, as CI runs them
```

## Layout

```text
cmd/server/        entry point
internal/          the service; see docs/architecture.md
e2e/               end-to-end tests through the real wiring
examples/reports/  sample payloads in both formats, sent by the e2e tests
docs/              user documentation
```

## Tests

- `make test` runs everything with the race detector.
- `make test-e2e` runs the end-to-end tests. They build the service with
  `app.New` from environment-style configuration and send requests to a
  real HTTP server; only the report stream is replaced. Prefer an
  end-to-end test for behavior a user would notice.
- A new example payload in `examples/reports` needs a test in `e2e` that
  sends it.

To try a change by hand, `make run` starts the server locally, and
`make report` sends the example reports to it. `make demo` does both in
Docker Compose, with the settings in `.env` (copy `.env.example`).

## Conventions

- Configuration is read only in `internal/config`, and dependencies are
  built only in `internal/app`. Other packages take what they need as
  arguments or small interfaces.
- Every new environment variable has the `APP_` prefix, is validated at
  startup, and is documented in `docs/configuration.md`. A change to the
  endpoint or the output lines is documented in `docs/reports.md`.
- stdout carries only accepted reports. Logs go to stderr through `slog`,
  with snake_case keys and the request context
  (`slog.InfoContext(ctx, ...)`).
- Report contents are untrusted input; see
  [design notes](docs/architecture.md#design-notes).
- Error responses are seen by clients. Return a generic message for server
  errors, and log the details.

## Commits and releases

Pull request titles follow [Conventional Commits](https://www.conventionalcommits.org/)
(`feat:`, `fix:`, `docs:`, ...); a check enforces it. Each merge to `main` is
tagged and released automatically: `fix:`, `perf:`, `chore:`, `docs:`,
`test:`, and `ci:` bump the patch version; `feat:`, `refactor:`, and other
types bump the minor version; and a breaking change, marked with `!`
(`feat!:`, `fix!:`, `refactor!:`) or a `BREAKING CHANGE:` footer, bumps the
major version. A `!` on `chore`, `docs`, `test`, or `ci` is still released
as a patch, so use one of the other types for breaking changes. A release is
tagged only after its binaries and image build. Releases attach
`browser-reporting-api_linux_amd64.tar.gz`,
`browser-reporting-api_linux_arm64.tar.gz`, and `checksums.txt`.

A breaking change to the configuration, the endpoint, or the output lines
needs an upgrade guide in `docs/`. A new major version also changes the
module path (`/v3`), as Go requires.
