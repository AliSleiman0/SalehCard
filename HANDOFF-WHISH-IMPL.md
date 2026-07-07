# HANDOFF — Whish Pay implementation (code-complete, awaiting Whish test API)

**Created:** 2026-07-06 · **Status:** ✅ implemented & verified in **stub mode**;
⏳ **blocked** on receiving Whish's **test/sandbox API credentials** to run the
live sandbox e2e. **Not committed** (working tree on `feat/usdt-onchain-payments`).

> Supersedes the planning doc `HANDOFF-WHISH.md` (kept for reference: it has the
> transcribed Whish v1.4.2 API spec in §3 and the LACPA gotchas in §5). This doc
> is the resume point.

---

## 1. TL;DR

Whish Pay (Lebanese wallet, **redirect-hosted checkout + server callback**) is
built end-to-end across backend, Flutter app, and admin, mirroring the existing
USDT flow. It settles **both** wallet top-ups and order checkout, opens the
hosted page in the **external browser**, and is **dark until configured**
(`whishEnabled:false` when creds absent). Everything works against the **stub**
provider (fake `collectUrl`, auto-confirm on re-poll). The **only** thing left is
to point it at the real **Whish sandbox** once we have the test `channel`/`secret`
and run the sandbox test matrix through a public tunnel — then request our own
**prod** merchant account.

---

## 2. What's built (all reusing the `payment` module + settlement path)

**Backend (`api/`)**
- **`internal/platform/whish/`** — new ports-and-adapters package (like
  `platform/tron`): `Provider` port (`Initiate`/`GetStatus`), real HTTP adapter
  (`client.go`/`adapter.go`, ported from LACPA), dev `stub.go`, `New(Config)`
  switch, `config.go`/`types.go`.
- **`internal/modules/payment/`**
  - `token.go` — HMAC-SHA256 callback tokens (`SignExternalID`/`VerifyExternalID`,
    constant-time). The token in the callback query string is the auth for the
    public webhook.
  - `model.go` — `Intent` gained `provider` / `externalId` / `redirectUrl` /
    `providerRef` / `payerPhone`; new `failed` terminal state; `CanTransition`
    extended; `ProviderOf()` (legacy empty ⇒ usdt).
  - `repository.go` — `GetByExternalID`, `UpdateAfterInitiate`,
    `ClaimWhishConfirmed`, `MarkFailed`, `ListWhishExpiryCandidates`, + unique
    partial `(provider, externalId)` index.
  - `service.go` — `CreateWhish{TopUp,Order}Intent` (skip HD derivation → set
    externalId → `Initiate` → persist `redirectUrl`; on Initiate error `MarkFailed`
    so the idem key isn't stuck), **`HandleCallback`** (verify token → re-poll
    `GetStatus` as ground truth → settle idempotently, resume `confirming`, no-op
    on terminal re-delivery), `SweepWhishExpired` (re-poll-before-expire),
    provider-aware wallet method (`walletMethod`).
  - `handler.go`/`routes.go` — `POST /api/v1/payments/whish/topup-intents`
    (authed) + **public** `GET /api/v1/payments/webhooks/whish/{success,failure}`
    (mounted outside the auth group), `whishEnabled` in the config view, Whish
    fields on `IntentView`.
  - `sweeper.go` — `WhishSweeper` (own ticker; runs when Whish enabled, since the
    USDT `Watcher` only runs when USDT is on).
  - `admin.go` — provider / redirectUrl / payerPhone on the admin view.
