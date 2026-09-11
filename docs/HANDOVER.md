# Handover

Rewrite this file (do not append) at the end of each session or when context nears 200–250k tokens.

## Last updated
2026-09-11 — Phase 5 (admin portal: user disable/enable/delete/reset-link, extraction monitoring +
retry, system category CRUD, audit log, four admin pages) is DONE, verified (admin smoke 104
checks, user smoke 120 checks, Playwright admin flow 69 steps at 1280px + 400px, `e2e/phase3.mjs`
still green, `make test`/`make lint`/`make build` green) and committed on local `main`.
NEXT: live-credential verification (PLAN §7) — blocked until the user supplies the Gemini key,
Cloudinary URL and OAuth client IDs. Do not start it without them.

## Status by phase
- Phase 0 Scaffolding: done
- Phase 1 Auth: done (OAuth code complete; untested against live Google/Microsoft/Apple)
- Phase 2 Core receipt flow: done (Gemini + Cloudinary untested live; fake/local verified)
- Phase 2.5 Restructure: done
- Phase 3 Insights and tracking: done
- Phase 5 Admin portal: DONE (this session)
- Phase 4 Post-MVP: not started
- PLAN §7 live checks (3 real Indian receipts + Amazon PDF through Gemini, Cloudinary upload, OAuth
  logins): NOT done, waiting on credentials

## Phase 5 as built
Backend (`backend/common`):
- `services/user/cascade.go` — `user.DeleteCascade(ctx, uid, store)`: receipts (+ files through the
  `storage.Provider`, nil = keep files), categories, budgets, warranties, notifications, every
  `auth_tokens` row for the user, then the user doc. Used by `DELETE /api/me` (user portal) and
  `DELETE /api/admin/users/{id}` (admin). The old per-domain cascade in
  `user/internal/services/auth` is gone.
- `services/auth`: `EnsureActive` (403 `account_disabled` / 401 when deleted), `SetDisabled`
  (revokes refresh tokens when disabling), `ResetLink` (1h reset token → `APP_URL/reset-password?
  token=`). Login/refresh/OAuth already rejected disabled users.
- `pkg/middleware`: `SetAuthGuard(func(ctx, claims) error)` runs inside `EnsureAuth` after the JWT
  parses; errors implementing `HTTPStatus()/ErrorCode()` (`gohttp.APIError`) keep their status.
  The user router installs it (`routersetup.go#activeUserGuard`, one projected `_id` lookup per
  request). The admin router does not (admins are checked at login/refresh only).
- `services/category`: `GetSystem`, `SystemBySlug`, `CreateSystem`, `UpdateSystem` (others slug
  immutable), `DeleteSystem` (others protected). User `Create` now checks the live system slugs
  (not the static seed list) before appending `-custom`.
- `services/receipt`: `FindByID`, `ReassignCategoryAll`, `CountByCategory`, `ExtractionByDay`,
  `ExtractionByModel`, `ExtractionTotalsSince`, `ListFailed` (aggregation rows). Confidence
  averages exclude failed receipts.
- `services/adminaudit` + `models.AdminAuditEntry` (`admin_audit_log`), `services/user`
  `IsDisabled`, `EmailsByIDs`, `ListFilter.Status`; `mail.Discard()`.
Backend (`backend/user`): `POST /openapi/receipts/{id}/reextract` (secret header, 404 when
`CONFIG_RELOAD_SECRET` unset, 401 on mismatch) → `receipt.ReextractByID`; `receipt.Store()` exposes
the provider for the cascade.
Backend (`backend/admin`):
- config gains `APP_URL`, `STORAGE_PROVIDER`, `LOCAL_STORAGE_DIR` (default `../user/.data/uploads`),
  `CLOUDINARY_URL`; `app.Run` builds the storage provider and calls `users.Configure(store)`.
- `services/audit.Log(r, action, targetType, targetID, details)` — detached 5s context, never fails
  the request; called after every successful mutation (users, extraction retry, categories, and
  system-config upsert/delete/reload). Reset links and config values are never written.
- `services/{users,extraction,categories}` hold the logic; handlers under `http/api/{users,
  extraction,categories,audit}`. `middleware.RequireRole("superadmin")` is a RouteSpec extra on
  user delete only. Retry POSTs to `USER_API_URL/openapi/receipts/{id}/reextract`; returns 503
  `retry_unavailable` when the secret/URL is unset, 502 when the user API is down.
