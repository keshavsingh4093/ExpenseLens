# Backend restructure → dbc-pro conventions

Goal: make `backend/` look and behave like `~/repos/dbcpro/backend/user` so the team can work on
both with the same muscle memory, and reuse the shared infra library.

Reference repos on this machine (read-only, do NOT modify):
- `~/repos/dbcpro/backend/user` — the layout to mirror (`cmd/server`, `internal/app`,
  `internal/config`, `internal/http/{api,openapi,router}`, `internal/pkg/*`, `internal/services/*`).
- `~/repos/dbcpro/backend/CLAUDE.md` — boot-path and `.env` policy rules (apply them here too).
- `~/repos/infra/golang` — module `gitlab.tezminds.com/infra/golang`; consumed via a local
  `replace`. Provides `pkg/http` (RouteSpec, LoadAPIs/LoadOpenAPIs, response envelope, NewServer),
  `pkg/middleware` (ConfigureAuth, EnsureAuth, CORS, Logging, WithTimeout, RecoveryAndClean),
  `pkg/database` (ConfigureMongo, Connect, GetDBConnection — mongo-driver **v1**),
  `pkg/utils/logger`, `pkg/utils/config` (GetEnv), `pkg/oauth` (Google/Microsoft/Apple verifiers).

## Target layout (single service, mirrors `dbcpro/backend/user`)

```
backend/
  go.mod                        module gitlab.tezminds.com/smart-receipt/backend ; go 1.25.0
                                replace gitlab.tezminds.com/infra/golang => ../../infra/golang
  Makefile                      build / run / dev(air) / test / ensure-indexes
  .env.example                  bootstrap keys + documented optional keys
  cmd/server/main.go            app.Run()
  cmd/scripts/ensure-indexes/   app.ConnectForTooling + app.EnsureAllIndexes (+ category seed)
  internal/app/server.go        Run(): boot sequence (below)
  internal/app/indexes.go       EnsureAllIndexes, ConnectForTooling
  internal/config/config.go     AppConfig, Load(), Reload(), GetConfigVal(env → dynamic → fallback)
  internal/http/router/routersetup.go   InitRouter(): ConfigureAuth, Logging, LoadOpenAPIs, LoadAPIs
  internal/http/openapi/openapi.go      GetRouteSpecs() aggregator (unauthenticated, /openapi/…)
  internal/http/openapi/health/
  internal/http/openapi/auth/           register, login, refresh, logout, forgot/reset, providers,
                                        {provider}/start, {provider}/callback
  internal/http/openapi/files/          signed local-storage file server (/openapi/files/*)
  internal/http/api/api.go              GetRouteSpecs() aggregator (authenticated, /api/…)
  internal/http/api/user/               /api/me, /api/me/preferences, /api/me/password, DELETE /api/me
  internal/http/api/receipts/           /api/receipts…
  internal/http/api/categories/         /api/categories…
  internal/http/api/analytics/          /api/analytics/…
  internal/pkg/config/                  BaseConfig LoadBase, dynamic system_config (GetDynamic,
                                        InitDynamicConfig, EnsureConfigIndexes), LoadCORS
  internal/pkg/middleware/auth.go       UserClaims{UserID,Email,Name,RegisteredClaims}, NewUserClaims,
                                        GetUserFromContext
  internal/pkg/models/                  user.go, receipt.go, category.go, token.go (bson v1,
                                        primitive.ObjectID)
  internal/pkg/validate/                email/password/objectid helpers (+ struct validation shim)
  internal/pkg/storage/                 Provider interface, local (signed URLs), cloudinary
  internal/pkg/extraction/              Extractor interface, gemini, fake, Pool
  internal/pkg/mail/                    Sender: log | smtp
  internal/services/auth/               service.go (register/login/session/refresh/reset/social),
                                        helper.go (auth_tokens collection, EnsureIndexes), oauth.go
  internal/services/user/               service.go, helper.go (users collection)
  internal/services/category/           service.go, helper.go (+ Seed)
  internal/services/receipt/            service.go (upload/extract/confirm/…), helper.go (repo)
  internal/services/analytics/          service.go (aggregations), helper.go
```

