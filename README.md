# Equipment Checkout System

A Go REST API for managing equipment, categories, and local users, with PostgreSQL persistence and a foundation for checkout, return, reservation, and maintenance workflows.

The development environment uses Docker Compose to start PostgreSQL, apply Goose migrations, and run the API. Authentication and authorization are not implemented yet; `X-Actor-User-ID` is development attribution only.

## Project structure

```text
.
├── Makefile              Development commands
├── compose.yaml          PostgreSQL, migration, and API services
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
make seed-dev       # optional development data
make compose-down
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

`make reset-dev CONFIRM=YES` deletes all development data while preserving migration history.