- `ensure-indexes` now also builds `admin_audit_log` indexes (run `make ensure-indexes`; it was
  run against the dev DB `expenselens` this session, so both services boot without warnings).
Frontend (`frontend/admin`):
- `features/users`: status chips (All/Active/Disabled), detail modal with Disable/Enable, Reset
  link (read-only input + copy button, single-use note), Delete (superadmin only; inline confirm).
- `features/extraction/ExtractionPage.tsx`: range chips, 4 tiles, stacked daily columns
  (succeeded = `--viz-ok`, failed = `--viz-failed`, tokens in `admin/src/index.css`, validated
  with the dataviz palette script on both surfaces), table toggle, by-model table, failed list
  with Retry (refetches every 15s).
- `features/categories`: system list with receipt counts, add/edit modal (name, slug, colour,
  icon picker from `@common/categories/icons`), Others locked.
- `features/audit`: table with action select, target-id input, admin chip filter (click an admin
  or target cell), paging.
- `AdminShell` nav: Dashboard, Users, Extraction, Categories, System Config, Audit log.

## Verification done this session
- `make ensure-indexes` against dev DB `expenselens` (Phase 3 + audit indexes now exist).
- Backend: `go build`/`go vet` ×3, `make test`, gofmt clean. Frontend: `tsc` ×3, oxlint (only the
  pre-existing `incompatible-library` warning), vitest, `make build`.
- `admin_smoke.py` (104 checks, :8094 + user API :8093, scratch DB `expenselens_smoke`): everything
  from Phase 2.5 plus disable → live token 403 / login 403 / refresh rejected / status filters,
  enable, reset link → consumed via `/openapi/auth/reset-password` → login with new password →
  link single use; extraction stats 7d/30d (day fill, model breakdown, failurePct), failed list
  with owner email, retry → receipt back to `review`, ops route secret checks; system category
  create/dup 409/validation/user visibility/`-custom` slug/update/others protection/delete moves
  receipts; plain admin 403 on delete, superadmin delete → all six collections empty for the user
  and files removed from disk; audit entries for every action, filters, paging, actions list.
- `smoke.py` (120 checks) rerun: green.
- Playwright `e2e/admin.mjs` (69 steps, Chrome at /usr/bin/google-chrome): seeds users/receipts via
  the user API (one forced `failed` via mongosh), then all user actions, extraction chart hover +
  range + table toggle + retry, categories CRUD, system config, delete user, audit rows/filters,
  plain-admin delete disabled, six pages at 400px with no overflow, dialogs fit at 400px. The only
  console error is the expected 409 from the duplicate-slug check (filtered).
- `e2e/phase3.mjs` rerun against the new user API (Vite on :5175): green.
- Scratchpad: `/tmp/claude-1001/-home-keshav-repos-smart-receipt/0a2de432-76e5-465e-9be3-f5ec693bd3ac/scratchpad/`
  (`run_smoke.sh` boots the whole smoke env, `smoke.py`, `admin_smoke.py`, `e2e/{admin,phase3,flow,
  receipts}.mjs` — user flows read `USER_URL`, admin reads `ADMIN_URL`; `shot-admin-*.png`). If gone:
  recreate from `docs/PLAN.md` §4.

## Decisions made this session
- Disabled users are rejected on every `/api` request (DB lookup in the auth guard) rather than
  waiting for the 15-minute access token to expire; the check lives in the common auth service.
- Deleting a user from either portal goes through the single `user.DeleteCascade`; auth tokens are
  deleted (not just marked used) because the subject no longer exists.
- Admin retry goes through the user API's secret-guarded ops route (the admin process has no
  worker pool). Both services must share `CONFIG_RELOAD_SECRET`.
- The admin API must be configured with the same storage settings as the user API; with the local
  provider `LOCAL_STORAGE_DIR` is relative to each service's cwd, so use an absolute path in dev.
- Audit details never contain secrets: reset-link entries carry no details, config entries only
  the key and the `sensitive` flag.
- Extraction chart: succeeded = categorical slot 1 blue, failed = status critical red (icon + label
  in the legend, not colour alone); confidence averages exclude failures.
