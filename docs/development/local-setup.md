# Local setup

This setup runs PostgreSQL and Keycloak in Docker Compose and runs the Go API on the host.

## Start dependencies and API

From the repository root:

```sh
go mod download
docker compose up -d --wait postgres keycloak
export WAYPOINT_DATABASE_URL='postgres://postgres:secret@localhost:5433/waypoint?sslmode=disable'
go run ./cmd/waypoint serve --migrate \
  --jwt.issuer=http://localhost:8081/realms/waypoint \
  --jwt.audience=waypoint-api
```

The API listens at `http://localhost:8080`. `--migrate` applies embedded pending migrations before the server starts. To migrate separately, run `go run ./cmd/waypoint migrate`; this command reads `WAYPOINT_DATABASE_URL`.

Check process health and database readiness:

```sh
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

`/healthz` reports process health; `/readyz` succeeds only while PostgreSQL is reachable.

## Register a local account and create a profile

Open <http://localhost:8081/realms/waypoint/account/> and choose **Sign in**, then **Register** if prompted. The local realm starts without application users. Registering creates a Keycloak identity; the Waypoint profile is created separately using that identity's access token.

Request a token with the `waypoint-manual` direct-grant client:

```sh
curl --fail-with-body http://localhost:8081/realms/waypoint/protocol/openid-connect/token \
  --data-urlencode 'grant_type=password' \
  --data-urlencode 'client_id=waypoint-manual' \
  --data-urlencode 'username=YOUR_USERNAME' \
  --data-urlencode 'password=YOUR_PASSWORD'
```

Put the returned `access_token` in `TOKEN`, then create and inspect the Waypoint profile:

```sh
TOKEN='PASTE_ACCESS_TOKEN_HERE'
curl --fail-with-body http://localhost:8080/users \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"user_name":"some-guy", "display_name":"Some Guy"}'
curl --fail-with-body http://localhost:8080/me -H "Authorization: Bearer $TOKEN"
```

The API uses the token's `email`, `iss`, and `sub` claims. The request body supplies only `user_name` and `display_name`. See the [API reference](../api/README.md) for other routes.

## Compose services and overrides

| Service    | Local address    | Notes                                                                                             |
| ---------- | ---------------- | ------------------------------------------------------------------------------------------------- |
| PostgreSQL | `localhost:5433` | Database `waypoint`, user `postgres`, password `secret` by default                                |
| Keycloak   | `localhost:8081` | Realm `waypoint`; admin console at `/admin/`, default master-realm login `admin` / `secret`       |
| pgAdmin    | `localhost:5050` | Optional; start with `docker compose up -d pgadmin`; default login `admin@example.com` / `secret` |

Compose accepts the following environment overrides, set in the repository-root `.env` file or exported in the shell. Keep this complete list and its defaults in sync with [docker-compose.yaml](../../docker-compose.yaml).

| Service    | Variable                   | Default             | Purpose                                                                       |
| ---------- | -------------------------- | ------------------- | ----------------------------------------------------------------------------- |
| PostgreSQL | `POSTGRES_DB`              | `waypoint`          | Database name; also used by pgAdmin's connection configuration                |
| PostgreSQL | `POSTGRES_USER`            | `postgres`          | Database user; also used by pgAdmin's connection configuration                |
| PostgreSQL | `POSTGRES_PASSWORD`        | `secret`            | Database password; also used by pgAdmin's connection configuration            |
| PostgreSQL | `POSTGRES_PORT`            | `5433`              | Host port mapped to container port `5432`                                     |
| Keycloak   | `KEYCLOAK_ADMIN_USERNAME`  | `admin`             | Master-realm bootstrap administrator; passed as `KC_BOOTSTRAP_ADMIN_USERNAME` |
| Keycloak   | `KEYCLOAK_ADMIN_PASSWORD`  | `secret`            | Bootstrap administrator password; passed as `KC_BOOTSTRAP_ADMIN_PASSWORD`     |
| Keycloak   | `KEYCLOAK_PORT`            | `8081`              | Host port mapped to container port `8080`                                     |
| pgAdmin    | `PGADMIN_DEFAULT_EMAIL`    | `admin@example.com` | Initial pgAdmin login email                                                   |
| pgAdmin    | `PGADMIN_DEFAULT_PASSWORD` | `secret`            | Initial pgAdmin login password                                                |
| pgAdmin    | `PGADMIN_PORT`             | `5050`              | Host port mapped to container port `80`                                       |

Compose also sets two fixed pgAdmin container environment values: `PGADMIN_REPLACE_SERVERS_ON_STARTUP=True` reloads the supplied server configuration at startup, and `PGPASS_FILE=/config/pgpass` selects the supplied database password file. These values are not exposed as `.env` overrides.

The API reads exported environment variables, not `.env` files. If you override database credentials or `POSTGRES_PORT`, update `WAYPOINT_DATABASE_URL` to match. If you override `KEYCLOAK_PORT`, update the JWT issuer URL and the browser/token URLs above to use that port.

The API environment variables and flags are declared in [cmd/waypoint/serve.go](../../cmd/waypoint/serve.go). Run `go run ./cmd/waypoint serve --help` for the available settings and defaults.

Compose services persist data in named volumes as declared in `docker-compose.yaml`. The realm JSON is imported only when Keycloak first creates the realm; changing Compose JSON does not overwrite an existing realm. Use the admin console for an existing realm. Stop services with `docker compose down`; this keeps volumes. `docker compose down -v` also deletes persisted data.

## Frontend and social login

The initial realm import configures `waypoint-fe` as a confidential OpenID Connect client with the standard authorization code flow and required S256 PKCE. It uses `http://localhost:3000`, the callback `http://localhost:3000/api/auth/callback/keycloak`, and the post-logout redirect `http://localhost:3000`. Configure the frontend with issuer `http://localhost:8081/realms/waypoint`, client ID `waypoint-fe`, and the same `KEYCLOAK_FE_CLIENT_SECRET` value supplied to Compose. Keep this secret in the frontend's server-side auth configuration.

