ENV_FILE := server/.env
MIGRATIONS_DIR := server/db/migrations
MIGRATION_FILES = $(wildcard $(MIGRATIONS_DIR)/*.sql)
SEED_FILE ?= server/db/seeds/development.sql
RESET_FILE ?= server/db/scripts/reset-development.sql
COMPOSE := docker compose --env-file "$(ENV_FILE)"
GOOSE := go run github.com/pressly/goose/v3/cmd/goose@v3.27.2
SQLC := go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1

.DEFAULT_GOAL := help

-include $(ENV_FILE)

REQUIRED_DATABASE_VARIABLES := POSTGRES_DB POSTGRES_USER POSTGRES_PASSWORD POSTGRES_PORT DATABASE_URL
REQUIRED_KEYCLOAK_VARIABLES := KEYCLOAK_HTTP_PORT KEYCLOAK_POSTGRES_DB KEYCLOAK_POSTGRES_USER KEYCLOAK_POSTGRES_PASSWORD KEYCLOAK_POSTGRES_VOLUME_NAME KEYCLOAK_BOOTSTRAP_ADMIN_USERNAME KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD KEYCLOAK_EQUIPMENT_ADMIN_PASSWORD KEYCLOAK_SAMPLE_BORROWER_PASSWORD KEYCLOAK_AUDIT_VIEWER_PASSWORD KEYCLOAK_USER_SYNC_CLIENT_SECRET
REQUIRED_OIDC_VARIABLES := OIDC_ISSUER_URL OIDC_JWKS_URL OIDC_AUDIENCE OIDC_HTTP_TIMEOUT OIDC_CLOCK_SKEW
REQUIRED_KEYCLOAK_ADMIN_VARIABLES := KEYCLOAK_ADMIN_URL KEYCLOAK_REALM KEYCLOAK_USER_SYNC_CLIENT_ID KEYCLOAK_USER_SYNC_CLIENT_SECRET KEYCLOAK_ADMIN_TIMEOUT

.PHONY: help check-db-env check-keycloak-env check-oidc-env check-keycloak-admin-env compose-config compose-up compose-down compose-down-volumes db-up keycloak-up keycloak-stop keycloak-reset migrate-up migrate-status migrate-down seed-dev reset-dev sqlc inspect-token reconcile-users run build test

# Validate that the environment file exists and contains every required database setting.
check-db-env:
	$(if $(wildcard $(ENV_FILE)),,$(error Missing $(ENV_FILE); copy server/.env.example and provide local database values))
	$(foreach variable,$(REQUIRED_DATABASE_VARIABLES),$(if $(strip $($(variable))),,$(error $(variable) is missing from $(ENV_FILE))))
	@echo Database configuration loaded from $(ENV_FILE).

# Validate the Keycloak and dedicated Keycloak database settings.
check-keycloak-env:
	$(if $(wildcard $(ENV_FILE)),,$(error Missing $(ENV_FILE); copy server/.env.example and provide local Keycloak values))
	$(foreach variable,$(REQUIRED_KEYCLOAK_VARIABLES),$(if $(strip $($(variable))),,$(error $(variable) is missing from $(ENV_FILE))))
	@echo Keycloak configuration loaded from $(ENV_FILE).

# Validate the host API's OIDC resource-server settings.
check-oidc-env:
	$(if $(wildcard $(ENV_FILE)),,$(error Missing $(ENV_FILE); copy server/.env.example and provide local OIDC values))
	$(foreach variable,$(REQUIRED_OIDC_VARIABLES),$(if $(strip $($(variable))),,$(error $(variable) is missing from $(ENV_FILE))))
	@echo OIDC configuration loaded from $(ENV_FILE).

# Validate the host application's Keycloak Admin API settings.
check-keycloak-admin-env:
	$(if $(wildcard $(ENV_FILE)),,$(error Missing $(ENV_FILE); copy server/.env.example and provide local Keycloak Admin API values))
	$(foreach variable,$(REQUIRED_KEYCLOAK_ADMIN_VARIABLES),$(if $(strip $($(variable))),,$(error $(variable) is missing from $(ENV_FILE))))
	@echo Keycloak Admin API configuration loaded from $(ENV_FILE).

# Validate the resolved Compose configuration without starting services.
compose-config: check-db-env check-keycloak-env check-oidc-env check-keycloak-admin-env
	$(COMPOSE) config --quiet

# Build and start the complete development environment and wait for healthy services.
compose-up: compose-config
	$(COMPOSE) up --detach --build --wait

# Stop the Compose project without removing its named volumes.
compose-down: check-db-env check-keycloak-env
	$(COMPOSE) down

# Stop the Compose project and delete its named and anonymous volumes after explicit confirmation.
compose-down-volumes: check-db-env check-keycloak-env
	$(if $(filter YES,$(CONFIRM)),,$(error Destructive command. Run: make compose-down-volumes CONFIRM=YES))
	$(COMPOSE) down --volumes

# Start only PostgreSQL for running the API directly on the host.
db-up: check-db-env
	$(COMPOSE) up --detach --wait postgres

# Start only Keycloak and its dedicated PostgreSQL database.
keycloak-up: check-db-env check-keycloak-env
	$(COMPOSE) up --detach --wait keycloak-postgres keycloak
	$(COMPOSE) run --rm --no-deps keycloak-bootstrap

# Stop only Keycloak and its database without deleting their persistent volume.
keycloak-stop: check-db-env check-keycloak-env
	$(COMPOSE) stop keycloak keycloak-postgres

# Recreate only the Keycloak development database and automatically import/bootstrap its realm.
keycloak-reset: check-db-env check-keycloak-env
	$(if $(filter YES,$(CONFIRM)),,$(error Destructive command. Run: make keycloak-reset CONFIRM=YES))
	$(COMPOSE) rm --force --stop keycloak-bootstrap keycloak keycloak-postgres
	@if docker volume inspect "$(KEYCLOAK_POSTGRES_VOLUME_NAME)" >/dev/null 2>&1; then docker volume rm "$(KEYCLOAK_POSTGRES_VOLUME_NAME)"; fi
	$(COMPOSE) up --detach --wait keycloak-postgres keycloak
	$(COMPOSE) run --rm --no-deps keycloak-bootstrap

# Apply every pending migration to the configured PostgreSQL database.
migrate-up: check-db-env
	@$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

# Display the applied or pending state of every migration, or explain when none exist.
migrate-status: check-db-env
	@$(if $(MIGRATION_FILES),$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status,echo No migration files found in $(MIGRATIONS_DIR).)

# Roll back exactly one applied migration after explicit confirmation.
migrate-down: check-db-env
	$(if $(filter YES,$(CONFIRM)),,$(error Destructive command. Run: make migrate-down CONFIRM=YES))
	@$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

# Load repeatable sample data into an already-running, migrated development database.
seed-dev: check-db-env
	$(if $(filter development,$(APP_ENV)),,$(error seed-dev requires APP_ENV=development))
	$(if $(wildcard $(SEED_FILE)),,$(error Missing seed file: $(SEED_FILE)))
	@$(COMPOSE) exec -T postgres psql --username "$(POSTGRES_USER)" --dbname "$(POSTGRES_DB)" --set=ON_ERROR_STOP=1 --single-transaction --quiet --tuples-only --no-align < "$(SEED_FILE)"

# Delete all application rows and reset identity counters after explicit confirmation.
reset-dev: check-db-env
	$(if $(filter development,$(APP_ENV)),,$(error reset-dev requires APP_ENV=development))
	$(if $(filter YES,$(CONFIRM)),,$(error Destructive command. Run: make reset-dev CONFIRM=YES))
	$(if $(wildcard $(RESET_FILE)),,$(error Missing reset file: $(RESET_FILE)))
	@$(COMPOSE) exec -T postgres psql --username "$(POSTGRES_USER)" --dbname "$(POSTGRES_DB)" --set=ON_ERROR_STOP=1 --single-transaction < "$(RESET_FILE)"

# Generate pgx/v5-compatible Go code from the reviewed queries.
sqlc:
	@$(SQLC) generate -f server/db/sqlc.yaml

# Prompt securely for an access token and decode only allowlisted metadata.
inspect-token:
	cd server && go run . inspect-token

# Reconcile local user intent into the configured Keycloak realm once.
reconcile-users: check-db-env check-oidc-env check-keycloak-admin-env
	cd server && go run . reconcile-users

# Start the API server locally through its serve command.
run: check-db-env check-oidc-env check-keycloak-admin-env
	cd server && go run . serve

# Compile every Go package from the server module directory.
build:
	cd server && go build ./...

# Run the complete Go test suite from the server module directory.
test:
	cd server && go test ./...

# Print a small message for quickly confirming that Make is working.
hello:
	@echo Hello World!

# Display the supported development commands.
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Configuration:"
	@echo "  compose-config       Validate the resolved Compose configuration"
	@echo ""
	@echo "Environment:"
	@echo "  compose-up           Build and start the complete environment"
	@echo "  compose-down         Stop the complete environment and preserve volumes"
	@echo "  compose-down-volumes Stop the environment and delete volumes (CONFIRM=YES)"
	@echo "  db-up                Start only the application PostgreSQL database"
	@echo "  keycloak-up          Start and bootstrap Keycloak"
	@echo "  keycloak-stop        Stop Keycloak and preserve its database"
	@echo "  keycloak-reset       Recreate the Keycloak database (CONFIRM=YES)"
	@echo ""
	@echo "Database:"
	@echo "  migrate-up           Apply pending migrations"
	@echo "  migrate-status       Show migration status"
	@echo "  migrate-down         Roll back one migration (CONFIRM=YES)"
	@echo "  seed-dev             Load repeatable development data"
	@echo "  reset-dev            Reset development data (CONFIRM=YES)"
	@echo ""
	@echo "Development:"
	@echo "  sqlc                 Generate Go database code"
	@echo "  inspect-token        Decode allowlisted access-token metadata"
	@echo "  reconcile-users      Reconcile local users into Keycloak once"
	@echo "  run                  Run the API on the host"
	@echo "  build                Build all Go packages"
	@echo "  test                 Run all Go tests"
