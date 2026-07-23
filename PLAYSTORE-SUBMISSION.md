# Play Store Submission Playbook — FlashCash Global Customer App (`/app`)

> **Rebrand (2026-07-24):** the app is now **FlashCash Global**, Play package
> **`flashcash.global`** (NOT `com.salehcard.salehcard_app`, which only survives as the
> Gradle `namespace`). References to the old package below are historical — see
> DEVOPS-TODO items 3 + 21 for what the rename still owes.

_Date: 2026-07-11 · Companion to `PLAYSTORE-READINESS.md`. Everything in this file happens
in the Play Console / Firebase console / Google Cloud — none of it lives in the repo. The
repo-side blockers (readiness #1/#2/#4/#5 + the cleartext/FCM quick wins) shipped 2026-07-11._

---

## 1. Data Safety form — exact answers

Fill the form exactly like this. Mis-declaring is a frequent takedown cause, and Google
cross-checks the declared types against what the app observably sends.

### Data collected

| Play category | Data type | Collected | Shared | Purpose to declare |
| --- | --- | --- | --- | --- |
| Personal info | Name | Yes | No | Account management |
| Personal info | Phone number (required — phone-OTP login) | Yes | No | Account management, authentication (OTP SMS) |
| Personal info | Email address (optional) | Yes | No | Account management |
| Financial info | User payment info — wallet balance, top-ups | Yes | No | Payments, fraud prevention |
| Financial info | Purchase history — orders, transactions | Yes | No | Order fulfillment, support |
| Photos | KYC government-ID photos (front/back) — **mark as sensitive info** | Yes | No | Identity verification (KYC / compliance) |
| Device or other IDs | FCM registration token (`app/lib/core/push/push_service.dart`) | Yes | No | Push notifications |

### Explicitly NOT collected — answer "No"

- **Location** (precise or approximate)
- **Contacts**
- **Advertising ID** — the app ships **no ads SDKs and no analytics SDKs**

"Shared" is **No** across the board: Azure (hosting + blob storage), Monty (SMS), Firebase
(push), and the RapidAPI game-ID verifier are **service providers acting on our
instructions** — under Play's definitions that is still "collected", not "shared". Nothing
is sold or shared for advertising.

### Global questions

| Question | Answer |
| --- | --- |
| Is all user data encrypted in transit? | **Yes** — HTTPS to `salehcard-api.azurewebsites.net` |
| Do you provide a way for users to request data deletion? | **Yes** — in-app (account menu → Delete account) **and** web: `https://salehcard-api.azurewebsites.net/delete-account` |
| Is an account required to use the app? | **Yes** — account creation required (phone number + OTP) |

### ⚠️ Do NOT over-attest at-rest security

KYC ID images live in a **public-read** Azure Blob container behind unguessable URLs
(accepted v1 trade-off — see CLAUDE.md). Until that container is locked down, do **not**
attest anywhere (Data Safety free-text, review appeals, listing copy) that data is
"encrypted at rest" or "access-controlled" — attest only encryption **in transit**, which
is true. The privacy policy (`api/internal/server/static/privacy.html`) is deliberately
worded honestly for the same reason — keep every Console answer consistent with it.

---

## 2. Content rating questionnaire (IARC)

- **User-generated content: declare YES** — product reviews are UGC. Mitigations to
  declare: login required to post, one review per product per user, and an **admin
  moderation queue** (`/api/admin/reviews`, decisions audited).
- Violence / sexuality / drugs / real or simulated gambling: **No** to all.
- The questionnaire will likely land a low age rating (Everyone / PEGI 3), but the app is
  functionally adult (financial services; the privacy policy states 18+). Set the **target
  audience to 18+** and do not opt into any children/families program.

## 3. Financial-features declaration

The wallet plus wallet-to-wallet transfer (`send_money_screen.dart`) puts the app in
Google's financial-features declaration. Prepare with the owner **before** submitting:

- The **legal basis for operating in Lebanon**: business registration details, and whether
  any money-transfer authorization applies to a closed-loop wallet + digital-goods store.
- A one-paragraph description ready to paste: prepaid **closed-loop** wallet funded via
  admin-approved out-of-band channels; balance spendable only on digital goods in-app plus
  peer transfer between app users; **no cash-out**.
- **The USDT gating decision comes first** (readiness #7 — **still unmade**): if crypto
  top-up ships in the Play build, the declaration surface grows to crypto ("facilitating an
  exchange") and rejection risk rises sharply. The standing recommendation is to gate USDT
  **out** of the v1 Play build.

---

## 4. Store listing assets checklist

| Asset | Requirement | Source / suggestion |
| --- | --- | --- |
| App icon | 512×512 32-bit PNG | export from `app/assets/branding/logo_mark_padded.png` (or `logo_mark.png`) |
| Feature graphic | 1024×500 PNG/JPEG | compose from `app/assets/branding/logo_lockup.png` on a brand background |
| Phone screenshots | **≥2** (up to 8) | suggested: home/catalog, product detail (verified-nickname state), wallet, orders. **Do not screenshot the USDT deposit screen** regardless of the gating decision. |
| Short description | ≤80 chars, **EN + AR** | |
| Full description | ≤4000 chars, **EN + AR** | mirror actual features; no at-rest-encryption claims (§1 warning) |
| Privacy policy URL | mandatory | `https://salehcard-api.azurewebsites.net/privacy` |
| Category | **Shopping** | |
| Contact email | mandatory | the same support email set in prod admin Settings (`DEVOPS-TODO.md` #18) |

## 5. Testing timeline

- **Personal developer account** (created after Nov 2023): Google requires a **closed test
  running 14 days with ≥12 opted-in testers** before production access can even be
  requested. **Organization accounts are exempt.** Start the closed track immediately after
  the first AAB upload.
- Run the **internal testing** track first — the pre-launch report's automated device crawl
  catches crashes before any human tester (or reviewer) sees them.

## 6. Release build + versioning

```bash
cd app
flutter build appbundle --dart-define=API_BASE_URL=https://salehcard-api.azurewebsites.net/api/v1
```

- **Never build a release without the dart-define** — the default base URL is a dev LAN IP
  and the app hangs on infinite loading on real phones (memory:
  `salehcard-flutter-apk-prod-url`).
- Requires `app/android/upload-keystore.jks` + `app/android/key.properties` (manual step
  §7.1, both gitignored) — the release build **fails loudly** without them (no debug
  fallback). Debug builds are unaffected.
- **versionCode bump rule:** `app/pubspec.yaml` → `version: 1.0.0+1`. The number after `+`
  is the Android `versionCode` and **must strictly increase on every Play upload** (Play
  rejects a reused code); the `1.0.0` part is the user-visible `versionName`. Bump `+N`
  every upload; bump the semantic part on user-facing releases.
- Output: `app/build/app/outputs/bundle/release/app-release.aab`. Verify the signer with
  `keytool -printcert -jarfile app-release.aab` — it must show the upload cert, **not**
  `androiddebugkey`.

---

## 7. Manual steps — the full Console-side checklist (in order)

1. **Keystore**: generate the upload keystore + `app/android/key.properties` (recipe in the
   hardening plan / `keytool -genkeypair`); record creds in the gitignored
   `DEPLOY-CREDS.local.md`; back the keystore up to `C:\Users\user\.salehcard-secrets\`.
2. **Firebase console** (project `salehcard-app`): confirm/register the Android app
   — **done 2026-07-24 for `flashcash.global`** (appId `1:184899958988:android:1b1749afb3bcb962742a69`;
   the old `com.salehcard.salehcard_app` client is still in the project for legacy installs).
   Add the **debug + upload + Play App Signing SHA-1s**,
   download `google-services.json` → commit to `app/android/app/` (client identifiers only,
   not secret — the same key is already committed in `firebase_options.dart`).
3. **Google Cloud**: restrict the Android API key to the package + SHA-1s
   (`DEVOPS-TODO.md` #3).
4. **Play Console**: create the app, enroll in **Play App Signing** on first upload, set
   the privacy-policy URL, complete Data Safety (§1), content rating (§2), the financial
   declaration (§3), upload assets (§4), then internal → closed testing (§5).
5. **Prod admin Settings**: set the real support email — the public legal pages fall back
   to a placeholder until then (`DEVOPS-TODO.md` #18).
6. **Legal review**: owner/legal read-through of the EN+AR privacy + deletion page copy
   before submission.
7. **USDT gating decision** (readiness #7): decide whether crypto top-up ships in the Play
   build — the top remaining rejection risk.
8. **Push e2e**: validate push end-to-end on a **Play-signed release build**
   (`DEVOPS-TODO.md` #19) — or drop the push claim from the listing for v1.
