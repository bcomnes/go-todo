.PHONY: all build deps generate help test validate \
        migrate-up migrate-down migrate-list migrate-create release dev

CHECK_FILES ?= $(shell go list ./... | grep -v /vendor/)

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

# Capture any extra argument passed to migrate-up; if none, default to "max".
migrate-up-version := $(if $(strip $(word 2,$(MAKECMDGOALS))),$(word 2,$(MAKECMDGOALS)),"max")
# Remove that extra argument from the make targets.
$(eval $(word 2,$(MAKECMDGOALS)):;@:)

migrate-up: ## Run database migrations up (loads .env if present)
	@set -o allexport; \
	if [ -f .env ]; then source .env; fi; \
	go tool github.com/bcomnes/gostgrator/cmd/gostgrator-pg \
		-migration-pattern "./internal/database/migrations/*.sql" \
		migrate "$(migrate-up-version)"

migrate-down: ## Rollback the last migration (loads .env if present)
	@set -o allexport; \
	if [ -f .env ]; then source .env; fi; \
	go tool github.com/bcomnes/gostgrator/cmd/gostgrator-pg \
	   -migration-pattern "./internal/database/migrations/*.sql" \
	   down 1

migrate-list: ## List all migrations (loads .env if present)
	@set -o allexport; \
	if [ -f .env ]; then source .env; fi; \
	go tool github.com/bcomnes/gostgrator/cmd/gostgrator-pg \
		-migration-pattern "./internal/database/migrations/*.sql" \
		list

# Capture all extra arguments after 'migrate-create' into desc.
desc := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
# Remove desc from target list.
$(eval $(desc):;@:)

migrate-create: ## Create a new migration (usage: make migrate-create "add_users")
	$(if $(strip $(desc)),, $(error Please provide a migration description. Usage: make migrate-create "add_users"))
	go tool github.com/bcomnes/gostgrator/cmd/gostgrator-pg \
	   -mode "int" \
	   -migration-pattern "./internal/database/migrations/*.sql" \
	   new "$(desc)"
