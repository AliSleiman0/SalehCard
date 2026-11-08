# HANDOFF: iOS recharge-only build, resubmitted after the 3.1.1 rejection (2026-11-08)

**Start here for anything iOS / App Store.** This replaces the "If App Review rejects" section of
`HANDOFF-2026-10-14-IOS-APPSTORE.md`, which still holds the signing, Codemagic and demo-account basics.
Memory: `[[salehcard-ios-codemagic-plan]]`.

## Status (2026-11-08, 22:00 Beirut)

| Item | State |
| --- | --- |
| App Store Connect | **1.0.6 (9) "Waiting for Review"**, resubmitted 2026-11-08 22:00. Submission id `b106f53e-6e3d-4820-9e01-b57e93d2794c` |
| Release mode | **Automatically release** when approved, so approval puts it straight on the App Store |
| Code | PR #110 merged to `main` (`60cef37`) |
| Build | Codemagic build #5 (manual start; see "Codemagic gotcha") → uploaded as build **9**. Also in the TestFlight internal "Team" group |
| Android / Google Play | Unaffected. FlashCash Global is live in production on the Entrizo Play account, no policy issues |
| Apple account | Individual (Ali Sleiman, Team `BFHJD7572H`). App id `6822703792`, bundle `com.flashcashglobal.app` |

**Next session, first thing:**
1. Open App Store Connect → FlashCash Global → App Review and read Apple's latest message.
2. If approved, go to "Approved". If rejected again, go to "If Apple objects again".

## Why iOS is recharge-only
On 2026-10-29 App Review rejected **1.0.6 (8)** under **Guideline 3.1.1**. Apple's two reasons:
- the app sells digital items, codes or currencies used in other apps or on third-party platforms;
- it sells gift cards redeemable for digital content on third-party platforms.

Apple's next step read "review the app concept". In-App Purchase doesn't fix this: Apple doesn't let an app
sell other apps' currency, and its 15–30% cut would erase the margin. Prepaid mobile airtime is a real-world
service (3.1.3(e)), so it can be sold without IAP. Ding and Recharge.com work the same way on the App Store.

**Why Whish Money can sell more on iOS:** it's a licensed financial company publishing from its own
organisation account, and Apple treats it as a payment app. FlashCash publishes from an individual account
and was reviewed as a digital-goods store. The org-account route was dropped in October 2026; don't
re-litigate it.

## What changed in the app (PR #110; app only, no API change)
- **iOS shows only the `telecom` root domain**: Alfa/Touch(MTC) direct recharge, vouchers, airtime,
  validity and u-share. That's 50 products on prod, all with `rootDomain: telecom`; all 630 products were
  scanned on 2026-11-08 with no mismatches. Android is unchanged.
- **`app/lib/core/config/regulated_finance.dart`** is the single switch:
  - `restrictToRealWorldServices` is true on iOS. It's a **build-time platform check, never a remote
    flag**, because changing what a reviewed build shows from the server breaches guideline 2.3.1.
  - `hideRegulatedFinance` is now an alias for it.
  - `iosAllowedRootDomains = {'telecom'}`, plus `catalogRootDomainScope` and `isRootDomainAllowed()`.
    `isRootDomainAllowed()` fails closed: an empty or unknown domain is hidden.
  - `RegulatedCatalogFilter.categoryAllowed()` resolves allowed slugs via `GET /categories?rootDomain=telecom`.
  - `isTopUpMethodHiddenOnRestricted()` hides crypto and Whish top-up methods.
  - The debug-only `--dart-define=FORCE_IOS_SCOPE=true` applies the iOS scope on Android; release builds
    compile it out.
- **Catalog** (`catalog_repository_impl.dart`):
  - Every product query is pinned server-side to `rootDomain=telecom`, so lists, search and totals are right.
  - Asking for another domain returns empty without making a request.
  - There's also a client backstop on `Product.rootDomain` (new DTO field), and a non-telecom product
    opens as not-found.
- **Categories, offers, orders:**
  - Categories and offers are filtered.
  - Order list and detail hide any order with a non-allowed line (`checkout/presentation/providers.dart`).
  - Profile hides saved player IDs and the edit mode.
- **Strings (both platforms):** no more "gift cards" in promo, empty cart, search prompt, notification
  promo or the item fallback label (EN + AR).
- **iOS branding:**
  - New phone + lightning app icon (`ios/Runner/Assets.xcassets/AppIcon.appiconset`).
  - In-app mark and lockup (`assets/branding/logo_mark_ios.png`, `logo_lockup_ios.png`) and the native
    LaunchImage. The old lockup still said "SALEH CARD".
  - Android keeps the gamepad mark.
  - `flutter_launcher_icons` / `flutter_native_splash` are now **`ios: false`**. Don't flip them back, or
    they will overwrite the iOS assets.
