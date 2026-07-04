# CLAUDE.md

Guidance for Claude Code (and humans) working in this repo. Read the linked docs
for depth — this file is the orientation layer.

## What this is

**SalehCard** — a digital-goods e-commerce platform (gaming gift cards, top-ups,
money transfers). Monorepo:

- **`/api`** — Go 1.22 + chi + MongoDB (driver v2). Module path `github.com/AliSleiman0/salehcard/api`.
- **`/app`** — **Flutter customer mobile client** (Riverpod, go_router) on the Go API — the
  customer-facing app going forward (supersedes the `/web` storefront). See `app/HANDOFF-DESIGN.md`.
- **`/web`** — React 18 + TS + Vite storefront (customer-facing), port **5173** (legacy; being retired).
- **`/admin`** — React 18 + TS + Vite admin console, port **5174** (separate app).
- **`/deploy`** — docker-compose for MongoDB + env examples.

> **Live in production on Azure** (CI/CD auto-deploys on push to `main`). See `DEPLOYMENT.md`,
> `HANDOFF-OTP-DEPLOY.md`, and the latest session handoff (`HANDOFF-2026-06-27.md`).

## Read these first

- **`BACKLOG.md`** — the prioritized product backlog (BL-1…BL-15). When asked to
  work on a backlog item, start from its entry there.
- **`DEVOPS-TODO.md`** / **`QA-TODO.md`** — ALL pending ops work and ALL deferred
  manual/e2e checks, consolidated (to be done at the end of dev). Add new
  deferred items to these files, not to handoff docs.
- **`README.md`** — stack, prerequisites, quick start.
- **`CONVENTIONS.md`** — backend module layout, response envelope, frontend design-system rules. **Follow it.**
- **`HANDOFF-2026-06-27.md`** — **latest session state** (mobile app, phone-OTP + Monty SMS live, CI/CD, NAT). Start here.
- **`app/HANDOFF-DESIGN.md`** — the Flutter mobile client: modules, what's stubbed, run/verify.
- **`DEPLOYMENT.md`** + **`HANDOFF-OTP-DEPLOY.md`** — Azure prod resources, redeploy recipes, OTP/SMS rollout.
- **`HANDOFF.md`** — earlier funnel state (Step 3: refunds, admin manual-completion, etc.).
- **`MANUAL-TEST.md`** / **`PURCHASE-FUNNEL.md`** — manual funnel test + assessment.

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
These are **local/dev only** — they live in your local Mongo and do **not** exist in prod.

> **Prod admin console** is a separate Azure Static Web App
> (`https://purple-bay-0d1a52d03.7.azurestaticapps.net`) backed by prod Cosmos, so the
> `.local` seed accounts fail there. Prod admin login is a **different** account
> (`admin@salehcard.com`, promoted from a real sign-up during deploy). Its password is in
> the gitignored **`DEPLOY-CREDS.local.md`** — never hardcode it here. (That file flags the
> password as chat-exposed → change it after login; see its rotation checklist.)

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
- **MongoDB has no multi-document transactions** (dev = local standalone Mongo; **prod = Azure
  Cosmos DB for MongoDB vCore**, `retrywrites=false`). Achieve consistency with **document-atomic
  `FindOneAndUpdate`** (e.g. code claim: `available→delivered`; wallet debit: `$inc` guarded by
  `$gte`) plus explicit compensation. Never select-then-update. Indexes are created at startup by
  each module's `EnsureIndexes` (sparse-unique `email`/`phone`, TTL on `otp_codes`/`refresh_tokens`).
- **Auth**: JWT access token in memory + rotating refresh token in an httpOnly cookie.
  `auth.AuthRequired` gates customer routes; `auth.AdminOnly` gates `/api/admin/*` and
  **dev-bypasses only when `ENV=development` AND `JWT_SECRET` is empty** (logs a warning).
  `ENV` defaults to **production** (fail-closed) and `config.Validate()` refuses to boot
  outside development without `JWT_SECRET` + `ALLOWED_ORIGINS`. `api/.env` sets
  `ENV=development` and ships a `JWT_SECRET`, so to bypass admin auth in dev run with
  `JWT_SECRET=` empty.
- **Phone-OTP auth** (`internal/modules/user`): `POST /auth/otp/request` + `/auth/otp/verify`
  (find-or-create by phone, optional password-set) + `/auth/login-phone`, alongside email/password.
  Users carry a sparse-unique `phone`; codes live in `otp_codes`. The mobile login offers
  **code or password**. Native clients send `X-Client: mobile` to get the refresh token in-body.