Mapping from the current code:
| Current | Target |
|---|---|
| `internal/server` (chi, JSON helpers, Validate) | infra `gohttp` + `internal/pkg/validate`; drop |
| `internal/db` | infra `pkg/database`; drop |
| `internal/auth/{service,oauth}.go` | `internal/services/auth/{service,oauth}.go` |
| `internal/auth/{jwt,password,tokens}.go` | `internal/services/auth/helper.go` (+ `internal/pkg/middleware/auth.go` for claims) |
| `internal/auth/handler.go` | `internal/http/openapi/auth/auth.go` + `internal/http/api/user/user.go` |
| `internal/users/{model,repo,handler}.go` | `internal/pkg/models/user.go`, `internal/services/user/*`, `internal/http/api/user/*` |
| `internal/categories/*` | `internal/pkg/models/category.go`, `internal/services/category/*`, `internal/http/api/categories/*` |
| `internal/receipts/*` | `internal/pkg/models/receipt.go`, `internal/services/receipt/*`, `internal/http/api/receipts/*` |
| `internal/analytics/*` | `internal/services/analytics/*`, `internal/http/api/analytics/*` |
| `internal/storage`, `internal/extraction`, `internal/mail` | `internal/pkg/{storage,extraction,mail}` |

## Rules to carry over from dbc-pro

1. **No index builds on the boot path.** `Run()` may only *check* indexes and warn. Index creation
   and the category seed live in `app.EnsureAllIndexes`, run by `cmd/scripts/ensure-indexes`
   (`make ensure-indexes`) before starting the server.
2. **Boot sequence in `app.Run()`**: `config.Load()` → `logger.InitLogger(nil,nil,"")` →
   `db.ConfigureMongo` + `db.Connect()` → `InitDynamicConfig(mongoDB)` (warn on failure) →
   `config.Reload()` → validate production secrets → init storage provider + extractor + worker
   pool (workerCtx, stopped in defer) → `router.InitRouter()` → `LoadCORS` →
   `gohttp.NewServer(gohttp.ServerConfig{Port, LogFile: "logs/app.log", CORSConfig}, r).Run()`.
3. **Config layering**: `.env` (bootstrap: `ENV`, `PORT`, `MONGO_URI`, `MONGO_DATABASE`,
   `JWT_SECRET`) → `system_config` collection (dynamic) → code defaults, via `GetConfigVal`.
   For local dev it is fine to put every key in `.env.local`; document all keys in `.env.example`.
4. **Handlers**: package-level `var svc = domain.New()`, `GetRouteSpecs() []gohttp.RouteSpec`,
   `handleX(svc) http.HandlerFunc` closures, `chi.URLParam` for params, claims via
   `middleware.GetUserFromContext(ctx)`. Responses via `gohttp.Success/Created/NoContent/Error/
   ErrorWithCode/BadRequest/...`. Validation failures: 422 with
   `{success:false, error, code:"validation_failed", fields:{...}}` (use `gohttp.JSON` with a map).
5. **Services**: `Service` struct + `New()`; Mongo access in `helper.go` through
   `db.GetDBConnection().Collection(name)`; `EnsureIndexes(ctx)` per service; models in
   `internal/pkg/models`.
6. **Auth**: keep our semantics (access JWT 15m + rotating httpOnly refresh cookie with grace,
   OIDC via go-oidc, argon2id) but issue tokens with `UserClaims` so infra `EnsureAuth` validates
   them. Cookie path becomes `/openapi/auth`. Multipart upload routes set `TimeOut: 120`.
7. **Paths**: unauthenticated under `/openapi/…`, authenticated under `/api/…` (no `/api/v1`).
   Health at `GET /openapi/health`.

## Frontend changes required

- `lib/api/client.ts`: `BASE = ''`; feature api modules use `/openapi/auth/…` and `/api/…`.
  Unwrap the envelope: return `json.data`; on `!success` throw `ApiRequestError{status, code,
  message: error, fields}`. Refresh endpoint: `POST /openapi/auth/refresh`.
- `vite.config.ts` proxy: both `/api` and `/openapi` → `http://localhost:8090`.
- `authApi.oauthStartUrl` → `/openapi/auth/${provider}/start`; API callback redirect unchanged
  (`APP_URL/auth/callback`).
- Image URLs come from the API (`/openapi/files/...` in local mode) — no change.

## Verification checklist

- `go build ./... && go vet ./... && go test ./...` in `backend/` (keep the existing unit tests:
  password/JWT, local storage, extraction parseWire, analytics range/rollup).
- `make ensure-indexes` then `EXTRACTOR=fake make run` → `GET /openapi/health` returns
  `{success:true,data:{status:"healthy"}}`.
- Re-run the API smoke (44 checks) adapted to new paths + envelope, then the Playwright flows
  (auth/settings, full receipt flow) at 1280px and 400px. Commit as
  "Restructure backend to dbc-pro layout".

## Non-goals (for this restructure)
- No admin/help services, no Postgres, no workspace/tenant middleware, no plan gates.
- No behaviour changes beyond paths and response envelope.
