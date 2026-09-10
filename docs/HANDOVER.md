# Handover

Rewrite this file (do not append) at the end of each session or when context nears 200–250k tokens.

## Last updated
2026-09-10 — Phases 0, 1, 2 complete and committed (4 commits on local `main`, no remote by
choice). NEXT: Phase 2.5, restructure the backend to dbc-pro conventions — spec in
`docs/BACKEND_RESTRUCTURE.md`. Do it before Phase 3.

## Status by phase
- Phase 0 Scaffolding: done
- Phase 1 Auth: done (OAuth code complete; untested against live Google/Microsoft/Apple)
- Phase 2 Core receipt flow: done (Gemini + Cloudinary untested live; fake/local verified)
- Phase 3 Insights: not started
- Phase 4 Post-MVP: not started

## What exists
Backend (`backend/internal/*`): config, db, server (chi, error envelope, validation), auth
(argon2id, JWT, rotating refresh with 30s grace, reset tokens, go-oidc providers), users (/me),
mail (log/smtp), storage (local signed URLs at `/api/v1/files/*`, cloudinary), extraction
(interface, gemini REST w/ responseSchema, fake, worker pool), categories (seeded 9 system,
user CRUD, reassign on delete), receipts (upload per-file or grouped, sha256 duplicate check,
async extraction with 3 attempts, review/confirm, list w/ regex search + filters + sort + paging,
images add/remove, reextract, account cascade), analytics (summary KPIs w/ deltas, trend roll-up
day/week/month, by category, top merchants, stats). `cmd/api/main.go` wires all of it.

Frontend (`frontend/src`): app shell (sidebar / bottom tabs, user block, theme), auth pages +
provider, Settings (Profile/Security/Preferences/Notifications/Data/static), Upload page
(dropzone, camera, PDF, session rows, recent uploads), Review page (polls while processing,
ReceiptForm with items/tags, discard/cancel/save), Receipts list (debounced search, chips,
category filter, sort, pagination), Receipt Details (carousel, meta, items, notes, tags, edit
inline, delete, reextract, add/remove pages), Dashboard (StatTile x4, TrendChart area/line,
CategoryDonut with legend, recent receipts, budget empty state). Charts follow the dataviz skill
rules; palette validated for light+dark.

Placeholders still: Analytics, Budgets, Categories, Warranty, Reports, Notifications, More.

## Verification done
- `go test ./...`, `go vet`, frontend `tsc -b`, `oxlint`, `vite build`, `vitest` all green.
- API smoke script (44 checks, fake extractor) — `scratchpad/smoke.py`; recreate from
  `docs/PLAN.md` §4 if the scratchpad is gone.
- Playwright (playwright-core + `/usr/bin/google-chrome`) flows: auth/settings and full receipt
  flow (upload → review → confirm → details → list/search → dashboard charts → mobile → delete).

## Decisions made this session
- API port 8090 (8080 taken on this machine). Vite proxies `/api` to it.
- Dev extractor: `EXTRACTOR=fake` in env for offline work; `gemini` needs `GEMINI_API_KEY`.
- Counted statuses for analytics: `review` + `confirmed` (processing/failed excluded).
- Category colors: 8 validated slots by slug (frontend `categories/colors.ts`), backend seed
  stores the light hex; "others" neutral gray.
- Single-file uploads auto-navigate to Review unless duplicate.

## Known issues / gotchas
- Never `pkill -f` a pattern present in your own command line; use `fuser -k PORT/tcp`.
- React StrictMode double effects: keep one-shot network calls single-flighted.
- Local storage provider writes to `backend/.data/uploads` by default (gitignored).
- Tailwind v4 important modifier is a suffix (`w-auto!`).
- HEIC uploads are accepted but browsers cannot render them; Gemini can read them. Consider
  server-side conversion later.
- Gemini model default `gemini-3.1-pro-preview`; verify with the models list endpoint once a key
  exists. Consumer Google AI Pro plan ≠ API quota.

## Next steps (exact)
1. Phase 2.5: follow `docs/BACKEND_RESTRUCTURE.md` end to end (read `~/repos/dbcpro/backend/user`
   and `~/repos/infra/golang/pkg/{http,middleware,database}` first). Commit when smoke + Playwright
   pass. Update this file.
2. Phase 3 backend (in the NEW layout): `internal/budgets` (model, repo, upsert, list by month with spent via
   analytics), `internal/warranties`, `internal/notifications` (+ daily scheduler goroutine for
   budget 80/100% and warranty 30/7-day), `internal/reports` (CSV, PDF). Wire + indexes + cascade.
3. Phase 3 frontend: Analytics page (range tabs, TrendChart, top merchants bar list, donut, stats
   row), Budgets page, Warranty page, Notifications page + badge, Categories page, More page,
   Reports/Export, dashboard Budget Overview widget.
4. Ask user for Gemini key + Cloudinary URL + OAuth creds to run live verification.

## Commands
- `make dev` (API :8090 + Vite :5173), `make test`, `make lint`, `make build`.
- Offline dev: `EXTRACTOR=fake make dev-backend`.
- Smoke DB: run API with `MONGO_DB=expenselens_smoke`, drop after with
  `mongosh --quiet expenselens_smoke --eval 'db.dropDatabase()'`.
