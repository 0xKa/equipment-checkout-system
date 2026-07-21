ENV_FILE := server/.env
MIGRATIONS_DIR := server/db/migrations
GOOSE := go run github.com/pressly/goose/v3/cmd/goose@v3.27.2

# Import the same simple KEY=value file used by Compose without making it mandatory for Go-only targets.
-include $(ENV_FILE)

REQUIRED_DATABASE_VARIABLES := POSTGRES_DB POSTGRES_USER POSTGRES_PASSWORD POSTGRES_PORT DATABASE_URL
MIGRATION_FILES = $(wildcard $(MIGRATIONS_DIR)/*.sql)

.PHONY: check-db-env compose-up compose-down migrate-up migrate-down migrate-status

check-db-env:
	$(if $(wildcard $(ENV_FILE)),,$(error Missing $(ENV_FILE); copy server/.env.example and provide local database values))
	$(foreach variable,$(REQUIRED_DATABASE_VARIABLES),$(if $(strip $($(variable))),,$(error $(variable) is missing from $(ENV_FILE))))
	@echo Database configuration loaded from $(ENV_FILE).

compose-up: check-db-env
	docker compose --env-file $(ENV_FILE) up --detach --wait postgres

# Deliberately omit --volumes so ordinary shutdown preserves local data.
compose-down: check-db-env
	docker compose --env-file $(ENV_FILE) down

migrate-up: check-db-env
	@$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

# Goose down reverses exactly one applied migration.
migrate-down: check-db-env
	@$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

migrate-status: check-db-env
	@$(if $(MIGRATION_FILES),$(GOOSE) -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status,echo No migration files found in $(MIGRATIONS_DIR).)

# Local
hello:
	@echo Hello World!

test:
	cd server && go test ./...

build:
	cd server && go build ./...

run:
	cd server && go run . serve
