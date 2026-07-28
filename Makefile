ENV_FILE := server/.env
MIGRATIONS_DIR := server/db/migrations
MIGRATION_FILES = $(wildcard $(MIGRATIONS_DIR)/*.sql)
SEED_FILE ?= server/db/seeds/development.sql
RESET_FILE ?= server/db/scripts/reset-development.sql
GOOSE := go run github.com/pressly/goose/v3/cmd/goose@v3.27.2
SQLC := go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1

-include $(ENV_FILE)

REQUIRED_DATABASE_VARIABLES := POSTGRES_DB POSTGRES_USER POSTGRES_PASSWORD POSTGRES_PORT DATABASE_URL
REQUIRED_KEYCLOAK_VARIABLES := KEYCLOAK_HTTP_PORT KEYCLOAK_POSTGRES_DB KEYCLOAK_POSTGRES_USER KEYCLOAK_POSTGRES_PASSWORD KEYCLOAK_POSTGRES_VOLUME_NAME KEYCLOAK_BOOTSTRAP_ADMIN_USERNAME KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD KEYCLOAK_EQUIPMENT_ADMIN_PASSWORD KEYCLOAK_SAMPLE_BORROWER_PASSWORD KEYCLOAK_AUDIT_VIEWER_PASSWORD

.PHONY: check-db-env check-keycloak-env compose-up keycloak-up keycloak-stop keycloak-reset db-up compose-down compose-down-volumes migrate-up migrate-down migrate-status seed-dev reset-dev sqlc

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

# Build and start the complete development environment and wait for healthy services.
compose-up: check-db-env check-keycloak-env
	docker compose --env-file $(ENV_FILE) up --detach --build --wait
	docker compose --env-file $(ENV_FILE) run --rm --no-deps keycloak-bootstrap

# Start only Keycloak and its dedicated PostgreSQL database.
keycloak-up: check-db-env check-keycloak-env
	docker compose --env-file $(ENV_FILE) up --detach --wait keycloak-postgres keycloak
	docker compose --env-file $(ENV_FILE) run --rm --no-deps keycloak-bootstrap

# Stop only Keycloak and its database without deleting their persistent volume.
keycloak-stop: check-db-env check-keycloak-env
	docker compose --env-file $(ENV_FILE) stop keycloak keycloak-postgres

# Recreate only the Keycloak development database and automatically import/bootstrap its realm.
keycloak-reset: check-db-env check-keycloak-env
	$(if $(filter YES,$(CONFIRM)),,$(error Destructive command. Run: make keycloak-reset CONFIRM=YES))
	docker compose --env-file $(ENV_FILE) rm --force --stop keycloak-bootstrap keycloak keycloak-postgres
	@if docker volume inspect "$(KEYCLOAK_POSTGRES_VOLUME_NAME)" >/dev/null 2>&1; then docker volume rm "$(KEYCLOAK_POSTGRES_VOLUME_NAME)"; fi
	docker compose --env-file $(ENV_FILE) up --detach --wait keycloak-postgres keycloak
	docker compose --env-file $(ENV_FILE) run --rm --no-deps keycloak-bootstrap

# Start only PostgreSQL for running the API directly on the host.
db-up: check-db-env
	docker compose --env-file $(ENV_FILE) up --detach --wait postgres

# Stop the Compose project without removing its named volumes.
compose-down: check-db-env check-keycloak-env
	docker compose --env-file $(ENV_FILE) down

# Stop the Compose project and delete its named and anonymous volumes after explicit confirmation.
compose-down-volumes: check-db-env check-keycloak-env
	$(if $(filter YES,$(CONFIRM)),,$(error Destructive command. Run: make compose-down-volumes CONFIRM=YES))
	docker compose --env-file $(ENV_FILE) down --volumes

# Apply every pending migration to the configured PostgreSQL database.
migrate-up: check-db-env
	@$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

# Roll back exactly one applied migration from the configured PostgreSQL database.
migrate-down: check-db-env
	@$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

# Display the applied or pending state of every migration, or explain when none exist.
migrate-status: check-db-env
	@$(if $(MIGRATION_FILES),$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status,echo No migration files found in $(MIGRATIONS_DIR).)

# Load repeatable sample data into an already-running, migrated development database.
seed-dev: check-db-env
	$(if $(filter development,$(APP_ENV)),,$(error seed-dev requires APP_ENV=development))
	$(if $(wildcard $(SEED_FILE)),,$(error Missing seed file: $(SEED_FILE)))
	@docker compose --env-file "$(ENV_FILE)" exec -T postgres psql --username "$(POSTGRES_USER)" --dbname "$(POSTGRES_DB)" --set=ON_ERROR_STOP=1 --single-transaction --quiet --tuples-only --no-align < "$(SEED_FILE)"

# Delete all application rows and reset identity counters after explicit confirmation.
reset-dev: check-db-env
	$(if $(filter development,$(APP_ENV)),,$(error reset-dev requires APP_ENV=development))
	$(if $(filter YES,$(CONFIRM)),,$(error Destructive command. Run: make reset-dev CONFIRM=YES))
	$(if $(wildcard $(RESET_FILE)),,$(error Missing reset file: $(RESET_FILE)))
	@docker compose --env-file "$(ENV_FILE)" exec -T postgres psql --username "$(POSTGRES_USER)" --dbname "$(POSTGRES_DB)" --set=ON_ERROR_STOP=1 --single-transaction < "$(RESET_FILE)"

# Generate pgx/v5-compatible Go code from the reviewed queries.
sqlc:
	@$(SQLC) generate -f server/sqlc.yaml

# Print a small message for quickly confirming that Make is working.
hello:
	@echo Hello World!

# Run the complete Go test suite from the server module directory.
test:
	cd server && go test ./...

# Compile every Go package from the server module directory.
build:
	cd server && go build ./...

# Start the API server locally through its serve command.
run:
	cd server && go run . serve
