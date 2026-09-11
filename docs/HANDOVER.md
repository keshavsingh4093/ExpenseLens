# Handover

Rewrite this file (do not append) at the end of each session or when context nears 200–250k tokens.

## Last updated
2026-09-11 — Phase 5.1 "Admin hardening" is DONE, verified and committed on local `main`:
(1) gofmt sweep, (2) emailed password-reset links from the admin portal, (3) admin account
management (Admins page + `/api/admin/admins`, bearer guard for disabled admins). Verified with
`admin_smoke.py` (160 checks), `smoke.py` (unchanged, green), Playwright `e2e/admin.mjs` (Admins
flow + Email link at 1280px and 400px, no browser errors), `make test`/`make lint`/`make build`.
NEXT: live-credential verification (PLAN §7) — still blocked until the user supplies the Gemini key,
Cloudinary URL and OAuth client IDs. Do not start it without them. After that: Phase 4.

## Status by phase
- Phase 0 Scaffolding: done
- Phase 1 Auth: done (OAuth code complete; untested against live Google/Microsoft/Apple)
- Phase 2 Core receipt flow: done (Gemini + Cloudinary untested live; fake/local verified)
- Phase 2.5 Restructure: done
- Phase 3 Insights and tracking: done
- Phase 5 Admin portal: done; Phase 5.1 Admin hardening: DONE (this session)
- Phase 4 Post-MVP: not started
- PLAN §7 live checks (3 real Indian receipts + Amazon PDF through Gemini, Cloudinary upload, OAuth
  logins): NOT done, waiting on credentials

## Phase 5.1 as built (this session)
Commits: `chore: gofmt` (five long-line nits in `backend/common`), then one feature commit.

Backend (`backend/common`):
- `services/adminuser`: `List(ListFilter{Query,Page,PageSize})` (total, newest first),
  `SetDisabled`, `CountSuperadmins` (active only: `role=superadmin, disabled≠true`), `IsDisabled`
  (projected lookup for the guard). `models.AdminRoleAdmin/Superadmin`, `ValidAdminRole`;
  `PublicAdmin` now carries `disabled`.
- `services/auth`: `EmailResetLink(ctx, uid)` issues the 1h reset token and sends it with the same
  `resetMessage` as `ForgotPassword` (shared helper); `ResetLink` unchanged. `issueResetLink` is the
  common core.
- `services/auth/tokens.go`: **`Revoke`/`RevokeAll` now also set `expiresAt = now`.** Before, they
  only set `usedAt`, which the 30s refresh grace window (`Consume`) treated as "just rotated", so a
  revoked refresh token still worked for 30s. Disabled users/admins were already rejected by the
  `Disabled` checks, so only the new "set password revokes sessions" path exposed it.
Backend (`backend/admin`):
- config: `Mail mail.Config` from `MAIL_PROVIDER` (default `log`), `MAIL_FROM`, `SMTP_HOST/PORT/
  USER/PASS` via `GetConfigVal` (same keys as the user backend; added to `.env.example`).
- `services/users`: `core()` now uses `mail.New(cfg.Mail)`; `EmailResetLink`. Handler
  `POST /api/admin/users/{id}/reset-password-link` reads an optional body (`{"send": true}`; empty
  body = old behaviour; unknown fields 400). Send → `{sent: true, expiresAt}` without the link,
  audit `{emailed: true}`; otherwise `{link, expiresAt}` and audit `{emailed: false}`.
- `services/adminauth`: `EnsureActive` (401 unknown / 403 `account_disabled`), `SetPassword`
  (validates, hashes, revokes admin refresh tokens), `RevokeSessions`. `CreateAdmin` now returns
  422 `APIError`s for bad password/role (the `create-admin` script prints the field messages).
- `services/admins` + `http/api/admins`: `GET /api/admin/admins` (any admin), superadmin-only
  `POST /api/admin/admins`, `PATCH /{id}` (name/role), `POST /{id}/disable|enable`,
  `POST /{id}/reset-password`. `guardSuperadmin`: target == actor → 409 `self_target`; target is an
  active superadmin and `CountSuperadmins() <= 1` → 409 `last_superadmin` (checked in the DB, so a
  stale superadmin claim on an already-demoted actor cannot remove the last one). Disable revokes
  refresh tokens. Audit `admin.create|update|disable|enable|reset_password`, targetType `admin`,
  details only email/name/role.
