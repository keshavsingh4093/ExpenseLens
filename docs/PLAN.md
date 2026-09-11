# ExpenseLens — Implementation Plan

Source of truth for scope, architecture, and progress. Update checkboxes as work lands.
Design reference: the ExpenseLens mockup shared on 2026-09-10 (desktop sidebar layout + mobile
bottom-tab layout). Screens: Dashboard, Upload Receipt, Review Receipt, Receipts, Receipt Details,
Analytics, Budgets, Warranty, Settings, Categories, Reports, Notifications.

## 1. Decisions (locked 2026-09-10)

| Area | Decision |
|---|---|
| Frontend | React 19, Vite, TypeScript, Tailwind v4, React Router v7, TanStack Query v5, react-hook-form + zod, Recharts, react-dropzone, lucide-react. **`frontend/{common,user,admin}`** npm workspaces; shared UI/api/theme in `common` (alias `@common`) |
| Backend | Go, chi, MongoDB driver v2. **`backend/{common,user,admin}` per dbc-pro conventions** (`docs/BACKEND_RESTRUCTURE.md`, landed 2026-09-10); shared code in `common` (own http/middleware/database/logger/config helpers); no code from outside this repo. Paths `/openapi/…` (open) and `/api/…` (auth); envelope `{success,data,error,code,fields}` |
| Database | MongoDB (docker-compose locally, Atlas or self-hosted in prod) |
| File storage | Cloudinary via `storage.Provider` interface; `local` provider for dev/tests |
| Extraction | Gemini API via `extraction.Extractor` interface. Default model `gemini-3.1-pro-preview`; `gemini-3.7-flash` as cheaper option. Model name from `GEMINI_MODEL` env. |
| Auth | Email/password (argon2id), Google, Microsoft, Apple. Access JWT 15 min + refresh token 30 days in httpOnly cookie, rotated on use. |
| Mobile | Responsive web only. Bottom tab bar under `md` breakpoint, sidebar above. Camera via `<input capture>`. |
| Ports | User API `:8090` (8080 is taken on the dev machine), user Vite `:5173` proxies `/api` + `/openapi` to it. Admin API `:8091`, admin Vite `:5174` |
| Money | int64 paise + currency code. Display via `Intl.NumberFormat`. |
| Currency/locale | INR, Asia/Kolkata defaults; user-configurable in Settings. |

