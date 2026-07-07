# HANDOFF — Implement Whish Pay in SalehCard

**Created:** 2026-07-06 · **Status:** not started (planning inputs only)

Goal for the next session: add **Whish Pay** as a real payment method in SalehCard
(Lebanese wallet payment, redirect-hosted checkout), the way LACPA already did it.
This doc is self-contained — you should not need to re-read the PDF or re-explore
LACPA to start, but both references are pinned below.

---

## 0. TL;DR of the two references

- **Whish spec (the protocol):** `C:\Users\user\Downloads\WHISH PAY Web Service - Technical Specification - v1.4.2.pdf`
  — REST/JSON over HTTPS. The endpoints + payloads you need are transcribed in §3
  so you don't have to open it, but keep the PDF for the fine print (error codes,
  dialog fields).
- **LACPA implementation (the blueprint):** `C:\Users\user\Lacpa\Backend\payments\`
  — a sibling Go + Mongo project (module `github.com/AliSleiman0/Lacpa`). Whish is a
  drop-in provider behind its hexagonal payments module. **This is the code to port.**
  Exact files in §4.

> ⚠️ Whish is **fiat, redirect + callback** — architecturally different from the
> **on-chain, watcher-poll** USDT flow already merged in SalehCard (PR #51,
> `api/internal/modules/payment`). Read §2 before writing code: the key decision is
> whether to extend that module or run Whish alongside it.

---

## 1. What already exists in SalehCard (don't rebuild it)

PR #51 (`feat/usdt-onchain-payments`) added an async payment module. Whish should
reuse as much of it as fits:

- **`api/internal/modules/payment/`** — `Intent` aggregate + state machine
  (`pending → confirming → confirmed / expired`, `CanTransition` guard), Mongo
  `Store` (collection `payment_intents`), `Service` (idempotent create + settlement
  with wallet-compensation), event-free settlement that calls into wallet/order via
  narrow ports (`topUpCrediter`, `OrderSettler`).
- **`wallet.WalletService.TopUp(userID, amount, method, ref)`** — credit + ledger
  with a unique partial `(method, ref)` index that makes settlement retries a safe
  no-op. Whish settlement should credit with `method: "whish"` and `ref: intentID`.
- **Order integration** — `order.FulfillPaidOrder` / `FailUnpaidOrder` already
  implement `payment.OrderSettler`; a Whish order can reuse them unchanged.
- **`platform/tron`** is the *provider* precedent (port + adapters + `New(Config)`
  switch), but its `Reader.FindPayment` port is **poll-based** and does NOT fit
  Whish. Whish needs a different provider shape (see §2).
- **What is NOT there yet and Whish needs:** any **inbound webhook/callback route**
  (grep `webhook`/`callback` in `api/` → nothing), and any **redirect-URL** concept
  on the Intent. Both are net-new for SalehCard (the USDT flow had neither).

Conventions to follow (all already in the repo — see `CONVENTIONS.md`, `CLAUDE.md`):
- Modules split model/repository/service/handler/routes; deps wired explicitly in
  `server.go`; no multi-doc transactions → document-atomic `FindOneAndUpdate` +
  compensation. Money in the payment module is **int64 micro-USDT**; convert at the
  wallet boundary and when talking to Whish (Whish wants a float amount).
- Public (unauthenticated) routes are mounted outside the `/api/admin` group and
  without `auth.AuthRequired` (e.g. `/health`, `/api/v1/auth`). The webhook route is
  public — its HMAC token in the query string is the auth (see §5).
- Any custom request header must be in the CORS `AllowedHeaders` in `server.go`
  (the USDT PR did NOT need this; Whish's server-to-server callbacks aren't
  browser-preflighted, so likely N/A — but the browser redirect return page might).

---

## 2. THE architectural decision (resolve first)

SalehCard's `payment` module was built for **on-chain**: `Service` has a `Reader`
(poll the chain) and a `Watcher` (background loop). Whish is **redirect + callback**:

| | USDT (built) | Whish (to build) |
|---|---|---|
| Start | derive address, show QR | `POST /payment/whish` → `collectUrl`, **redirect the browser** |
| Confirm | watcher polls TronGrid | Whish fires an **unsigned GET callback**; you re-poll `GET /payment/collect/status` for ground truth |
| Intent needs | `address` | **`redirectUrl`** (collectUrl) |
| Server needs | watcher loop | **public callback route** + HMAC token |

**Recommended approach:** generalize `modules/payment` to host *both* provider kinds
rather than a parallel module.
- Add a provider **port** with `Initiate(intent) → {redirectURL, providerRef}` and
  `GetStatus(intent) → {success|failed|pending, payerPhone}` (this is exactly
  LACPA's `providers.Provider` interface — port it near-verbatim). The USDT path
  keeps its `Reader`; Whish implements this new port.
- Add `RedirectURL string` (+ optional `ProviderRef`) to the `Intent` model.
- Add a `HandleCallback` service method (port LACPA `service.go` `HandleCallback`):
  verify the HMAC token → look up the intent by `externalId` → re-poll
  `GetStatus` as source of truth → transition + settle (reuse the existing
  `settle` → wallet/order paths). Idempotent re-acks for already-terminal intents.
- The watcher stays USDT-only; for Whish add an **expiry sweep** (Whish intents that
  never get a callback should expire) — LACPA has `BulkMarkExpired` for this.

**Confirm with the user (same questions as USDT):**
1. Does Whish settle **wallet top-ups**, **order checkout**, or **both**?
   (USDT chose both. The existing settler ports support both.)
2. Sandbox first (Whish sandbox base URL) before requesting SalehCard's own prod
   merchant credentials? (Almost certainly yes.)

---

## 3. Whish API reference (transcribed from the v1.4.2 spec)

**Environments**
- Production: `https://api.whish.money/itel-service/api`
- Sandbox:    `https://api.sandbox.whish.money/itel-service/api`

