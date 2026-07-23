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

## 3. Restrict the Firebase Android API key ✅ DONE 2026-07-12

`app/lib/firebase_options.dart` commits the Android client key
(`AIzaSy…AuUsQ`) — an identifier, not a secret, but it should be locked to the
app. Applied via `gcloud services api-keys update` (key uid
`406142c0-8556-403e-a84f-15427c52119a`, project `salehcard-app`): Application
restriction = **Android apps**, package `com.salehcard.salehcard_app`, with two
SHA-1s — debug `8C:D4:52:…:1F:F6` + upload `A1:AF:3B:…:6F:AB`. All 27 existing
API targets preserved. Moots the GitGuardian flag on PR #27.
- ⚠️ **TODO at step 4 (Play launch):** add the **Play App Signing** SHA-1 to this
  same key, or Play-Store-distributed installs (re-signed by Google) get their
  Google/Firebase API calls rejected. Command:
  `gcloud services api-keys update 406142c0-8556-403e-a84f-15427c52119a --project=salehcard-app --allowed-application=sha1_fingerprint=<PLAY_SHA1>,package_name=com.salehcard.salehcard_app --allowed-application=... (re-pass debug+upload too, update replaces the android list)`.

**Update 2026-07-11 (Play Store hardening):** the release signing config is now
in-repo and the upload keystore exists after the WP1 manual step
(`PLAYSTORE-SUBMISSION.md` §7.1). Add BOTH the **upload-key SHA-1** and the
**Play App Signing SHA-1** (Play Console → Setup → App signing, available after
the first upload) to this API-key restriction **and** to the Firebase Android
app (`salehcard-app` → Project settings → Android app) — Play re-signs the AAB,
so the restricted key + FCM need the Play cert, not just the upload cert.

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
- ✅ Release signing config **shipped in-repo 2026-07-11**
  (`app/android/app/build.gradle.kts` loads the gitignored `key.properties`; no
  debug fallback — a release build without the keystore fails loudly). The
  keystore itself is generated manually; keystore location + creds live in the
  gitignored `DEPLOY-CREDS.local.md` (backup in `C:\Users\user\.salehcard-secrets\`).
- ✅ Distribution channel decided: **Play Console** (internal → closed testing
  first). The full Console-side playbook is `PLAYSTORE-SUBMISSION.md`; readiness
  status in `PLAYSTORE-READINESS.md`.
- Note: push notifications require the shipped app build to include the
  BL-11 code + real `firebase_options.dart` (already on `main`) — plus
  `google-services.json` (item 3 update) and the e2e check in item 19.

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

The feature ships DISABLED (neither `USDT_XPUB` nor `USDT_ADDRESS` set → intent
creation refuses, no watcher runs, `GET /api/v1/payments/config` answers
`usdtEnabled:false`). There are now **two mutually exclusive addressing modes**
(both set refuses to boot in every env; each open intent is stamped with its
mode, so flipping between modes mid-flight is safe — open intents keep settling
under their own rules through the grace window):

- **Shared-address mode** (`USDT_ADDRESS`) — the launch plan: every customer
  pays ONE fixed address (the owner's real wallet), matched by exact salted
  amount. No sweep runbook needed (funds land in the owner's wallet directly);
  transfers matching nothing appear in admin → Payments → Unmatched deposits
  for manual attribution.
- **Derived-address mode** (`USDT_XPUB`) — the later upgrade: a unique
  watch-only address per payment. Needs the offline wallet/xpub ceremony +
  periodic sweeps.

### 14a. Shared-address go-live (current plan — client's wallet address)

1. Azure App Service app settings on the API:
   - `USDT_PROVIDER=trongrid`  (⚠️ NEVER `stub` in prod)
   - `USDT_ADDRESS=TLRaHegyg2grMQqX85nJyCzbdRtvM5nCDn`  (the client's TRC20
     address — re-verify with him it is EXACTLY this before setting; validated
     at boot, but validation only catches typos, not a wrong-but-valid address)
   - `TRONGRID_API_KEY=<key>` (trongrid.io account)
   - `USDT_XPUB` must remain UNSET (both set refuses to boot)
2. Confirm **Always On** (watcher is in-process; idle-unload kills it).
3. Boot check: log must show
   `payment: on-chain USDT payments enabled (shared-address mode) ... address=TLRa...`.
4. Index check on Cosmos: `payment_intents` — the old unconditional-unique
   `address_1` index is dropped + recreated partial (`addressMode:"derived"`)
   automatically at boot; new unique partial `amountExpectedMicros` (over
   `sharedOpen`) must exist; new `usdt_deposits` collection (unique
   `network+txHash`). If the boot log warns the equality partial filter was
   rejected, the address index fallback is non-unique — acceptable (see
   repository.go comment).
5. **Never flip to "neither set"** while payments are open: that disables the
   watcher entirely and strands open intents. Flip directly between modes.
6. Ops note: the admin console → Payments → "Unmatched deposits" tab is the
   reconciliation queue (attribute = credit the customer's wallet; ignore =
   dust/spam). Customers who round the salted amount land there — expect some.

### 14b. Derived-address upgrade (later, when the client produces an xpub)

Flip `USDT_ADDRESS` → unset, `USDT_XPUB` → set (keep trongrid). No deploy, no
data migration; open shared intents drain through their grace window. Steps:

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

## 15. BEP20 (BSC) second USDT network — prod rollout

Ships DARK: prod behavior is byte-identical until the `USDT_BEP20_*` app
settings are set. BEP20 is **additive** to the TRC20 mode above (independent —
either can run alone) and is always shared-address mode: customers pick the
network at payment time; identity is the same exact-salted-amount scheme,
scoped per network (compound unique index `network+amountExpectedMicros`).
BEP20 USDT has **18 decimals** on-chain; the adapter normalizes to micro-USDT.

### Rollout order (prod is live — sequence matters)

1. **Deploy the code FIRST with no new env vars.** Verify prod unchanged
   (TRC20 boot log line still present, watcher running) and the index
   migrations applied — on Cosmos run `getIndexes()`:
   - `payment_intents`: `network_1_amountExpectedMicros_1` (unique partial,
     sharedOpen) EXISTS and the old `amountExpectedMicros_1` is GONE.
   - `wallet_transactions`: `method_1_ref_1_usdt_bep20` (unique partial,
     `method:"usdt_bep20"`) EXISTS next to the original `method_1_ref_1`.
   ⚠️ If the wallet index was rejected, DO NOT flip the env vars — without it
   BEP20 settlement retries have no double-credit guard. Fallback: replace
   both wallet dedup indexes with one `$in`-filtered index and re-verify.
   Deploy at low traffic: there is an ms-scale drop→create window on the
   amount-uniqueness index at boot.
2. ~~Ali creates a free etherscan.io account → API key~~ **DISCOVERED AT
   ROLLOUT (2026-07-10): Etherscan's FREE plan does not cover BSC** — chainid
   56 returns "Free API access is not supported for this chain", and the
   legacy api.bscscan.com V1 API is fully shut down. The provider to use is
   **`jsonrpc`** (free public BSC nodes, no key, default endpoints built in;
   override via `BSC_RPC_ENDPOINTS`). The etherscan adapter is kept for a
   future paid plan. NOTE: the canonical bsc-dataseed.* public nodes do NOT
   serve eth_getLogs at all — the built-in defaults (rpc-bsc.48.club +
   NodeReal's documented public endpoint) were verified live on 2026-07-10
   with 4,000-block filtered getLogs ranges.
3. **Confirm the BEP20 address with the client character-for-character**:
   `0x5e0a66cedc7688aab52c87dc02bff97f6575d7dc` (the BSC address he sent
   first, before the TRC20 one). Boot validation catches malformed addresses,
   not wrong-but-valid ones; funds sent to a wrong address are unrecoverable.
4. Azure App Service app settings on the API:
   - `USDT_BEP20_PROVIDER=jsonrpc`  (⚠️ NEVER `stub` in prod — refused at
     boot; an unknown provider name is refused too)
   - `USDT_BEP20_ADDRESS=0x5e0a66cedc7688aab52c87dc02bff97f6575d7dc`
   - optional: `BSC_RPC_ENDPOINTS` (CSV; defaults to the official public
     dataseeds), `USDT_BEP20_MIN_CONFIRMATIONS` (default 15 ≈ seconds on BSC)
   - `ETHERSCAN_API_KEY` is UNUSED with the jsonrpc provider (removable; only
     needed if ever switching to a paid etherscan plan)
   - **State as of 2026-07-10**: address + `provider=etherscan` + a free key
     were set before the free-tier limitation surfaced — BEP20 is
     enabled-but-broken (every chain call fails). Flip the provider to
     `jsonrpc` once the adapter deploys.
5. Boot check: a SECOND enable line must appear —
   `payment: on-chain USDT payments enabled (shared-address mode) provider=jsonrpc network=bep20 address=0x5e0a...`
   and the `shared-address transfer list failed` warns must stop.
6. First ~20 min after the flip = backfill: the scanner walks the open window
   in 16k-block steps ("scan window truncated by the per-tick chunk cap"
   warns taper to zero). Any transfers sent during the enabled-but-broken
   outage still inside the 7-day grace get swept up automatically (settle or
   land in Unmatched deposits); older ones need manual bscscan reconciliation.
7. Canary: $1 real BEP20 top-up end-to-end (Binance → BEP20 withdrawal):
   exact salted amount auto-confirms, ledger row `method=usdt_bep20`,
   admin intent row shows a BEP20 badge + bscscan links.
8. Ops notes: the same Unmatched-deposits queue serves both networks (rows
   carry the network; attribute/ignore unchanged). The app shows a network
   picker only when BOTH networks are enabled — flipping BEP20 off later
   hides it again (open BEP20 intents keep settling until you also break
   the lister; avoid flipping while intents are open, same rule as §14a.5).

## 16. Data hygiene — corrupted legacy `qty` field on one product

`Z3x Samsung (Box or Dongle) Activation` (category `gsm-tool`, prod
`_id` in the `products` collection) has a legacy `inputFields` entry
`{key:"qty", type:"quantity", constraints:{min:0,max:0}}` — a known
migration-import corruption shape (see
`migration/internal/transform/inputfields.go`'s `corrupt()` check). The
order-service fix that made quantity-type fields authoritative
(`api/internal/modules/order/service.go`'s `validateQuantityField`)
treats `{0,0}` as "no real constraint" so this product stays orderable
today, but the underlying bad data should still be cleaned up via the
admin product editor — either remove the bogus `qty` field or set a
real `{min,max}` range. Low urgency (not currently blocking sales).

## 17. Reseller go-live — prod ops (after the pricing + un-hide PRs deploy)

Order matters: the pricing PR (#72, api+app) merges first so the catalog is
reseller-aware before anyone can be promoted; the un-hide PR (admin+web+docs)
follows.

1. After PR #72 deploys: smoke `GET /api/v1/products` anonymously — response
   shape unchanged (no `offerPrice` leakage without a reseller token).
2. After the un-hide PR deploys: the admin console shows the **Resellers** nav
   item for super admins. Grant `resellers.view`/`resellers.manage` to any
   custom roles that need it (`/roles`).
3. Create the real tier definitions in prod (admin → Resellers → tier editor).
   **Business decision required first: tier names + margin percents** — the
   dev seed's Bronze 5 / Silver 8 / Gold 12 are placeholders, not policy.
   Note: during a live sale a reseller can pay MORE than a retail customer
   (offers never stack on reseller pricing) — set margins with that in mind.
4. Promote the first real reseller (Users → detail → Role & access →
   Reseller), assign their tier, and have them verify in the app: catalog
   shows their price; a small real order charges exactly the displayed price.
5. Their first top-up request should show the amber reseller chip in
   `/topups` (QA-TODO "Reseller go-live" §8).
6. The customer APK needs no rebuild for pricing (server-side), but the
   provider-invalidation fix rides the next app release — until then a
   freshly-promoted reseller should restart the app once after login.

## 18. Set the real support email in prod admin Settings (Play Store hardening)

The public legal pages served by the API (`/privacy`, `/delete-account` — live
once the account-deletion/legal-pages PR deploys) render the support email from
`app_settings.supportEmail` (admin console → Settings). Until it is set in prod,
**the pages fall back to the `support@salehcard.com` placeholder** — set the
real mailbox before the Play listing goes live, and use the same address as the
listing's contact email (`PLAYSTORE-SUBMISSION.md` §4). Verify: open
`https://salehcard-api.azurewebsites.net/privacy` and check the contact section
updates without a restart.

## 19. Push e2e validation on a Play-signed release build (Play Store hardening)

The FCM Gradle wiring shipped 2026-07-11 (`com.google.gms.google-services`
plugin in `app/android/settings.gradle.kts` + app `build.gradle.kts`), but
`google-services.json` is still a Firebase-console download (item 3 update), and
**Play App Signing re-signs the AAB with a different cert** — so push against a
locally built AAB proves nothing about the store build. Before claiming push in
the listing: install a **Play-delivered build** (internal testing track),
trigger any business event (e.g. approve a top-up), and confirm a tray
notification arrives on the device. If the GMS plugin misbehaves under the new
AGP, the guarded FlutterFire programmatic init is the fallback — the e2e check
is required either way.

## 21. Re-register the app package in Firebase after the `flashcash.global` rename (BLOCKS push)

The Play Console app is registered under package **`flashcash.global`**, so the
Android `applicationId` was changed from `com.salehcard.salehcard_app` →
`flashcash.global` (2026-07-23, `app/android/app/build.gradle.kts`; `namespace`
kept as the old value so R/MainActivity classes are untouched). The Firebase
project **`salehcard-app`** only knows the OLD package, so `google-services.json`
had no matching client and hard-failed the release build. **Interim fix:** the
file was moved aside to `app/android/app/google-services.json.disabled` so the
GMS plugin is skipped and the build compiles (back to the FlutterFire
programmatic-init path). **Push (FCM) will NOT deliver to the new package until
fixed.** To restore push: in the Firebase console, add an Android app with
package `flashcash.global` to project `salehcard-app`, download the new
`google-services.json`, restore it as `app/android/app/google-services.json`
(delete the `.disabled` one), regenerate `firebase_options.dart` if needed
(`flutterfire configure`), then rebuild + re-run item 19's push e2e check.

## 20. Upstream supplier integration — prod rollout (DESIGN-SUPPLIERS.md, all 4 phases)

The whole supplier stack (panel adapters jentel/speedcard/gift4card + async
settler + `/suppliers` admin surface + umanage telecom adapter) is code-complete
and merged, but **inert in prod until credentials are set** — every supplier id
resolves to the parking stub with no env, which is exactly today's behavior
(api-mode orders park for manual completion). To go live:

1. **Set supplier credentials** as Azure app settings on `salehcard-api`
   (secrets — never in repo; add to `DEPLOY-CREDS.local.md`):
   - Panels: `SUPPLIER_JENTEL_TOKEN`, `SUPPLIER_SPEEDCARD_TOKEN`,
     `SUPPLIER_GIFT4CARD_TOKEN` (the `api-token` values; the four tokens shared
     in chat are chat-exposed — owner accepted, not rotating). Default ids
     10/11/12 and base URLs are baked in; override with `SUPPLIER_*_ID` /
     `SUPPLIER_*_BASE_URL` only if needed.
   - umanage (telecom): `SUPPLIER_UMANAGE_KEY` + `SUPPLIER_UMANAGE_SECRET`
     (X-API-Key / X-API-Secret). `SUPPLIER_UMANAGE_STORE_ID` optional (resolved
     at boot via GET /stores when 0). Id 13, LBP.
   - Optional settler knobs: `SUPPLIER_SETTLER_INTERVAL` (default 60s),
     `SUPPLIER_SETTLER_GIVEUP` (default 24h).
   A supplier with no credential stays a parking stub → zero risk in setting
   them one at a time.

2. **IP-allowlist the prod NAT egress IP at every supplier.** The panels enforce
   IP allowlisting (error 123) and umanage may too. Prod already egresses via the
   fixed **NAT Gateway IP** (set up for Monty SMS — see `HANDOFF-OTP-DEPLOY.md`);
   register that one IP with jentel/speedcard/gift4card and umanage at
   onboarding. An un-allowlisted caller → the balance probe shows `ip_blocked` and
   orders park (never a failed/refunded order). Dev boxes are NOT allowlisted, so
   the stub stays the dev default.

3. **RBAC:** the new `suppliers` domain (`suppliers.view`/`.manage`) exists in
   the catalog. Super admins get it automatically (nil adminRoleId → `["*"]`);
   grant it to any custom roles that should see `/suppliers`. No migration.

4. **Prepaid balances:** each supplier is prepaid — keep a wallet funded at each
   panel/umanage. A drained supplier wallet silently PARKS paid customers'
   orders; the `/suppliers` balance cards + dashboard supplier strip + the
   per-supplier `lowBalanceThreshold` (Settings) are the operational guardrail —
   set a sensible threshold per supplier after go-live.

5. **First-supplier canary:** enable ONE panel first (e.g. jentel), map a single
   low-value product (admin product editor → Fulfill via supplier API + upstream
   id, or `/suppliers` → Browse & import which creates it hidden), place one real
   wallet order end-to-end, confirm it completes with a delivered code and an
   `order.supplier_settle`/completion audit entry, THEN widen. umanage is a
   separate strategic rail (bridge-adjacent) — ship panels first.

Verify after setting creds: `/suppliers` renders live balances (not
`unreachable`/`ip_blocked`), and the dashboard shows the supplier balance strip.