- **`internal/modules/wallet/repository.go`** — `(method,ref)` unique-partial
  index **auto-migrated** from `usdt_trc20`-only to `{usdt_trc20, whish}` (drops
  stale `method_1_ref_1` + recreates; Mongo/Cosmos won't rewrite it in place).
- **`internal/modules/order/`** — `PaymentMethodWhish` (validated `WhishEnabled()`),
  post-persist Whish-intent branch, `usdtIntents` port extended with
  `WhishEnabled()` + `CreateWhishOrderIntent`; `FulfillPaidOrder`/`FailUnpaidOrder`
  reused unchanged.
- **`internal/config/config.go`** — `WHISH_*` + `PAYMENTS_*` fields + `Validate`
  (refuses stub+creds in prod; requires callback config with the real provider).
- **`internal/server/server.go`** + `cmd/server/main.go` — build Whish provider +
  tokens, `SetWhish`, start the sweeper.

**Flutter app (`app/`)** — `url_launcher` added; `WhishRedirectScreen` (opens
`redirectUrl` externally, reuses the USDT poll-to-terminal UX);
`/payments/whish-redirect` route; `createWhishTopUpIntentController`; top-up
(`whish` channel branch) + checkout (`whish` pay row) entry points; entity/DTO/
config extended (`provider`, `redirectUrl`, `payerPhone`, `failed`, `whishEnabled`).

**Admin (`admin/`)** — provider badge, `failed` filter, Whish rows show payer
phone (tronscan links are USDT-only), API types extended.

**Docs** — `api/.env.example` (WHISH_*/PAYMENTS_*), `DEVOPS-TODO.md` §15 (prod
rollout), `QA-TODO.md` (sandbox test matrix), plan file at
`~/.claude/plans/salehcard-handoff-whish-md-read-it-and-prancy-cat.md`.

---

## 3. Verified now vs. pending

**✅ Verified (stub mode, no external dependency)**
- `cd api && go build ./... && go vet ./... && go test ./...` — green, incl. 8 new
  Whish tests: token sign/verify, callback success-credits-**once**, replay no-op,
  failure-fails-order, pending no-op, bad-token→400, initiate-failure→failed.
- Live boot smoke test: server starts with `payment: Whish redirect payments
  enabled provider=stub`, the sweeper starts, the **index migration runs clean**
  (no `wallet: failed to ensure indexes` warning), the public webhook route is
  wired and token-authed (missing/bad token → 400), `/health` → 200.
- `flutter analyze` clean; `admin pnpm build` clean.

**⏳ Pending (needs the Whish test API creds)** — the live **sandbox** e2e through
a public tunnel. See the full matrix in `QA-TODO.md` → "Whish Pay — sandbox/
redirect test matrix". The sandbox delivers no OTP; use the fixed test values:
success phone `96170902894` / OTP `111111`, any other OTP = failure.

---

## 4. RESUME when the Whish test API arrives

1. **Get the sandbox creds** from Whish → `channel`, `secret`, `websiteUrl`.
   (The LACPA reference creds in `HANDOFF-WHISH.md` §6 are **not ours** — sandbox
   testing only, and chat-exposed → ask LACPA to rotate. Prefer our own sandbox
   creds if Whish issues them.)
2. **Start a public tunnel** (Whish sandbox can't reach localhost):
   `ngrok http 8090` (or cloudflared) → note the https URL.
3. **Set dev env** (`api/.env`, gitignored — copy the block from `.env.example`):
   ```
   WHISH_PROVIDER=whish
   WHISH_BASE_URL=https://api.sandbox.whish.money/itel-service/api
   WHISH_CHANNEL=<sandbox channel>
   WHISH_SECRET=<sandbox secret>
   WHISH_WEBSITE_URL=salehcard.com
   PAYMENTS_WEBHOOK_BASE_URL=<the ngrok https URL>
   PAYMENTS_HMAC_SECRET=<openssl rand -hex 32>
   WHISH_INTENT_EXPIRY=2m        # short, so the abandon→expiry case is quick to test
   ```
   Keep `ENV=development` (otherwise `Validate` requires JWT/CORS etc.).
4. **Run the app against the local API** (see the `salehcard-flutter-emulator-run`
   memory), top up via the **Whish** channel → the hosted page opens in the
   browser → pay with `96170902894` / `111111` → return → the screen polls to
   confirmed, wallet updates.
5. **Walk the `QA-TODO.md` Whish matrix**: success top-up + order, failure (wrong
   OTP), abandon→expiry, browser-lands-on-failure-but-paid (the re-poll guard),
   callback idempotency (replay → credited once), bad token → 400, rate limits,
   manual-vs-auto whish channel.
6. **Then prod**: follow `DEVOPS-TODO.md` §15 — obtain SalehCard's **own** Whish
   **merchant** account, set Azure app settings (`WHISH_PROVIDER=whish`, prod
   `WHISH_BASE_URL`, our merchant creds, `PAYMENTS_WEBHOOK_BASE_URL=<prod API>`,
   a prod `PAYMENTS_HMAC_SECRET`), confirm Always On, and **check whether Whish
   IP-allowlists the caller** (like Monty) — if so register the NAT Gateway IP.

---

## 5. Open risks / decisions to carry
- **Whish caller-IP allowlist?** Unconfirmed. If Whish allowlists the merchant's
  calling IP (Monty does), `Initiate`/`GetStatus` will fail from prod until the NAT
  Gateway IP is registered. Confirm with Whish before go-live.
- **LACPA creds are chat-exposed** → ask LACPA to rotate `channel 10200046` + its
  secret. Never ship them; SalehCard needs its own merchant account for prod.
- **Lost-callback-but-paid** race is handled: the expiry sweep re-polls `GetStatus`
  before expiring, so a paid intent whose callback was lost still settles.

---

## 6. Git state
- Branch: `feat/usdt-onchain-payments`. **Uncommitted** — the Whish work sits on
  top of the (also uncommitted) USDT work. When ready, commit per-stack
  (platform/whish → payment module → wallet/config/order/server → app → admin →
  docs), or split onto a fresh `feat/whish-payments` branch. Feature stays dark
  until `WHISH_*` is configured, so it's safe to merge before the sandbox run.
- New files: `api/internal/platform/whish/*.go`,
  `api/internal/modules/payment/{token,sweeper,whish_test}.go`,
  `app/lib/features/payments/presentation/screens/whish_redirect_screen.dart`,
  this doc.
