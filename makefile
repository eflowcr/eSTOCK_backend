# eSTOCK Backend — Makefile
# Run from backend/ so paths (db/migrations, .env) resolve correctly.

.PHONY: help server docs build-docker migrate-up migrate-down migrate-up-1 migrate-down-1 create-migration migrate-force

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
	@echo "  server             Run the backend server (go run cmd/main.go)"
	@echo "  docs               Show API docs URLs (route list, OpenAPI, Swagger UI)"
	@echo "  build-docker       Docker buildx and push"
	@echo ""
	@echo "Requires: .env with DATABASE_URL, DB_SOURCE, or DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME; migrate CLI (go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest)"

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

all: build-docker