# Equipment Checkout System

A work-in-progress equipment checkout REST API built with Go, Echo, PostgreSQL, pgx, and sqlc.

## Current features

- Process health endpoint
- Persistent item and category CRUD
- Persistent local user management and activation status
- Optional item/category assignment
- Required case-insensitive unique usernames and optional case-insensitive unique emails
- Provisional local actor resolution through `X-Actor-User-ID`
- Strict JSON decoding and consistent success/error envelopes
- Request IDs, structured logging, panic recovery, secure headers, and graceful shutdown
- PostgreSQL Compose service, Goose migrations, development seed data, and a Postman collection

PostgreSQL is required. The API does not fall back to in-memory storage and does not apply migrations during startup.

## Local setup

Create the ignored environment file:

```powershell
Copy-Item server/.env.example server/.env
```

From the repository root, start PostgreSQL and apply the schema:

```text
make compose-up
make migrate-up
make migrate-status
```

Optionally load the repeatable development seed:

```text
make seed-dev
```

Start the API:

```text
make run
```

Or run it directly from PowerShell when GNU Make is unavailable:

```powershell
Set-Location server
go run . serve
```

The API listens on `http://localhost:8080` by default.

## Configuration

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `development` | Selects development or production logging |
| `HTTP_HOST` | `localhost` | Address the server listens on |
| `HTTP_PORT` | `8080` | HTTP server port |
| `DATABASE_URL` | Required | PostgreSQL connection URL |
| `POSTGRES_DB` | Required by Compose/Make | Local database name |
| `POSTGRES_USER` | Required by Compose/Make | Local database user |
| `POSTGRES_PASSWORD` | Required by Compose/Make | Local database password |
| `POSTGRES_PORT` | Required by Compose/Make | Published PostgreSQL port |
| `DB_MAX_CONNECTIONS` | `10` | Maximum pool connections |
| `DB_MIN_CONNECTIONS` | `1` | Minimum pool connections |
| `DB_MAX_CONNECTION_LIFETIME` | `30m` | Maximum pooled connection lifetime |

## API endpoints

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/health` | Process liveness |
| `POST` | `/api/v1/items` | Create an item |
| `GET` | `/api/v1/items` | List items |
| `GET` | `/api/v1/items/:id` | Retrieve an item |
| `PUT` | `/api/v1/items/:id` | Replace client-editable item fields |
| `DELETE` | `/api/v1/items/:id` | Delete an unreferenced item |
| `POST` | `/api/v1/categories` | Create a category |
| `GET` | `/api/v1/categories` | List categories |
| `GET` | `/api/v1/categories/:id` | Retrieve a category |
| `PUT` | `/api/v1/categories/:id` | Replace category fields |
| `DELETE` | `/api/v1/categories/:id` | Delete an unused category |
| `POST` | `/api/v1/users` | Create an active local user |
| `GET` | `/api/v1/users` | List all users |
| `GET` | `/api/v1/users?is_active=true` | Filter users by active state |
| `GET` | `/api/v1/users/:id` | Retrieve a user |
| `PUT` | `/api/v1/users/:id` | Replace profile fields while preserving status |
| `PATCH` | `/api/v1/users/:id/status` | Activate or deactivate a user |
| `GET` | `/api/v1/me` | Resolve the provisional actor |

Successful resource responses use a `data` envelope. Errors use:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "...",
    "request_id": "..."
  }
}
```

## Local users

A user requires `username` and `display_name`; `email` is optional:

```json
{
  "username": "equipment.operator",
  "email": "operator@example.test",
  "display_name": "Equipment Operator"
}
```

Usernames are required and unique without regard to case. An absent email is represented as JSON `null`; present emails are unique without regard to case and stored lowercase. New users are active; profile `PUT` requires a username and does not change activation status.

Change status separately:

```json
{
  "is_active": false
}
```

Users are retained for workflow history and are not deleted through the API.

## Provisional actor header

`GET /api/v1/me` requires a positive local user ID:

```http
X-Actor-User-ID: 1
```

The API resolves that ID to an existing active user. This header provides development/internal attribution only. It is client-spoofable, is not authentication, grants no role or permission, and must not be exposed as a production security boundary. Keycloak will replace this source in a later milestone.

## Development commands

```text
make build
make test
make sqlc
make migrate-up
make migrate-down
make migrate-status
make compose-up
make compose-down
```

`make reset-dev CONFIRM=YES` destructively truncates all application tables and resets identities while retaining Goose migration history.

## Project structure

```text
server/
├── cmd/            CLI commands and application composition
├── config/         Environment parsing and validation
├── db/migrations/  Authoritative Goose schema
├── db/queries/     Reviewed sqlc queries
├── db/sqlcgen/     Generated code; do not edit manually
├── handlers/       HTTP decoding and response translation
├── logger/         Zap/slog configuration
├── middleware/     Request safeguards and actor resolution
├── routes/         Endpoint registration
├── services/       Validation, domain orchestration, and persistence mapping
├── types/          Domain and API contracts
└── utils/          Shared HTTP and PostgreSQL helpers
```

## Current limitations

- There is no authentication or authorization.
- Checkout, return, location, reservation, maintenance, condition-report, status-history, and audit API workflows are not implemented.
- PostgreSQL is the only Compose service; the API is currently run locally.
- There are no Go test files yet.
