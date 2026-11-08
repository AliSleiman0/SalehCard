# HANDOFF: iOS recharge-only build after the 3.1.1 rejection (2026-11-08)

Supersedes the "If App Review rejects" section of `HANDOFF-2026-10-14-IOS-APPSTORE.md`.
Memory: `[[salehcard-ios-codemagic-plan]]`.

## Why
On 2026-10-29 App Review rejected iOS **1.0.6 (8)** under **Guideline 3.1.1**. Apple's two reasons:
- the app sells digital items, codes or currencies used in other apps or on third-party platforms;
- it sells gift cards redeemable for digital content on third-party platforms.

Apple's next step reads "review the app concept". Adding IAP doesn't fix this, because Apple doesn't let an
app sell other apps' currency. Prepaid mobile airtime is a real-world service (3.1.3(e)), so it can be sold
without IAP.

## What changed (app only, no API change)
- **iOS shows only the `telecom` root domain**: Alfa/Touch recharge, airtime, validity and u-share (50
  products on prod, all with `rootDomain: telecom`, checked 2026-11-08). Android is unchanged.
- **`app/lib/core/config/regulated_finance.dart`:**
  - `restrictToRealWorldServices` is true on iOS. It's a build-time platform check, never a remote flag
    (guideline 2.3.1).
  - Allowlist `iosAllowedRootDomains = {telecom}`, plus `catalogRootDomainScope`, `isRootDomainAllowed`
    (empty or unknown domains are hidden), `RegulatedCatalogFilter.categoryAllowed()`, and
    `isTopUpMethodHiddenOnRestricted` (crypto + Whish).
  - Debug-only `--dart-define=FORCE_IOS_SCOPE=true` applies the iOS scope on Android; release builds
    compile it out.
- **Catalog:** every product query is pinned server-side to `rootDomain=telecom`, so lists, search and
  totals are correct. Products are also filtered on the phone as a safety net, and a non-telecom product
  opens as not-found.
- Categories, offers and orders are filtered: game or gift-card orders don't appear in history on iOS.
  Profile hides saved player IDs.
- **Strings** (both platforms) no longer say "gift cards": promo, empty cart, search prompt, notification
  promo, and the item fallback label.
- **Branding on iOS:** new phone + lightning icon (`ios/Runner/Assets.xcassets/AppIcon.appiconset`),
  in-app mark and lockup (`assets/branding/logo_*_ios.png`), and native LaunchImage. Android keeps the
  gamepad. `flutter_launcher_icons` / `flutter_native_splash` are now `ios: false` so regenerating
  doesn't overwrite them.

## Release steps
1. Merge the PR to `main`. Codemagic `ios-release` builds automatically and uploads **1.0.6 (≥9)** to App
   Store Connect / TestFlight.
2. On an iPhone via TestFlight, signed in with the demo account, check:
   - home shows only telecom;
   - searching "pubg" finds nothing;
   - Offers and Orders show nothing outside telecom;
   - the profile has no player IDs;
   - top-up methods include neither USDT nor Whish;
   - the new icon and splash appear.
3. **Demo account** `appreview@flashcashglobal.com`:
   - Refund or cancel the game order **#311558** in admin.
   - Clear its saved player IDs.
   - Place one small Alfa/Touch order, so a finished purchase shows.
   - Top the balance back up.
4. Retake screenshots from the iOS build: iPhone 6.3", 1206×2622, 24-bit JPEG, no alpha.
5. Update the listing (below), attach the new build to version 1.0.6, and paste the review notes.
6. Reply to Apple's 10-29 message (below), then **Resubmit to App Review**.

## App Store listing (iOS)
- **Subtitle:** `Lebanon Mobile Recharge`
- **Primary category:** Utilities
- **Keywords:** `alfa,touch,mtc,recharge,top up,airtime,prepaid,lebanon,validity,ushare,mobile credit`
- **Promotional text:** Recharge your Alfa or Touch line in seconds.
- **Description:**
  > FlashCash Global lets you recharge prepaid Alfa and Touch mobile lines in Lebanon.
  >
  > • Recharge any Alfa or Touch number directly, or get a carrier recharge PIN to dial on your phone
  > • Airtime, validity and u-share bundles
  > • Pay from your account balance, topped up in cash or by bank transfer
  > • Track every recharge in your order history
  > • Arabic and English

## Review notes (App Review Information → Notes)
> FlashCash Global for iOS sells prepaid mobile airtime for the Lebanese carriers Alfa and Touch only.
> Airtime is a real-world telecom service consumed on the user's SIM card. The app does not sell digital
> content, codes, currencies or gift cards for use in other apps or platforms; the product categories
> mentioned in our previous review are not included in this build.
>
> • "Direct" products credit the phone number the user enters.
> • "Voucher" products deliver a carrier-issued recharge PIN that the user dials on their phone to add
>   credit to their line.
> • Purchases are paid from an account balance that customers top up outside the app (in cash at our
>   agents or by bank transfer). The balance can only be spent on mobile recharge. No digital content is unlocked inside this or
>   any other app.
>
> Demo account: phone 01 999 888, password [in DEPLOY-CREDS.local.md] (or email
> appreview@flashcashglobal.com). Identity verification is approved and the account has a balance. To test:
> Home → TELECOM → Alfa (or Touch) → choose a package → enter a number → Buy now.

## Reply to App Review (Resolution Center, message of 2026-10-29)
> Hello App Review Team,
>
> Thank you for your feedback. We have revised the app concept for iOS. FlashCash Global now offers only
> prepaid mobile recharge for the Lebanese carriers Alfa and Touch (airtime, validity and data bundles,
> and carrier recharge PINs). These are real-world telecom services consumed on the user's SIM card, not
> digital items, codes or gift cards used in other apps or on third-party platforms.
>
> We have removed all game top-ups, in-game items and gift cards from the iOS app, updated the app
> icon, description, keywords and screenshots to match, and attached a new build of version 1.0.6.
> Details for testing are in the App Review Information notes.
>
> Thank you for your guidance.

## If Apple objects again
- **To "codes" (voucher PINs):** remove the `alfa-voucher` / `mtc-voucher` slugs on iOS and keep direct
  recharge only. That means filtering those slugs in `RegulatedCatalogFilter._load()`.
- **Authorisation (5.3):** be ready to show that you're allowed to resell Alfa/Touch credit (a distributor
  agreement).
- **Never mention** crypto, money transfer or Whish in anything Apple sees.
