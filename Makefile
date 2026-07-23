ENV_FILE := server/.env
MIGRATIONS_DIR := server/db/migrations
MIGRATION_FILES = $(wildcard $(MIGRATIONS_DIR)/*.sql)
SEED_FILE ?= server/db/seeds/development.sql
RESET_FILE ?= server/db/scripts/reset-development.sql
GOOSE := go run github.com/pressly/goose/v3/cmd/goose@v3.27.2
SQLC := go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1

-include $(ENV_FILE)

REQUIRED_DATABASE_VARIABLES := POSTGRES_DB POSTGRES_USER POSTGRES_PASSWORD POSTGRES_PORT DATABASE_URL

.PHONY: check-db-env compose-up db-up compose-down migrate-up migrate-down migrate-status seed-dev reset-dev sqlc

# Validate that the environment file exists and contains every required database setting.
check-db-env:
	$(if $(wildcard $(ENV_FILE)),,$(error Missing $(ENV_FILE); copy server/.env.example and provide local database values))
	$(foreach variable,$(REQUIRED_DATABASE_VARIABLES),$(if $(strip $($(variable))),,$(error $(variable) is missing from $(ENV_FILE))))
	@echo Database configuration loaded from $(ENV_FILE).

# Build and start PostgreSQL, apply migrations, and wait for the API to become ready.
compose-up: check-db-env
	docker compose --env-file $(ENV_FILE) up --detach --build --wait

# Start only PostgreSQL for running the API directly on the host.
db-up: check-db-env
	docker compose --env-file $(ENV_FILE) up --detach --wait postgres

# Stop the Compose project without removing its named volume, preserving local database data.
compose-down: check-db-env
	docker compose --env-file $(ENV_FILE) down

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
