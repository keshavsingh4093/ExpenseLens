# Handover

Rewrite this file (do not append) at the end of each session or when context nears 200–250k tokens.

## Last updated
2026-09-10 — Phase 3 (budgets, analytics screen, warranty, notifications + scheduler, reports,
categories screen, mobile More, settings export) is DONE, verified (API smoke 121 checks, Playwright
at 1280px + 400px, `make test`/`make lint`/`make build` green) and committed on local `main`.
NEXT: Phase 5 (admin portal) per `docs/PLAN.md`, or live verification once Gemini/Cloudinary/OAuth
credentials are supplied.

## Status by phase
- Phase 0 Scaffolding: done
- Phase 1 Auth: done (OAuth code complete; untested against live Google/Microsoft/Apple)
- Phase 2 Core receipt flow: done (Gemini + Cloudinary untested live; fake/local verified)
- Phase 2.5 Restructure: done
- Phase 3 Insights and tracking: DONE (this session)
- Phase 5 Admin portal: scaffold only (login, dashboard stats, users list/detail, system config)
- Phase 4 Post-MVP: not started

## Layout (what exists now)
Backend — three Go modules, no code from outside the repo, mongo-driver v2:
- `backend/common`: `pkg/{http,middleware,database,logger,config,validate,models,storage,
  extraction,mail}`; `services/{auth,user,category,receipt,systemconfig,adminuser,budget,warranty,
  notification}` (collection access, each with `Indexes()`/`EnsureIndexes`). Added this session:
  `user.IDsWithWeeklySummary` (scheduler).
- `backend/user` (:8090): `internal/services/{auth,receipt,analytics,budget,warranty,notification,
  report,scheduler}`, `internal/http/api/{user,receipts,categories,analytics,budgets,warranties,
  notifications,reports}`, `internal/http/openapi/{health,auth,files,configreload,scheduler}`,
  `internal/app/{server.go,indexes.go,hooks.go}`. Dependency `github.com/go-pdf/fpdf v0.9.0` (PDF).
- `backend/admin` (:8091): unchanged this session.

Frontend — npm workspaces `frontend/{common,user,admin}`:
- `common/src` additions: `api/client.ts#downloadFile` (bearer + one refresh retry → blob → anchor),
  `ui/ProgressBar.tsx`, `format/percent.ts#percentTone`, `categories/icons.ts` (30 lucide icons for
  the picker; `CategoryIcon.tsx` imports it).
- `user/src/features`: `analytics/AnalyticsPage.tsx`, `budgets/{api,BudgetsPage,BudgetOverview}`,
  `warranty/{api,WarrantyPage}`, `notifications/{api,useUnreadCount,NotificationsPage}`,
  `categories/CategoriesPage.tsx`, `reports/{api,ReportsPage}`, `more/MorePage.tsx`; settings Export
  tab + Data & Privacy now call `reportsApi.exportAll`; Dashboard "Budget Overview" renders
  `BudgetOverview`; `AppShell` shows the unread badge (sidebar Notifications item + mobile More tab,
  polled every 60s). No `Placeholder` routes remain.

## Phase 3 as built (matches the design in the previous handover)
- Budgets: `List(uid, month)` joins `analytics.ByCategory` over the month; `Upsert` validates
  YYYY-MM / limit>0 / category visible; currency from user prefs. `CheckAlerts` emits `budget_alert`
  at ≥80% / ≥100% with dedupe keys `budget:<id>:80|100`, honouring `preferences.notifications.
  budgetAlerts`; called from the receipt `OnConfirmed` hook and by the scheduler for
  `UsersWithBudgets(currentMonth)`.
- Warranties: expiry = purchase + durationMonths when no explicit expiry; receipt link verifies
  ownership and copies the first image key (URL via storage provider, `warranty.Configure(store)`);
  status/daysLeft derived per request in the user's timezone; `List` returns `counts` per status.
  Updating dates clears `remindersSent`. `SendReminders(todayUTC)` → `warranty_expiring` at ≤30d
  (tag "30") and ≤7d (tag "7"), dedupe `warranty:<id>:<tag>`, honouring `warrantyReminders`.
- Notifications: `notification.Notify(...)` is best-effort (logs, never fails callers); duplicate
  dedupe key → silently skipped. Extraction hook emits `extraction_done` / `extraction_failed`
  (no dedupe, so re-extract notifies again).
- Receipt hooks are package-level: `receipt.SetHooks(receipt.Hooks{OnExtracted, OnConfirmed})`,
  wired in `app/hooks.go`. The old `Service.OnExtracted` field is gone.
- Scheduler (`services/scheduler`): `Start()` in `app.Run`, `Stop` in `OnShutdown`; first run 10s
  after boot, then hourly; `RunDaily` runs once per UTC day per process unless forced. Mondays also
  send `weekly_summary` (dedupe `weekly:YYYY-Www`) to users with `weeklySummary` on.
  `POST /openapi/scheduler/run` (header `X-Config-Reload-Secret` = `CONFIG_RELOAD_SECRET`, 404 when
  unset, 401 on mismatch) forces it and returns `{budgetAlerts, warrantyReminders[, weeklySummaries]}`.