- `routersetup.go` installs `commonmw.SetAuthGuard(activeAdminGuard)` → `adminauth.EnsureActive`
  (one projected `admin_users` lookup per `/api` request).
- `create-admin` doc comment: bootstrap path for the first superadmin only.
Frontend (`frontend/admin`):
- `features/users`: `usersApi.emailResetLink`; "Email link" button beside "Reset link"; success
  Alert "Sent to <email>, expires HH:MM" (clears the copy-link box); mail errors → error Alert.
- `features/admins/{api.ts,AdminsPage.tsx}`: table (name + "(you)", email, role, status, last
  login, actions), search, paging; "Add admin" modal (name, email, role select, password with
  "Generate" = 16 chars from Web Crypto; inline field errors from 422); row actions for superadmins:
  role `<select>` (PATCH on change), Disable/Enable, "Set password" modal (success note says other
  sessions were signed out). The signed-in admin's own row has no role select and no Disable.
  Plain admins: info Alert "…read-only for you", no Add button, no Actions column.
- `AdminShell` nav: Dashboard, Users, **Admins**, Extraction, Categories, System Config, Audit log.
  Router: `/admins`. `Admin` type has `disabled?`.

## Verification done this session
- `gofmt -l` clean ×3, `go build`/`go vet` ×3, `make test` (auth package rerun with `-count=1`),
  `make lint` (only the pre-existing `incompatible-library` warning), `make build`.
- `admin_smoke.py` (160 checks): everything from Phase 5 plus emailed reset link (200, `sent`, no
  link, mail text in `admin.log`, `send:false`, 404, unknown field 400, audit `emailed` flags);
  admins list/search/paging; plain admin can read but gets 403 on all five mutating routes;
  create (422 short password / bad role / bad email, 201 with email normalised, 409 duplicate);
  login as the new admin; empty patch 400; rename; promote; self-disable/self-demote 409;
  disable → live token 403 `account_disabled` on `/api/admin/me` and the list, refresh 401, login
  403; enable → login works; demote then the stale-superadmin token gets 409 `last_superadmin` on
  disable and demote of the last superadmin; set password 422 / 200 → old refresh 401, old password
  401, new password 200; audit has all five `admin.*` actions with targetType `admin` and no
  password anywhere. Existing audit counts updated (`targetId` filter now 8, reset-link entries
  carry `{emailed}`).
- `smoke.py` (user API) rerun unchanged: green.
- Playwright `e2e/admin.mjs` (Chrome at /usr/bin/google-chrome, Vite admin on :5174 →
  API :8094): all previous steps + "Email link" → "Sent to e2e-main@example.com, expires …";
  Admins page: own row (you) without disable/demote, add-admin (inline short-password error,
  Generate → 16 chars, create, API login with the generated password), role select → superadmin,
  disable → API login 403, enable, set password → old fails / new works, search; plain admin sees
  the read-only hint with no actions; 400px: seven pages without overflow, Email-link Alert, Add
  admin dialog + Generate button, admins page fit the viewport. Console filter ignores the
  intentional 401/409/422 responses. "no browser errors".
- Scratchpad: `/tmp/claude-1001/-home-keshav-repos-smart-receipt/5b4c2b22-32f5-480d-9129-ced08c2c6d72/scratchpad/`
  (`run_smoke.sh`, `smoke.py`, `admin_smoke.py`, `e2e/{admin,phase3,flow,receipts}.mjs`,
  `shot-admin-*.png`). The previous session's copy at `…/0a2de432-…/scratchpad/` still exists and is
  older. If both are gone: recreate from `docs/PLAN.md` §4.

## Decisions made this session
- Emailed reset links reuse the exact forgot-password mail (subject/body) via a shared helper in
  the common auth service; the admin never sees the link when it is emailed, and the audit only
  records `emailed: true|false`.
- The admin API mail sender is built from the same `MAIL_*`/`SMTP_*` keys as the user API; in dev
  the `log` provider writes the link to the admin server log.
