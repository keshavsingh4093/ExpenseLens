.PHONY: dev build test vet ensure-indexes

dev: ## run the user API with go run
	go run ./cmd/server

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server
	CGO_ENABLED=0 go build -o bin/ensure-indexes ./cmd/scripts/ensure-indexes

test:
	go test ./...

vet:
	go vet ./...

ensure-indexes: ## build indexes + seed system categories (never on the boot path)
	go run ./cmd/scripts/ensure-indexes
