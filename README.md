# ExpenseLens (smart-receipt)

Receipt-first personal expense tracker. Go + MongoDB API, React + Vite frontend.

See `docs/PLAN.md` for scope and architecture, `docs/HANDOVER.md` for current status.

## Run locally

Requires Go 1.23+, Node 22+, and MongoDB on `localhost:27017` (local service or `make mongo`).

```sh
cp backend/.env.example backend/.env   # fill GEMINI_API_KEY, OAuth creds as needed
make dev                               # API on :8090, frontend on :5173
make test
make lint
```
