# DevOps TODO — do at the end of dev

Single collection point for all pending ops/infra work. Supersedes the scattered
"deferred" notes in the handoff files. Items are ordered by impact; each says
where it came from and how to verify it. Prod context: App Service
**salehcard-api** in RG **salehcard-prod**, sub `1adb4811-6234-4822-b7f2-8411ed2cb999`.

> **Corporate-proxy reminder:** local `az` cannot mint tokens (Python TLS rejects
> the proxy CA). Use the **Azure portal** or **Cloud Shell** for everything below
> that touches Azure. See `DEPLOYMENT.md`.

## 1. FCM push app settings (BL-11) ✅ DONE 2026-07-05

`PUSH_PROVIDER=fcm` + `FCM_CREDENTIALS_JSON` applied to `salehcard-api` via local
`az` (Norton is gone — local CLI works again; ignore the Cloud Shell detour notes).
Boot log captured via live `az webapp log tail` across a restart: **no**
`push provider misconfigured — falling back to log sender` warning → the FCM adapter
constructed successfully. Remaining soft-verify: trigger any business event and
confirm a tray push actually arrives on a device (needs an app build with
`firebase_options.dart`).

## 2. `BUSINESS_TZ=Asia/Beirut` app setting (BL-10) ✅ DONE 2026-07-05

Applied in the same settings batch as item 1 (one restart). Dashboard "today"
KPIs now bucket on Beirut time.

## 3. Restrict the Firebase Android API key

`app/lib/firebase_options.dart` commits the Android client key
(`AIzaSy…AuUsQ`) — an identifier, not a secret, but it should be locked to the
app. Google Cloud console (project `salehcard-app`) → APIs & Services →
Credentials → the auto-created Android key → Application restrictions →
**Android apps** → add package `com.salehcard.salehcard_app` + the release
signing SHA-1 (and debug SHA-1 for dev builds). This also moots the
GitGuardian flag on PR #27.

## 4. Service-account key custody ✅ DONE 2026-07-05

