# smart-receipt (ExpenseLens)

Receipt-first personal expense tracker. INR-first, Indian market, responsive web app.

## Read these first, every session
1. `docs/PLAN.md` — scope, architecture, data model, API, phased task list with checkboxes.
2. `docs/HANDOVER.md` — where the last session stopped, what is in flight, gotchas.

## Session protocol
- Work through `docs/PLAN.md` tasks in order unless the user redirects. Tick checkboxes as tasks land.
- When context usage reaches ~200–250k tokens, or a task finishes and the session is likely to end,
  rewrite `docs/HANDOVER.md` (not append) with: status per phase, files touched, decisions made,
  known issues, exact next steps, and any commands needed to run/verify.
- Keep `docs/PLAN.md` the source of truth for decisions. If a decision changes, edit the plan, don't
  just mention it in chat.

## Stack
- Frontend: React 19 + Vite + TypeScript + Tailwind v4. Lives in `frontend/` (workspaces `common`, `user`, `admin`).
- Backend: Go, chi router, MongoDB (official driver v2). Lives in `backend/` (modules `common`, `user`, `admin`).
- Storage: pluggable `storage.Provider` interface. Cloudinary in prod, local disk in dev.
- Extraction: pluggable `extraction.Extractor` interface. Gemini API (`gemini-3.1-pro-preview`
  default, model name from env).
- Auth: email/password + Google + Microsoft + Apple. JWT access token + rotating refresh cookie.

## Conventions
- Layout: `backend/{common,user,admin}` (Go modules, `replace ../common`) and
  `frontend/{common,user,admin}` (npm workspaces, `@common` alias) per
  `docs/BACKEND_RESTRUCTURE.md` (dbc-pro conventions). Shared code lives in `common`. Handlers:
  `GetRouteSpecs()` + `var svc = domain.New()` closures, claims via `middleware.GetUserFromContext`;
  no business logic in handlers. Collection access in `common/services/<domain>`, portal logic in
  `<portal>/internal/services/<domain>`.
- No code from outside this repository: no `replace` directives to sibling repos, no copied files
  from `~/repos/dbcpro` or `~/repos/infra`. Read them for conventions only.
- Never build Mongo indexes on the server boot path (see dbc-pro `backend/CLAUDE.md`); use the
  `ensure-indexes` tool.
- All money stored as int64 minor units (paise) + ISO currency code. Never floats.
- Dates stored as UTC `time.Time`; receipt date also kept as `YYYY-MM-DD` string in the user's tz.
- API paths: `/openapi/…` unauthenticated, `/api/…` authenticated. Envelope
  `{success, data, error, code, fields}`; validation failures are 422 with `fields`.
- Config: `.env` holds only bootstrap keys (`ENV`, `PORT`, `MONGO_URI`, `MONGO_DATABASE`,
  `JWT_SECRET`); everything else via `config.GetConfigVal` (env → `system_config` → default).
- Frontend: feature folders under `<app>/src/features/<name>`, shared UI/charts/api/auth in
  `frontend/common/src`, TanStack Query for server state. Tokens in `common/tailwind.preset.css`.
- Run: `make ensure-indexes` once, then `make dev` (user) / `make dev-admin` (admin). See Makefile.
- Tests: `make test`. Go tests next to code; frontend uses Vitest.

## Do not
- Do not commit `.env` files. `.env.example` files are the contract.
- Git is local only; do not add remotes or push unless the user asks.
- Do not hardcode Cloudinary or Gemini specifics outside their provider packages.
