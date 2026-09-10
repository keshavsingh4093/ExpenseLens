.PHONY: dev build test vet ensure-indexes create-admin

dev: ## run the admin API with go run
	go run ./cmd/server

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server
	CGO_ENABLED=0 go build -o bin/ensure-indexes ./cmd/scripts/ensure-indexes
	CGO_ENABLED=0 go build -o bin/create-admin ./cmd/scripts/create-admin

test:
	go test ./...

vet:
	go vet ./...

ensure-indexes: ## build admin indexes (never on the boot path)
	go run ./cmd/scripts/ensure-indexes

create-admin: ## create an admin: make create-admin EMAIL=a@b.c PASSWORD=secret123 NAME="Admin"
	go run ./cmd/scripts/create-admin -email "$(EMAIL)" -password "$(PASSWORD)" -name "$(NAME)" -role "$(ROLE)"