**Required headers on every request**
| Header | Value |
|---|---|
| `channel` | provided by Whish (numeric) |
| `secret` | provided by Whish (hex) |
| `websiteUrl` | your site/app name (e.g. `salehcard.com`) |
| `User-Agent` | `Whish/1.0 (https://whish.money; support@whish.money)` |
| `Content-Type` | `application/json` |

**Unified response envelope:** `{ status: bool, code: string|null, dialog: {...}|null, data: {...} }`
(`code` is an operation-specific failure code; `dialog` is optional user-facing copy.)

**Post Payment** — `POST /payment/whish`
Request body:
```json
{
  "amount": 1,                // Double, in `currency`
  "currency": "USD",          // "USD" | "LBP"
  "invoice": "Order #1",      // description
  "externalId": 1,            // Long, UNIQUE per request, your reference id
  "successCallbackUrl": "https://<you>/api/v1/payments/webhooks/whish/success?...",
  "failureCallbackUrl": "https://<you>/api/v1/payments/webhooks/whish/failure?...",
  "successRedirectUrl": "https://<you>/payments/success?...",
  "failureRedirectUrl": "https://<you>/payments/failure?..."
}
```
Response: `data.collectUrl` — the hosted payment page URL to redirect the user to.
- Callback URLs = server-to-server GET Whish fires with the result.
- Redirect URLs = where the user's browser lands after paying.
- You may append any query params (e.g. your token); Whish echoes them back unchanged.

**Get Status** — `POST /payment/collect/status`
Request: `{ "currency": "USD", "externalId": 1 }`
Response: `data.collectStatus` ∈ `success | failed | pending`, and
`data.payerPhoneNumber` (Long; decode leniently — LACPA handles string OR number).

**Get Balance** — `GET /payment/account/balance?currency=USD` (or `LBP`) → `data.balance` (Double).

**Sandbox test cases**
- Success: on the payment page use phone `96170902894` with OTP `111111`.
- Failure: any phone with an OTP other than `111111`.
- **No OTP is delivered in sandbox** (use the fixed one above).

---

## 4. LACPA files to port (the blueprint)

