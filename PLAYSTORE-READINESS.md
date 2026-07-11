# Play Store Readiness Assessment — SalehCard Customer App (`/app`)

_Date: 2026-07-11 · Scope: Flutter customer app (`/app`), Android. Bridge app (`/bridge`) covered separately at the bottom._
_Updated 2026-07-11: repo-side hard blockers fixed (playstore-hardening); Console-side steps split out to `PLAYSTORE-SUBMISSION.md`._

## Verdict

**Not ready to publish as-is.** The code is functionally mature, but there are hard
technical blockers and — more importantly — several **policy risks** that map directly onto
the most common rejection reasons for finance apps. Plan for 1–2 rejection cycles unless the
policy items below are handled up front.

**Update 2026-07-11:** hard blockers #1, #2, #4, #5 and the cleartext/FCM should-fixes are
now fixed in-repo (details per item below). Remaining before submission: the Console-side
steps (#3, #6 — see `PLAYSTORE-SUBMISSION.md`), the manual keystore/Firebase/Play-Console
steps (`PLAYSTORE-SUBMISSION.md` §7), and the **USDT gating decision (#7) — still unmade,
now the top remaining rejection risk**.

---

## 🔴 Hard blockers (cannot ship, or will be auto-rejected)

### 1. Release build is debug-signed — ✅ FIXED 2026-07-11
`app/android/app/build.gradle.kts` now loads the gitignored `app/android/key.properties`
into a real `release` signing config (no debug fallback — a release build without the
keystore fails loudly at `:app:validateSigningRelease`; debug builds are unaffected).
**Keystore generation itself is still a manual step** (never committed) — recipe + Play App
Signing enrollment in `PLAYSTORE-SUBMISSION.md` §6–7; creds go to `DEPLOY-CREDS.local.md`.
Ops tracking: `DEVOPS-TODO.md` #6.

### 2. No Privacy Policy — ✅ FIXED 2026-07-11
Bilingual (EN + AR, RTL) privacy policy is served by the Go API at
**`https://salehcard-api.azurewebsites.net/privacy`** (live once the API PR deploys) —
`api/internal/server/pages.go` + `api/internal/server/static/privacy.html`. Honestly worded
(no at-rest claims for the KYC images — see #3), with the support email
admin-configurable via `app_settings` (placeholder fallback until set — `DEVOPS-TODO.md`
#18). Linked in-app from the signup legal line and the account menu.

### 3. Data Safety form
Must be completed **accurately**, declaring collection of:
- Personal info — name / phone / email
- Financial info — wallet / payments
- Photos — KYC ID documents
- Device IDs — the FCM token registered in `push_service.dart`

Mis-declaring here is a frequent takedown cause.
**→ The exact answers to file are in `PLAYSTORE-SUBMISSION.md` §1** (Console-side step).
⚠️ **Related weakness (stands):** KYC ID images currently sit in a **public-read blob
container with unguessable URLs** (per CLAUDE.md). Fix/tighten before attesting "data is
encrypted / not shared" — until then attest in-transit encryption only.

### 4. No in-app account deletion — ✅ FIXED 2026-07-11
Both required paths now exist:
- **In-app**: account menu → Delete account → double-confirm bottom sheet →
  `DELETE /api/v1/users/me` — anonymize + soft-delete, KYC photos purged from blob storage,
  all sessions revoked; wallet balance > 0 blocks deletion (409). Implementation:
  `api/internal/modules/user/delete.go`,
  `app/lib/features/account/presentation/widgets/delete_account_sheet.dart`.
- **Web URL**: **`https://salehcard-api.azurewebsites.net/delete-account`**
  (`api/internal/server/static/delete-account.html`) — covers the can't-log-in path via the
  support email.

### 5. Target API level — ✅ VERIFIED 2026-07-11
The app inherits `flutter.targetSdkVersion` (`build.gradle.kts:49`), which **resolves to
API 36 on Flutter 3.44.3** — above the API 35 requirement. Nothing to change in the repo;
re-check only if the Flutter SDK is ever pinned back.

### 6. Store listing gates
Content-rating questionnaire + assets (feature graphic, ≥2 screenshots, full description) — standard
Console requirements, not yet prepared.
**→ Asset checklist + rating/financial-declaration guidance in `PLAYSTORE-SUBMISSION.md` §2–5.**

---

## 🟠 High policy risk (finance + crypto — expect manual scrutiny)

### 7. USDT / crypto surface  ← biggest rejection risk
`usdt_deposit_screen.dart` shows a crypto deposit address to fund the wallet. Google Play's
financial-services/crypto policy is strict — accepting crypto to credit a spendable balance can be
read as "facilitating a crypto exchange," pulling in licensing/declaration requirements.
**Recommendation:** gate crypto top-up **out of the Play build** for v1 (keep card / Whish / manual
rails); reintroduce only with a compliance story.

**Status 2026-07-11: the gating decision is STILL UNMADE — with #1/#2/#4/#5 fixed, this is
now the top remaining rejection risk.** Decide before the first AAB upload; see
`PLAYSTORE-SUBMISSION.md` §3 + §7.7.

### 8. Money transfer / financial services
`send_money_screen.dart` + wallet-to-wallet transfer puts this in Google's "financial services"
bucket. Some regions require a declaration that you're licensed/authorized. Operation is
Lebanon-based — be ready to state the legal basis in the Console financial-features declaration.

### 9. KYC ID capture
Collecting government IDs is sensitive-data territory — must be transparent in the privacy policy
and Data Safety, with a clear purpose. Allowed, but invites a manual review.

### 10. Play Billing question (lower risk)
Gift cards / top-ups redeemable **outside** the app are generally exempt from Play Billing as
real-world goods — defensible, but be ready to justify if asked.

---

## 🟡 Should-fix before submitting

- **`usesCleartextTraffic="true"`** — ✅ DONE 2026-07-11: removed from the main manifest;
  now declared only in `app/android/app/src/debug/AndroidManifest.xml`, so debug builds keep
  the plain-http dev LAN while `--profile`/`--release` builds are HTTPS-only (use an https
  dart-define for profile builds against a remote API).
- **FCM native config gap** — ⚙️ PARTIALLY DONE 2026-07-11: the
  `com.google.gms.google-services` Gradle plugin is now wired (`app/android/settings.gradle.kts`
  + app `build.gradle.kts`). **`google-services.json` itself is still an ops step** —
  download from the Firebase console after registering SHA-1s (`DEVOPS-TODO.md` #3) and
  commit to `app/android/app/`. Init remains guarded (won't crash without it), but validate
  push end-to-end on a Play-signed release build (`DEVOPS-TODO.md` #19) — or drop the push
  claim from the listing for v1.
- **Camera/photos** — `kyc_form_screen.dart` uses `image_picker` (camera + gallery). Confirm the
  runtime permission flow on Android 13/14 (photo picker needs no storage permission; camera source
  uses the system camera intent). Usually fine — verify.
- **`applicationId` is permanent** — `com.salehcard.salehcard_app` can never change after first
  publish. The scaffold `// TODO` next to it is removed; the ID itself must still be
  confirmed final before first publish.
- **Pre-launch report** — run the internal testing track first; Google's automated device crawl
  catches crashes before production.

---

## ⚠️ Separate issue: the `/bridge` app must NOT go to Play Store

`/bridge` uses an **AccessibilityService to automate USSD dialing**
(`AlfaUssdAccessibilityService.kt`). This is squarely against Google's Accessibility API policy (and
restricted-permission rules around USSD/call). It will be rejected and could jeopardize the developer
account. Keep it a **privately side-loaded owner/operator tool** — never on the Play Store.

---

## Process note (timeline)

If the Play Console developer account is a **personal** account created recently, Google now requires
a **14-day closed test with ≥12 testers** before promotion to production. **Org** accounts are exempt.
Start closed testing early.

---

## Suggested order of work

1. Release signing + Play App Signing enrollment. _(✅ signing config in-repo; keystore + enrollment manual — `PLAYSTORE-SUBMISSION.md` §7.1/§7.4)_
2. Add in-app account deletion + deletion web URL. _(✅ shipped 2026-07-11)_
3. Write privacy policy; complete Data Safety + content rating. _(✅ policy shipped; Data Safety + rating are Console steps — `PLAYSTORE-SUBMISSION.md` §1–2)_
4. **Decide crypto strategy** — recommend gating USDT out of the v1 Play build. _(⚠️ still open — top remaining risk)_
5. Fix cleartext traffic; verify target API 35; validate push on a release AAB. _(✅ cleartext fixed; ✅ targetSdk = 36; push e2e still open — `DEVOPS-TODO.md` #19)_
6. Prepare store assets; run internal/closed testing; then production. _(open — `PLAYSTORE-SUBMISSION.md` §4–5)_

---

## Quick reference — file pointers

| Concern | Location |
| --- | --- |
| Release signing (fixed) | `app/android/app/build.gradle.kts` (key.properties loader + `signingConfigs`) |
| Cleartext traffic (debug-only now) | `app/android/app/src/debug/AndroidManifest.xml` (removed from `src/main`) |
| applicationId (permanent) | `app/android/app/build.gradle.kts` (`defaultConfig`) |
| targetSdk (verified 36) | `app/android/app/build.gradle.kts:49` (`flutter.targetSdkVersion`) |
| FCM wiring (json pending) | `app/android/settings.gradle.kts`, `app/lib/firebase_options.dart`, `app/lib/core/push/push_service.dart` |
| Privacy policy page | `api/internal/server/pages.go`, `api/internal/server/static/privacy.html` |
| Account deletion (API + web + app) | `api/internal/modules/user/delete.go`, `api/internal/server/static/delete-account.html`, `app/lib/features/account/presentation/widgets/delete_account_sheet.dart` |
| USDT crypto top-up | `app/lib/features/payments/presentation/screens/usdt_deposit_screen.dart` |
| Money transfer | `app/lib/features/wallet/presentation/screens/send_money_screen.dart` |
| KYC ID capture | `app/lib/features/kyc/presentation/screens/kyc_form_screen.dart` |
| Console-side playbook | `PLAYSTORE-SUBMISSION.md` |
| Signing/API ops notes | `DEVOPS-TODO.md` #3, #6, #18, #19 |
