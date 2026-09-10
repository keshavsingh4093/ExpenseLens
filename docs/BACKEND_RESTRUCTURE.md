# Restructure → dbc-pro conventions (backend AND frontend)

Decided 2026-09-10. Three rules from the user:
1. `backend/` and `frontend/` each get a **`user` / `common` / `admin`** split.
2. Everything shared across portals lives in **`common`**.
3. **No code from outside this repository.** No `replace` directives to sibling repos
   (`~/repos/infra`, `~/repos/dbcpro`), no copying files from them. Third-party modules from the
   Go module proxy / npm registry are fine. dbc-pro and infra may be *read* as references for
   API shape and conventions only.

Reference (read-only): `~/repos/dbcpro/backend/{user,common,admin}` and
`~/repos/dbcpro/frontend/{common,reactjs/user,reactjs/admin}`; `~/repos/infra/golang/pkg/{http,
middleware,database,utils/logger}` for the shape of the helpers we re-implement in `common`.

## Backend target layout

```
backend/
  common/                                  Go module: smartreceipt/backend/common
    go.mod
    pkg/http/        RouteSpec, LoadAPIs, LoadOpenAPIs, response envelope (Success/Created/
                     NoContent/Error/ErrorWithCode/BadRequest/…/ValidationError(fields)), NewServer
                     (graceful shutdown, CORS wrap)
    pkg/middleware/  ConfigureAuth(secret, claimsFactory), EnsureAuth, WithTimeout,
                     RecoveryAndClean, Logging(slog), CORS(config), RequestID
    pkg/database/    ConfigureMongo, Connect, Disconnect, GetDBConnection (mongo-driver v2),
                     IndexSet + EnsureIndexes helper
    pkg/logger/      thin slog wrapper: Init, Info/Warn/Error/Debug/Fatal(ctx, msg, …)
    pkg/config/      BaseConfig + LoadBase (.env → ENV_FILE/.env.local overlay), dynamic
                     system_config (InitDynamicConfig, GetDynamic, Reload), LoadCORS
    pkg/validate/    Email, Password, RequiredString, ObjectID, struct validation → fields map
    pkg/models/      user.go, receipt.go, category.go, auth_token.go, admin_user.go, system_config.go
    pkg/storage/     Provider interface, local (signed URLs), cloudinary
    pkg/extraction/  Extractor interface, gemini, fake, Pool
    pkg/mail/        Sender: log | smtp
    services/auth/   password hashing, access JWT, opaque tokens (refresh/reset), OIDC providers,
                     session issue/refresh/logout — reused by user and admin portals
    services/user/   users collection access (FindByID/Email/Provider, Create, Update, Link, Delete)
    services/category/  categories collection (+ Seed of system defaults)
    services/receipt/   receipts collection (repo-level helpers used by user API and admin stats)
  user/                                    Go module: smartreceipt/backend/user  (replace ../common)
    go.mod, Makefile, .env.example
    cmd/server/main.go                     app.Run()
    cmd/scripts/ensure-indexes/main.go     app.ConnectForTooling + app.EnsureAllIndexes (+ seed)
    internal/app/{server.go,indexes.go}
    internal/config/config.go              AppConfig, Load, Reload, GetConfigVal
    internal/http/router/routersetup.go
    internal/http/openapi/{openapi.go,health,auth,files}
    internal/http/api/{api.go,user,receipts,categories,analytics}
    internal/pkg/middleware/auth.go        UserClaims{UserID,Email,Name} + NewUserClaims + GetUserFromContext
    internal/services/{receipt,analytics}  user-portal business logic (upload/extract/confirm, aggregates)
  admin/                                   Go module: smartreceipt/backend/admin (replace ../common)
    same skeleton as user: cmd/server, cmd/scripts/ensure-indexes, internal/{app,config,http,pkg,services}
    Phase 2.5 scope: /openapi/health, /openapi/admin/login (admin_users collection, argon2id,
    AdminClaims), /api/admin/me, /api/admin/users (list/search users), /api/admin/stats
    (user count, receipts count, extraction failures), /api/admin/system-config (get/set keys).
    Default port 8091.
```

Mapping from the current flat code:
| Current `backend/internal/…` | Target |
|---|---|
| `server` (chi, JSON helpers, Validate) | `common/pkg/http`, `common/pkg/validate` |
| `db` | `common/pkg/database` |
| `config` | `common/pkg/config` (base + dynamic) + `user/internal/config` (AppConfig) |
| `auth/{password,jwt,tokens,oauth,service}.go` | `common/services/auth` |
| `auth/handler.go` | `user/internal/http/openapi/auth` + `user/internal/http/api/user` |
| `auth/middleware.go` | `common/pkg/middleware` (EnsureAuth) + `user/internal/pkg/middleware/auth.go` (claims) |
| `users/{model,repo}.go`, `users/handler.go` | `common/pkg/models/user.go`, `common/services/user`, `user/internal/http/api/user` |
| `categories/*` | `common/pkg/models/category.go`, `common/services/category`, `user/internal/http/api/categories` |
| `receipts/{model,repo}.go` | `common/pkg/models/receipt.go`, `common/services/receipt` |
| `receipts/{service,handler}.go` | `user/internal/services/receipt`, `user/internal/http/api/receipts` |
| `analytics/*` | `user/internal/services/analytics`, `user/internal/http/api/analytics` |
| `storage`, `extraction`, `mail` | `common/pkg/{storage,extraction,mail}` |

