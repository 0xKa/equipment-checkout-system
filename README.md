# Equipment Checkout System

A Go REST API for managing equipment, categories, local users, and transactional checkout/return workflows with PostgreSQL persistence.

The development environment uses Docker Compose to run the API, its PostgreSQL
database and Goose migrations, plus Keycloak with a separate PostgreSQL
database. Every `/api/v1` route authenticates a Keycloak access token, resolves
its exact external identity to an existing or safely JIT-provisioned local
user, and authorizes the operation with application capabilities derived from
Keycloak client roles.

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
| `employee` | `/me`, item/category reads, self checkout/return, and own checkout list/get |
| `auditor` | `/me`, item/category reads, all checkout list/get, and item-wide checkout history |
| `inventory_admin` | All current routes, including on-behalf checkout/return and local-user management |

Employees must send their own local user ID as `borrower_user_id`; a mismatch
returns `403 forbidden`. Administrators may check out equipment for any active
local borrower. Employees may return and read only their own checkout records;
an attempt to get another borrower's checkout returns `404`. Employee lists
are borrower-filtered in PostgreSQL, while administrators and auditors retain
bounded all-record reads. Item-wide history remains administrator/auditor
only.

The authenticated actor, borrower, and return recipient remain distinct.
Client-supplied local user IDs are never an identity source and cannot
override the token-derived actor used for history and audit attribution.

Local-user routes manage application profiles, not Keycloak credentials.
`POST /api/v1/users` still creates an unlinked borrower profile. An existing
unlinked row never becomes login-capable through a username or email match; it
requires an explicit reviewed `(issuer, subject)` link.

For a previously unseen Keycloak identity, the first valid token with a
recognized application role provisions one active local profile. The exact
`(issuer, subject)` pair is the identity key. `preferred_username` is required,
`name` initializes the display name with a username fallback, and email is
stored only when `email_verified=true`. A normalized username or email
collision returns `403 identity_conflict` and never attaches the identity to
the existing row. Missing or invalid required profile data returns
`403 identity_profile_invalid`.

Token profile claims initialize a JIT row only. Later logins reuse the exact
link without overwriting the local username, email, display name, or
`is_active`; a locally inactive linked account receives
`403 account_inactive`. The optional `last_seen_at` field is intentionally not
implemented because the application has no current operational use for
login-presence tracking.

The Postman collection applies `Bearer {{accessToken}}` to API requests at the
collection level while explicitly leaving the health requests unauthenticated.
Paste a current Keycloak access token into the `accessToken` collection
variable before exercising `/api/v1`.
