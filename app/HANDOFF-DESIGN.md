# SalehCard Mobile — Handoff

Gets the next session from a cold start to productive. **All 6 Claude Design
modules are built, device-verified, and merged to `main`.** This doc covers what
exists, what's still stubbed, where to go next, and how to run/verify.

---

## 1. Where we are (2026-06-26)

- **`/app`** is the Flutter customer client on the existing Go + MongoDB API — the
  replacement for the `/web` storefront. Clean architecture, Riverpod, go_router.
- **All 6 modules done + verified on the emulator + merged to `main`** via **PR #6**
  (merge commit `9589675`). Branch `feat/flutter-mobile-skeleton` still exists
  (not deleted). `main` carries the full app.
- Build pattern used throughout: a subagent builds each module in fresh context;
  the parent **device-verifies on the emulator** (EN/AR × light/dark × each state)
  and commits `app/` only. Device verification repeatedly caught runtime bugs that
  `flutter analyze` missed — **keep doing it**.

### Modules (all on `main`)
| Module | Feature dir | Backend |
|---|---|---|
| Foundation | shell, cart, home, phone-OTP auth | `/auth/otp/*`, `/auth/login-phone` (+ email login) |
| Shop | `features/checkout/` | `POST /orders`, `GET /products` |
| Orders | `features/orders/` (+ checkout Order types) | `GET /orders`, `GET /orders/{id}` |
| Wallet | `features/wallet/` | `GET /wallet`, `POST /wallet/topups` |
| Account | `features/account/` | `GET`/`PATCH /users/me` |
| Browse | `features/browse/` | `GET /categories?withCounts` |
| KYC | `features/kyc/` | none (fully stubbed) |

## 2. What's stubbed (the real next-work list)

Each stub sits **behind a real repository interface** — swapping to HTTP is a
one-file change at the named provider. Wire these when the backend endpoints exist:

| Stub | Swap point (provider) | Notes |
|---|---|---|
| **KYC** (all of it) | `kycRepositoryProvider` — `features/kyc/presentation/providers.dart` | `KycRepositoryStub` holds status in memory; needs real submit + status endpoints. |
| **Send money** (peer transfer) | `transferRepositoryProvider` — `features/wallet/presentation/providers.dart` | Form is real; stub returns `NOT_IMPLEMENTED`. No peer-transfer endpoint exists. |
| **Notifications** | `notificationRepositoryProvider` — `features/browse/presentation/providers.dart` | Stub returns sample data. Needs `GET /notifications`. |
| **Customer text search** | `features/browse/presentation/screens/search_screen.dart` | Client-side filter over the loaded catalog page. Needs a `GET /products?q=` param + a datasource method. |
| **Order refund** | display-only in order detail | Refund is admin/501; the refunded card is render-only. |
| **USDT verify** | — | Backend mock-approves card AND usdt top-ups instantly; no poll flow (by design). |
| **Profile name** | — | No backend field; profile omits a name (email read-only). |
| **OTP SMS delivery** | sender selection in `api/.../user/routes.go` (`internal/platform/sms`) | Phone-OTP auth is **real** now (`POST /auth/otp/request`, `/auth/otp/verify`, `/auth/login-phone`). Sending is config-gated: set `TWILIO_ACCOUNT_SID`/`TWILIO_AUTH_TOKEN`/`TWILIO_FROM` (preferred) or `SMS_GATEWAY_BASE_URL` (mip gateway); with neither set the code is logged to the API console (dev). |

## 3. Suggested next-session work (pick with the user)

1. **Wire a stub to a real backend** — most valuable is whichever endpoint the
   backend team ships first (notifications and customer search are the smallest;
   KYC and send-money are larger because they need new API surface).
2. **Retire `/web`** — the storefront this app replaces. Confirm parity first.
3. **Polish pass** — empty/error states, skeleton loaders, image placeholders,
   pull-to-refresh consistency, a real font (see §5).
4. **Release prep** — iOS build (untested), release signing, store metadata.
   (App icon + splash are done — branded gamepad mark, native + in-app splash.)
5. **Finish OTP rollout** — phone-OTP auth is implemented (the demo bridge is
   gone). Remaining: set live `TWILIO_*` creds so codes actually send, then
   deploy the updated API to Azure so the release APK can log in (it currently
   targets Azure, which lacks the new `/auth/otp/*` + `/auth/login-phone` routes).

Confirm scope with the user before starting — don't assume.

## 4. Architecture (keep it intact)

- **Per-feature clean arch**: `data/{datasources,dtos(.g.dart),repositories}` →
  `domain/{entities,repositories,usecases}` → `presentation/{providers,screens}`.
  Clone an existing feature (`features/wallet/` or `features/account/` are the
  cleanest recent examples). DTOs: `@JsonSerializable` + `part '*.g.dart'` + a
  `toEntity()`; after editing run `dart run build_runner build --delete-conflicting-outputs`.
- **Riverpod**: layered providers (remoteDataSource → repository → usecase →
  `FutureProvider.autoDispose` that throws the `Failure` on `Left`); `Notifier`
  controllers for submit/edit state (`{submitting, failure}`).
- **Networking**: Dio + `unwrap(response)` envelope helper; Bearer JWT interceptor;
  `Idempotency-Key` header on `POST /orders`.
