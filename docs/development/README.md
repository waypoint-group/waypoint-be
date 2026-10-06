# Developer guide

## Prerequisites

- Go 1.27.1, as declared in `go.mod`.
- Docker with Compose for the local database and Keycloak, and for integration tests (which use Testcontainers).
- `just` for the repository shortcuts.
- `sqlc` to regenerate database access code after editing query or schema source. `golangci-lint` is needed only for `just lint`.

For a running local stack and manual token workflow, see [local setup](local-setup.md). For frontend client configuration and social-login callbacks, see [frontend authentication](../frontend/auth.md).

## Common commands

Run these from the repository root:

| Command                        | Purpose                                                              |
| ------------------------------ | -------------------------------------------------------------------- |
| `just run serve --migrate ...` | Run the API through the CLI; pass server and JWT flags after `serve` |
| `just run migrate`             | Apply pending migrations                                             |
| `just build`                   | Build `./build/waypoint`                                             |
| `just test-unit`               | Run all packages without the integration build tag                   |
| `just test`                    | Run with `-tags integration`; requires Docker                        |
| `just vet`                     | Run `go vet ./...`                                                   |
| `just fmt`                     | Run `go fmt .`                                                       |
| `just lint`                    | Run golangci-lint (optional tool)                                    |
| `just check`                   | Run format, vet, and integration tests                               |

`just run` forwards arguments to `go run ./cmd/waypoint`. Direct equivalents are available in `justfile`. To inspect flags, use `go run ./cmd/waypoint --help` or `go run ./cmd/waypoint serve --help`.

## Tests

Unit tests live beside the packages they exercise. Integration tests are under `tests/integration` and have the `integration` build tag. Their `TestMain` starts one PostgreSQL and one Keycloak container; each Waypoint fixture gets its own database in the shared PostgreSQL container. Docker must be running and able to pull the images.

Run one integration test with, for example, `go test -tags integration ./tests/integration -run TestHealth` (replace the test name with one present in the package). Tests use a separate test realm in `tests/testlib/testdata/waypoint-test-realm.json`; local Compose realm settings are imported from [waypoint-dev-realm.json](../../tests/testlib/testdata/waypoint-dev-realm.json).

## Change workflow

1. Read the relevant package and its tests before changing behavior.
2. If the database contract changes, update SQL migrations and `db/queries`, then regenerate `internal/db/sqlc` with `sqlc generate`.
3. Keep HTTP parsing and status mapping in handlers, business rules in services, and persistence operations in generated queries wrapped by `internal/db`.
4. Add or update tests at the layer where the behavior is observable. Integration tests cover PostgreSQL and real token validation.
5. Run the narrowest useful check, then `just check` when the full Docker-backed suite is available.

Generated files in `internal/db/sqlc` come from `db/queries`, `db/migrations`, and `sqlc.yaml`. Edit those inputs rather than hand-editing generated query code.
