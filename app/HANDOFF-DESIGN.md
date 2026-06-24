# SalehCard Mobile — Design Handoff

Handoff for the **next session: applying the real design** to the Flutter app.
The functional walking skeleton is done and verified; this doc gets you from a
cold start to theming productively.

---

## 1. Where we are

- **`/app`** is a Flutter customer client on the existing Go API — the planned
  replacement for the `/web` storefront. Branch **`feat/flutter-mobile-skeleton`**,
  **PR #4** → `main` (open).
- **Walking skeleton COMPLETE & verified on a physical Android phone**: login →
  product list → product detail, auth guard, English/Arabic LTR↔RTL toggle. All
  API calls return 200 (`/auth/login`, `/products`, `/products/{id}`).
- **UI is intentionally un-styled** — neutral Material 3, no real design yet.
  That is the entire job for next session.

## 2. What's needed from the user (design inputs)

The user will provide **design screenshots**. Before building, get clarity on:
- Screens covered (at minimum: splash/login, product list, product detail).
- Color palette (primary/secondary/surface/error), light **and** dark? (skeleton
  is light-only today).
- Typography (font family — is there a brand font to bundle? Arabic font too).
- Component look: cards, buttons, list rows, badges (e.g. "Out of stock"), app bar.
- Spacing/radius/elevation tokens.
- Bottom nav / tab bar? (not in skeleton — would be new).

If a brand font is provided, drop files under `app/assets/fonts/`, declare in
`pubspec.yaml` `flutter: fonts:`, and wire into the theme's `textTheme`.

## 3. The design approach (keep the architecture intact)

Theming should be **centralized**, not scattered into widgets:
1. **`lib/core/theme/app_theme.dart`** — the single source of truth. Today it
   returns one `ThemeData` from a seed color. Expand it into a real `ThemeData`
   (ColorScheme, textTheme, `cardTheme`, `appBarTheme`, `filledButtonTheme`,
   `inputDecorationTheme`, etc.). Add `dark()` if the design needs it.
2. **Design tokens** — if useful, add `lib/core/theme/app_tokens.dart` (spacing,
   radii, brand colors) and reference them from `app_theme.dart`. Mirrors the
   `/web` CSS-variable design system conceptually, but as Dart constants.
3. **Shared widgets** — put reusable styled components in
   `lib/core/widgets/` (e.g. `PrimaryButton`, `ProductCard`, `AppBadge`) and use
   them in screens instead of restyling Material widgets inline. The skeleton's
   screens currently use raw `ListTile`/`Card`/`FilledButton` — replace those
   with the shared components as they're built.

**Do not** hardcode strings (use `AppLocalizations`/`l10n`) or colors (use
`Theme.of(context).colorScheme`) in screens.

## 4. File map (where the visible UI lives)

```
lib/app.dart                              # MaterialApp.router: theme/locale wired here
lib/core/theme/app_theme.dart            # <-- PRIMARY theming entry point
lib/core/i18n/arb/app_{en,ar}.arb        # UI strings (add new keys here, regen)
lib/core/format/money.dart               # USD formatting
lib/core/locale/locale_controller.dart   # en/ar toggle (drives RTL)
lib/features/auth/presentation/screens/login_screen.dart
lib/features/catalog/presentation/screens/product_list_screen.dart    # _ProductTile, _ErrorView
lib/features/catalog/presentation/screens/product_detail_screen.dart
```

Entities the UI binds to: `Product` (`title` is `I18nString`, `images`, `variants`,
`stock`, `available`, `fromPrice`, `rating`) and `Variant` (`denomination`, `price`).

## 5. Run it from a cold start

> Flutter is at **`C:\src\flutter`** (3.44.3). It's on the user PATH, but if a
> fresh shell can't find it: `$env:Path = "C:\src\flutter\bin;" + $env:Path`.
> `ANDROID_HOME` = `%LOCALAPPDATA%\Android\Sdk`. adb at
> `%LOCALAPPDATA%\Android\Sdk\platform-tools\adb.exe`.

```powershell
# 1. Backend (from repo root) — MongoDB must be on :27017 (already running normally)
cd C:\Users\user\salehcard\api ; $env:PORT='8090' ; go run ./cmd/server

# 2. Flutter analyze + tests
cd C:\Users\user\salehcard\app ; flutter analyze ; flutter test

# 3. Build + run on the phone (USB connected, USB debugging on)
#    Device networking: WiFi and the PC's wired Ethernet are on ISOLATED L2
#    segments — the phone CANNOT reach the API over the LAN. Use adb reverse:
adb reverse tcp:8090 tcp:8090
flutter run --dart-define=API_BASE_URL=http://127.0.0.1:8090/api/v1
#    (flutter run gives hot reload — ideal for design iteration. Re-run
#     `adb reverse` after any reconnect.)
```

For a standalone APK: `flutter build apk --debug --dart-define=API_BASE_URL=http://127.0.0.1:8090/api/v1`
→ `build/app/outputs/flutter-apk/app-debug.apk`.

**Login:** `customer@salehcard.local` / `password123`.

**Driving the UI headless** (no hands on phone): `adb shell input tap X Y`,
`adb shell input text '...'`, `adb shell screencap -p /sdcard/s.png` + `adb pull`.
Then `Read` the PNG. Coordinates are in **device** pixels (1080×2316 on the test
SM-S908N), independent of how the screenshot renders. Keyboard stays open and
shifts the layout — tap fields using keyboard-open coordinates.

## 6. Gotchas to remember while styling

- **RTL is real**, not string swaps. Test every screen in Arabic (top-bar toggle).
  Use `EdgeInsetsDirectional`, `start/end`, and logical alignment — never hardcode
  left/right. Flutter mirrors automatically from the active locale.
- **Corporate-proxy TLS:** Gradle builds need the JKS truststore at
  `C:\Users\user\.android\flutter-cacerts.jks` (configured globally in
  `~/.gradle/gradle.properties`). If a build fails with PKIX/SSL errors, that's
  the cause — see the project memory / `DEPLOYMENT.md` proxy notes. Don't use
  `Windows-ROOT` (it fails).
- **Images** are remote URLs (`Image.network`) with an `errorBuilder` fallback —
  keep a graceful placeholder in the design.
- **Money is USD-only**; use `formatUsd()`.
- **Data quirk:** some seed variant denominations are malformed (e.g.
  `40000-4e+12`). Backend data issue, not the app — don't design around it.
- After editing `.arb` files or any `*.g.dart`-generating change, run
  `dart run build_runner build` (DTOs) — l10n regenerates on build automatically.

## 7. Out of scope (future slices, not design)

Cart, checkout (dynamic `inputFields` form + `Idempotency-Key`), orders history,
wallet, registration, profile, multi-page pagination, iOS build, release signing.
Design the visual language now so these inherit it.
