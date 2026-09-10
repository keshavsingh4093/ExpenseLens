# Handover

Rewrite this file (do not append) at the end of each session or when context nears 200–250k tokens.

## Last updated
2026-09-10 — Phase 2.5 (restructure to `user`/`common`/`admin` on both sides) is DONE and
committed on local `main` (commit `d7da324`). Phase 3 was just started: only the shared
(`backend/common`) models and collection services for budgets, warranties and notifications exist
(committed as WIP, see below). Nothing else of Phase 3 exists yet. NEXT: continue Phase 3 per the
plan in "Next steps".

## Status by phase
- Phase 0 Scaffolding: done
- Phase 1 Auth: done (OAuth code complete; untested against live Google/Microsoft/Apple)
- Phase 2 Core receipt flow: done (Gemini + Cloudinary untested live; fake/local verified)
- Phase 2.5 Restructure: done, verified, committed
- Phase 3 Insights: STARTED — common layer only (see "Phase 3 progress")
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
  (user, receipt, category, auth_token, admin_user, system_config, budget, warranty, notification),
  `pkg/storage` (local signed URLs + cloudinary), `pkg/extraction` (gemini, fake, pool), `pkg/mail`;
  `services/auth` (argon2id, SignToken, opaque TokenStore for refresh/reset/admin_refresh, OIDC
  providers, session service), `services/{user,category,receipt,systemconfig,adminuser,budget,
  warranty,notification}` (collection access, each with `Indexes()` + `EnsureIndexes`).
- `backend/user` (:8090): `cmd/server`, `cmd/scripts/ensure-indexes` (indexes + category seed),
  `internal/app` (boot sequence per spec; only *checks* indexes and warns), `internal/config`
  (AppConfig, Load/Reload/GetConfigVal/ValidateProductionSecrets), `internal/http/router`,
  `internal/http/openapi/{health,auth,files,configreload}`, `internal/http/api/{user,receipts,
  categories,analytics}`, `internal/pkg/middleware` (UserClaims w/ `type:"user"`, GetUserFromContext,
  UserID), `internal/services/{auth (facade + providers + delete cascade), receipt (upload/extract/
  confirm/images; deps via `receipt.Configure`; `OnExtracted` hook field), analytics}`.
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
  fns, `useAuth<T>()`, `RequireAuth`, `GuestOnly`), `theme/`, `ui/`, `charts/` (StatTile, TrendChart,
  CategoryDonut, donutSlices), `format/`, `hooks/`, `categories/` (palette, icons, types),
  `analytics/types.ts`, `types.ts` (ReceiptStatus). `tailwind.preset.css` holds the tokens; each
  app's `index.css` imports tailwind + preset + `@source '../../common/src'`.
- `user/`: the Phase 0–2 app, imports `@common/…`; `features/auth/useAuth.ts` is the typed wrapper.
  Placeholders still: Analytics, Budgets, Categories, Warranty, Reports, Notifications, More
  (`app/router.tsx` uses `<Placeholder/>`); Dashboard "Budget Overview" card is an empty state;
  Settings → Export page is a stub.
- `admin/`: login, dashboard (stat tiles + status breakdown), users (search, paging, detail modal),
  system config (grouped list, add/edit modal, delete confirm, reload). Port 5174 → :8091.
- Vite proxies `/api` and `/openapi`; `VITE_API_PROXY` overrides the backend target (used by e2e).

## Phase 3 progress (WIP commit on top of d7da324)
Written, built (`go build`/`go vet` in `backend/common` green), NOT yet exercised by any test:
- `common/pkg/models/{budget,warranty,notification}.go` — Budget{userId,categoryId,month YYYY-MM,
  limit paise,currency}; Warranty{productName,purchaseDate,expiryDate (YYYY-MM-DD),durationMonths,
  receiptId?,itemId?,imageKey,notes,remindersSent["30","7"]} + derived `status`/`daysLeft`;
  Notification{type,title,body,data,dedupeKey,readAt} with type consts.
- `common/services/budget` (ListByMonth, Get, Upsert via FindOneAndUpdate, Delete,
  UsersWithBudgets(month), ListAllForUser, DeleteAllForUser; unique index {userId,categoryId,month}),
  `common/services/warranty` (Insert, List, Get, Update, Delete, ExpiringBetween(from,to) across
  users, MarkReminderSent, DeleteAllForUser), `common/services/notification` (Insert → ErrDuplicate
  on dedupe key, List paged, UnreadCount, MarkRead, MarkAllRead, DeleteAllForUser; unique partial
  index on {userId,dedupeKey}).
