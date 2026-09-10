.PHONY: dev dev-backend dev-frontend test test-backend test-frontend lint build mongo

# Local MongoDB is expected on localhost:27017 (systemd service or `make mongo` via docker).

dev: ## run backend and frontend together
	@$(MAKE) -j2 dev-backend dev-frontend

dev-backend:
	cd backend && go run ./cmd/api

dev-frontend:
	cd frontend && npm run dev

test: test-backend test-frontend

test-backend:
	cd backend && go test ./...

test-frontend:
	cd frontend && npm test

lint:
	cd backend && go vet ./...
	cd frontend && npm run lint && npx tsc -b

build:
	cd backend && CGO_ENABLED=0 go build -o bin/api ./cmd/api
	cd frontend && npm run build

mongo: ## start MongoDB via docker compose (only if docker is available)
	docker compose up -d mongo