- **Routing**: `StatefulShellRoute.indexedStack` (tabs `/home /browse /cart /account`)
  + top-level pushed routes (`/product/:id /checkout /order-success/:id /orders
  /orders/:id /wallet /wallet/* /search /notifications /kyc /kyc/form /profile`) in
  `core/router/app_router.dart`, with an auth `redirect` guard.
- **Design tokens**: `core/theme/app_tokens.dart` (`brandGradient`, `cta`, `accent`,
  `danger`, radii `rMd/rLg/rPill`) + `AppColors` ThemeExtension via `context.colors.*`;
  theme via `themeControllerProvider` (defaults dark).
- **No hardcoded user-facing strings or colors.** Strings → ARB +
  `AppLocalizations.of(context)` (`core/i18n/arb/app_{en,ar}.arb`, then
  `flutter gen-l10n`). Colors → `context.colors.*` / `AppTokens`. Money → `formatUsd()`.
- **RTL is real.** Use `EdgeInsetsDirectional`, `start/end`, logical alignment;
  Flutter mirrors from `localeControllerProvider`. Test every screen in Arabic.

## 5. Run from a cold start

> Flutter **3.44.3** at `C:\src\flutter` (on PATH; else
> `$env:Path = "C:\src\flutter\bin;" + $env:Path`). `ANDROID_HOME` =
> `%LOCALAPPDATA%\Android\Sdk`. adb at `%LOCALAPPDATA%\Android\Sdk\platform-tools\adb.exe`.

```powershell
# 1. Backend (repo root) — MongoDB on :27017, API on :8090 (web/.env already targets 8090)
cd C:\Users\user\salehcard\api ; $env:PORT='8090' ; go run ./cmd/server
#    (first run only) seed dev data: cd api ; go run ./cmd/seed

# 2. Verify gates
cd C:\Users\user\salehcard\app
dart run build_runner build --delete-conflicting-outputs ; flutter gen-l10n ; flutter analyze ; flutter test

# 3. Run on the EMULATOR (this session used emulator-5554 / Medium_Phone_API_36.1)
adb reverse tcp:8090 tcp:8090     # tunnels device localhost:8090 -> PC; re-run after any reconnect
flutter run --dart-define=API_BASE_URL=http://127.0.0.1:8090/api/v1 -d emulator-5554
```

**Login:** `customer@salehcard.local` / `password123` (the phone-login UI maps to
this seeded account). Session persists across restarts (token in secure storage);
locale + theme reset to defaults (English, dark) on a fresh launch.

**Physical phone** also works (SM-S908N): WiFi and the PC's wired Ethernet are on
**isolated L2 segments**, so the phone can't reach the API over the LAN — use the
same `adb reverse` over USB.

## 6. Device verification (how this session caught bugs)

`flutter run` is backgrounded (no stdin → no `r`/`R` hot reload from the harness);
to reload after a code change, kill `dart` and re-`flutter run` (incremental, fast).
Drive headless:
```powershell
$adb shell input tap X Y ; $adb shell input text "abc" ; $adb shell input keyevent 4  # back/hide-kbd
$adb shell screencap -p /sdcard/_s.png ; $adb pull /sdcard/_s.png dest.png
# downscale to width 460 with System.Drawing, then Read the PNG.
```
**Emulator is 1080×2400.** Screenshot downscaled to width 460 → **real X = downscaledX × 2.348**
(same for Y). Bottom nav sits at ~y 2266 (real). In RTL the bottom-nav order mirrors.
Keyboard shifts layout — account for it. **Check the `flutter run` log for runtime
exceptions** (`grep -iE 'exception|overflow|infinite'`) — that's what surfaced the
Account `_PlayerIdEditor` infinite-width crash.

## 7. Gotchas

- **Corporate-proxy TLS (Gradle):** builds need the JKS truststore at
  `C:\Users\user\.android\flutter-cacerts.jks` (configured globally in
  `~/.gradle/gradle.properties`). PKIX/SSL build errors → that's the cause. Do NOT
  use `Windows-ROOT` (it fails). See project memory / `DEPLOYMENT.md`.
- **autocrlf phantom files:** `core.autocrlf=true`, no `.gitattributes`. Some `api/`
  `.go` files can show as "modified" with **zero content change** (CRLF only) —
  `git diff --ignore-cr-at-eol` is empty. Don't commit those; `git checkout -- <file>`
  to clear. (Cleared this session, but they can reappear.)
- **git remote moved:** `origin` points at lowercase `salehcard`; the repo is now
  `https://github.com/AliSleiman0/SalehCard.git`. Push works via redirect; tidy with
  `git remote set-url origin https://github.com/AliSleiman0/SalehCard.git`.
- **Images** are remote URLs (`Image.network`) with `errorBuilder` fallbacks. On the
  emulator some category image hosts fail TLS handshake → the initials-chip fallback
  shows (by design, not a bug).
- **Money is USD-only** (`formatUsd()`). Some seed variant denominations are
  malformed (e.g. `40000-4e+12`) — backend data quality, not an app bug.
- **Commit hygiene:** stage only `app/` for app work; end commit messages with the
  `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>` trailer;
  never commit temp screenshots.

## 8. Reference docs
`CLAUDE.md` (orientation) · `CONVENTIONS.md` (module layout, design system) ·
`app/design/MANIFEST.md` (per-screen → API map) · plan at
`~/.claude/plans/use-the-claude-design-mcp-gentle-hartmanis.md` · project memory
`salehcard-flutter-mobile.md`.