- **Tests:** `app/test/regulated_finance_test.dart` covers the allowlist, server scoping, the empty
  out-of-scope result, not-found, and Android passthrough. The full suite passes (83/83).

## What's in App Store Connect now
- **App Information:**
  - Subtitle `Lebanon Mobile Recharge`.
  - Primary category **Utilities** (was Shopping).
  - English is the only localisation.
- **Version 1.0.6:**
  - Promo text: "Recharge your Alfa or Touch line in seconds: airtime, validity and data bundles for Lebanon."
  - Keywords: `alfa,touch,mtc,recharge,top up,airtime,prepaid,lebanon,validity,ushare,mobile credit`
  - Description: recharge only (what you can buy, how it works, why FlashCash).
  - Build **9**.
- **Screenshots (iPhone 6.3", 5 of them):**
  - Home, Touch product, Checkout, Orders, Account menu.
  - They are frames from the user's WhatsApp screen recording, upscaled to 1206×2622, so they're slightly
    soft.
  - Swap in native screenshots any time; metadata-only changes don't need a new build.
  - Local copies: `C:\Users\user\Downloads\FlashCash-iOS-screenshots\`.
  - The iPad and Watch slots are empty, which is fine because the app is iPhone-only.
- **Review notes** (2,500 chars), keeping Apple's 2.1 structure:
  - intro: the concept is revised;
  - 1. recording: it stops at the delete confirmation, so the demo account survives;
  - 2. purpose;
  - 3. how to test;
  - 4. external services (Azure, Monty);
  - 5. regional (Lebanon only, EN/AR);
  - 6. regulated: a closed-loop balance funded outside the app, no money transfer.
- **Attachment:** `flashcash-ios-review-recording.mp4`, a 61 s recording on an iPhone with build 9.
  - The user's WhatsApp-compressed copy is at `C:\Users\user\Downloads\WhatsApp Video 2026-11-08 at 9.40.18 PM.mp4`.
  - It shows sign-in → telecom-only home → MTC Validity 1 Year purchase → order Processing → delete
    dialog cancelled.
- **Reply to Apple:** posted 2026-11-08 21:58 on the submission thread. It says the concept is revised
  to Alfa/Touch prepaid recharge only, game top-ups and gift cards are removed, the icon, text and
  screenshots are updated, and build 1.0.6 (9) is attached.

## App Review demo account
- Phone **`01 999 888`** (sign in with the password, because it's a landline that can't get SMS).
  Email `appreview@flashcashglobal.com`.
- The password is in App Store Connect → version → App Review Information → Sign-In Information, and in
  `DEPLOY-CREDS.local.md`. Never paste it into docs or chat.
- **Never delete this account**: reviewers need it. The recording shows the deletion flow up to the
  confirmation, then Cancel.
- 2026-11-08 cleanup:
  - Game orders `375ae521` (BLACK BOOSTER $15.06) and `9d1b4621` (LAKA CHAT $2.56) were refunded with the
    reason "App Review demo account cleanup".
  - `33311558` (#311558) had already been refunded on 10-14.
  - The old "App Review Demo Card" gift-card orders are still there, hidden on iOS by the order filter.
- **Open item:** the recording's order **#1E280E**, MTC Validity 1 Year $14.50 to the user's own line, is
  **Processing** in admin. Complete it (real recharge) or refund it (see QA-TODO).

## Codemagic gotcha
`codemagic.yaml` says `ios-release` triggers on pushes to `main`, but **the merge of PR #110 did not start
a build**. Start it by hand: codemagic.io → Applications → SalehCard → **Start new build** → branch `main`,
workflow "iOS Release (TestFlight)". The build number is `4 + Codemagic build index`, so index 5 gave
build 9 and the next one gives build 10. Tracked in DEVOPS-TODO §23.

## Approved
- Nothing to do if auto-release fires. Check the live listing shows the new icon and screenshots.
- Merging `ios-first-release` is moot: everything is on `main`.
- Optional: replace the soft screenshots with native iPhone screenshots.
- Optional: add the 6.9" set.

## If Apple objects again
- **To "codes" (recharge PIN vouchers):**
  - Hide the voucher categories on iOS and keep direct recharge only.
  - The smallest change is to drop the `alfa-voucher` / `mtc-voucher` slugs inside
    `RegulatedCatalogFilter._load()`.
  - Products are also scoped by `rootDomain`, so `CatalogRepositoryImpl._visible()` must filter those
    slugs as well.
  - Then rebuild, which gives build 10.
- **Authorisation (5.3):** be ready to show you're allowed to resell Alfa/Touch credit (a distributor
  agreement).
- **Stored-value balance:** if Apple questions the wallet, point to the notes. The balance is closed-loop,
  can only be spent on recharge, and is funded outside the app.
- **Never mention** crypto, money transfer or Whish in anything Apple sees (listing, notes, replies,
  TestFlight text).
