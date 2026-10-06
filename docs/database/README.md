# Database guide

Waypoint uses PostgreSQL 18 and `pgx/v5`. `internal/db.Database` owns a connection pool and exposes generated sqlc queries; `internal/db.Transaction` binds generated queries to a transaction. Services use transactions when related changes or permission checks must be atomic.

## Schema and constraints

The [numbered migrations](../../db/migrations/) define the schema and database-enforced constraints. Consult the [query sources](../../db/queries/) for persistence operations and the relevant domain service for rules enforced by application code.

For example, creating a user persists a profile in `users` and its external identity in `user_identities`; uniqueness constraints prevent duplicate registrations. For any change involving related records, inspect the applicable foreign keys and their deletion behavior rather than assuming that deletion cascades.

## Migrations

Migrations are versioned paired files in `db/migrations` and embedded into the binary. Add the next sequential `<version>_<description>.up.sql` and `.down.sql` pair. Keep down migrations accurate and review their data-loss effects. Apply migrations locally with `go run ./cmd/waypoint migrate` or `just run migrate`, configured through `WAYPOINT_DATABASE_URL`; `serve --migrate` runs them before connecting the app pool. Integration fixtures always migrate fresh databases.

Do not edit an already released migration to change an existing schema; add a new migration. Migrations run in order and are tracked by golang-migrate.

## SQL and generated code

1. Write named queries in `db/queries/*.sql` with sqlc annotations.
2. Update the migration schema source when the database changes.
3. Run `sqlc generate` using `sqlc.yaml`.
4. Review generated changes under `internal/db/sqlc` and the service call sites.

`sqlc.yaml` uses PostgreSQL, pgx/v5, emits a query interface, and maps PostgreSQL UUIDs to the repository's `uuid` package. Generated files should not be edited by hand.

The canonical local database URL is `postgres://postgres:secret@localhost:5433/waypoint?sslmode=disable`; Compose allows credentials, database, and port overrides. See [local setup](../development/local-setup.md).
