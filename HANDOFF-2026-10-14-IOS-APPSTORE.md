# HANDOFF — FlashCash Global iOS: first App Store submission (2026-10-14)

Supersedes the 2026-08-09 Codemagic handoff (deleted 2026-10-16; in git history). Memory: `[[salehcard-ios-codemagic-plan]]`.

## TL;DR

| Item | State |
| --- | --- |
| App Store Connect | **1.0.3 (build 5) "Waiting for Review"**, submitted 2026-10-14 22:43 |
| Apple account | **Individual** — Ali Sleiman, Team ID `BFHJD7572H` (not the Entrizo org) |
| App record | app id `6822703792`, bundle `com.flashcashglobal.app`, SKU `flashcash-ios` |
| Crypto + money transfer on iOS | **Hidden** (client-side gate, iOS only). Android unchanged |
| Code | branch `ios-first-release` (pushed, **not merged** to `main`) |
| CI | Codemagic `ios-release` → App Store Connect / TestFlight. Manual start until merged |
| TestFlight | internal group "Team": sleimana181@gmail.com, clashroyale084@gmail.com |
| Availability / price | Lebanon only, Free, Shopping, 18+ |
| Push notifications on iOS | Not wired yet (deferred) |

## Why crypto and money transfer are hidden on iOS

Apple only accepts crypto wallets (guideline 3.1.5) and financial-transfer services (3.2.1) from
developers enrolled as an **organization**. The iOS app ships from an individual account, so iOS builds
hide the Cryptocurrency and Money Transfers root domains, their products and offers, the USDT payment
option and crypto top-up methods. Re-enabling them on iOS needs an organization account **and** a new
build — there is deliberately no remote flag (a server toggle flipped after review would breach 2.3.1).

**Rule for all Apple-facing text** (listing, review notes, TestFlight info): never mention crypto, money
transfer or Whish. Whish is a third-party Lebanese payment provider to be integrated later.

## Code (branch `ios-first-release`)

- `app/lib/core/config/regulated_finance.dart` — `hideRegulatedFinance` (iOS only),
  `regulatedRootDomains = {wallets_crypto, money_transfers}`, crypto top-up name matcher,
  `RegulatedCatalogFilter` (resolves blocked category slugs once, cached).
- Applied in: category repository, catalog repository (lists, pages, single product → not found),
  offers provider, `paymentConfigProvider` (USDT forced off), wallet top-up methods.
- Test: `app/test/regulated_finance_test.dart`.
- iOS project: iPhone only (`TARGETED_DEVICE_FAMILY = 1`), `ITSAppUsesNonExemptEncryption = false`.
- `codemagic.yaml`: integration key name `Apple Distribution`; `xcode-project use-profiles`; build number
  `4 + $BUILD_NUMBER`; **`--dart-define=API_BASE_URL=https://salehcard-api.azurewebsites.net/api/v1`**
  (without it the app targets the dev LAN IP and hangs); `submit_to_testflight: false`.

## Signing and CI

- Apple portal: App ID `com.flashcashglobal.app`; App Store profile "FlashCash Global App Store"
  (distribution cert expires 2027-10-14).
- Codemagic Personal Account (sleimana181@gmail.com): app `6a79af1c587b87cf000c1d82`, Developer Portal key
  "Apple Distribution", profile imported as `flashcash_appstore`.
- Workflow auto-triggers only on pushes to `main`. Until the branch is merged, start builds manually from
  the Codemagic UI on `ios-first-release`.

## App Store listing (as submitted)

- Subtitle "Game credit & gift cards"; description, promo text and keywords cover gift cards, game credit
  and mobile recharge only. Support/marketing URL `https://www.flashcashglobal.com`, privacy
  `https://salehcard-api.azurewebsites.net/privacy`, copyright "2026 Ali Sleiman".
- Age rating: questionnaire all "none" except user-generated content = yes (product reviews), override 18+.
- App Privacy (published): name, email, phone, photos (ID / receipts), other user content (reviews),
  user ID, device ID (push), purchase history, other data (player IDs) — all App Functionality, linked,
  not used for tracking.
- Screenshots: required slot is **iPhone 6.3" (1206 × 2622)**. Take them on an iPhone 17 Pro Max
  (1320 × 2868), downscale, and upload as **24-bit JPEG** — PNGs with an alpha channel fail processing.
  Sending via WhatsApp as a photo compresses to 736 × 1600; send as a document instead.

## App Review demo account

- Customer `appreview@flashcashglobal.com`, phone `+9611999888` (landline format, can't receive SMS, so it
  can't be taken over by OTP). Reviewer enters **01 999 888** and signs in with the password. Reset the
  password with `api/cmd/setpass` if needed.
- ID check approved, $20 wallet credit (admin adjustment, reason "Apple App Review demo account balance").
- Test order **#311558** ($0.89, player ID "e") was created on prod while taking screenshots — cancel /
  refund it.

## If App Review rejects

Most likely guideline **3.1.1**: digital goods bought with a wallet funded outside Apple's IAP. The review
notes say codes are redeemed outside the app in the third-party stores and games they belong to. Bring the
rejection text back and decide the response; don't revisit the account decision.

## Next steps

1. Wait for review (Apple says up to 48 h).
2. Merge `ios-first-release` into `main`.
3. iOS push: APNs key → Firebase project `salehcard-app`, add `aps-environment` entitlement + App ID
   capability, then a new build.
4. Optional: add the 6.9" (1320 × 2868) screenshot set for better quality on large phones.
