# Equipment Checkout System

A Go REST API for managing equipment, categories, local users, and transactional checkout/return workflows with PostgreSQL persistence.

The development environment uses Docker Compose to run the API, its PostgreSQL
database and Goose migrations, plus Keycloak with a separate PostgreSQL
database. Authentication and authorization are not implemented yet;
`X-Actor-User-ID` is development attribution only.

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
    ├── middleware/       HTTP safeguards and actor resolution
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
