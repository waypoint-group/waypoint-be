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
docker compose up -d --wait postgres keycloak
export WAYPOINT_DATABASE_URL='postgres://postgres:secret@localhost:5433/waypoint?sslmode=disable'
go run ./cmd/waypoint serve --migrate \
  --jwt.issuer=http://localhost:8081/realms/waypoint \
  --jwt.audience=waypoint-api
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

### Manual authentication testing

Open <http://localhost:8081/realms/waypoint/account/>, choose **Sign in** if shown,
then **Register** to create your own Keycloak account. The manual realm contains
no seeded application users. Registering creates your login identity; the
Waypoint API creates your application profile separately.

Request an access token using your registered credentials:

```sh
curl --fail-with-body http://localhost:8081/realms/waypoint/protocol/openid-connect/token \
  --data-urlencode 'grant_type=password' \
  --data-urlencode 'client_id=waypoint-manual' \
  --data-urlencode 'username=YOUR_USERNAME' \
  --data-urlencode 'password=YOUR_PASSWORD'
```

Copy the response's `access_token` into `TOKEN`, then create your Waypoint profile:

```sh
TOKEN='PASTE_ACCESS_TOKEN_HERE'
curl --fail-with-body http://localhost:8080/users \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","display_name":"Your Name"}'
curl --fail-with-body http://localhost:8080/me -H "Authorization: Bearer $TOKEN"
```

The `waypoint-manual` client enables direct grants for local command-line testing.
Browser registration uses Keycloak's built-in account client.
The administration console is at <http://localhost:8081/admin/>, with local
defaults `admin` / `secret`. This administrator belongs to the `master` realm.
Compose supports `KEYCLOAK_PORT`, `KEYCLOAK_ADMIN_USERNAME`, and
`KEYCLOAK_ADMIN_PASSWORD` overrides. If you change the port, update the issuer and
URLs above too.

Keycloak stores accounts in the `keycloak_data` volume. The inline realm JSON in
Compose is imported only on first creation; subsequent restarts preserve the
existing realm. Use the admin console to change an existing realm.

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
