# Repository Guidelines

## Project Structure & Module Organization

Waypoint is a Go 1.27.1 and PostgreSQL backend for workspace messaging. `cmd/waypoint/` contains the CLI and server entry point. `internal/waypoint/` wires configuration, routing, and services; domain packages include `internal/user/`, `internal/workspace/`, and `internal/message/`. Keep HTTP parsing and status mapping in handlers, business rules in services, and persistence in `internal/db/`.

SQL sources live in `db/queries/` and `db/migrations/`; generated access code lives in `internal/db/sqlc/`. Unit tests sit beside source files. Integration tests, shared fixtures, and realm assets live under `tests/`. Consult `docs/` for architecture, API, database, and development details.

## Build, Test, and Development Commands

Run commands from the repository root:

- `just build`: compile the CLI to `build/waypoint`.
- `docker compose up -d --wait postgres keycloak`: start local dependencies after configuring the required secrets in `docs/development/local-setup.md`.
- `just run serve --migrate`: start the API and apply migrations; configure `WAYPOINT_DATABASE_URL`, `WAYPOINT_JWT_ISSUER`, and `WAYPOINT_JWT_AUDIENCE` first.
- `just test-unit`: run tests without the integration tag.
- `just test`: run the suite with integration tests; requires Docker.
- `go fmt ./...`, `just vet`, `just lint`: format all packages, run Go vet, and run golangci-lint. Note that `just fmt` currently targets only `.`.
- `sqlc generate`: regenerate database access code after SQL changes.

## Coding Style & Naming Conventions

Use Go formatting with tabs, lowercase package names, exported `PascalCase` identifiers, and unexported `camelCase` identifiers. Follow existing domain file names such as `handler.go`, `service.go`, and `model.go`. Add sequential migration pairs such as `000007_description.up.sql` and `.down.sql`; do not alter released migrations or manually edit generated sqlc files.

## Testing Guidelines

Use Go's `testing` package, `*_test.go` files, and descriptive `Test...` functions. Integration tests use the `integration` build tag and Testcontainers for PostgreSQL and Keycloak. Cover changed behavior at its observable layer. Run focused tests first, then the full suite when Docker is available. No numeric coverage threshold is documented.

## Commit & Pull Request Guidelines

Prefer the history's `feat:`, `fix:`, `refactor:`, `perf:`, and `docs:` prefixes, optionally scoped, such as `refactor(services/user): ...`. Keep commits focused. In PRs, describe behavior changes, link relevant issues, report checks run, and highlight migration or configuration changes.
