# Architecture overview

Waypoint is a single Go HTTP service backed by PostgreSQL. `cmd/waypoint` parses CLI configuration, discovers the configured OIDC provider and JWKS endpoint, then constructs the app in `internal/waypoint`. The app wires a standard-library `http.ServeMux`, domain services, and a PostgreSQL pool.

## Request and persistence flow

```text
cmd/waypoint
    -> internal/waypoint (configuration, services, routes)
        -> domain packages (HTTP handlers and services; for example, internal/user)
            -> internal/db (pool and transaction wrapper)
                -> internal/db/sqlc (generated PostgreSQL queries)
                    -> PostgreSQL
```

Handlers decode HTTP requests, validate path IDs, authenticate where required, and map domain errors to HTTP responses. Services own business rules and coordinate transactions. `internal/db/sqlc` is generated from `db/queries` and `db/migrations`; migrations are embedded by `db/migrations`.

## Package responsibilities

Domain packages under `internal/` group related handlers, services, and models. For example, `internal/user` owns local profiles and their external identities. Apply the same handler/service/persistence boundaries to each domain; adding a domain does not change that division of responsibility.

Shared infrastructure supports those packages:

- `cmd/waypoint`: `serve` and `migrate` CLI commands, flags, environment mapping, and OIDC/JWKS setup.
- `internal/waypoint`: app construction, route registration, configuration, service lifecycle.
- `internal/middleware`: OIDC discovery types and RS256 bearer token validation.
- `internal/httpx`: request JSON, response JSON, and path UUID helpers.
- `internal/db`: PostgreSQL connection pool, transaction wrapper, and constraint error helpers.
- `db/queries`, `db/migrations`: handwritten persistence contract and schema history.
- `tests/integration`, `tests/testlib`: container-backed HTTP integration tests and fixtures.

Service construction in [services.go](../../internal/waypoint/services.go) and route registration in [router.go](../../internal/waypoint/router.go) show which capabilities are exposed by the running app. A domain package or database table alone does not imply an HTTP endpoint.
