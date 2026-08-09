DB_URL ?= "postgres://postgres:postgres@localhost:5432/typstlab?sslmode=disable"
MIGRATIONS_DIR = migrations

.PHONY: run test lint migrate-create migrate-up migrate-down migrate-force help

## help: Print available Makefile commands
help:
	@echo "Available commands:"
	@echo "  make run            - Run application locally"
	@echo "  make test           - Run all tests"
	@echo "  make lint           - Run golangci-lint"
	@echo "  make migrate-create - Create a new pair of migration files (.up.sql / .down.sql)"
	@echo "  make migrate-up     - Apply all pending migrations"
	@echo "  make migrate-down   - Roll back 1 migration"
	@echo "  make migrate-force  - Force migration version (useful for dirty state recovery)"

## run: Run the application
run:
	go run ./cmd/server/main.go

## test: Run unit and integration tests
test:
	go test -v ./...

## lint: Run linter
lint:
	golangci-lint run

## migrate-create: Create a new pair of up/down SQL migrations
migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $$name

## migrate-up: Apply all pending migrations to the local database
migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database $(DB_URL) up

## migrate-down: Roll back 1 migration
migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database $(DB_URL) down 1

## migrate-force: Force migration version
migrate-force:
	@read -p "Enter version to force: " version; \
	migrate -path $(MIGRATIONS_DIR) -database $(DB_URL) force $$version
