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
- **Admin SMS 2FA** (`internal/modules/user`): a **setting-gated** second factor for admin logins.
  When the `app_settings` flag **`adminSmsTwoFactorEnabled`** is on (toggled from the admin console
  Settings → Security; **off by default** so the phoneless dev seed admin still logs in),
  `Login`/`LoginByPhone` divert admins (only `role==admin`; customers/mobile untouched) into
  `begin2FAChallenge` instead of `issueTokens`: it stores an OTP in `otp_codes`, sends it via the
  same `sms.Sender`, and returns a **pending challenge** (a short-lived `Issue2FAToken` JWT +
  masked `phoneHint`, no tokens/cookie). The client completes it at **`POST /auth/2fa/verify`**
  `{pendingToken, code}` (→ tokens), or **`/auth/2fa/resend`** (throttled). **Fails closed**: an
  admin with no phone gets `403 ADMIN_2FA_NO_PHONE`; the Settings toggle likewise refuses to enable
  2FA unless the acting admin's JWT carries a phone (so re-login after setting a phone). In prod
  this sends **real, billed Monty SMS**. Prod admin `admin@salehcard.com` phone set to `+961 78991778`.
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
- **Per-product input fields** (`Product.InputFields` — key/label/type/sensitive): the fields a
  customer enters at checkout for non-`code` products (Account ID, Zone ID, Email, Server…). Admins
  define them in the product editor (add/remove, editable key + type + label). Captured values are
  stored **structured + labeled** on the order — `OrderItem.Fields []{key,label,value}`, labels
  resolved server-side from the product spec, `Sensitive` fields never persisted (spec §3.2) — so the
  operator fulfilling an `account_credit`/manual order sees them; `playerId` snapshots the first field
  (feeds `Fulfillment.CreditedToID`).
- **Game-ID verification** (`internal/platform/idcheck`, ports & adapters like `sms`): a `Verifier`
  port with `rapidapi` (RapidAPI "ID Game Checker") + `stub` adapters, selected by **`IDCHECK_PROVIDER`**
  (default `stub`; prod needs **`RAPIDAPI_KEY`**, server-side only — never in the app/admin bundle). A
  product opts in via `Product.Verification{provider, app}` (the `check_name` hook; `app` = the game
  slug — `pubgm-global` / `dfm-garena` / `free-fire`). The app calls **`POST
  /api/v1/products/{id}/verify-account`** `{playerId}` (AuthRequired + per-IP rate-limited — each call
  is billed) → resolved nickname; the product page shows it and unlocks Buy Now only on a hit.
  **Fail-open**: a positively-bad ID blocks, but a timeout/outage returns `unavailable` so a
  third-party hiccup never halts the sale; the playerId is never logged. Only single-ID games verify —
  multi-param games (Mobile Legends id+zone, Genshin id+server) are collect-only.
- **Product reviews** (`internal/modules/review`): a logged-in customer posts one editable 1–5★
  rating + text per product — `POST /api/v1/reviews`, `GET /api/v1/reviews/mine`, public list at
  `GET /api/v1/products/{id}/reviews`; the app has a star-input + write-review sheet. Admins moderate
  under `/api/admin/reviews` (decisions audited).
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
  see `admin/BACKLOG.md` for the honest wired-vs-missing split. **The settings module
  backend is real** (GET/PUT `/api/admin/settings` over the `app_settings` singleton —
  store config, loyalty, admin SMS-2FA); the Settings page also lists admins from
  `/api/admin/users?role=admin`. `/web` still has
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
