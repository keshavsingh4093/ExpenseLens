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
- Frontend: React 19 + Vite + TypeScript + Tailwind v4. Lives in `frontend/`.
- Backend: Go 1.23+, chi router, MongoDB (official driver v2). Lives in `backend/`.
- Storage: pluggable `storage.Provider` interface. Cloudinary in prod, local disk in dev.
- Extraction: pluggable `extraction.Extractor` interface. Gemini API (`gemini-3.1-pro-preview`
  default, model name from env).
- Auth: email/password + Google + Microsoft + Apple. JWT access token + rotating refresh cookie.

## Conventions
- Backend layout: `backend/cmd/api`, `backend/internal/<domain>` (handler.go, service.go, repo.go,
  model.go per domain). No business logic in handlers.
- All money stored as int64 minor units (paise) + ISO currency code. Never floats.
- Dates stored as UTC `time.Time`; receipt date also kept as `YYYY-MM-DD` string in the user's tz.
- API is JSON under `/api/v1`. Errors: `{ "error": { "code": "...", "message": "..." } }`.
- Frontend: feature folders under `src/features/<name>`, shared UI in `src/components/ui`,
  API client in `src/lib/api`, TanStack Query for server state.
- Run: `make dev` at repo root starts mongo (docker), backend, and frontend. See Makefile.
- Tests: `make test`. Go tests next to code; frontend uses Vitest.

## Do not
- Do not commit `.env` files. `.env.example` files are the contract.
- Do not hardcode Cloudinary or Gemini specifics outside their provider packages.
