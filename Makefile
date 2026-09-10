.PHONY: dev dev-backend dev-frontend dev-admin dev-admin-backend dev-admin-frontend \
        test test-backend test-frontend lint lint-backend lint-frontend build build-backend build-frontend \
        ensure-indexes install mongo

# Local MongoDB is expected on localhost:27017 (systemd service or `make mongo` via docker).
# Layout: backend/{common,user,admin} Go modules, frontend/{common,user,admin} npm workspaces.

BACKENDS = common user admin

dev: ## user API :8090 + user frontend :5173
	@$(MAKE) -j2 dev-backend dev-frontend

dev-admin: ## admin API :8091 + admin frontend :5174
	@$(MAKE) -j2 dev-admin-backend dev-admin-frontend

dev-backend:
	cd backend/user && go run ./cmd/server

dev-frontend:
	cd frontend && npm run dev:user

dev-admin-backend:
	cd backend/admin && go run ./cmd/server

dev-admin-frontend:
	cd frontend && npm run dev:admin

install: ## frontend workspaces + go module downloads
	cd frontend && npm install
	for m in $(BACKENDS); do (cd backend/$$m && go mod download); done

ensure-indexes: ## build Mongo indexes + seed categories for both services (never on the boot path)
	cd backend/user && go run ./cmd/scripts/ensure-indexes
	cd backend/admin && go run ./cmd/scripts/ensure-indexes

create-admin: ## make create-admin EMAIL=a@b.c PASSWORD=secret123 NAME="Admin" [ROLE=superadmin]
	cd backend/admin && go run ./cmd/scripts/create-admin -email "$(EMAIL)" -password "$(PASSWORD)" -name "$(NAME)" -role "$(ROLE)"

test: test-backend test-frontend

test-backend:
	for m in $(BACKENDS); do (cd backend/$$m && go test ./...) || exit 1; done

test-frontend:
	cd frontend && npm test

lint: lint-backend lint-frontend

lint-backend:
	for m in $(BACKENDS); do (cd backend/$$m && go vet ./...) || exit 1; done

lint-frontend:
	cd frontend && npm run lint && npm run typecheck

build: build-backend build-frontend

build-backend:
	cd backend/user && CGO_ENABLED=0 go build -o bin/server ./cmd/server && CGO_ENABLED=0 go build -o bin/ensure-indexes ./cmd/scripts/ensure-indexes
	cd backend/admin && CGO_ENABLED=0 go build -o bin/server ./cmd/server && CGO_ENABLED=0 go build -o bin/ensure-indexes ./cmd/scripts/ensure-indexes && CGO_ENABLED=0 go build -o bin/create-admin ./cmd/scripts/create-admin

build-frontend:
	cd frontend && npm run build

mongo: ## start MongoDB via docker compose (only if docker is available)
	docker compose up -d mongo