Rules carried over from dbc-pro:
1. **No index builds on the boot path.** `app.Run()` may only check + warn. Index creation and the
   category seed run in `cmd/scripts/ensure-indexes` (`make ensure-indexes`).
2. **Boot sequence**: `config.Load()` → `logger.Init()` → `database.ConfigureMongo` + `Connect()` →
   `InitDynamicConfig` (warn on failure) → `config.Reload()` → validate production secrets →
   storage + extractor + worker pool (workerCtx, stopped in defer) → `router.InitRouter()` →
   `LoadCORS` → `gohttp.NewServer(ServerConfig{Port, CORS}, r).Run()`.
3. **Config layering**: `.env` bootstrap (`ENV`, `PORT`, `MONGO_URI`, `MONGO_DATABASE`, `JWT_SECRET`)
   → `system_config` collection → code defaults, via `GetConfigVal`. Local dev may put every key
   in `.env.local`; `.env.example` documents all keys.
4. **Handlers**: package-level `var svc = domain.New()`, `GetRouteSpecs() []gohttp.RouteSpec`,
   `handleX(svc) http.HandlerFunc` closures, claims via `middleware.GetUserFromContext`. Responses
   via the `common/pkg/http` envelope `{success, data, error, code}`; validation → 422 with `fields`.
5. **Services**: `Service` struct + `New()`; collection access in `helper.go` via
   `database.GetDBConnection().Collection(name)`; `EnsureIndexes(ctx)` per service.
6. **Auth semantics unchanged**: access JWT 15m + rotating httpOnly refresh cookie (30s grace),
   go-oidc social login, argon2id. Tokens carry `UserClaims` so `EnsureAuth` validates them.
   Cookie path `/openapi/auth`. Multipart routes set `TimeOut: 120`.
7. **Paths**: `/openapi/…` unauthenticated, `/api/…` authenticated. Health `GET /openapi/health`.

Keep mongo-driver **v2** (we own `common/pkg/database`, so there is no reason to downgrade).

## Frontend target layout

```
frontend/
  package.json                npm workspaces: ["common", "user", "admin"]; root scripts fan out
  common/                     package @expenselens/common (no build step; consumed via alias)
    package.json, tsconfig.json, tailwind.preset.css (shared @theme tokens + utilities)
    src/api/client.ts         fetch wrapper, envelope unwrap, refresh single-flight (base path
                              configurable: user → /openapi,/api ; admin → /openapi/admin,/api/admin)
    src/auth/                 AuthProvider/useAuth generic over a User type, RequireAuth, GuestOnly
    src/theme/                ThemeProvider, themeContext, useTheme
    src/ui/                   Button, Field/Input/Select, Toggle, Alert, Skeleton, EmptyState,
                              Chip, Modal/ConfirmDialog, PageHeader, StatusBadge
    src/charts/               StatTile, TrendChart, CategoryDonut, donutSlices
    src/format/               money.ts, date.ts (+ tests)
    src/hooks/                useDebounce
    src/categories/           colors.ts (validated palette), CategoryIcon/Badge, types
    src/index.ts
  user/                       the current app (Vite): src/app, src/features/{auth,dashboard,upload,
                              receipts,analytics,budgets,warranty,categories,settings,notifications,
                              reports}; imports shared code from '@common/…'; port 5173
  admin/                      new Vite app: login, dashboard (stats), users list, system config
                              editor; imports '@common/…'; port 5174; proxies /openapi + /api → :8091
```

Wiring: each app's `vite.config.ts` sets `resolve.alias['@common'] = ../common/src` and
`server.fs.allow` includes `../common`; `tsconfig.app.json` `paths` `{"@common/*": ["../common/src/*"]}`;
Tailwind v4 `@import '../../common/tailwind.preset.css'` from each app's `index.css` so tokens stay
single-sourced. Root `package.json` scripts: `dev:user`, `dev:admin`, `build`, `lint`, `test`.

Frontend API changes: `BASE=''`; user endpoints become `/openapi/auth/…`, `/api/…`; unwrap
`json.data`; on `!success` throw `ApiRequestError{status, code, message: error, fields}`;
Vite proxy forwards both `/api` and `/openapi`.

## Root tooling
- Root `Makefile`: `dev` (user backend :8090 + user frontend :5173), `dev-admin` (admin backend
  :8091 + admin frontend :5174), `test`, `lint`, `build`, `ensure-indexes` (both services).
- `docker-compose.yml` unchanged (Mongo only).

## Verification checklist
- `cd backend/common && go test ./...`; `cd backend/user && go build ./... && go vet ./... && go test ./...`;
  same for `backend/admin`.
- `make ensure-indexes`, then `EXTRACTOR=fake make dev-backend` → `GET :8090/openapi/health`
  returns `{success:true,data:{status:"healthy"}}`; admin `:8091/openapi/health` likewise.
- API smoke (44 checks) adapted to new paths + envelope; Playwright flows (auth/settings, full
  receipt flow) at 1280px and 400px; admin login + users list smoke.
- Commit as "Restructure to user/common/admin layout (Phase 2.5)".

## Non-goals for Phase 2.5
- No Postgres, no workspaces/tenants, no plan gates, no help portal.
- No behaviour changes in the user portal beyond paths and envelope.
