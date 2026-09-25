# HANDOFF — FlashCash Global iOS build via Codemagic (2026-08-09)

New track of work, unrelated to the Play Store saga (see `HANDOFF-2026-08-07-PLAYSTORE.md` +
`[[salehcard-playstore-org-account-path]]` memory for that — Android is live: Alpha
closed-testing approved, Production submitted for review, Lebanon-only for now). This session
just scoped **iOS**; nothing iOS-side has been built yet. Owner has a physical **iPhone 17 Pro
Max** available for testing once a build exists (via TestFlight).

## TL;DR

| Item | State |
| --- | --- |
| iOS platform in the Flutter project (`app/ios/`) | 🔴 **does not exist** — never scaffolded. Confirmed via `HANDOFF-DESIGN.md` line 56: "iOS build (untested)" |
| Build machine | This dev box is **Windows** — Xcode (required to build/sign iOS) is macOS-only, no way around it |
| Chosen path | **Codemagic** cloud macOS CI — code stays on Windows/GitHub, Codemagic's cloud Mac runs `flutter build ipa` |
| Apple Developer Program account | 🔴 Not created yet — **owner action, $99/yr, blocking everything below** |
| Codemagic account | 🔴 Not created yet |
| Firebase iOS app registration | 🔴 Not done — project already has an Android client in Firebase project `salehcard-app` (see `salehcard-flashcash-app-rebrand` memory); an iOS client needs adding to the same project if push (FCM) should work on iOS too |
| Test device | ✅ iPhone 17 Pro Max, owner-side, ready for TestFlight once a build ships |

## Why Codemagic (decision made this session)

Compared free tiers for macOS build minutes — Codemagic wins clearly for this use case:

| Service | Free tier | Effective macOS build time/month |
| --- | --- | --- |
| **Codemagic** | 500 min/month on macOS M2 (personal account only, not Teams) | ~500 min — **picked** |
| Bitrise | 300 credits/month (Hobby); 1 credit = 30s macOS | ~150 min, capped at 1 private app |
| GitHub Actions | 2,000 min/month but macOS runs at a **10x** multiplier | ~200 min effective |

A single Flutter iOS build (compile + archive + export) is typically 10–20 min, so Codemagic's
free tier covers roughly 25–50 builds/month at no cost. It's also the CI most purpose-built for
Flutter (first-class `codemagic.yaml` support, Flutter version pinning, built-in code-signing
management via Apple Developer Portal API integration — no manual cert/profile wrangling
required like the other two).

## Exact next steps

1. **Owner action (blocking, do first): enroll in the Apple Developer Program**
   ($99/yr) at developer.apple.com. Needed before anything else — App Store Connect,
   provisioning, and TestFlight all require it. Use the Entrizo org identity if this should be
   an organization-level Apple account (parallels the Play Console org-account decision), or an
   individual account if that's not needed for iOS. Confirm with owner which is intended.
2. **Scaffold the iOS platform in the Flutter project.** From a machine with Flutter installed
   (this Windows box is fine for this step — it's just file generation, not compilation):
   ```bash
   cd app
   flutter create --platforms=ios .
   ```
   This generates `app/ios/` (Xcode project, `Info.plist`, `Runner.xcodeproj`, etc.) without
   touching existing Android files. Verify it doesn't collide with anything — check
   `pubspec.yaml`'s `version: 1.0.3+4` carries through, and decide the iOS bundle identifier
   (recommend mirroring Android's `com.flashcashglobal.app` for consistency, though iOS/Android
   bundle IDs are independent and don't have to match).
3. **Create a Codemagic account**, connect it to the GitHub repo (`AliSleiman0/SalehCard`).
   Codemagic can auto-detect the Flutter project once `app/ios/` exists.
4. **Set up code signing in Codemagic.** Two options — Codemagic's own docs recommend
   letting it manage signing automatically via an Apple Developer Portal API key (App Store
   Connect → Users and Access → Integrations → generate a key with Admin access), which lets
   Codemagic auto-create/renew certificates and provisioning profiles. Avoids manual
   `.p12`/`.mobileprovision` handling entirely.
5. **Register an iOS app in the Firebase project** (`salehcard-app`, same project as the
   existing Android client) if push notifications should work on iOS — Firebase Console →
   Project settings → Add app → iOS, using the bundle ID chosen in step 2. Download
   `GoogleService-Info.plist` and add it to `app/ios/Runner/` (or wire through
   `flutterfire configure` if that's already the pattern used for Android's
   `firebase_options.dart` — check how that file was generated originally).
6. **Write `codemagic.yaml`** at the repo root (or `app/codemagic.yaml` — confirm which
   Codemagic expects given the monorepo layout, since `/app` is a subdirectory not repo root).
   Minimal Flutter iOS workflow: `flutter build ipa --release`, then publish to TestFlight via
   Codemagic's App Store Connect integration (needs the same API key from step 4, or a
   dedicated App Store Connect API key with appropriate scope).
7. **First build**: trigger it, fix whatever breaks (this is genuinely untested — expect iOS
   permission strings (camera/photo library for KYC docs, notifications) to need adding to
   `Info.plist`, and possibly platform-specific code paths in the Flutter codebase that assume
   Android).
8. **TestFlight**: once a build lands in App Store Connect, install via TestFlight on the
   iPhone 17 Pro Max and do a real smoke test — login/OTP, catalog browse, checkout, wallet.
9. Only after that: App Store Connect listing (screenshots, description, age rating,
   privacy nutrition labels — Apple's equivalent of Play's Data Safety form) and first
   submission for Apple review.

## Open questions for the owner (surface before deep work)

- Individual or organization Apple Developer account? (Mirrors the Play Console org-account
  question already resolved for Android — see `[[salehcard-playstore-org-account-path]]`.)
- Should the iOS bundle ID mirror `com.flashcashglobal.app`, or does it not matter?
- Is push notification support on iOS in scope for v1, or can that be deferred (skips step 5)?
- Financial-features declarations exist on Play; Apple has its own App Review Guidelines around
  crypto/financial apps (App Store Review Guideline 3.1.5 covers crypto exchanges/wallets) —
  worth a pass through those once the build exists, separate from Google's requirements already
  documented in `[[salehcard-playstore-org-account-path]]`.

## Reference

- Android/Play Store status (separate, already live): `HANDOFF-2026-08-07-PLAYSTORE.md`,
  `[[salehcard-playstore-org-account-path]]` memory.
- Firebase project: `salehcard-app` (existing Android client per
  `salehcard-flashcash-app-rebrand` memory, appId `1:184899958988:android:d242dff62ae6af75742a69`).
- Flutter app: `app/pubspec.yaml` — `version: 1.0.3+4`, Dart SDK `^3.12.2`.
- `app/HANDOFF-DESIGN.md` line 56 — original note flagging iOS as untested/deferred.
- Codemagic docs: https://docs.codemagic.io/ (pricing: https://docs.codemagic.io/billing/pricing/)