- Reports: CSV (UTF-8 BOM, columns date,time,merchant,category,status,currency,subtotal,tax,total,
  paymentMethod,gstin,items,tags,notes,id; amounts as decimals) and PDF (fpdf core fonts, currency
  code not ₹: KPI tiles, daily bar chart, by-category and top-merchant tables). `GET /api/me/export`
  streams the all-data JSON (user, categories, receipts with image URLs, budgets, warranties, last
  100 notifications). Account deletion cascade now also clears budgets/warranties/notifications.
- Indexes for budgets/warranties/notifications are in `app/indexes.go`; `make ensure-indexes` builds
  them (already run against the dev DB `expenselens`? NO — only the smoke DB; run it once for dev).

## Phase 5 design (decided 2026-09-10, follow this)
Admin portal already has: login/refresh/logout, `/api/admin/{me,stats,users,users/{id},system-config…}`,
frontend login/dashboard/users/config. Build the rest in `backend/admin` + `frontend/admin`:
- Users: `POST /api/admin/users/{id}/disable|enable` (sets `users.disabled`; the user API must reject
  disabled users at login/refresh and in `EnsureAuth`-level lookups — add the check in the common
  auth service, not the handlers), `DELETE /api/admin/users/{id}` (cascade = same steps as
  `backend/user/internal/services/auth.DeleteAccount`; move that cascade into a common helper
  `common/services/user/cascade.go` that takes a `storage.Provider`, so both portals share it),
  `POST /api/admin/users/{id}/reset-password-link` (issues the existing reset token via the common
  auth service and returns the URL; never emails from admin). Superadmin only for delete.
- Extraction monitoring: `GET /api/admin/extraction/stats?range` (volume per day, failure rate,
  model breakdown, average confidence; aggregate on `receipts.extracted`) and
  `GET /api/admin/extraction/failed?page` + `POST /api/admin/extraction/{receiptId}/retry`
  (admin API cannot reach the user API's worker pool; retry = set `status: processing` and call a
  shared re-extract path in `common` **or** enqueue via the user API's `/openapi/config/reload`-style
  secret route `POST /openapi/receipts/{id}/reextract` guarded by `CONFIG_RELOAD_SECRET`. Prefer the
  secret route; document it in PLAN §4).
- Default categories: `GET|POST /api/admin/categories`, `PATCH|DELETE /api/admin/categories/{id}`
  on system categories (`userId` null). Deleting a system category reassigns receipts of ALL users to
  `others`; `others` itself cannot be deleted. Slug must stay unique among system categories.
- Audit log: collection `admin_audit_log` {adminId, action, targetType, targetId, details, ip, at};
  written by every mutating admin route via a helper in `admin/internal/services/audit`;
  `GET /api/admin/audit?page&action&adminId`. Index `{at: -1}`, `{targetId, at: -1}`.
- Frontend (`frontend/admin`): Users page gains disable/enable/delete/reset-link actions in the
  detail modal; new pages Extraction (stat tiles + failure list + retry), Categories (system CRUD
  with the shared icon picker from `@common/categories/icons`), Audit log (table with filters);
  add them to `AdminShell` nav.
- Verify with `admin_smoke.py` (extend; :8094, scratch DB) + `e2e/admin.mjs` at 1280px and 400px.
  Live-credential checks (Gemini, Cloudinary, OAuth) come LAST, only after the user provides keys.

## Verification done this session
- `make test` (go tests + vitest), `make lint` (go vet ×3, oxlint clean except the pre-existing
  `incompatible-library` warning, tsc ×3), `make build` (user + admin bundles) — all green.
- API smoke — 121 checks (was 76), fake extractor, local storage, scratch DB `expenselens_smoke`,
  server on :8093 with `CONFIG_RELOAD_SECRET=smoke-secret`: everything from Phase 2.5 plus
  notifications (extraction_done, read/read-all/unread-count/paging, 404), budgets (validation,
  upsert-in-place, progress math, 80% then 100% alerts via confirm hook, dedupe, pref off → no
  alert, delete), warranties (validation, expiry derivation, receipt link + image URL, derived
  status/counts/filters, patch recomputes expiry, unlink, delete), scheduler run (reminders,
  idempotent rerun, secret), CSV/PDF/JSON exports, and the cascade (all collections empty after
  `DELETE /api/me`).
- Playwright (`e2e/phase3.mjs`, Chrome at /usr/bin/google-chrome): register → upload+confirm 2
  receipts (category Groceries) → notifications list/badge/click-through/mark-all-read → budgets
  add/edit/over-budget/month nav → dashboard widget → analytics range tabs, donut, merchants →
  categories add/rename/delete → warranty add from receipt item (auto-fills product name), expiring
  tab switch, tab filters → CSV + monthly PDF + JSON downloads → 400px: More, Budgets, Analytics,
  Warranty, Categories, Notifications, Reports (no horizontal overflow), More → Warranty link,
  budget modal. No browser errors. `receipts.mjs` and `flow.mjs` (Phase 2) still pass.
