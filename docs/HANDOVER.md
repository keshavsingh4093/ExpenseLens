# Handover

Rewrite this file (do not append) at the end of each session or when context nears 200–250k tokens.

## Last updated
2026-09-10 — Phase 2.5 (restructure to `user`/`common`/`admin` on both sides) is DONE and
committed on local `main`. NEXT: Phase 3 (Budgets, Analytics screen, Warranty, Notifications,
Reports, Categories screen, More screen) in the new layout.

## Status by phase
- Phase 0 Scaffolding: done
- Phase 1 Auth: done (OAuth code complete; untested against live Google/Microsoft/Apple)
- Phase 2 Core receipt flow: done (Gemini + Cloudinary untested live; fake/local verified)
- Phase 2.5 Restructure: done (see below)
- Phase 3 Insights: not started
- Phase 5 Admin portal: scaffold only (login, dashboard stats, users list/detail, system config)
- Phase 4 Post-MVP: not started

## Layout (what exists now)
Backend — three Go modules, no code from outside the repo, mongo-driver v2:
- `backend/common` (`smartreceipt/backend/common`): `pkg/http` (RouteSpec + LoadAPIs/LoadOpenAPIs,
  envelope helpers, `APIError` + `WriteError`, `Decode`, `NewServer` w/ graceful shutdown + CORS),
  `pkg/middleware` (ConfigureAuth/EnsureAuth/ParseToken, WithTimeout w/ guarded writer,
  RecoveryAndClean, Logging, RequestID, CORS), `pkg/database` (ConfigureMongo/Connect/
  GetDBConnection, IndexSet, EnsureIndexes, MissingIndexes), `pkg/logger` (slog wrapper),
  `pkg/config` (BaseConfig/LoadBase w/ ENV_FILE + .env.local overlay, GetConfigVal = env →
  system_config → default, dynamic config cache, LoadCORS), `pkg/validate`, `pkg/models`
  (user, receipt, category, auth_token, admin_user, system_config), `pkg/storage` (local signed URLs
  + cloudinary), `pkg/extraction` (gemini, fake, pool), `pkg/mail`; `services/auth` (argon2id,
  SignToken, opaque TokenStore for refresh/reset/admin_refresh, OIDC providers, session service),
  `services/{user,category,receipt,systemconfig,adminuser}` (collection access, `EnsureIndexes`).
- `backend/user` (:8090): `cmd/server`, `cmd/scripts/ensure-indexes` (indexes + category seed),
  `internal/app` (boot sequence per spec; only *checks* indexes and warns), `internal/config`
  (AppConfig, Load/Reload/GetConfigVal/ValidateProductionSecrets), `internal/http/router`,
  `internal/http/openapi/{health,auth,files,configreload}`, `internal/http/api/{user,receipts,
  categories,analytics}`, `internal/pkg/middleware` (UserClaims w/ `type:"user"`, GetUserFromContext,
  UserID), `internal/services/{auth (facade + providers + delete cascade), receipt (upload/extract/
  confirm/images; deps via `receipt.Configure`), analytics}`.
- `backend/admin` (:8091): same skeleton; `cmd/scripts/create-admin`; AdminClaims (`type:"admin"`,
  rejected by the user API and vice versa); routes `/openapi/admin/login`, `/openapi/admin/auth/
  {refresh,logout}` (cookie `el_admin_refresh`, path `/openapi/admin/auth`), `/api/admin/{me,users,
  users/{id},stats,system-config[/{key}|/reload]}`. Sensitive config values are masked; re-saving a
  masked value keeps the secret. Reload also pings the user backend's `/openapi/config/reload` when
  `CONFIG_RELOAD_SECRET` is set on both sides.

Frontend — npm workspaces in `frontend/` (`common`, `user`, `admin`):
- `common/src`: `api/client.ts` (absolute paths `/openapi/…` | `/api/…`, envelope unwrap, throws
  `ApiRequestError{status,code,message,fields}`, single-flight refresh on 401 for `/api/*`,
  `configureApi({refreshPath})`), `auth/` (generic `AuthProvider<TUser>` taking `bootstrap`/`logout`
  fns, `useAuth<T>()`, `RequireAuth`, `GuestOnly`), `theme/`, `ui/`, `charts/`, `format/`, `hooks/`,
  `categories/` (palette, icons, types), `analytics/types.ts`, `types.ts` (ReceiptStatus).
  `tailwind.preset.css` holds the tokens; each app's `index.css` imports tailwind + preset + `@source`.
- `user/`: the Phase 0–2 app, imports `@common/…`; `features/auth/useAuth.ts` is the typed wrapper.
- `admin/`: login, dashboard (stat tiles + status breakdown), users (search, paging, detail modal),
  system config (grouped list, add/edit modal, delete confirm, reload). Port 5174 → :8091.
- Vite proxies `/api` and `/openapi`; `VITE_API_PROXY` overrides the backend target (used by e2e).

## Verification done this session
- `make lint` (go vet ×3, oxlint, tsc for 3 workspaces) and `make test` (go tests: auth, extraction,
  storage, analytics; vitest: money format) green. `npm run build` for user + admin green.
- API smoke — 76 checks, fake extractor, local storage, scratch DB `expenselens_smoke`:
  scratchpad `smoke.py` (user API on :8093) — envelope, 401/404/422 shapes, register/login/refresh
  rotation + 30s grace, logout, profile/preferences/password, categories CRUD, upload/duplicate/
  grouped, patch/confirm, list/search/filter/sort, analytics, images, reextract, delete, cascade.