The realm import attaches the `waypoint-api` scope by default to this client. Its audience mapper adds `waypoint-api` to access tokens and introspection, but not ID tokens or lightweight access tokens. It is not a realm-wide default scope.

Google and GitHub providers are also created by the realm import. Before the first startup, set all five required values in your ignored `.env` file or exported environment:

- `KEYCLOAK_FE_CLIENT_SECRET`: a randomly generated secret shared with the frontend server.
- `KEYCLOAK_GOOGLE_CLIENT_ID`: your Google OAuth application's client ID.
- `KEYCLOAK_GOOGLE_CLIENT_SECRET`: that Google application's client secret.
- `KEYCLOAK_GITHUB_CLIENT_ID`: your GitHub OAuth application's client ID.
- `KEYCLOAK_GITHUB_CLIENT_SECRET`: that GitHub application's client secret.

These values have no defaults. Compose rejects missing or empty values when loading the configuration, including commands targeting other services.

Register these callback URLs in the respective Google and GitHub OAuth apps:

- `http://localhost:8081/realms/waypoint/broker/google/endpoint`
- `http://localhost:8081/realms/waypoint/broker/github/endpoint`

These callbacks use the existing `waypoint` realm. Update the port if overriding `KEYCLOAK_PORT`. Provider callbacks are generated by Keycloak rather than stored as redirect URI settings in the realm import. Blank provider options retain Keycloak's defaults. Changing environment variables after the realm has been created does not update stored credentials; update them in the admin console.

The import includes built-in scope definitions exported from Keycloak 26.7.3.

Database, Keycloak, and pgAdmin data persist in named volumes. The realm JSON is imported only when Keycloak first creates the realm; changing Compose JSON does not overwrite an existing realm. Use the admin console for an existing realm. Stop services with `docker compose down`; this keeps volumes. `docker compose down -v` also deletes persisted data.