- **SMS provider layer** (`internal/platform/sms`, ports & adapters): a `Sender` port with
  `monty` / `twilio` / `log` adapters, selected by **`SMS_PROVIDER`** (default `log` → codes are
  logged, not sent — add a provider = new adapter file + one case in `New`). Prod uses **Monty**
  (Lebanon; sender `Arya`). Monty **IP-allowlists the caller**, so prod egresses via a NAT Gateway
  (one fixed IP) — see `HANDOFF-OTP-DEPLOY.md`. The **admin bulk-SMS broadcast** (Users → select →
  SMS, `POST /api/admin/users/bulk-sms`) reuses this same `Sender`, so in prod it sends **real,
  billed SMS on the live Monty provider** (not `log` like dev). Guardrails: `BULK_SMS_MAX` caps
  recipients (default **200** → `400 BULK_SMS_LIMIT`), a **160-char single-segment** limit (client +
  server), and an admin confirm dialog. Phoneless / `StatusDeleted` users are skipped; the send is a
  detached serial fan-out; audited as `user.bulk_sms`. The old `platform/email` package is now
  **dormant** (kept for future receipts; all `EMAIL_*`/`SMTP_*`/`SENDGRID_*` settings are unused).
- **Orders / payment / fulfillment** (implemented): server **re-prices from the catalog**
  (never trusts client price/fulfillment); `Idempotency-Key` header dedupes via a
  partial-unique index; order-first → atomic code claim → `completed`, with compensation
  (release codes + wallet refund) on failure; `account_credit`/`transfer` → `processing`,
  which admins complete via `PUT /api/admin/orders/{id}/status` or reverse via
  `POST /api/admin/orders/{id}/refund` (guarded `TransitionStatus` FindOneAndUpdate =
  the double-refund lock; **delivered codes are never re-pooled on refund**).
  **Payments are wallet-only** — card/usdt are rejected (`PAYMENT_METHOD_UNAVAILABLE`)
  until a real gateway exists; only the wallet ledger moves real balance.
- **KYC gates every purchase** (`kyc.Gate` checked first in `PlaceOrder` → `403
  KYC_REQUIRED`). KYC is **not part of signup**: users skip it, see a home-screen banner
  ("verify to purchase" → `/kyc`), and can also start it from the account menu; checkout
  shows a blocking verify panel until an admin approves the submission (`/api/admin/kyc`).
- **Wallet funding is an admin-approved request queue** (`topup_requests`): customers file
  amount + out-of-band channel (whish/omt/cash/usdt/other) via `POST /api/v1/wallet/topups`;
  admins approve (atomic pending-claim → credit → **mandatory** ledger row, with
  compensation) or reject at `/api/admin/wallet/topups`. There is NO instant top-up.
- **Admin audit log** (`audit` module, `admin_audit_log`): role/status changes, wallet &
  reseller adjustments, product deletes, order refunds/completions, KYC/review/top-up
  decisions, bulk-SMS broadcasts are recorded (actor from JWT) and browsable at `/audit` + `GET
  /api/admin/audit-log`. Guardrails: last-admin demote/suspend → `409 LAST_ADMIN`;
  admin money adjustments reverse the balance if the ledger insert fails.
- **CORS gotcha**: any custom request header must be in the server's CORS `AllowedHeaders`
  (`internal/server/server.go`) or the browser preflight blocks it — `Idempotency-Key` is
  there for this reason. curl won't catch this (no preflight); test in a real browser.
- **Frontend design system is CSS-variable based** (`web/src/styles/`, `admin/src/styles/`),
  **NOT** Tailwind `dark:` utilities. Components emit ported class names (`.abtn .acard
  .tbl .bdg .st-* .ff-*` etc.). Theme via `[data-theme]`; locale forced to English on load.
- **Frontend data**: catalog + orders + wallet are wired to the API via React-Query hooks
  + `adapt*` mappers (`features/*/lib/adapt*.ts`). **The `/admin` console is fully
  API-wired** (no mock data file; the old `admin/src/lib/mock/demo.ts` is deleted) —
  see `admin/BACKLOG.md` for the honest wired-vs-missing split (the settings module
  backend is still a 501 stub; only its admin-accounts list is real). `/web` still has
  `web/src/lib/mock/demo.ts` remnants behind `// TODO` markers.
- **Customer routes guarded** by `RequireAuth` in `web/src/app/router.tsx`: `/dashboard`,
  `/wallet`, `/orders`, `/orders/:id`, `/checkout`, `/order-success/:id`.

## Git / workflow

- Default branch is `main`. Commit/push only when asked.
- End commit messages with the `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>` trailer.
- **CI/CD**: `.github/workflows/ci.yml` (build/lint/test on PRs — note golangci-lint runs, stricter
  than `go vet`) + `deploy.yml` (**auto-deploys to Azure on push to `main`**, path-filtered per
  `api`/`web`/`admin`, via OIDC — no stored secrets). Merging a PR to `main` ships prod.
- **GitHub repo is `AliSleiman0/SalehCard`** (capital S/C — renamed from lowercase; `origin` still
  redirects). OIDC subject matching is **case-sensitive** — keep Azure federated creds on `SalehCard`.
- **Corporate-proxy gotcha**: `az` fails TLS locally; run Azure CLI in **Azure Cloud Shell**
  (browser, pre-auth, no proxy). `git`/`gh`/`docker`/Go trust the Windows cert store and work.