- `common/services/receipt`: added `FindAll(uid, from, to)` (counted statuses, ≤5000, for CSV) and
  `ListAllForUser(uid)` (for JSON export).
Not written yet: everything in `backend/user` and the frontend for Phase 3; indexes not yet added to
`user/internal/app/indexes.go`; account-deletion cascade not extended.

## Phase 3 design (decided, follow this)
Backend (`backend/user`):
- `internal/services/budget`: `List(uid, month)` → rows {Budget, spent, percent, remaining} using
  `analytics.ByCategory` over Range{month-01, month-31} + summary {totalLimit,totalSpent};
  `Upsert` (validate YYYY-MM, limit>0, category visible via category.Get); `Delete`;
  `CheckAlerts(uid, month)` → notify `budget_alert` at ≥80% / ≥100% with dedupe keys
  `budget:<id>:80|100`, honouring `preferences.notifications.budgetAlerts`.
- `internal/services/warranty`: Create (expiry = purchase + durationMonths when expiry absent;
  if receiptId given verify ownership, copy first image key), List(uid, statusFilter) with derived
  status: expired if expiry < today, expiring if daysLeft ≤ 30, else active (today in the user's
  preferences.timezone); Update; Delete; decorate ImageURL via storage.
- `internal/services/notification`: `Notify(uid, type, title, body, data, dedupeKey)`, List/Read/
  ReadAll/UnreadCount. Emit `extraction_done` / `extraction_failed` from the receipt `OnExtracted`
  hook. Change receipt hooks to package-level (`receipt.SetHooks{OnExtracted, OnConfirmed}`) and call
  `budget.CheckAlerts` from `OnConfirmed`.
- `internal/services/scheduler`: goroutine started in `app.Run` (workerCtx, stopped in OnShutdown);
  runs 10s after boot then hourly, doing the daily job once per UTC day: budgets → CheckAlerts for
  `UsersWithBudgets(currentMonth)`; warranties → `ExpiringBetween(today, today+30d)`, notify
  `warranty_expiring` at ≤30 and ≤7 days (dedupe `warranty:<id>:30|7`, MarkReminderSent, honour
  `warrantyReminders`); Mondays → `weekly_summary` for users with `weeklySummary` (analytics.Sum of
  last 7 days).
- `internal/services/report`: `CSV(uid, rg)` (date,time,merchant,category,status,currency,subtotal,
  tax,total,paymentMethod,gstin,items "name x qty = amount; …",tags,notes,id; amounts as decimals)
  and `PDF(uid, rg, tz)` via `github.com/go-pdf/fpdf v0.9.0` (registry package, allowed): title +
  period, totals row, by-category table, top merchants, simple daily bar chart; use currency code,
  not "₹" (core fonts). Add fpdf to `backend/user/go.mod`.
- Handlers (`internal/http/api`, register in `api.go`): `budgets` (GET /api/budgets?month, PUT
  /api/budgets {categoryId,month,limit}, DELETE /api/budgets/{id}); `warranties` (GET
  /api/warranties?status, GET/POST, PATCH|DELETE /{id}); `notifications` (GET /api/notifications
  ?page&pageSize → {notifications,total,unread,page,pageSize}, GET /unread-count, POST /{id}/read,
  POST /read-all); `reports` (GET /api/reports/export?format=csv|pdf&range|from&to&tz → attachment);
  `user`: add GET /api/me/export (JSON of user+categories+receipts+budgets+warranties+notifications,
  attachment). Extend `authsvc.DeleteAccount` cascade with budget/warranty/notification
  DeleteAllForUser. Add budget/warranty/notification Indexes to `app/indexes.go` (AllIndexSets +
  EnsureAllIndexes) and run `make ensure-indexes`.
Frontend (`frontend/user`, widgets shared via `frontend/common`):
- Add `downloadFile(path, filename)` to `common/src/api/client.ts` (fetch w/ bearer + one refresh
  retry → blob → anchor click) for CSV/PDF/JSON downloads.
