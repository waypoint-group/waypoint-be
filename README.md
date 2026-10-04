# Waypoint backend

Waypoint is a Go and PostgreSQL backend for workspace based messaging.

## Quick start

Requirements: Go 1.27.1 (see `go.mod`), Docker with Compose, and a reachable OIDC provider. For the local provider and database, run:

```sh
docker compose up -d --wait postgres keycloak
export WAYPOINT_DATABASE_URL='postgres://postgres:secret@localhost:5433/waypoint?sslmode=disable'
go run ./cmd/waypoint serve --migrate \
  --jwt.issuer=http://localhost:8081/realms/waypoint \
  --jwt.audience=waypoint-api
```

The API listens on port 8080. See [local setup](docs/development/local-setup.md) for account registration, configuration, and smoke checks.

## Documentation

- [Developer guide](docs/development/README.md): setup, commands, tests, and code changes
- [Architecture](docs/architecture/overview.md): packages, request flow, and implemented capabilities
- [HTTP API](docs/api/README.md): routes, authentication, and response behavior
- [Database](docs/database/README.md): schema, migrations, and sqlc workflow
