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