- Pages replacing placeholders: Analytics (range chips 7d/30d/3m/6m/1y/all, TrendChart, donut +
  legend, top merchants bar list, stats row: avg daily/highest day/transactions/avg per tx),
  Budgets (month prev/next, progress bars per category, add/edit modal, delete, summary tile),
  Warranty (tabs Active/Expiring soon/Expired, cards with days left + receipt link, add/edit modal
  with product/purchase date/duration or expiry/receipt+item picker from recent receipts),
  Notifications (list, unread styling, mark read on click, mark all read; unread badge on sidebar
  item and mobile More tab via `GET /api/notifications/unread-count` polled every 60s), Categories
  (system read-only list, custom add/edit/delete with icon picker from the lucide set in
  `common/src/categories/CategoryIcon.tsx` + `<input type=color>`), More (mobile links: Budgets,
  Categories, Warranty, Reports, Notifications, Settings + sign out), Reports (range + format →
  download; monthly PDF), Settings → Export (link to Reports + "Export all data" JSON), Data &
  Privacy "Export" button, Dashboard Budget Overview widget (top 3 progress bars for current month).
- Keep the validated category palette for charts (`common/src/categories/colors.ts`).

## Verification done this session (Phase 2.5)
- `make lint` (go vet ×3, oxlint, tsc for 3 workspaces) and `make test` (go tests: auth, extraction,
  storage, analytics; vitest: money format) green. `npm run build` for user + admin green.
- API smoke — 76 checks, fake extractor, local storage, scratch DB `expenselens_smoke`:
  scratchpad `smoke.py` (user API on :8093) — envelope, 401/404/422 shapes, register/login/refresh
  rotation + 30s grace, logout, profile/preferences/password, categories CRUD, upload/duplicate/
  grouped, patch/confirm, list/search/filter/sort, analytics, images, reextract, delete, cascade.
- Admin smoke — 27 checks (`admin_smoke.py`, :8094): login, cross-portal token rejection, me, users
  list/search/paging/detail, stats, system-config upsert/mask/list/delete/reload, refresh, logout.
- Playwright (playwright-core + `/usr/bin/google-chrome`, scripts in scratchpad `e2e/`):
  `flow.mjs`, `receipts.mjs`, `admin.mjs` — all steps pass, no browser errors (the two 401 console
  lines in flow.mjs are the expected no-cookie refresh probe on first load).
- Scratchpad of this session: `/tmp/claude-1001/-home-keshav-repos-smart-receipt/9b608705-c89e-464f-96fb-f87808d8b73f/scratchpad/`
  (`smoke.py`, `admin_smoke.py`, `e2e/{flow,receipts,admin,reload,nav}.mjs`, `e2e/node_modules/
  playwright-core`, `e2e/r0..r3.png`). If gone: recreate smoke from `docs/PLAN.md` §4 (paths
  `/openapi/auth/*`, `/api/*`; unwrap `data`; errors carry `error`/`code`/`fields`).

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
- Session-length discipline: rewrite this file at ~200–250k context; this session overran that.

## Next steps (exact)
1. Phase 3 backend in `backend/user` exactly per "Phase 3 design": services (budget, warranty,
   notification, scheduler, report), handlers + `api.go`, `/api/me/export`, hooks in receipt
   service, cascade, indexes. `cd backend/user && go mod tidy && go build ./... && go vet ./...`.
2. Extend `smoke.py` with budgets/warranties/notifications/reports checks; run against a fresh
   `expenselens_smoke` DB after `make ensure-indexes` (env `MONGO_DATABASE=expenselens_smoke`).
3. Phase 3 frontend per design; `npm run typecheck && npm run lint && npm run build`; Playwright
   flows at 1280px and 400px for the new screens.
4. Tick Phase 3 boxes in `docs/PLAN.md`, rewrite this file, commit
   "Phase 3: budgets, analytics, warranty, notifications, reports, categories, more".
5. Ask user for Gemini key + Cloudinary URL + OAuth creds to run live verification.

## Commands
- `make install` (npm workspaces + go mod download), `make ensure-indexes`,
  `make create-admin EMAIL=… PASSWORD=… NAME=… ROLE=superadmin`.
- `make dev` (user API :8090 + Vite :5173), `make dev-admin` (:8091 + :5174), `make test`,
  `make lint`, `make build`.
- Offline dev: `EXTRACTOR=fake make dev-backend`.
- Smoke DB: run APIs with `MONGO_DATABASE=expenselens_smoke` (+ `PORT`, `API_URL`), drop after with
  `mongosh --quiet expenselens_smoke --eval 'db.dropDatabase()'`. E2E: start Vite with
  `VITE_API_PROXY=http://localhost:<port>`.
