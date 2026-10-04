# HTTP API

The server uses Go's `net/http` `ServeMux` method and path patterns. The local default address is `http://localhost:8080`. JSON responses use `application/json`; error responses are plain text. Request bodies are capped at 1 MiB for routes that read/decode them.

## Routes

| Method and path                                     | Authentication                                             | Behavior                                                         |
| --------------------------------------------------- | ---------------------------------------------------------- | ---------------------------------------------------------------- |
| `GET /healthz`                                      | None                                                       | Process health                                                   |
| `GET /readyz`                                       | None                                                       | PostgreSQL readiness                                             |
| `POST /users`                                       | Bearer JWT                                                 | Create a local profile from token email/identity and body fields |
| `GET /users`                                        | None                                                       | List all user profiles                                           |
| `GET /users/{id}`                                   | None                                                       | Get a profile by UUID                                            |
| `GET /me`                                           | Bearer JWT                                                 | Get the profile and workspace memberships for the token identity |
| `POST /workspaces`                                  | Bearer JWT with existing local profile                     | Create workspace; caller becomes owner                           |
| `GET /workspaces/{workspace_id}`                    | Bearer JWT with existing local profile and membership      | Get workspace and members                                        |
| `POST /workspaces/{workspace_id}/users/{user_id}`   | Bearer JWT; caller must be owner                           | Add existing user as member                                      |
| `DELETE /workspaces/{workspace_id}/users/{user_id}` | Bearer JWT; caller must be owner                           | Remove a member                                                  |
| `DELETE /workspaces/{workspace_id}`                 | Bearer JWT; caller must be sole remaining member and owner | Delete workspace                                                 |

## Authentication

Authenticated routes require one `Authorization: Bearer <token>` header. At startup the service fetches OIDC discovery metadata from the configured issuer, then resolves keys from its JWKS URI. Tokens must use RS256 and have a valid signature, exact issuer, configured audience, expiration, and nonempty subject. `/users` requires an email claim to create a usable profile; other identity lookups use `iss` and `sub`.

A valid access token must map to a local user for the request to be authorized; otherwise, the service returns `403 Forbidden`. See [local setup](../development/local-setup.md) for a working Keycloak example.

## Request and response shapes

Create a user with `POST /users`:

```json
{ "user_name": "some-guy", "display_name": "Some Guy" }
```

Create a workspace with `POST /workspaces`:

```json
{ "name": "Engineering" }
```

The user endpoints return user identifiers and profile values using `id`, `email`, `user_name`, `display_name`, and `created_at`. `GET /me` wraps the user as `user` and includes a `workspace_memberships` array. Workspace responses include `id`, `name`, `members`, and `created_at`; each member has `user_id` and `role`.

## Common statuses

- `200`: successful reads.
- `201`: successful resource creation, such as creating a user.
- `204`: successful operation with no response body.
- `400`: malformed JSON, invalid UUID, or invalid input.
- `401`: missing or invalid bearer token on protected routes.
- `403`: authenticated user lacks a required local identity or permission.
- `404`: requested resource was not found.
- `409`: operation conflicts with existing state, such as registering an existing user identity.
- `500`: unexpected server or database failure.

For exact request validation and error mapping, use the relevant domain's handlers and errors as the source of truth. Follow [route registration](../../internal/waypoint/router.go) to the handler for a given endpoint; for example, user creation is handled in `internal/user`. Internal service operations are exposed over HTTP only when a route is registered for them.