Note on Gemini billing: the consumer "Google AI Pro" subscription does not include API quota. API
calls need a key from Google AI Studio with billing enabled (or the free tier's rate limits). Verify
`GET /v1beta/models` with the key before assuming a model name works.

## 2. Architecture

```
browser ──HTTPS──> Go API (chi) ──> MongoDB
                     │
                     ├─> storage.Provider  (Cloudinary | local disk)
                     └─> extraction.Extractor (Gemini)  [async worker pool]
```

- Two Go binaries (`backend/user`, `backend/admin`) share `backend/common`. Routes are `/openapi/*`
  (unauthenticated) and `/api/*` (bearer JWT). In prod each can also serve its built frontend from `/`.
- Upload flow: client POSTs file → API stores original via `storage.Provider` → creates receipt with
  `status: "processing"` → enqueues extraction job (in-process worker pool, channel-backed; can be
  swapped for a queue later) → worker sends file to Gemini with a JSON schema → receipt updated with
  `extracted` payload + `status: "review"` → client polls `GET /receipts/{id}` (SSE later) → user
  edits and confirms → `status: "confirmed"`.
- Idempotent uploads: SHA-256 of file bytes stored on the receipt; duplicate hash for the same user
  returns the existing receipt with a `duplicate: true` flag (Phase 4 turns this into a UI prompt).

### Backend layout (Phase 2.5, see `docs/BACKEND_RESTRUCTURE.md`)
```
backend/
  common/   module smartreceipt/backend/common
    pkg/{http,middleware,database,logger,config,validate,models,storage,extraction,mail}
    services/{auth,user,category,receipt,systemconfig,adminuser,adminaudit,…}   collection access + shared logic
    services/user/cascade.go   DeleteCascade: the one account-deletion path for both portals (Phase 5)
  user/     module smartreceipt/backend/user (replace ../common)   API :8090
    cmd/server, cmd/scripts/ensure-indexes
    internal/{app,config,http/{router,openapi/*,api/*},pkg/middleware,services/{auth,receipt,analytics}}
    internal/services/{budget,warranty,notification,report,scheduler} + internal/http/api/{budgets,
    warranties,notifications,reports} + internal/http/openapi/scheduler (Phase 3)
  admin/    module smartreceipt/backend/admin (replace ../common)  API :8091
    cmd/server, cmd/scripts/{ensure-indexes,create-admin}
    internal/{app,config,http/{router,openapi/*,api/*},pkg/middleware,services/{adminauth,stats}}
    internal/services/{users,extraction,categories,audit} + internal/http/api/{users,extraction,
    categories,audit} (Phase 5); services/admins + http/api/admins (Phase 5.1 admin accounts)
```
Indexes are never built on the boot path: `make ensure-indexes` (per service) builds them and seeds
the system categories; `app.Run()` only warns about missing ones.

### Frontend layout (npm workspaces: `common`, `user`, `admin`)
```
frontend/
  common/src/     @expenselens/common, consumed via the `@common` alias (no build step)
    api/          fetch wrapper (envelope unwrap, single-flight refresh), queryClient
    auth/         AuthProvider<TUser>, useAuth, RequireAuth, GuestOnly
    theme/ ui/ charts/ format/ hooks/ categories/ analytics/types
  common/tailwind.preset.css   shared @theme tokens, dark variant, card/text-muted utilities
  user/src/       app/ (shell, router), features/{auth,dashboard,upload,receipts,categories,settings,…}
  admin/src/      app/ (AdminShell, router), features/{auth,dashboard,users,admins,config,extraction,categories,audit}
```

## 3. Data model (MongoDB)

All docs carry `createdAt`, `updatedAt`. All user-owned docs carry `userId` (ObjectID) and are
indexed `{userId: 1, ...}`.

**users**
`email` (unique, lowercased), `emailVerified`, `passwordHash?`, `name`, `avatarUrl?`,
`providers: [{provider, providerUserId, linkedAt}]`,
`preferences: {currency, language, theme, timezone, notifications: {budgetAlerts, warrantyReminders, weeklySummary}}`,
`disabled?` (admin-set; login, refresh and every `/api` call reject the user via the bearer-token guard)

**refresh_tokens**
`userId`, `tokenHash`, `expiresAt`, `revokedAt?`, `replacedBy?`, `userAgent`, `ip`. TTL index on `expiresAt`.

**receipts**
`userId`, `status: processing|review|confirmed|failed`, `merchant`, `merchantNormalized`,
`date` (YYYY-MM-DD), `dateTime?`, `categoryId`, `currency`, `subtotal?`, `tax?`, `total`,
`paymentMethod?: cash|upi|card|netbanking|wallet|other`,
`items: [{id, name, quantity?, unitPrice?, amount}]`,
`images: [{storageKey, url, contentType, size, hash, order}]`,
`notes?`, `tags: [string]`,
`extracted?: {raw: object, confidence: 0..1, model, extractedAt, error?}`,
`fileHash` (for duplicate detection), `gstin?`, `taxBreakdown?: [{label, rate?, amount}]`.
Indexes: `{userId, date desc}`, `{userId, categoryId}`, `{userId, fileHash}`, text index on
`merchant, items.name, notes, tags`.

**categories**
`userId?` (null = system default), `name`, `slug`, `icon`, `color`, `isDefault`. Seed: Food, Fuel,
Shopping, Medical, Entertainment, Travel, Utilities, Groceries, Others.

**budgets**
`userId`, `categoryId`, `month` (YYYY-MM), `limit` (paise), `currency`. Unique `{userId, categoryId, month}`.

**warranties**
`userId`, `productName`, `purchaseDate`, `expiryDate`, `durationMonths`, `receiptId?`, `itemId?`,
`imageUrl?`, `notes?`, `remindAt: [dates]`. Status derived from expiryDate.

**notifications**
`userId`, `type: budget_alert|warranty_expiring|extraction_done|extraction_failed|weekly_summary`,
`title`, `body`, `data`, `readAt?`.

**admin_users** (admin portal accounts, separate from `users`)
`email` (unique, lowercased), `name`, `passwordHash` (argon2id), `role: superadmin|admin`,
`disabled?` (Phase 5.1: set from the Admins page; login, refresh and every `/api/admin` call reject
the admin through the bearer guard, and disabling revokes their refresh tokens), `lastLoginAt?`.
Refresh tokens live in `auth_tokens` with `kind: admin_refresh`. The first superadmin is created by
`cmd/scripts/create-admin`; further admins are created from the portal.

**admin_audit_log** (Phase 5)
`adminId`, `adminEmail`, `action` (e.g. `user.disable`, `category.delete`, `system_config.upsert`),
`targetType: user|receipt|category|system_config|admin`, `targetId`, `details` (never secrets: no
reset links, no config values, no passwords; reset-link entries carry `{emailed: bool}`), `ip`, `at`. Indexes `{at:-1}`, `{targetId,at:-1}`, `{adminId,at:-1}`,
`{action,at:-1}`. Written after every successful mutating admin route.

## 4. API surface

Unauthenticated routes live under `/openapi`, authenticated ones under `/api`. Responses use
`{success, data, error, code, fields}`; the lists below omit the prefix.

Auth (`/openapi/auth`): `POST /register`, `POST /login`, `POST /logout`, `POST /refresh`,
`GET /providers`, `GET /{google|microsoft|apple}/start`, `GET|POST /{provider}/callback`,
`POST /forgot-password`, `POST /reset-password`. Refresh cookie path is `/openapi/auth`.

Users (`/api/me`): `GET/PATCH /me`, `PATCH /me/preferences`, `POST /me/password`, `DELETE /me`.

Files: `GET /openapi/files/*` (signed URLs, local storage provider only).
Ops: `POST /openapi/config/reload` (header `X-Config-Reload-Secret`, only when `CONFIG_RELOAD_SECRET` set).

Admin API (`:8091`): `POST /openapi/admin/login`, `POST /openapi/admin/auth/{refresh,logout}`,
`GET /api/admin/me`, `GET /api/admin/users?q&page&pageSize`, `GET /api/admin/users/{id}`,
`GET /api/admin/stats`, `GET /api/admin/system-config`, `PUT|DELETE /api/admin/system-config/{key}`,
`POST /api/admin/system-config/reload`.
Phase 5: `GET /api/admin/users?status=active|disabled`, `POST /api/admin/users/{id}/disable|enable`,
`POST /api/admin/users/{id}/reset-password-link` → `{link, expiresAt}` (1h, single use; the admin
copies it out of band), or with body `{"send": true}` → `{sent: true, expiresAt}` and the link is
emailed through `common/pkg/mail` with the forgot-password wording instead of being returned (Phase
5.1; `MAIL_PROVIDER=log` prints it to the admin server log), `DELETE /api/admin/users/{id}`
(superadmin only; shared cascade incl. stored files),
`GET /api/admin/extraction/stats?range=7d|30d|90d` → `{totals, failurePct, byDay, byModel, extractor,
model}`, `GET /api/admin/extraction/failed?page&pageSize`, `POST /api/admin/extraction/{receiptId}/retry`
(forwards to the user API ops route below; 503 `retry_unavailable` when the secret/URL is unset),
`GET|POST /api/admin/categories`, `PATCH|DELETE /api/admin/categories/{id}` (system categories;
delete moves every user's receipts to `others` and returns `receiptsMoved`; `others` cannot be
deleted or re-slugged → 409 `others_protected`; duplicate slug → 409 `category_exists`),
`GET /api/admin/audit?action&adminId&targetId&page&pageSize`, `GET /api/admin/audit/actions`.
Phase 5.1 admin accounts: `GET /api/admin/admins?q&page&pageSize` (any admin), and superadmin-only
`POST /api/admin/admins {email,name,password,role}` (201; 409 `admin_exists`), `PATCH
/api/admin/admins/{id} {name?, role?}`, `POST /api/admin/admins/{id}/disable|enable` (disable revokes
the admin's refresh tokens; the bearer guard rejects their access token on the next request with 403
`account_disabled`), `POST /api/admin/admins/{id}/reset-password {password}` (revokes their refresh
tokens). Rules: you cannot disable or demote yourself (409 `self_target`) or the last active
superadmin (409 `last_superadmin`); plain admins get 403 `forbidden` on every mutating route. Every
mutation is audited (`admin.create|update|disable|enable|reset_password`, targetType `admin`).
`cmd/scripts/create-admin` (`make create-admin`) is only needed to bootstrap the first superadmin.
The admin API needs the same `STORAGE_PROVIDER`/`LOCAL_STORAGE_DIR`/`CLOUDINARY_URL` as the user API
(to delete files on account deletion), `APP_URL` (reset links), `MAIL_*`/`SMTP_*` (emailed reset
links; same keys as the user API), and `USER_API_URL` + `CONFIG_RELOAD_SECRET` (retry, reload).

Receipts: `POST /receipts/upload` (multipart, 1..n files or pdf) → `{receiptId, status}`;
`GET /receipts?q&category&from&to&sort&page`; `GET /receipts/{id}`; `PATCH /receipts/{id}`;
`DELETE /receipts/{id}`; `POST /receipts/{id}/confirm`; `POST /receipts/{id}/reextract`;
`POST /receipts/{id}/images`; `DELETE /receipts/{id}/images/{imageId}`; `GET /receipts/recent-uploads`.

Categories: `GET /categories`, `POST /categories`, `PATCH /categories/{id}`, `DELETE /categories/{id}`.

Budgets: `GET /budgets?month&tz` → `{month, budgets:[{…budget, spent, remaining, percent, count}], summary}`,
`PUT /budgets {categoryId, month, limit}` (upsert), `DELETE /budgets/{id}`.

Warranties: `GET /warranties?status` → `{warranties, counts}`, `POST`, `GET|PATCH|DELETE /{id}`.
Expiry = purchaseDate + durationMonths when `expiryDate` is absent; status/daysLeft derived per request.

Analytics: `GET /analytics/summary` (KPI tiles with deltas), `GET /analytics/trend?range&granularity`,
`GET /analytics/categories?range`, `GET /analytics/merchants?range&limit`,
`GET /analytics/stats?range` (avg daily, highest day, tx count).

Notifications: `GET /notifications?page&pageSize` → `{notifications,total,unread,page,pageSize}`,
`GET /notifications/unread-count`, `POST /notifications/{id}/read`, `POST /notifications/read-all`.

Reports: `GET /reports/export?format=csv|pdf&range|from&to&tz` (attachment). All-data JSON:
`GET /api/me/export`. Ops: `POST /openapi/scheduler/run` (header `X-Config-Reload-Secret`) forces the
daily scheduler job. `POST /openapi/receipts/{id}/reextract` (same header; 404 when the secret is
unset, 401 on mismatch) re-runs extraction for any user's receipt; the admin retry calls it.

## 5. Extraction contract (Gemini)

Request: file bytes inline (`inline_data`, ≤20MB) or Files API for larger; `responseMimeType:
application/json` with `responseSchema`:

```json
{
  "merchant": "string", "date": "YYYY-MM-DD", "time": "HH:MM|null",
  "currency": "ISO-4217", "subtotal": number|null, "tax": number|null, "total": number,
  "paymentMethod": "cash|upi|card|netbanking|wallet|other|null",
  "gstin": "string|null",
  "taxBreakdown": [{"label": "CGST 9%", "amount": number}],
  "items": [{"name": "string", "quantity": number|null, "unitPrice": number|null, "amount": number}],
  "suggestedCategory": "one of the category slugs passed in the prompt",
  "warrantyCandidates": [{"itemName": "string", "reason": "string"}],
  "confidence": 0..1
}
```
Prompt includes the user's category list, hints for Indian receipts (₹, GST, UPI), and instructs to
return null rather than guess. Amounts are converted to paise in Go. Retries: 2 with backoff;
on final failure receipt → `failed` with error stored, notification emitted.

## 6. Phases and tasks

### Phase 0 — Scaffolding
- [x] Repo: `git init`, `.gitignore`, `.editorconfig`, root `Makefile` (`dev`, `test`, `lint`, `build`)
- [x] `docker-compose.yml` with MongoDB 7 (+ mongo-express optional)
- [x] Backend: go module, config loader, chi server, health endpoint, structured logging (slog),
      mongo connect + index bootstrap, error envelope, request-id + CORS middleware
- [x] Backend `.env.example` (PORT, MONGO_URI, JWT_SECRET, STORAGE_PROVIDER, CLOUDINARY_URL,
      LOCAL_STORAGE_DIR, GEMINI_API_KEY, GEMINI_MODEL, OAUTH_* per provider, APP_URL, API_URL)
- [x] Frontend: Vite + React + TS + Tailwind v4, router, query client, layout shells (sidebar /
      bottom tabs), base UI components, API client with refresh handling
- [x] Design tokens: primary green `#16a34a`-ish from mockup, category colors, light/dark theme via
      `class` strategy

### Phase 1 — Auth and profile
- [x] Register/login/logout/refresh with argon2id + JWT + rotating refresh cookie
- [x] Google OAuth (authorization code), Microsoft (common tenant), Apple (form_post, client secret
      JWT generated from .p8 key). Account linking by verified email. (Code complete via go-oidc;
      untested against live providers until credentials are supplied.)
- [x] Forgot/reset password (email sender interface; log-to-console impl in dev)
- [x] `GET /me`, `PATCH /me`, preferences (currency, language, theme, timezone, notification toggles)
- [x] Frontend: Login, Register, OAuth buttons + callback route, protected routes, Settings →
      Profile / Security / Preferences screens (+ Notifications, Data & Privacy, static pages)

### Phase 2 — Core receipt flow
- [x] `storage.Provider` interface + local impl + Cloudinary impl + tests with local impl
      (Cloudinary impl compiles but is untested against a real account)
- [x] `extraction.Extractor` interface + Gemini impl + worker pool + fake extractor for tests
      (Gemini impl untested against the live API until a key is provided)
- [x] Upload endpoint (image/PDF, multi-file, size/type validation, hash, duplicate check)
- [x] Receipt CRUD, confirm, re-extract, image add/remove, list with search/filter/sort/pagination
- [x] Categories seed + CRUD (no Categories screen yet; see Phase 3)
- [x] Frontend: Upload screen (dropzone, camera, PDF, recent uploads), Review screen (image +
      editable fields + items editor), Receipts list (search, chips, sort), Receipt Details
      (carousel, items, notes, tags, edit/delete)
- [x] Dashboard: KPI tiles, trend chart, category donut, recent receipts, budget overview
      (budget widget reads Phase 3 data; render empty state until then)
- [x] `analytics/summary` + `trend` + `categories` + `merchants` + `stats` endpoints

### Phase 2.5 — Restructure to user/common/admin layout (decided 2026-09-10)
Spec: `docs/BACKEND_RESTRUCTURE.md`. Do this BEFORE Phase 3 so new domains land in the final shape.
Rules: `user`/`common`/`admin` on both sides; shared things in `common`; no code from outside this
repository (no local `replace` to sibling repos, no copied files; registry packages are fine).
- [x] `backend/common` module: pkg/{http,middleware,database,logger,config,validate,models,storage,extraction,mail}, services/{auth,user,category,receipt,systemconfig,adminuser}
- [x] `backend/user` module: cmd/server, cmd/scripts/ensure-indexes, internal/{app,config,http/{openapi,api,router},pkg/middleware,services/{auth,receipt,analytics}}
- [x] `backend/admin` module skeleton: health, admin login (admin_users, refresh cookie), /api/admin/{me,users,stats,system-config}; port 8091; `create-admin` script
- [x] `frontend/common` workspace: ui, charts, api client, auth provider, theme, format, categories palette
- [x] `frontend/user` (current app moved; imports `@common`) and `frontend/admin` (login, dashboard, users, system config)
- [x] Root Makefile + npm workspaces; `.env.example` per service
- [x] Unit tests ported; API smoke (76 checks) + Playwright flows green (user); admin smoke (27 checks) + admin Playwright; commit

### Phase 3 — Insights and tracking (landed 2026-09-10)
- [x] Budgets: API + Budgets screen (month nav, progress bars, add/edit)
- [x] Analytics screen: range tabs, trend, top merchants, category breakdown, stats row
- [x] Warranty: API + screen (Active / Expiring soon / Expired tabs, add form, link to receipt item)
- [x] Notifications: API + screen + unread badge; budget 80%/100% alerts; warranty 30/7-day reminders
      (scheduler goroutine, hourly tick, daily job once per UTC day; `POST /openapi/scheduler/run`
      forces it with the `CONFIG_RELOAD_SECRET` header)
- [x] Reports: CSV export; monthly PDF summary (`github.com/go-pdf/fpdf`)
- [x] Settings: Notifications, Data & Privacy (export all data, delete account), Help, About
      (Export tab links to Reports + "Export all data" JSON via `GET /api/me/export`)
- [x] Categories screen (list system + custom, add/edit/delete custom with icon + color)
- [x] Mobile "More" screen (links to Budgets, Categories, Warranty, Reports, Notifications, Settings)

Chart colors: the 8 system categories use the validated dataviz palette (see
`frontend/common/src/categories/colors.ts`, light + dark steps); "others" and custom categories
are neutral/stored. Keep this when adding charts in Phase 3.

Phase 3 lands in the new layout: collection access in `backend/common/services/<domain>`, business
logic + handlers in `backend/user/internal/{services,http/api}/<domain>`, screens in
`frontend/user/src/features/<domain>`, shared widgets in `frontend/common/src`.

### Phase 5 — Admin portal (landed 2026-09-10)
- [x] User management: search, view, disable/enable, delete with cascade, reset password link
      (status filter; delete is superadmin-only; disabled users are rejected at login, refresh and
      on every `/api` call through `commonmw.SetAuthGuard` → `auth.EnsureActive`)
- [x] System config editor (dynamic `system_config` keys, sensitive masking, reload endpoint)
      (built in Phase 2.5; mutations are now audited, values never logged)
- [x] Extraction monitoring: volume, failure rate, model, average confidence; retry failed
      (stacked daily succeeded/failed columns with table view; retry via the user API ops route)
- [x] Default categories management (system category CRUD)
- [x] Audit log of admin actions (filters: action, admin, target)

#### Phase 5.1 — Admin hardening (landed 2026-09-11)
- [x] gofmt sweep across `backend/{common,user,admin}` (separate `chore: gofmt` commit)
- [x] Emailed password-reset links: `{"send": true}` on the reset-link route sends the link through
      `common/pkg/mail` (admin config `MAIL_PROVIDER`, `MAIL_FROM`, `SMTP_*`), response carries no
      link, audit `{emailed: true|false}`; "Email link" button in the Users detail modal
- [x] Admin account management: `adminuser` List/SetDisabled/CountSuperadmins/IsDisabled, admin
      `admins` service + `/api/admin/admins` routes (superadmin-only mutations, self and
      last-superadmin protection, refresh revocation, audit), bearer guard `adminauth.EnsureActive`
      on the admin router (403 `account_disabled` on the next request), Admins page (table, add-admin
      modal with generated/typed password, role select, disable/enable, set password; read-only with
      a hint for plain admins); `create-admin` stays the bootstrap path for the first superadmin

### Phase 4 — Suggested features (post-MVP)
- [ ] Duplicate detection UI prompt on upload
- [ ] Warranty auto-suggest from `warrantyCandidates` on confirm
- [ ] Recurring expense detection (same merchant, similar amount, ~monthly)
- [ ] Multi-currency with conversion to home currency (rates provider interface)
- [ ] GST breakdown display + GST report export
- [ ] Email-forwarding inbox per user (inbound email provider interface)
- [ ] Shared/household receipts
- [ ] SSE for extraction status instead of polling
- [ ] PWA manifest + offline queue for uploads

## 7. Verification checklist per phase
- `make test` green; `go vet` and `eslint` clean.
- Manual: run `make dev`, exercise the phase's screens at desktop and 400px width.
- Extraction: at least 3 real Indian receipts (supermarket, restaurant, fuel) + 1 Amazon PDF invoice
  extract with correct total and ≥80% items.
