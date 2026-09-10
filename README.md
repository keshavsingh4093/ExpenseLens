# ExpenseLens (smart-receipt)

Receipt-first personal expense tracker. Go + MongoDB APIs, React + Vite frontends.

See `docs/PLAN.md` for scope and architecture, `docs/HANDOVER.md` for current status, and
`docs/BACKEND_RESTRUCTURE.md` for the user/common/admin layout conventions.

```
backend/   common (shared Go module) · user (API :8090) · admin (API :8091)
frontend/  common (shared UI/api)    · user (Vite :5173) · admin (Vite :5174)
```

## Run locally

Requires Go 1.26+, Node 22+, and MongoDB on `localhost:27017` (local service or `make mongo`).

```sh
cp backend/user/.env.example backend/user/.env     # set EXTRACTOR=fake for offline dev
cp backend/admin/.env.example backend/admin/.env
make install
make ensure-indexes                                 # indexes + system categories (run before first boot)
make create-admin EMAIL=admin@example.com PASSWORD=secret123 NAME=Admin ROLE=superadmin
make dev                                            # user API :8090 + user app :5173
make dev-admin                                      # admin API :8091 + admin app :5174
make test && make lint
```

API paths: `/openapi/…` unauthenticated, `/api/…` authenticated. Every JSON response uses the
`{success, data, error, code, fields}` envelope.