Under `C:\Users\user\Lacpa\Backend\payments\`:

- **`providers/whish/adapter.go`** — implements the `Provider` port over the client;
  maps `Initiate`/`GetStatus`. **Port almost verbatim.**
- **`providers/whish/client.go`** — thin HTTP wrapper: `Initiate` (`POST /payment/whish`),
  `CollectStatus` (`POST /payment/collect/status`), sets the 4 headers. **Port verbatim.**
- **`providers/whish/config.go`** — env loading (`WHISH_CHANNEL/SECRET/WEBSITE_URL/BASE_URL/USER_AGENT`).
- **`providers/whish/types.go`** — request/response DTOs (note `payerPhoneNumber` is `any`, decoded leniently).
- **`providers/provider.go`** — the `Provider` **port** (`Name`, `Initiate`, `GetStatus`) +
  `InitiateInput`/`InitiateResult`/`StatusResult`/`CollectStatus`. Adapt to SalehCard's `Intent`.
- **`service.go`** — study `CreateIntentDetailed` (idempotent create → provider Initiate →
  persist redirect URL → pending) and **`HandleCallback`** (verify token → re-poll status
  as ground truth → `CanTransition` → settle → idempotent re-ack). This is the core to adapt.
- **`token.go`** — HMAC-SHA256 tokens appended to the unsigned Whish callbacks
  (`SignExternalID`/`VerifyExternalID`, constant-time compare). **Port verbatim** —
  SalehCard has no equivalent yet.
- **`ports/http.go`** — shows the webhook routes MUST be registered **before** the authed
  group (Fiber prefix-middleware gotcha; chi is explicit but the ordering lesson stands):
  `GET /api/payments/webhooks/whish/success` + `/failure`.
- **`store/mongo.go`** — `GetByExternalID`, `UpdateAfterInitiate`, `UpdateTerminal`,
  `BulkMarkExpired` (the expiry sweep). SalehCard's store has analogues; add
  `external_id` + `redirect_url` handling.
- **`subscribers/`** — LACPA fans confirmed payments to side effects via events;
  SalehCard settles synchronously in `Service.settle` instead — credit the wallet /
  fulfill the order there, no pub/sub needed.

LACPA env reference (`Backend/payments/.env.example`): `WHISH_BASE_URL`, `WHISH_CHANNEL`,
`WHISH_SECRET`, `WHISH_WEBSITE_URL`, `WHISH_USER_AGENT`, `PAYMENTS_WEBHOOK_BASE_URL`
(public host Whish can reach — in local dev use an ngrok/cloudflared tunnel; Whish
sandbox cannot reach localhost), `PAYMENTS_HMAC_SECRET` (`openssl rand -hex 32`),
`PAYMENTS_SUCCESS_REDIRECT_URL`, `PAYMENTS_FAILURE_REDIRECT_URL`.

---

## 5. Gotchas carried from LACPA (will bite otherwise)

- **`externalId` must be a unique int64** per request. LACPA: `time.UnixMilli() << 16 |
  16-bit random jitter`. SalehCard intents use ObjectID — add an `externalId` field for Whish.
- **Callbacks are UNSIGNED.** Append `?externalId=<id>&token=<HMAC(externalId)>` to the
  callback URLs and verify the token on receipt. That token *is* the authentication for
  the public webhook route.
- **Re-poll is the source of truth.** The browser can land on the *failure* redirect even
  when the payment actually succeeded — always call `GET /payment/collect/status` in the
  callback handler and trust that, not which URL fired.
- **Webhook route is public** — mount it outside the auth group. In local dev, Whish
  sandbox can't reach `localhost` → run a tunnel and set `PAYMENTS_WEBHOOK_BASE_URL` to it.
- **Currency:** Whish supports USD + LBP; SalehCard is USD-only → always send `"USD"`.
- **Idempotency:** re-emit/settle on gateway re-delivery of an already-terminal intent
  (subscribers/settlement must be idempotent — the unique `(method, ref)` wallet index
  already gives you this).

---

## 6. Credentials the user shared (READ CAREFULLY)

```
channel:    10200046
secret:     9bdaee6ad2b7401a903d69b09a04e7b0
websiteUrl: lacpa.academy
```
These are **LACPA's** Whish merchant credentials (note `websiteUrl: lacpa.academy`), given
for **reference / sandbox testing only**. **SalehCard must obtain its OWN Whish merchant
account + channel/secret before prod.** Do not ship LACPA's creds. They are also
**chat-exposed** → flag to the user that LACPA should rotate them. Never hardcode any of
these — load from env (`WHISH_*`), like every other secret; put prod values in Azure app
settings and the gitignored creds file, not in the repo.

---

## 7. Suggested plan for the session

1. Confirm settlement scope with the user (§2) + sandbox-first.
2. Add the `Provider` port + `whish` adapter/client/config/types (port from LACPA §4).
3. Extend `Intent` (`externalId`, `redirectUrl`, `providerRef`) + store methods.
4. Add `Tokens` (HMAC) + `Service.CreateWhishIntent` + `Service.HandleCallback` +
   the public webhook routes + the browser redirect landing pages (app + `/web`? — app only, per USDT scope).
5. Client: a "Pay with Whish" option that opens `collectUrl` (in-app webview or external
   browser) and a return/poll screen (mirror the USDT deposit screen's poll-to-terminal UX).
6. Config: `WHISH_*` + `PAYMENTS_WEBHOOK_BASE_URL` + `PAYMENTS_HMAC_SECRET`; `Validate`
   requires them in prod when Whish is enabled; feature dark until configured.
7. Admin: extend the payments page to show Whish intents (method/provider column).
8. Docs: add prod-rollout items to `DEVOPS-TODO.md` and the sandbox/redirect test matrix
   to `QA-TODO.md`.

**Verify:** `cd api && go build ./... && go vet ./... && go test ./...`; sandbox e2e with
the fixed test phone/OTP through a tunnel; then real merchant creds. Follow PR #51's
pattern (feature branch, per-stack commits, stub/sandbox e2e before prod).

---

## 8. Context pointers

- SalehCard is Lebanon-based (Beirut TZ, Monty SMS) — Whish is a natural local rail,
  same audience as the existing manual `whish` **top-up channel** (which this would
  automate — see `wallet/model.go` `TopUpChannels`).
- Prior art / state: PR #51 (USDT) is the template for everything here. Read its diff.
  The USDT feature is dark until `USDT_XPUB` is set; Whish should be dark until `WHISH_*` is set.
- Related memory: `salehcard-usdt-onchain-payments`, `salehcard-phone-otp-auth` (Monty SMS
  / Lebanon egress-IP allowlist — Whish may have a similar caller-IP allowlist; check).
