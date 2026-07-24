.PHONY: all build deps dev help test validate web-build web-typecheck \
        migrate-up migrate-down migrate-list migrate-create

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: deps build test ## Run all validation steps.

build: web-build ## Build frontend assets and all Go packages.
	go build ./...

deps: node_modules/.package-lock.json ## Resolve and verify project dependencies.
	go mod tidy
	go mod verify

node_modules/.package-lock.json: package.json package-lock.json
	npm ci

.web-assets.stamp: package.json package-lock.json tsconfig.json pkg/web/global.client.ts pkg/web/global.css | node_modules/.package-lock.json
	npm run build
	@touch $@

web-typecheck: node_modules/.package-lock.json ## Type-check browser TypeScript.
	npm run test:tsc

web-build: web-typecheck .web-assets.stamp ## Type-check and build browser assets with esbuild.

dev: web-build ## Build assets and run the HTTP server (loads .env through the application).
	go run ./cmd/server

test: web-build ## Build assets and run tests.
	go test -v ./...

validate: all ## Run all local validation.

MIGRATION_VERSION ?= max

migrate-up: ## Run PostgreSQL migrations up (use MIGRATION_VERSION=N to stop at N).
	@set -a; \
	if [ -f .env ]; then . ./.env; fi; \
	set +a; \
	go tool github.com/bcomnes/gostgrator/pg \
		-migration-pattern "./migrations/*.sql" \
		-schema-table "public.schemaversion" \
		migrate "$(MIGRATION_VERSION)"

migrate-down: ## Roll back the last migration (loads .env if present).
	@set -a; \
	if [ -f .env ]; then . ./.env; fi; \
	set +a; \
	go tool github.com/bcomnes/gostgrator/pg \
		-migration-pattern "./migrations/*.sql" \
		-schema-table "public.schemaversion" \
		down 1

migrate-list: ## List migrations and the current database version.
	@set -a; \
	if [ -f .env ]; then . ./.env; fi; \
	set +a; \
	go tool github.com/bcomnes/gostgrator/pg \
		-migration-pattern "./migrations/*.sql" \
		-schema-table "public.schemaversion" \
		list

migrate-create: ## Create a migration pair (usage: make migrate-create DESC=add_table).
	$(if $(strip $(DESC)),,$(error DESC is required. Usage: make migrate-create DESC=add_table))
	go tool github.com/bcomnes/gostgrator/pg \
		-mode int \
		-migration-pattern "./migrations/*.sql" \
		-schema-table "public.schemaversion" \
		new "$(DESC)"
