# HANDOFF — FlashCash Global Play Store submission (2026-07-25)

Session state for getting the **FlashCash Global** Android app (`flashcash.global`) into
Google Play **closed testing**. The binary, screenshots, feature graphic, and store copy are
all done. The app is currently **REJECTED** on Play for reasons that are **all store-listing /
account settings — not the binary**. This doc is the orientation for finishing that.

Related memory: `[[salehcash-flashcash-app-rebrand]]`, `[[salehcash-android-build-machine]]`.
Store copy lives in `PLAYSTORE-LISTING.md`; build recipe in `BUILD-MACHINE.md`.

---

## TL;DR — what's done vs. what's blocking

| Item | State |
| --- | --- |
| Rebrand SalehCard → FlashCash Global (api/web/admin/app) | ✅ shipped to main + prod (PR #105, commit e85c35b) |
| Signed release AAB **1.0.3+4** (`flashcash.global`) | ✅ built, at `~/Downloads/flashcash-release/2026-07-24/flashcash-global-1.0.3+4.aab` |
| Real in-app screenshots (phone + tablet) | ✅ `~/Downloads/flashcash-release/store-screenshots/{phone,tablet}/` |
| Feature graphic 1024×500 | ✅ `~/Downloads/flashcash-release/store-screenshots/feature-graphic.png` |
| Store copy (short + full, EN + AR), ranking-keyword clean | ✅ `PLAYSTORE-LISTING.md` (commit 4ce1181) |
| **Rejection A** — screenshots = "marketing illustrations" | ✅ FIXED (real captures) |
| **Rejection B** — feature graphic placeholder + description names brands | 🟡 graphic FIXED; description needs live-text check (see below) |
| **Rejection C** — Play Console Requirements: financial features → org account | 🔴 **OPEN — blocking, decision needed from owner** |

---

## The build (no new build needed)

- **Upload this:** `C:\Users\Admin\Downloads\flashcash-release\2026-07-24\flashcash-global-1.0.3+4.aab`
  (57.1 MB / 59,879,307 bytes). Identical copy at
  `app/build/app/outputs/bundle/release/app-release.aab`.
- Package `flashcash.global`, version **1.0.3 (versionCode 4)**, **upload-key signed**
  (signer SHA1 `A1:AF:3B:…:6F:AB`, matches the 1.0.2+3 signer — verified).
- Built with `--dart-define=API_BASE_URL=https://salehcard-api.azurewebsites.net/api/v1`
  (prod). **Never omit** the dart-define — default is a dead dev LAN IP.
- None of the three rejections require a rebuild. Only bump the version if you actually
  change app code.

Rebuild recipe (if ever needed): see `BUILD-MACHINE.md`. Keystore + `key.properties` are on
this box under `app/android/` (gitignored).

---

## The three Play rejections

### A. Screenshots were "marketing illustrations instead of in-app experience" — FIXED
Replaced with **real captures** from the running app (local build + seeded demo data):
`store-screenshots/phone/` (1080×1920) and `store-screenshots/tablet/` (1200×1920), 4 each:
`1-login → 2-home → 3-product → 4-wallet`. All **24-bit RGB, no alpha** (Play rejects 32-bit
RGBA). Same tablet set is valid for both the 7" and 10" slots. See `store-screenshots/README.md`.

### B. Feature graphic placeholder + description names third-party brands
- **Feature graphic — FIXED.** Generated `store-screenshots/feature-graphic.png`, exactly
  **1024×500, 24-bit RGB**: real app home screen on the right, FlashCash logo mark + "Flash**Cash**
  Global" wordmark + tagline + bilingual pill on the left. Genuinely shows the in-app experience.
  Regenerate with `scratchpad/feature_graphic.ps1` (System.Drawing) if a different look is wanted.
- **Description brand mention — NEEDS A CHECK.** The full description currently in
  `PLAYSTORE-LISTING.md` (commit 4ce1181) **names no brands** (says "mobile titles" / "the
  stores you use"). So the text **live in Play Console is probably an older draft** that named
  games (PUBG / Free Fire) or a store. **Action:** compare the live Console text to 4ce1181; if
  it's old, paste in the current clean copy. If you *want* to name games (converts better), add:
  *"FlashCash Global is an independent retailer and is not affiliated with, endorsed by, or
  sponsored by any of the games or brands mentioned."*
- Also earlier: a **short-description promotion warning** ("keywords that indicate ranking") —
  caused by the literal string **"top"** inside "top-up". Fixed by rewording to "game credit" /
  "recharge" throughout EN + AR (commit 4ce1181). Current short desc (74 chars):
  `Gaming gift cards, game credit and mobile recharge, paid from your wallet.`

### C. 🔴 Violation of Play Console Requirements → organization account (BLOCKING)
Google flagged the app as offering **financial services** (their list: banking, loans, stock
trading, **cryptocurrency wallets/exchanges**). Since **2024-08-31**, a *new* developer account
providing those must be an **organization** account (registered business + D-U-N-S), not an
individual. Triggered by **either the app category (Finance) or a Financial-features declaration**
(most likely **cryptocurrency** — the app has a USDT/crypto rail in its history — or the wallet
read as banking).

**Two paths — this is the owner's decision, not a copy tweak:**

- **Path A (recommended, matches what the app actually is):** reposition as a **digital-goods
  store**. The wallet is **prepaid, closed-loop, no cash-out** = store credit, not a regulated
  product. Steps: set **app category → Shopping** (not Finance); in **App content → Financial
  features** declare accurately (do **not** check cryptocurrency / banking / loans / investment).
  Valid **only if crypto/USDT is genuinely off in this build** (it is, per the listing decision).
  Then resubmit as an individual.
- **Path B:** if crypto / money-transfer are meant to be real, live financial services → must
  register an **organization** account and transfer the app to it. Bigger process.

**Info still needed from the owner to give exact click-steps** (asked, not yet answered):
1. **App content → Financial features** — what is currently declared/checked?
2. **Store settings → App category** — is it set to *Finance*?

---

## Reviewer / submission details (for Play Console)
- **Privacy policy URL:** `https://salehcard-api.azurewebsites.net/privacy` (rebranded EN+AR, live).
- **Reviewer sign-in:** app uses **phone-OTP** login (find-or-create). Provide the reviewer a
  test phone + note that a one-time SMS code is required. In prod SMS is real (Monty). Coordinate
  a reviewer test number/flow before submitting — do **not** hand out a real admin number.
- Content rating: intended **18+**.

---

## Other open items (flagged, not yet requested to fix)
- **Push notifications are dead** on `flashcash.global` until the Google **API-key application
  restriction** is widened to the new package. Needs the **Play App Signing SHA-1** (from Console
  after first upload) + `gcloud` — **not doable on this box**. See `[[salehcash-flashcash-app-rebrand]]`.
- **Prod data leftovers still say "SalehCard"** (data, not code — edit in the admin console, no deploy):
  - `app_settings.storeName` → **Settings**
  - `topup_methods` instructions → **Wallet / top-up methods**
- **Splash wordmark PNG** (`app/assets/branding/logo_lockup.png`) still reads "SALEH CARD" —
  needs new artwork (the logo *mark* is fine; only the lockup text is stale).

---

## Environment notes
- This Windows box is a working **Android build machine** as of 2026-07-24 (Flutter 3.44.8, JDK 17,
  Android SDK + NDK r28c, emulator + AEHD). Big downloads must use `curl.exe`, not IWR/sdkmanager.
  Emulator software GPU (swiftshader) is unstable. Full detail: `BUILD-MACHINE.md` +
  `[[salehcash-android-build-machine]]`.
- ⚠️ **Prod-safety note:** during screenshot capture, an emulator autofill once triggered a *Send
  code* against **prod** with the admin's phone number. Since then all screenshot/login testing is
  done against the **local API** (`SMS_PROVIDER=log`, read the code from the server log). Do not log
  into prod or create prod accounts on the owner's behalf.
