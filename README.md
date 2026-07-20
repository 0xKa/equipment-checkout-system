# Equipment Checkout System

A work-in-progress equipment checkout system built with Go. The current version provides a REST API for managing equipment items using in-memory storage.

## Current features

- Health endpoint
- Create, list, retrieve, update, and delete equipment items
- Request validation and consistent JSON errors
- Duplicate asset-tag protection
- Request IDs and structured logging
- Panic recovery, secure headers, and graceful shutdown
- Importable Postman collection

## Configuration

The default configuration works without a `.env` file. To customize it, copy the example file:

```powershell
cd server
Copy-Item .env.example .env
```

Available variables:

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `development` | Selects development or production logging |
| `HTTP_HOST` | `localhost` | Address the server listens on |
| `HTTP_PORT` | `8080` | HTTP server port |

## API endpoints

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/health` | Check API health |
| `POST` | `/api/v1/items` | Create an item |
| `GET` | `/api/v1/items` | List all items |
| `GET` | `/api/v1/items/:id` | Retrieve an item |
| `PUT` | `/api/v1/items/:id` | Replace an item's editable fields |
| `DELETE` | `/api/v1/items/:id` | Delete an item |


## Project structure

```text
server/
├── cmd/          Application commands and server startup
├── config/       Environment configuration
├── db/           In-memory item table
├── handlers/     HTTP handlers and error responses
├── logger/       Zap logger configuration
├── middleware/   Operational HTTP middleware
├── routes/       Endpoint registration
├── services/     Item business logic
└── types/        Domain and API types
```

## Current limitations

- Items are stored only in memory and are cleared whenever the server restarts.
- There is no authentication or authorization yet.
- Checkout and return workflows are not implemented yet.
- The project does not currently use an external database.

The project is still under development, so its API and structure may change as more features are added.