Both key files moved out of Downloads to `C:\Users\user\.salehcard-secrets\`
(value now also lives in the Azure app settings). Never commit either file.
Rotate the key if it ever leaks (generate new in Firebase console → Project
settings → Service accounts → update `FCM_CREDENTIALS_JSON` → delete old key).

## 5. Verify Cosmos picked up the new indexes (BL-11)

`notification.EnsureIndexes` creates `notifications` indexes incl. a **180-day
TTL on `createdAt`**, and unique `device_tokens.token`. Cosmos vCore TTL
support should be confirmed: check App Service Log Stream at startup for any
`EnsureIndexes` slog warnings; if TTL fails, notifications simply never expire
(harmless short-term, needs a cleanup job long-term).

## 6. Flutter app release/distribution path

The mobile app has **no CI/CD** — api/web/admin deploy on push to `main`, the
app ships manually. Before launch decide + set up:
- Release signing config (`app/android` keystore — none exists yet; debug-signed only).
- Distribution channel: Play Console (needs account + review) vs direct APK.
- Note: push notifications require the shipped app build to include the
  BL-11 code + real `firebase_options.dart` (already on `main`).

## 7. CI workflow action bumps (low priority) ✅ DONE 2026-07-05 (pending commit)

`actions/checkout` v4→v5 and `dorny/paths-filter` v3→v4 bumped in `ci.yml` +
`deploy.yml` (scoped to exactly the two actions the deprecation annotations
named). Sitting uncommitted on `main` — ships with the next push.

---

# BL-15 (shipped 2026-07-04, PRs #30/#34/#32/#33) — new ops items

All BL-15 config defaults are **launch-safe** (nothing *must* be set), but two
behaviors change on this deploy — read items 8 and 11 first.

## 8. ⚠️ Rate limiting is ON by default in prod (BL-15 P1)

`RATE_LIMIT_PROVIDER` defaults to **`mongo`**, so per-IP auth rate limiting is
**active immediately** after this deploy: 30 auth requests/min/IP and 5 OTP
requests/min/IP (the OTP cap protects Monty SMS spend). Returns `429
RATE_LIMITED`. A new `rate_counters` collection (TTL-indexed) is auto-created.

- **Verify the client IP resolves correctly.** The limiter keys on
  `middleware.RealIP` (X-Forwarded-For). If App Service / a CDN fronts the API
  and the true client IP is **not** propagated, all traffic can share one IP and
  hit a **global** throttle. Confirm real client IPs in Log Stream, or the app
  will appear to rate-limit everyone at once.
- **Tune / disable** via app settings: `RATE_LIMIT_AUTH_MAX`,
  `RATE_LIMIT_AUTH_WINDOW`, `RATE_LIMIT_OTP_MAX`, `RATE_LIMIT_OTP_WINDOW`; set
  `RATE_LIMIT_PROVIDER=noop` to turn it off.

## 9. Bulk-SMS — sends real, paid SMS in prod on deploy (BL-15 DevOps)

⚠️ The admin **bulk-SMS** broadcast (Users → select → SMS) reuses the **live Monty
provider** already configured for OTP, so unlike the old bulk-email (which was
`log`-only in prod), it **sends real, billed SMS the moment this ships**. Lebanon
SMS is charged per 160-char segment.

Guardrails (all on by default, no config needed):
- **`BULK_SMS_MAX`** — hard cap on recipients per send (default **200**); over it →
  `400 BULK_SMS_LIMIT`. Tune via an app setting (portal) if a larger blast is needed.
- Admin **confirm dialog** shows the recipient count before firing.
- **160-char** single-segment cap (enforced client + server).

The old `EMAIL_*` settings (`EMAIL_PROVIDER` / `SMTP_*` / `SENDGRID_*` / `EMAIL_FROM`)
are now **unused** — the `platform/email` package is dormant (kept for a future
"email me my receipt" need). Leave them unset.

## 10. Payment gateway + fulfillment frameworks — keep OFF in prod (BL-15 P4)

Both ship as hexagonal frameworks with a mock/reference adapter; the real
integrations are not built yet, so **leave the prod defaults as-is**:
- `PAYMENT_PROVIDER` defaults to `log` → checkout stays **wallet-only** (correct
  for launch). Only `mock` enables the sandbox card/usdt path — **do not set in
  prod**. A real gateway = a new adapter file + `PAYMENT_PROVIDER=<name>` + keys.
- `FULFILLMENT_MOCK` defaults **off** → api-mode orders keep parking for manual
  completion. Only `1` (with `FULFILLMENT_MOCK_ID`) enables the reference adapter.
- New additive `Order.paymentRef` field — no migration needed.

## 11. Confirm new Cosmos collections/indexes came up (BL-15)

`EnsureIndexes` auto-creates on this deploy: `app_settings` (settings), 
`reseller_prices` (unique `{userId, variantId}`), `rate_counters` (TTL on
`expiresAt`), and a new `codes.orderId` index. Check App Service Log Stream at
startup for any `EnsureIndexes` slog warnings (Cosmos vCore TTL support — same
caveat as item 5). All are additive; no data migration.

## 12. Admin SMS 2FA rollout (setting-gated)

The admin second factor reuses the OTP + Monty SMS path, so **prod sends real,
billed SMS** on the live Monty provider (dev with `SMS_PROVIDER=log` only logs).
Egress must stay on the NAT-gateway fixed IP allowlisted by Monty (same
dependency as OTP/bulk-SMS — see `HANDOFF-OTP-DEPLOY.md`). Rollout:
- Ship via PR → merge to `main` (auto-deploys). No new env var — the on/off flag
  is the `app_settings.adminSmsTwoFactorEnabled` boolean, toggled from the admin
  console (Settings → Security), **off by default**.
- Before enabling: confirm the prod admin (`admin@salehcard.com`) has a phone
  (`+961 78991778`). Enabling **fails closed** — any admin without a phone is
  blocked from login (`ADMIN_2FA_NO_PHONE`), and the toggle itself refuses to turn
  on unless the acting admin's JWT carries a phone (re-login after setting one).
- `app_settings` already exists (settings singleton); this is an additive boolean,
  no migration.

## 13. Product image storage — provision Azure Blob ✅ DONE 2026-07-05

Storage account `salehcardassets` (Standard_LRS, westeurope) + public-read
container `product-images` created; `STORAGE_PROVIDER=azure` +
`AZURE_STORAGE_CONNECTION_STRING` + `AZURE_STORAGE_CONTAINER` applied to
`salehcard-api`. Verified: boot log shows **no** `storage provider misconfigured`
warning, and a smoke-test blob uploaded + fetched anonymously (HTTP 200) + deleted.
Connection string recorded in `DEPLOY-CREDS.local.md`. Gotcha for posterity: the
`Microsoft.Storage` resource provider was NotRegistered on the subscription
(first-ever storage account) — `az provider register -n Microsoft.Storage` fixed
the misleading `SubscriptionNotFound` error. Remaining e2e check (→ QA-TODO):
upload a real product image through the admin editor and confirm the app renders it.

<details><summary>Original provisioning notes (superseded)</summary>

## Original item 13 text — provision Azure Blob (product images feature)

Product image uploads (admin editor → Go API compresses to a 1024px display JPEG
+ 256px thumbnail → blob storage) ship with a **hexagonal `platform/blob` port**:
`STORAGE_PROVIDER` selects `local` (dev default — writes to `UPLOADS_DIR`, served
by the API at `/uploads/*`) or `azure`. **Prod must use `azure`** (a Container App
has ephemeral local disk). One-time setup in **Azure Cloud Shell** (local `az` is
broken by the proxy TLS MITM — see CLAUDE.md):

```bash
az storage account create -n salehcardassets -g salehcard-prod -l <region> \
  --sku Standard_LRS --allow-blob-public-access true
az storage container create --account-name salehcardassets -n product-images \
  --public-access blob
az storage account show-connection-string -n salehcardassets -g salehcard-prod
# then set on the API Container App / App Service:
#   STORAGE_PROVIDER=azure
#   AZURE_STORAGE_CONNECTION_STRING=<value from show-connection-string>
#   AZURE_STORAGE_CONTAINER=product-images   (default; can omit)
```

- The container is **public-read** (`--public-access blob`) so the app/admin can
  render image URLs directly (no SAS). Only the API (admin-only endpoint) writes.
- Record the connection string in the gitignored **`DEPLOY-CREDS.local.md`**.
- The adapter **falls open to `local`** on misconfig (logs a warning) — so a
  missing/blank connection string won't crash boot, it just won't persist to Azure.
  Verify the startup log shows no `storage provider misconfigured` warning.
- **Future work:** swap connection-string auth → managed identity; optional
  best-effort old-blob delete on replace (v1 leaves orphans — pennies).

</details>

## 14. On-chain USDT payments (TRC20) — prod rollout (payment module PR1/PR2)

The feature ships DISABLED (no `USDT_XPUB` app setting → intent creation refuses,
no watcher runs, `GET /api/v1/payments/config` answers `usdtEnabled:false`). To
enable in prod, in order:

1. **Generate the wallet OFFLINE** (owner, never on the server / never in the repo):
   create a fresh mnemonic on a hardware wallet or an offline BIP39 tool, derive the
   BIP44 TRON account `m/44'/195'/0'`, and export its **xpub** (watch-only). The
   mnemonic is the spend key — store it like the prod DB password. The server only
   ever sees the xpub.
2. Create a TronGrid account (trongrid.io) → API key.
3. Azure App Service app settings on the API:
   - `USDT_PROVIDER=trongrid`  (⚠️ NEVER `stub` in prod — `config.Validate` refuses
     to boot with stub+xpub outside development, because the stub auto-confirms
     unpaid intents)
   - `USDT_XPUB=<the account xpub>`
   - `TRONGRID_API_KEY=<key>`
   - optional tuning: `USDT_INTENT_EXPIRY=30m`, `USDT_WATCH_INTERVAL=25s`
4. Confirm **Always On** is enabled on the App Service (the watcher is an in-process
   background loop — idle-unload kills it). Single instance recommended: duplicate
   watchers are SAFE (atomic claims + unique (network,txHash) index prevent double
   credits) but waste TronGrid quota.
5. Boot check: the startup log must show
   `payment: on-chain USDT payments enabled ... address0=T...` — verify that address
   matches index 0/0 of the wallet in an independent tool (TronLink import of the
   xpub, or iancoleman.io/bip39 offline) BEFORE announcing the feature.
6. Confirm the new Cosmos collections/indexes came up: `payment_intents`
   (unique partial `network+txHash`, unique `address`, unique partial
   `userId+idempotencyKey`), `counters`, and the new unique partial
   `method+ref` (method=usdt_trc20) index on `wallet_transactions`.
7. **Sweep runbook** (moving customer deposits to treasury): import the mnemonic
   into TronLink → funds sit on the per-intent derived addresses (0/0, 0/1, …).
   Each address needs a little TRX for energy/bandwidth to move USDT out; sweep
   periodically, oldest first. The admin `/payments` page + tronscan links show
   every funded address.

## 15. Whish Pay (redirect + callback) — prod rollout (payment module Whish PR)

The feature ships DISABLED (no `WHISH_*` creds → intent creation refuses, no
sweeper runs, `GET /api/v1/payments/config` answers `whishEnabled:false`). To
enable in prod, in order:

1. **Obtain SalehCard's OWN Whish merchant account** → per-merchant `channel` +
   `secret` + `websiteUrl`. ⚠️ The LACPA reference creds used for sandbox testing
   (`channel 10200046`, `websiteUrl lacpa.academy`) are NOT SalehCard's and must
   never ship. They were chat-exposed → **ask LACPA to rotate them.**
2. Azure App Service app settings on the API:
   - `WHISH_PROVIDER=whish`  (⚠️ NEVER `stub` in prod — `config.Validate` refuses
     to boot with creds + stub outside development; the stub auto-confirms on
     re-poll)
   - `WHISH_CHANNEL` / `WHISH_SECRET` / `WHISH_WEBSITE_URL` = SalehCard's merchant creds
   - `WHISH_BASE_URL=https://api.whish.money/itel-service/api`  (prod, not sandbox)
   - `PAYMENTS_WEBHOOK_BASE_URL=https://<prod-api-host>`  (public host Whish can reach)
   - `PAYMENTS_HMAC_SECRET=<openssl rand -hex 32>`  (store like a DB password)
   - optional: `WHISH_SUCCESS_REDIRECT_URL` / `WHISH_FAILURE_REDIRECT_URL`,
     tuning `WHISH_INTENT_EXPIRY=30m`, `WHISH_SWEEP_INTERVAL=60s`
3. Confirm **Always On** on the App Service (the reconciliation sweeper is an
   in-process loop). Duplicate sweepers are SAFE (atomic claims + wallet
   `(method,ref)` index prevent double credits).
4. **Whish caller-IP allowlist?** Whish may IP-allowlist the merchant's calling IP
   (like Monty SMS does). If so, prod already egresses via the NAT Gateway fixed
   IP (see `HANDOFF-OTP-DEPLOY.md`) — register that IP with Whish. Confirm before
   go-live, or `Initiate`/`GetStatus` will fail.
5. Boot check: the startup log must show `payment: Whish redirect payments
   enabled provider=whish`. Confirm the new Cosmos index came up: unique partial
   `provider+externalId` on `payment_intents`. The `wallet_transactions`
   `(method,ref)` unique-partial index is auto-migrated from usdt_trc20-only to
   `method ∈ {usdt_trc20, whish}` at startup (`wallet.EnsureIndexes` drops the
   stale `method_1_ref_1` and recreates it widened — Mongo/Cosmos won't recreate
   a same-named index with a different partial filter otherwise). Watch for no
   `wallet: failed to ensure indexes` warning in the boot log.
6. **Reconciliation:** abandoned intents are swept to `expired` after
   `WHISH_INTENT_EXPIRY` (the sweep re-polls Whish first, so a lost-callback-but-
   paid intent still settles). Monitor the admin `/payments` page (filter Whish /
   Failed) for stuck intents.
