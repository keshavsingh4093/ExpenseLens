# Handover

Rewrite this file (do not append) at the end of each session or when context nears 200–250k tokens.

## Last updated
2026-09-10 — planning session, no code written yet.

## Status by phase
- Phase 0 Scaffolding: not started
- Phase 1 Auth: not started
- Phase 2 Core receipt flow: not started
- Phase 3 Insights: not started
- Phase 4 Post-MVP: not started

## What exists
- `CLAUDE.md` (session protocol, conventions), `docs/PLAN.md` (full plan), this file.
- `backend/` and `frontend/` are empty directories.

## Decisions made this session
See `docs/PLAN.md` §1. Key: Go + chi + Mongo, React + Vite + Tailwind, Cloudinary behind a
storage interface, Gemini behind an extraction interface, four auth providers.

## In flight / half-done
Nothing.

## Known issues / gotchas
- Gemini API access is separate from the consumer Google AI Pro subscription; user needs an AI
  Studio API key. Confirm the model name with the models list endpoint before wiring defaults.
- Apple Sign In needs: Services ID, Team ID, Key ID, `.p8` private key, and a verified return URL
  over HTTPS (use a tunnel like ngrok locally).

## Next steps (exact)
1. Phase 0: `git init`, Makefile, docker-compose (Mongo), Go module + chi server + config + mongo
   bootstrap, Vite app + Tailwind + router + layout shells.
2. Commit as "Phase 0 scaffolding".
3. Start Phase 1 auth.

## Commands
Not yet applicable. Target: `make dev`, `make test`.
