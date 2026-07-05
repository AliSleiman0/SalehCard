# DevOps TODO — do at the end of dev

Single collection point for all pending ops/infra work. Supersedes the scattered
"deferred" notes in the handoff files. Items are ordered by impact; each says
where it came from and how to verify it. Prod context: App Service
**salehcard-api** in RG **salehcard-prod**, sub `1adb4811-6234-4822-b7f2-8411ed2cb999`.

> **Corporate-proxy reminder:** local `az` cannot mint tokens (Python TLS rejects
> the proxy CA). Use the **Azure portal** or **Cloud Shell** for everything below
> that touches Azure. See `DEPLOYMENT.md`.

## 1. FCM push app settings (BL-11 — blocks prod push delivery)

The BL-11 API code is deployed but prod still runs `PUSH_PROVIDER=log`
(inbox notifications work; FCM tray delivery is off until this lands).

Portal → **salehcard-api** → Environment variables → add, then Apply (restarts app):

| Setting | Value |
|---|---|
| `PUSH_PROVIDER` | `fcm` |
| `FCM_CREDENTIALS_JSON` | one-line contents of `C:\Users\user\Downloads\salehcard-fcm-minified.json` |

Cloud Shell alternative:
`az webapp config appsettings set -g salehcard-prod -n salehcard-api --settings PUSH_PROVIDER=fcm FCM_CREDENTIALS_JSON='<paste one-line JSON>'`

Verify: `https://salehcard-api.azurewebsites.net/health` returns ok after
restart; trigger any business event and check Log Stream for a successful FCM
send (or absence of `push: falling back to log sender` warnings).

## 2. `BUSINESS_TZ=Asia/Beirut` app setting (BL-10)

Dashboard "today" KPIs (orders + wallet top-ups, `api/pkg/timeutil`) bucket on
UTC until this is set. Same portal screen as item 1 — add
`BUSINESS_TZ` = `Asia/Beirut`, Apply. (Batch it with item 1 to get one restart.)

## 3. Restrict the Firebase Android API key

`app/lib/firebase_options.dart` commits the Android client key
(`AIzaSy…AuUsQ`) — an identifier, not a secret, but it should be locked to the
app. Google Cloud console (project `salehcard-app`) → APIs & Services →
Credentials → the auto-created Android key → Application restrictions →
**Android apps** → add package `com.salehcard.salehcard_app` + the release
signing SHA-1 (and debug SHA-1 for dev builds). This also moots the
GitGuardian flag on PR #27.

## 4. Service-account key custody

The FCM service-account private key lives at
`C:\Users\user\Downloads\salehcard-app-firebase-adminsdk-fbsvc-185f034406.json`
(plus the minified copy). After item 1 is applied:
- Move both files out of Downloads into a proper secrets location (or delete
  them — the value then lives only in Azure app settings; a new key can always
  be generated in Firebase console → Project settings → Service accounts).
- Never commit either file. Rotate the key if it ever leaks
  (generate new → update `FCM_CREDENTIALS_JSON` → delete old key in console).

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

## 7. CI workflow action bumps (low priority)

Deploy runs annotate: *"Node.js 20 is deprecated… actions/checkout@v4,
dorny/paths-filter@v3"*. Bump to checkout@v5 / paths-filter current when
convenient. Cosmetic for now — runners force Node 24 and the workflows pass.

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
