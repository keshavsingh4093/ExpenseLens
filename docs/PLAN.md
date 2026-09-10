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
    services/{auth,user,category,receipt,systemconfig,adminuser}   collection access + shared logic
  user/     module smartreceipt/backend/user (replace ../common)   API :8090
    cmd/server, cmd/scripts/ensure-indexes
    internal/{app,config,http/{router,openapi/*,api/*},pkg/middleware,services/{auth,receipt,analytics}}
    (Phase 3 adds internal/services/{budget,warranty,notification,report} + internal/http/api/*)
  admin/    module smartreceipt/backend/admin (replace ../common)  API :8091
    cmd/server, cmd/scripts/{ensure-indexes,create-admin}
    internal/{app,config,http/{router,openapi/*,api/*},pkg/middleware,services/{adminauth,stats}}
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
  admin/src/      app/ (AdminShell, router), features/{auth,dashboard,users,config}
```

## 3. Data model (MongoDB)

All docs carry `createdAt`, `updatedAt`. All user-owned docs carry `userId` (ObjectID) and are
indexed `{userId: 1, ...}`.

**users**
`email` (unique, lowercased), `emailVerified`, `passwordHash?`, `name`, `avatarUrl?`,
`providers: [{provider, providerUserId, linkedAt}]`,
`preferences: {currency, language, theme, timezone, notifications: {budgetAlerts, warrantyReminders, weeklySummary}}`

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

Receipts: `POST /receipts/upload` (multipart, 1..n files or pdf) → `{receiptId, status}`;
`GET /receipts?q&category&from&to&sort&page`; `GET /receipts/{id}`; `PATCH /receipts/{id}`;
`DELETE /receipts/{id}`; `POST /receipts/{id}/confirm`; `POST /receipts/{id}/reextract`;
`POST /receipts/{id}/images`; `DELETE /receipts/{id}/images/{imageId}`; `GET /receipts/recent-uploads`.

Categories: `GET /categories`, `POST /categories`, `PATCH /categories/{id}`, `DELETE /categories/{id}`.

Budgets: `GET /budgets?month`, `PUT /budgets` (upsert), `DELETE /budgets/{id}`.

Warranties: `GET /warranties?status`, `POST`, `PATCH /{id}`, `DELETE /{id}`.

Analytics: `GET /analytics/summary` (KPI tiles with deltas), `GET /analytics/trend?range&granularity`,
`GET /analytics/categories?range`, `GET /analytics/merchants?range&limit`,
`GET /analytics/stats?range` (avg daily, highest day, tx count).

Notifications: `GET /notifications`, `POST /notifications/{id}/read`, `POST /notifications/read-all`.

Reports: `GET /reports/export?format=csv|pdf&from&to`.

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

### Phase 3 — Insights and tracking
- [ ] Budgets: API + Budgets screen (month nav, progress bars, add/edit)
- [ ] Analytics screen: range tabs, trend, top merchants, category breakdown, stats row
- [ ] Warranty: API + screen (Active / Expiring soon / Expired tabs, add form, link to receipt item)
- [ ] Notifications: API + screen + unread badge; budget 80%/100% alerts; warranty 30/7-day reminders
      (scheduler goroutine, daily)
- [ ] Reports: CSV export; monthly PDF summary
- [ ] Settings: Notifications, Data & Privacy (export all data, delete account), Help, About
      (pages exist since Phase 1; Export needs Reports)
- [ ] Categories screen (list system + custom, add/edit/delete custom with icon + color)
- [ ] Mobile "More" screen (links to Budgets, Categories, Warranty, Reports, Notifications, Settings)

Chart colors: the 8 system categories use the validated dataviz palette (see
`frontend/common/src/categories/colors.ts`, light + dark steps); "others" and custom categories
are neutral/stored. Keep this when adding charts in Phase 3.

Phase 3 lands in the new layout: collection access in `backend/common/services/<domain>`, business
logic + handlers in `backend/user/internal/{services,http/api}/<domain>`, screens in
`frontend/user/src/features/<domain>`, shared widgets in `frontend/common/src`.

### Phase 5 — Admin portal (after Phase 3)
- [ ] User management: search, view, disable/enable, delete with cascade, reset password link
- [ ] System config editor (dynamic `system_config` keys, sensitive masking, reload endpoint)
- [ ] Extraction monitoring: volume, failure rate, model, average confidence; retry failed
- [ ] Default categories management (system category CRUD)
- [ ] Audit log of admin actions

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
