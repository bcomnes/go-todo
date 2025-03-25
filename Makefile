.PHONY: all build deps generate help test validate \
        migrate-up migrate-down migrate-create

CHECK_FILES ?= $$(go list ./... | grep -v /vendor/)

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: deps generate build test ## Run all steps

build: ## Build all
	go build ./...

release: ## Run GoReleaser
	go tool github.com/goreleaser/goreleaser release --clean

dev: ## Run the development server
	go run ./cmd/server/main.go

deps: ## Download dependencies.
	go mod tidy

generate: ## Run code generation
	go generate ./...

test: ## Run tests
	go test -v $(CHECK_FILES)

migrate-up: ## Run database migrations up (loads .env if present)
	@set -o allexport; \
	if [ -f .env ]; then source .env; fi; \
	go tool github.com/bcomnes/gostgrator/cmd/gostgrator-pg migrate \
	    -conn "$$DATABASE_URL" \
		-migration-pattern "./internal/database/migrations/*.sql" \
		-to "max"

migrate-down: ## Rollback the last migration (loads .env if present)
	@set -o allexport; \
	if [ -f .env ]; then source .env; fi; \
	go tool github.com/bcomnes/gostgrator/cmd/gostgrator-pg down \
		-conn "$$DATABASE_URL" \
		-migration-pattern "./internal/database/migrations/*.sql" \
		down 1

migrate-create: ## Create a new migration (usage: make migrate-create name=add_users)
	@set -o allexport; \
	if [ -f .env ]; then source .env; fi; \
	go tool github.com/bcomnes/gostgrator/cmd/gostgrator-pg new \
		-mode "int" \
		-migration-pattern "./internal/database/migrations/*.sql" \
        -desc "$(name)"