- Admin smoke — 27 checks (`admin_smoke.py`, :8094): login, cross-portal token rejection, me, users
  list/search/paging/detail, stats, system-config upsert/mask/list/delete/reload, refresh, logout.
- Playwright (playwright-core + `/usr/bin/google-chrome`, scripts in scratchpad `e2e/`):
  `flow.mjs` (auth/settings/theme/password/reload/mobile/logout), `receipts.mjs` (upload → review →
  confirm → details → list/search → dashboard charts → mobile → delete), `admin.mjs` (login,
  dashboard, users search + detail, config add/edit/reload/delete, reload, mobile, sign out). All
  steps pass, no browser errors (the two 401 console lines in flow.mjs are the expected no-cookie
  refresh probe on first load).
- If the scratchpad is gone: recreate smoke from `docs/PLAN.md` §4 (paths are `/openapi/auth/*`,
  `/api/*`; unwrap `data`; errors carry `error`/`code`/`fields`).

## Decisions made this session
- Module paths `smartreceipt/backend/{common,user,admin}`; `replace ../common` in user/admin.
- `.env` bootstrap keys: `ENV` (alias `APP_ENV`), `PORT`, `MONGO_URI`, `MONGO_DATABASE` (alias
  `MONGO_DB`), `JWT_SECRET`, `LOG_JSON`. Everything else via `GetConfigVal` (env → system_config →
  code default); `.env.local` / `ENV_FILE` overlay for dev.
- Access tokens carry typed claims (`UserClaims`/`AdminClaims`) with a `type` claim validated in
  `Validate()`, so a user token is rejected by the admin API and vice versa (same JWT secret).
- Refresh cookie path `/openapi/auth` (user) and `/openapi/admin/auth` (admin); admin refresh TTL 12h,
  access 30m. Refresh grace 30s kept.
- Handlers: package-level `var svc = domain.New()`; services read `config.AppConfig` lazily; receipt
  service infra (store/extractor/pool) injected once via `receipt.Configure` in `app.Run`.
- Local file URLs moved to `/openapi/files/*` (signed, 2h). `MAX_UPLOAD_MB` applies per file.
- Indexes: `make ensure-indexes` per service builds + seeds; boot only warns via `MissingIndexes`.
- Frontend apps consume `common` through the `@common` alias (no build step); React deduped.
- Admin CORS default origin `http://localhost:5174`; user `http://localhost:5173` (`ALLOWED_ORIGINS`
  overrides, CSV).

## Known issues / gotchas
- Never `pkill -f` a pattern present in your own command line; use `fuser -k PORT/tcp`.
- React StrictMode double effects: keep one-shot network calls single-flighted (AuthProvider does).
- Local storage provider writes to `backend/user/.data/uploads` (gitignored) relative to the cwd
  the server runs from.
- Tailwind v4 important modifier is a suffix (`w-auto!`); `@source '../../common/src'` is required
  in each app's index.css or common classes are purged.
- HEIC uploads are accepted but browsers cannot render them; Gemini can read them.
- Gemini model default `gemini-3.1-pro-preview`; verify with the models list endpoint once a key
  exists. Consumer Google AI Pro plan ≠ API quota.
- oxlint warns `react(incompatible-library)` on `useForm` usages; harmless.
- First boot on an empty DB without `make ensure-indexes` works but logs "missing indexes" and has
  no system categories (extraction then leaves `categoryId` unset until seeded).

## Next steps (exact)
1. Phase 3 backend, in the new layout: `backend/common/services/{budget,warranty,notification}`
   (models in `common/pkg/models`, `EnsureIndexes` each, add to `user/internal/app/indexes.go`),
   `backend/user/internal/services/{budget,warranty,notification,report}` (spent via analytics,
   daily scheduler goroutine for 80/100% budget + 30/7-day warranty), `internal/http/api/{budgets,
   warranties,notifications,reports}` + register in `api.go`. Extend account cascade in
   `user/internal/services/auth.DeleteAccount`.
2. Phase 3 frontend (`frontend/user/src/features/…`): Analytics page (range tabs, TrendChart, top
   merchants bar list, donut, stats row), Budgets, Warranty, Notifications + badge, Categories, More,
   Reports/Export, dashboard Budget Overview widget. Shared widgets go to `frontend/common/src`.
3. Ask user for Gemini key + Cloudinary URL + OAuth creds to run live verification.
4. Phase 5 later: admin user disable/delete, extraction monitoring, system categories CRUD, audit log.

## Commands
- `make install` (npm workspaces + go mod download), `make ensure-indexes`,
  `make create-admin EMAIL=… PASSWORD=… NAME=… ROLE=superadmin`.
- `make dev` (user API :8090 + Vite :5173), `make dev-admin` (:8091 + :5174), `make test`,
  `make lint`, `make build`.
- Offline dev: `EXTRACTOR=fake make dev-backend`.
- Smoke DB: run APIs with `MONGO_DATABASE=expenselens_smoke` (+ `PORT`, `API_URL`), drop after with
  `mongosh --quiet expenselens_smoke --eval 'db.dropDatabase()'`. E2E: start Vite with
  `VITE_API_PROXY=http://localhost:<port>`.
