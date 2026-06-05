# CLAUDE.md

Guidance for Claude Code (and humans) working in this repo. Read the linked docs
for depth — this file is the orientation layer.

## What this is

**SalehCard** — a digital-goods e-commerce platform (gaming gift cards, top-ups,
money transfers). Monorepo:

- **`/api`** — Go 1.22 + chi + MongoDB (driver v2). Module path `github.com/AliSleiman0/salehcard/api`.
- **`/web`** — React 18 + TS + Vite storefront (customer-facing), port **5173**.
- **`/admin`** — React 18 + TS + Vite admin console, port **5174** (separate app).
- **`/deploy`** — docker-compose for MongoDB + env examples.

## Read these first

- **`README.md`** — stack, prerequisites, quick start.
- **`CONVENTIONS.md`** — backend module layout, response envelope, frontend design-system rules. **Follow it.**
- **`HANDOFF.md`** — current funnel state and the next steps (Step 3: refunds, admin manual-completion, etc.).
- **`MANUAL-TEST.md`** — A→Z manual browser test of the implemented funnel.
- **`PURCHASE-FUNNEL.md`** — overall funnel assessment.

## Build / run / test

MongoDB on `localhost:27017` (db `salehcard`). pnpm 9 + Node 20, Go 1.22.

```bash
# DB (or reuse a running mongo on :27017)
make up                      # docker compose mongodb

# Seed dev data (idempotent): 3 products + customer/admin users
cd api && go run ./cmd/seed

# API  — run on :8090 if 8080 is taken; web/.env already targets :8090
cd api && PORT=8090 go run ./cmd/server

# Storefront → http://localhost:5173
cd web && pnpm dev

# Admin console → http://localhost:5174
cd admin && pnpm dev
```

Verify gates (run the relevant ones for what you touched):
```bash
cd api && go build ./... && go vet ./... && go test ./...
cd web && pnpm build && pnpm lint && pnpm test     # 2 pre-existing react-refresh warnings are harmless
```

Seeded accounts (password `password123`): `customer@salehcard.local`, `admin@salehcard.local`.

> **Environment notes carried from the previous machine** (verify on a new box):
> pnpm was installed at `~/.local/bin` (not on PATH by default) — `export PATH="$HOME/.local/bin:$PATH"`.
> Ports 8080/27017 were occupied by other projects, so the API was run on **8090** and
> `web/.env` set `VITE_API_BASE_URL=http://localhost:8090`. `.env` files are gitignored —
> copy from each `.env.example`. `.claude/memory/` holds the prior session memory; see its
> README to restore it into `~/.claude`.

## Architecture & conventions that bite

- **Backend modules** (`api/internal/modules/<name>/`): `model / repository / service /
  handler / routes` split (see CONVENTIONS.md). Wire deps explicitly in `routes.go` /
  `server.go` — no DI container.
- **MongoDB is standalone → no multi-document transactions.** Achieve consistency with
  **document-atomic `FindOneAndUpdate`** (e.g. code claim: `available→delivered`; wallet
  debit: `$inc` guarded by `$gte`) plus explicit compensation. Never select-then-update.
- **Auth**: JWT access token in memory + rotating refresh token in an httpOnly cookie.
  `auth.AuthRequired` gates customer routes; `auth.AdminOnly` gates `/api/admin/*` and
  **dev-bypasses when `JWT_SECRET` is empty** (logs a warning). `api/.env` ships a
  `JWT_SECRET`, so to bypass admin auth in dev you must run with `JWT_SECRET=` empty.
- **Orders / payment / fulfillment** (implemented): server **re-prices from the catalog**
  (never trusts client price/fulfillment); `Idempotency-Key` header dedupes via a
  partial-unique index; order-first → atomic code claim → `completed`, with compensation
  (release codes + wallet refund) on failure; `account_credit`/`transfer` → `processing`
  (manual admin completion is a future step). Payment methods: card (mock approve), wallet
  (real ledger debit), usdt (auto-approve mock).
- **CORS gotcha**: any custom request header must be in the server's CORS `AllowedHeaders`
  (`internal/server/server.go`) or the browser preflight blocks it — `Idempotency-Key` is
  there for this reason. curl won't catch this (no preflight); test in a real browser.
- **Frontend design system is CSS-variable based** (`web/src/styles/`, `admin/src/styles/`),
  **NOT** Tailwind `dark:` utilities. Components emit ported class names (`.abtn .acard
  .tbl .bdg .st-* .ff-*` etc.). Theme via `[data-theme]`; locale forced to English on load.
- **Frontend data**: catalog + orders + wallet are wired to the API via React-Query hooks
  + `adapt*` mappers (`features/*/lib/adapt*.ts`). Remaining mock data lives in
  `web/src/lib/mock/demo.ts` / `admin/src/lib/mock/demo.ts` behind `// TODO` markers.
- **Customer routes guarded** by `RequireAuth` in `web/src/app/router.tsx`: `/dashboard`,
  `/wallet`, `/orders`, `/orders/:id`, `/checkout`, `/order-success/:id`.

## Git / workflow

- Default branch is `main`. Commit/push only when asked.
- End commit messages with the `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>` trailer.
