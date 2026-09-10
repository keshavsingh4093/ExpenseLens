# Handover

Rewrite this file (do not append) at the end of each session or when context nears 200–250k tokens.

## Last updated
2026-09-10 — Phase 0 and Phase 1 complete and committed.

## Status by phase
- Phase 0 Scaffolding: done
- Phase 1 Auth: done (OAuth code complete but unverified against live providers; needs credentials)
- Phase 2 Core receipt flow: not started
- Phase 3 Insights: not started
- Phase 4 Post-MVP: not started

## What exists
- Backend: `cmd/api`, `internal/{config,db,server,auth,users,mail}`. Auth = argon2id passwords,
  HS256 access JWT (15m), opaque refresh tokens in `auth_tokens` (rotated, 30s reuse grace),
  password reset tokens, OIDC providers via go-oidc (google/microsoft/apple), `/me` endpoints.
- Frontend: app shell (sidebar + bottom tabs), theme provider, auth provider with single-flight
  bootstrap via refresh cookie, login/register/forgot/reset/oauth-callback pages, Settings with
  Profile/Security/Preferences/Notifications/Data pages. All other routes are placeholders.
- Verified: Go tests, curl smoke of every auth endpoint, Playwright flow through Chrome
  (`/usr/bin/google-chrome`, script at scratchpad `e2e/flow.mjs` — recreate if needed).

## Decisions made this session
See `docs/PLAN.md` §1. Key: Go + chi + Mongo, React + Vite + Tailwind, Cloudinary behind a
storage interface, Gemini behind an extraction interface, four auth providers.

## In flight / half-done
Nothing.

## Known issues / gotchas
- Port 8080 is occupied on this machine by an unrelated process; API defaults to 8090.
- Docker is not installed; MongoDB 6 runs as a local systemd service on 27017.
- Never `pkill -f` with a pattern that appears in your own shell command; use `fuser -k PORT/tcp`.
- React StrictMode double-fires effects: any one-shot network call in an effect must be
  single-flighted (see `AuthProvider.tsx`).
- Gemini API access is separate from the consumer Google AI Pro subscription; user needs an AI
  Studio API key. Confirm the model name with the models list endpoint before wiring defaults.
- Apple Sign In needs: Services ID, Team ID, Key ID, `.p8` private key, and a verified return URL
  over HTTPS (use a tunnel like ngrok locally).

## Next steps (exact)
1. Phase 2 backend: `internal/storage` (Provider interface, local, cloudinary), `internal/extraction`
   (Extractor interface, gemini, fake, worker pool), `internal/categories` (seed + CRUD),
   `internal/receipts` (upload, CRUD, list/search, confirm, images), `internal/analytics`
   (summary/trend/categories for dashboard). Wire into main, add index sets.
2. Phase 2 frontend: Upload, Review, Receipts list, Receipt details, Dashboard.
3. Commit per sub-step; update this file.

## Commands
- `make dev` (API :8090 + Vite :5173), `make test`, `make lint`, `make build`.
- Smoke DB: run API with `MONGO_DB=expenselens_smoke` and drop it after
  (`mongosh --quiet expenselens_smoke --eval 'db.dropDatabase()'`).
