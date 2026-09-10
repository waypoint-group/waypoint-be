# Waypoint backend

Waypoint is a Go/PostgreSQL backend for workspace-based messaging.

## Dependencies

- Go **1.27.1**, as declared in `go.mod`.
- PostgreSQL **18**; Docker with Compose is the provided local setup.
- Development tools: **just** for task shortcuts, **sqlc** for SQL code generation
  and optionally **golangci-lint** for `just lint`.
- Optional database UI: **pgAdmin 4**, included in Compose.

## Local setup and usage

From the repository root, with Docker running:

```sh
go mod download
docker compose up -d --wait db
export WAYPOINT_DATABASE_URL='postgres://postgres:secret@localhost:5433/waypoint?sslmode=disable'
go run ./cmd/waypoint serve --migrate
```

This applies pending migrations and starts the API on port **8080**. To apply
migrations separately, run `go run ./cmd/waypoint migrate`.

```sh
curl http://localhost:8080/healthz  # OK
curl http://localhost:8080/readyz   # READY when PostgreSQL is reachable
```

The application reads exported environment variables, not `.env` files.
Compose accepts `.env` overrides for `POSTGRES_USER`, `POSTGRES_PASSWORD`,
`POSTGRES_DB`, and `POSTGRES_PORT`; update the connection URL to match.

For pgAdmin, run `docker compose up -d pgadmin`, open
<http://localhost:5050>, and log in with `admin@example.com` / `secret` (local
defaults). The database connection is preconfigured. Stop the containers with
`docker compose down`; database data persists in a named volume.

## Development

```sh
just run serve --migrate # Run through the CLI
just build               # Build ./build/waypoint
just test-unit           # go test ./...
just test                # go test --tags integration ./...
just vet                 # go vet ./...
just fmt                 # go fmt ./...
just lint                # Requires golangci-lint
```

Run the built binary with `./build/waypoint serve` using the same environment variables.
Use `go run ./cmd/waypoint --help` to inspect CLI commands.