- Superadmin rules are enforced in the DB (`CountSuperadmins`), not from JWT claims, so a demoted
  admin whose 30-minute access token still says `superadmin` cannot remove the last superadmin.
  `RequireRole` still reads the claim for the coarse 403; that is acceptable for the token TTL.
- Self-protection returns 409 `self_target` (checked before `last_superadmin`).
- Token revocation expires the token instead of only marking it used (see tokens.go note above).
- `create-admin` remains the only way to create the first superadmin; documented in PLAN §4.
- The uncommitted `backend/user/internal/app/server.go` edit mentioned in earlier handovers is NOT
  present in the working tree this session (`git status` showed only `backend/user/.air.toml`
  untracked), so nothing was stashed. `.air.toml` is still untracked and untouched.

## Known issues / gotchas
- Never `pkill -f` a pattern present in your own command line; use `fuser -k PORT/tcp`.
- Playwright: after a sidebar click, wait for a selector unique to the new page (e.g.
  `tbody tr[data-email=…]` on Admins); `tbody tr` can still match the previous route's table.
- `RequireRole` uses the JWT role claim: a promoted admin must log in again to use superadmin
  routes; a demoted one keeps the coarse role until the token expires (≤30 min) but the DB checks
  above still apply.
- `gohttp.Decode` rejects empty bodies; the reset-link handler reads the body itself to make it
  optional. Use the same pattern for any other "optional JSON body" route.
- React StrictMode double effects: keep one-shot network calls single-flighted (AuthProvider does).
- Local storage provider writes to `LOCAL_STORAGE_DIR` relative to the server cwd; the admin
  default `../user/.data/uploads` matches only when both run from their module dirs.
- The local provider stores a `.type` sidecar next to each file; count `*.png` etc. in tests.
- Tailwind v4 important modifier is a suffix (`w-auto!`); `@source '../../common/src'` is required
  in each app's index.css or common classes are purged.
- Playwright: `<option>` elements are never "visible"; wait with `{ state: 'attached' }`. Recharts
  bar `<g>` wrappers are never "visible" either — wait on `.recharts-bar-rectangle path`. Sidebar
  links exist (hidden) at 400px, so scope mobile selectors with `main` or `nav:visible`.
- The fake extractor never fails at runtime; force a failed receipt with a mongosh update.
- Port 5173 is occupied by the user's own Vite; e2e uses :5174 (admin) / :5175 (user) with
  `VITE_API_PROXY` pointing at the smoke APIs.
- HEIC uploads are accepted but browsers cannot render them; Gemini can read them.
- Gemini model default `gemini-3.1-pro-preview`; verify with the models list endpoint once a key
  exists. Consumer Google AI Pro plan ≠ API quota.
- oxlint warns `react(incompatible-library)` on `useForm` usages; harmless.
- Weekly summary and warranty reminder "days left" use the UTC date.

## Next steps (exact)
1. Ask the user for `GEMINI_API_KEY`, `CLOUDINARY_URL`, and the Google/Microsoft/Apple OAuth
   client IDs + secrets. Put them in `backend/user/.env` (and `CLOUDINARY_URL` +
   `STORAGE_PROVIDER=cloudinary` in `backend/admin/.env`). Never commit them.
2. PLAN §7: `GET https://generativelanguage.googleapis.com/v1beta/models` with the key to confirm
   the model name; run 3 real Indian receipts + 1 Amazon PDF through upload → review and check
   total + ≥80% items; confirm a Cloudinary upload/URL/delete round trip; sign in with each OAuth
   provider on :5173. Optionally set `MAIL_PROVIDER=smtp` + `SMTP_*` and send one emailed reset
   link from the admin portal.
3. Then Phase 4 candidates, in order of value: warranty auto-suggest from `warrantyCandidates` on
   the review page, duplicate-upload prompt, SSE for extraction status.

## Commands
- `make install`, `make ensure-indexes`, `make create-admin EMAIL=… PASSWORD=… NAME=… ROLE=superadmin`
  (first superadmin only; later admins via the Admins page).
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
  `<scratch>/e2e`: `S=<scratch> node admin.mjs` (fresh `run_smoke.sh` first) and
  `S=<scratch> USER_URL=http://localhost:5175 node phase3.mjs`. Stop servers with
  `fuser -k 5174/tcp 8093/tcp 8094/tcp`.
