# Equipment Checkout System

A Go REST API for managing equipment, categories, local users, and transactional checkout/return workflows with PostgreSQL persistence.

The development environment uses Docker Compose to run the API, its PostgreSQL
database and Goose migrations, plus Keycloak with a separate PostgreSQL
database. Every `/api/v1` route authenticates a Keycloak access token, resolves
its external identity to an explicitly linked local user, and authorizes the
operation with application capabilities derived from Keycloak client roles.

## Project structure

```text
.
├── Makefile              Development commands
├── compose.yaml          API, migration, PostgreSQL, and Keycloak services
├── keycloak/             Reproducible development realm and bootstrap script
├── postman/              Importable API collection
└── server/
    ├── cmd/              CLI commands and application wiring
    ├── config/           Environment configuration
    ├── db/
    │   ├── migrations/   Goose schema migrations
    │   ├── queries/      sqlc query definitions
    │   └── sqlcgen/      Generated database code
    ├── handlers/         HTTP request and response handling
    ├── middleware/       HTTP safeguards, authentication, and authorization
    ├── routes/           Route registration
    ├── services/         Business rules and database mapping
    ├── types/            Domain and API types
    ├── Dockerfile        API and migration image targets
    └── .env.example      Local configuration template
```

## Development commands

Create the ignored local environment file:

```powershell
Copy-Item server/.env.example server/.env
```

Run the complete Compose environment:

```text
make compose-up
make keycloak-up     # Keycloak and its PostgreSQL database only
make keycloak-stop   # preserve Keycloak data
make keycloak-reset CONFIRM=YES  # recreate only Keycloak data
make seed-dev       # optional development data
make compose-down
make compose-down-volumes CONFIRM=YES  # delete all Compose volumes
```

Run PostgreSQL in Compose and the API on the host:

```text
make db-up
make keycloak-up
make migrate-up
make run
```

Verify and maintain the project:

```text
make test
make build
make sqlc
make migrate-status
make migrate-down
```

`make reset-dev CONFIRM=YES` deletes all application rows while preserving
migration history. `make keycloak-reset CONFIRM=YES` deletes and recreates only
the Keycloak PostgreSQL volume. `make compose-down-volumes CONFIRM=YES` deletes
both application and Keycloak PostgreSQL volumes.

Keycloak imports the canonical development realm only when the `equipment`
realm is absent. Normal startup skips an existing realm, so changes to
`keycloak/realms/equipment-realm.json` do not update previously imported data.
After reviewing an intentional realm change, run
`make keycloak-reset CONFIRM=YES` to recreate only the Keycloak database and
import the canonical realm again.

The API validates OIDC configuration and checks the configured JWKS endpoint
once during startup. Host execution uses `OIDC_JWKS_URL` from `server/.env`;
Compose uses Keycloak's internal service address while preserving the same
public `OIDC_ISSUER_URL`. `/health` remains process-only and `/ready` remains
database-only.

## Authentication and current authorization

Every `/api/v1` request requires exactly one
`Authorization: Bearer <access-token>` credential. Use an access token issued
for the `equipment-api` audience, not an ID token. `/health` and `/ready`
remain public.

The API verifies the token, resolves its exact `(issuer, subject)` identity to
an active local user, and derives application capabilities from the
`equipment-api` client roles:

| Keycloak role | Current API access |
| --- | --- |
| `employee` | `/me` and item/category reads |
| `auditor` | `/me`, item/category reads, and checkout-history reads |
| `inventory_admin` | All current routes, including inventory mutations, local-user management, checkout, and return |

Checkout and return are intentionally inventory-admin-only in Milestone 6.
Milestone 7 adds the final self-service ownership rules. Client-supplied local
user IDs are never an identity source and cannot override the token-derived
actor.

Local-user routes manage application profiles, not Keycloak credentials.
`POST /api/v1/users` creates an unlinked local user, and profile or status
updates never link an identity automatically. A user becomes login-capable
only through an explicit reviewed `(issuer, subject)` link; username or email
matches are not used for linking.

The Postman collection applies `Bearer {{accessToken}}` to API requests at the
collection level while explicitly leaving the health requests unauthenticated.
Paste a current Keycloak access token into the `accessToken` collection
variable before exercising `/api/v1`.
