# Documentation

Start with the page that matches the task:

| Topic                          | Page                                       | What it covers                                      |
| ------------------------------ | ------------------------------------------ | --------------------------------------------------- |
| Local development              | [Development guide](development/README.md) | Setup, commands, and verification                   |
| Local services and manual auth | [Local setup](development/local-setup.md)  | Compose, environment, Keycloak, and first API calls |
| Frontend integration | [Frontend guide](frontend/README.md) | Interactions with the frontend app |
| System design                  | [Architecture](architecture/README.md)     | Package boundaries and responsibilities             |
| HTTP contract                  | [API reference](api/README.md)             | Routes, auth requirements, and status codes         |
| PostgreSQL                     | [Database guide](database/README.md)       | Data model, migrations, and generated queries       |

The root [README](../README.md) is the short project entry point. Keep detailed operational and implementation guidance in these topic pages.

## Keeping documentation current

When a code change affects documented behavior, interfaces, setup, or architecture, update the relevant documentation in the same change.

Describe shared rules by responsibility or behavior rather than listing every affected service or resource. When an example helps, use a small, stable operation such as creating a user and mark it as an example. Keep complete inventories in their authoritative source; retain explicit lists where completeness is the purpose, such as an API route reference, a local setup guide or an implementation plan’s acceptance criteria.
