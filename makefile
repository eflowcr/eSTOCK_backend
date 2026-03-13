# eSTOCK Backend — Makefile
# Run from backend/ so paths (db/migrations, .env) resolve correctly.

.PHONY: help server docs build-docker migrate-up migrate-down migrate-up-1 migrate-down-1 create-migration migrate-force db-drop-all sqlc sqlc-validate sqlc-clean sqlc-diff test test-unit test-articles-integration

# DB URL from .env: prefer DATABASE_URL or DB_SOURCE; else build from DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME.
get-db-url = $(shell \
	url=$$(grep -E '^(DATABASE_URL|DB_SOURCE)=' .env 2>/dev/null | head -1 | sed 's/^[^=]*=//'); \
	if [ -n "$$url" ]; then echo "$$url"; else \
		H=$$(grep '^DB_HOST=' .env 2>/dev/null | cut -d'=' -f2-); \
		Pt=$$(grep '^DB_PORT=' .env 2>/dev/null | cut -d'=' -f2-); \
		U=$$(grep '^DB_USER=' .env 2>/dev/null | cut -d'=' -f2-); \
		W=$$(grep '^DB_PASSWORD=' .env 2>/dev/null | cut -d'=' -f2-); \
		N=$$(grep '^DB_NAME=' .env 2>/dev/null | cut -d'=' -f2-); \
		if [ -n "$$H" ] && [ -n "$$N" ]; then echo "postgres://$${U}:$${W}@$${H}:$${Pt:-5432}/$${N}?sslmode=disable"; fi; \
	fi \
)

.DEFAULT_GOAL := help

help: ## Show available commands
	@echo "eSTOCK Backend — usage: make <target>"
	@echo ""
	@echo "  migrate-up         Apply all pending migrations"
	@echo "  migrate-down       Revert last migration"
	@echo "  migrate-up-1       Apply one migration"
	@echo "  migrate-down-1     Revert one migration"
	@echo "  create-migration   Create new migration (name=short_description)"
	@echo "  migrate-force      Force schema version (version=N)"
	@echo "  db-drop-all        Drop all tables in public schema (requires confirmation)"
	@echo "  server             Run the backend server (go run cmd/main.go)"
	@echo "  docs               Show API docs URLs (route list, OpenAPI, Swagger UI)"
	@echo "  build-docker       Docker buildx and push"
	@echo "  sqlc               Generate sqlc code from db/query/*.sql (writes db/sqlc/)"
	@echo "  sqlc-validate      Validate sqlc config and compile queries"
	@echo "  sqlc-clean         Remove generated db/sqlc/"
	@echo "  sqlc-diff          Show schema diff (optional)"
	@echo "  test                     Run all tests (unit only with -short)"
	@echo "  test-unit                Run unit tests only (no Docker)"
	@echo "  test-articles-integration  Run ArticlesRepositorySQLC integration tests (requires Docker)"
	@echo ""
	@echo "Requires: .env with DATABASE_URL, DB_SOURCE, or DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME; migrate CLI; sqlc (go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest)"

migrate-up: ## Apply all pending migrations
	@DB_URL="$(call get-db-url)"; \
	if [ -z "$$DB_URL" ]; then echo "Error: set DATABASE_URL, DB_SOURCE, or DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME in .env"; exit 1; fi; \
	echo "Applying migrations..."; \
	migrate -path db/migrations -database "$$DB_URL" -verbose up; \
	echo "Done."

migrate-down: ## Revert last migration
	@DB_URL="$(call get-db-url)"; \
	if [ -z "$$DB_URL" ]; then echo "Error: set DATABASE_URL, DB_SOURCE, or DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME in .env"; exit 1; fi; \
	echo "Reverting last migration..."; \
	migrate -path db/migrations -database "$$DB_URL" -verbose down; \
	echo "Done."

migrate-up-1: ## Apply one migration
	@DB_URL="$(call get-db-url)"; \
	if [ -z "$$DB_URL" ]; then echo "Error: set DATABASE_URL, DB_SOURCE, or DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME in .env"; exit 1; fi; \
	echo "Applying 1 migration..."; \
	migrate -path db/migrations -database "$$DB_URL" -verbose up 1; \
	echo "Done."