- Scratchpad: `/tmp/claude-1001/-home-keshav-repos-smart-receipt/ffdf630a-2941-4d4b-98cb-17d3b174519e/scratchpad/`
  (`smoke.py`, `admin_smoke.py`, `e2e/{phase3,flow,receipts,admin,nav,reload}.mjs`,
  `e2e/node_modules/playwright-core`, `e2e/r0..r3.png`, `shot-p3-*.png`). If gone: recreate the smoke
  from `docs/PLAN.md` §4.

## Decisions made this session
- Scheduler runs in-process (one goroutine, hourly tick, once-per-UTC-day daily job); every
  scheduled notification carries a dedupe key so extra runs (multi-instance, forced) are harmless.
- Ops endpoint `POST /openapi/scheduler/run` reuses `CONFIG_RELOAD_SECRET` rather than adding a key.
- Warranty API: an empty `expiryDate` string is treated as absent (client omits it in duration mode).
- Notification amounts and PDF amounts render as `INR 1,23,456.50` (ASCII, Indian grouping) via
  `budget.Money`; the UI keeps `Intl.NumberFormat`.
- Budget "spent" counts `review` + `confirmed` receipts (same as analytics), so a budget can move
  before the user confirms; alerts only fire on confirm and from the daily job.
- `validator` quirk: `omitempty` on a `*string` still validates `""` (pointer non-nil). Handlers that
  accept optional dates must nil-out empty strings before `validate.Struct` (done for warranties).

## Known issues / gotchas
- Never `pkill -f` a pattern present in your own command line; use `fuser -k PORT/tcp`.
- React StrictMode double effects: keep one-shot network calls single-flighted (AuthProvider does).
- Local storage provider writes to `LOCAL_STORAGE_DIR` relative to the server cwd (default
  `backend/user/.data/uploads`, gitignored).
- Tailwind v4 important modifier is a suffix (`w-auto!`); `@source '../../common/src'` is required
  in each app's index.css or common classes are purged.
- Playwright: `<option>` elements are never "visible"; wait with `{ state: 'attached' }`. Sidebar
  links exist (hidden) at 400px, so scope mobile selectors with `main`.
- The fake extractor picks the suggested category by file seed and dates receipts today; e2e sets
  the category explicitly on the review page.
- HEIC uploads are accepted but browsers cannot render them; Gemini can read them.
- Gemini model default `gemini-3.1-pro-preview`; verify with the models list endpoint once a key
  exists. Consumer Google AI Pro plan ≠ API quota.
- oxlint warns `react(incompatible-library)` on `useForm` usages; harmless.
- Weekly summary and warranty reminder "days left" use the UTC date; a user far from UTC may see a
  one-day difference between the notification body and the Warranty screen near midnight.
- `make ensure-indexes` has NOT been run against the dev DB `expenselens` since Phase 3 added
  budgets/warranties/notifications indexes; boot will warn until it is.

## Next steps (exact)
1. `make ensure-indexes` on the dev DB; `make dev` and click through Phase 3 screens with the real
   Gemini key when available (Phase 4 "warranty auto-suggest from warrantyCandidates" is a natural
   follow-up on the review page).
2. Phase 5 admin portal per `docs/PLAN.md` §6: user disable/enable + delete with cascade (reuse
   `authsvc.DeleteAccount` logic via common services), extraction monitoring (aggregate on
   `receipts.extracted`), default categories CRUD, audit log collection. Extend `admin_smoke.py`
   and `e2e/admin.mjs`.
3. Ask user for Gemini key + Cloudinary URL + OAuth creds to run live verification (PLAN §7).

## Commands
- `make install`, `make ensure-indexes`, `make create-admin EMAIL=… PASSWORD=… NAME=… ROLE=superadmin`.
- `make dev` (user API :8090 + Vite :5173), `make dev-admin` (:8091 + :5174), `make test`,
  `make lint`, `make build`. Offline dev: `EXTRACTOR=fake make dev-backend`.
- Smoke: `mongosh --quiet expenselens_smoke --eval 'db.dropDatabase()'`; in `backend/user`:
  `MONGO_DATABASE=expenselens_smoke go run ./cmd/scripts/ensure-indexes`, then run the server with
  `MONGO_DATABASE=expenselens_smoke PORT=8093 API_URL=http://localhost:8093 EXTRACTOR=fake
  STORAGE_PROVIDER=local LOCAL_STORAGE_DIR=<scratch>/uploads CONFIG_RELOAD_SECRET=smoke-secret`,
  then `python3 smoke.py`. E2E: `VITE_API_PROXY=http://localhost:8093 npm run dev:user` in
  `frontend/`, then `S=<scratch> node e2e/phase3.mjs` from `<scratch>/e2e`.