- Local dev hack: the user's uncommitted edit to `backend/user/internal/app/server.go` (providers
  commented out, Fatal → Warn) was stashed as `stash@{0}` during this session and popped back
  after the commits. It is NOT committed; `git stash list` should be empty after the pop.

## Known issues / gotchas
- Never `pkill -f` a pattern present in your own command line; use `fuser -k PORT/tcp`.
- React StrictMode double effects: keep one-shot network calls single-flighted (AuthProvider does).
- Local storage provider writes to `LOCAL_STORAGE_DIR` relative to the server cwd (default
  `backend/user/.data/uploads`, gitignored); the admin default `../user/.data/uploads` matches only
  when both run from their module dirs (`make dev*` does).
- The local provider stores a `.type` sidecar next to each file; count `*.png` etc. in tests.
- Tailwind v4 important modifier is a suffix (`w-auto!`); `@source '../../common/src'` is required
  in each app's index.css or common classes are purged.
- Playwright: `<option>` elements are never "visible"; wait with `{ state: 'attached' }`. Recharts
  bar `<g>` wrappers are never "visible" either — wait on `.recharts-bar-rectangle path` attached
  and hover the path. ResponsiveContainer can blank briefly after a range change; wait ~500ms
  before screenshots. Sidebar links exist (hidden) at 400px, so scope mobile selectors with `main`
  or `nav:visible`.
- The fake extractor never fails at runtime; force a failed receipt with a mongosh update (see
  `admin_smoke.py`).
- Port 5173 was occupied by the user's own Vite during this session; the e2e used :5175 (user) and
  :5174 (admin) with `VITE_API_PROXY` pointing at the smoke APIs.
- HEIC uploads are accepted but browsers cannot render them; Gemini can read them.
- Gemini model default `gemini-3.1-pro-preview`; verify with the models list endpoint once a key
  exists. Consumer Google AI Pro plan ≠ API quota.
- oxlint warns `react(incompatible-library)` on `useForm` usages; harmless.
- Weekly summary and warranty reminder "days left" use the UTC date; a user far from UTC may see a
  one-day difference near midnight.

## Next steps (exact)
1. Ask the user for `GEMINI_API_KEY`, `CLOUDINARY_URL`, and the Google/Microsoft/Apple OAuth
   client IDs + secrets. Put them in `backend/user/.env` (and `CLOUDINARY_URL` +
   `STORAGE_PROVIDER=cloudinary` in `backend/admin/.env`). Never commit them.
2. PLAN §7: `GET https://generativelanguage.googleapis.com/v1beta/models` with the key to confirm
   the model name; run 3 real Indian receipts + 1 Amazon PDF through upload → review and check
   total + ≥80% items; confirm a Cloudinary upload/URL/delete round trip (upload, then delete the
   receipt and the account cascade); sign in with each OAuth provider on :5173.
3. Then Phase 4 candidates, in order of value: warranty auto-suggest from `warrantyCandidates` on
   the review page, duplicate-upload prompt, SSE for extraction status.

## Commands
- `make install`, `make ensure-indexes`, `make create-admin EMAIL=… PASSWORD=… NAME=… ROLE=superadmin`.
- `make dev` (user API :8090 + Vite :5173), `make dev-admin` (:8091 + :5174), `make test`,
  `make lint`, `make build`. Offline dev: `EXTRACTOR=fake make dev-backend`.
- Admin retry/reload in dev needs `CONFIG_RELOAD_SECRET` set (same value) in both `.env` files.
- Smoke: `<scratch>/run_smoke.sh` (drops `expenselens_smoke`, builds both servers into the scratch
  dir, ensure-indexes, creates `admin@example.com/adminsecret1` (superadmin) and
  `viewer@example.com/viewersecret1` (admin), starts :8093 and :8094 with `CONFIG_RELOAD_SECRET=
  smoke-secret`, fake extractor, local storage in `<scratch>/uploads`), then `python3 smoke.py`
  and/or `python3 admin_smoke.py` (each expects a fresh env: rerun `run_smoke.sh` in between).
- E2E: in `frontend/`, `VITE_API_PROXY=http://localhost:8094 npm run dev -w admin -- --port 5174`
  and `VITE_API_PROXY=http://localhost:8093 npm run dev -w user -- --port 5175`; then from
  `<scratch>/e2e`: `S=<scratch> node admin.mjs` and `S=<scratch> USER_URL=http://localhost:5175
  node phase3.mjs`.