migrate-down-1: ## Revert one migration
	@DB_URL="$(call get-db-url)"; \
	if [ -z "$$DB_URL" ]; then echo "Error: set DATABASE_URL, DB_SOURCE, or DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME in .env"; exit 1; fi; \
	echo "Reverting 1 migration..."; \
	migrate -path db/migrations -database "$$DB_URL" -verbose down 1; \
	echo "Done."

create-migration: ## Create new migration (usage: make create-migration name=add_articles_table)
	@if [ -z "$(name)" ]; then read -p "Migration name: " name; fi; \
	name=$${name:-unnamed}; \
	latest=$$(ls -1 db/migrations/*.up.sql 2>/dev/null | sed 's/.*\/\([0-9]*\)_.*/\1/' | sort -n | tail -1); \
	latest=$${latest:-0}; next=$$(printf "%06d" $$((latest + 1))); \
	touch "db/migrations/$${next}_$${name}.up.sql" "db/migrations/$${next}_$${name}.down.sql"; \
	echo "Created db/migrations/$${next}_$${name}.up.sql and .down.sql"

migrate-force: ## Force schema version (usage: make migrate-force version=0)
	@DB_URL="$(call get-db-url)"; \
	if [ -z "$$DB_URL" ]; then echo "Error: set DATABASE_URL, DB_SOURCE, or DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME in .env"; exit 1; fi; \
	if [ -n "$(version)" ]; then v="$(version)"; else read -p "Version to force: " v; fi; \
	echo "Forcing version $$v..."; \
	migrate -path db/migrations -database "$$DB_URL" force $$v; \
	echo "Done."

db-drop-all: ## Drop all tables in public schema (run migrate-up after to restore)
	@DB_URL="$(call get-db-url)"; \
	if [ -z "$$DB_URL" ]; then echo "Error: set DATABASE_URL or DB_* in .env"; exit 1; fi; \
	echo "WARNING: This will drop ALL tables in public schema for $$DB_URL"; \
	read -p "Type 'yes' to confirm: " confirm; \
	if [ "$$confirm" = "yes" ]; then \
		psql "$$DB_URL" -f db/scripts/drop_all_tables.sql; \
		echo "Done. Run 'make migrate-up' to reapply migrations."; \
	else \
		echo "Aborted."; \
	fi

server: ## Run the backend server
	go run cmd/main.go

docs: ## Show API docs URLs (run server first, then open in browser)
	@echo "API docs (run 'make server' first):"
	@echo "  Route list:    http://localhost:8080/api/docs/routes"
	@echo "  OpenAPI spec:  http://localhost:8080/api/docs/openapi.json"
	@echo "  Swagger UI:    http://localhost:8080/swagger/index.html"

# Docker build (original target)
build-docker: ## Docker buildx and push
	docker buildx build --platform linux/amd64,linux/arm64 -t epracsupply/estock_backend:v1.0.2 . --push

# -----------------------------------------------------------------------------
# sqlc — type-safe SQL codegen (see eSTOCK doc/Roadmap - SQLC Backend.md)
# -----------------------------------------------------------------------------
sqlc: ## Generate sqlc code from db/query/*.sql into db/sqlc/
	@command -v sqlc >/dev/null 2>&1 || (echo "sqlc not found: run go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest" && exit 1)
	sqlc generate
	@echo "Generated code in db/sqlc/"

sqlc-validate: ## Validate sqlc config and compile queries (no codegen)
	@command -v sqlc >/dev/null 2>&1 || (echo "sqlc not found: run go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest" && exit 1)
	sqlc compile
	@echo "sqlc config and queries are valid"

sqlc-clean: ## Remove generated db/sqlc/
	rm -rf db/sqlc/
	@echo "Removed db/sqlc/"

sqlc-diff: ## Show database schema diff (requires DB connection)
	@command -v sqlc >/dev/null 2>&1 || (echo "sqlc not found" && exit 1)
	@DB_URL="$(call get-db-url)"; \
	if [ -z "$$DB_URL" ]; then echo "Error: set DATABASE_URL or DB_* in .env for sqlc diff"; exit 1; fi; \
	sqlc diff

test: ## Run all tests (use -short to skip integration tests)
	go test ./... -count=1

test-unit: ## Run unit tests only (skips integration tests that need Docker)
	go test ./... -short -count=1

test-articles-integration: ## Run ArticlesRepositorySQLC integration tests (requires Docker; skips if unavailable)
	go test -v ./repositories/... -run TestArticlesRepositorySQLC -count=1

all: build-docker