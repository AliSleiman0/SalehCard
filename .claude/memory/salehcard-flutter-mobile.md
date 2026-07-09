---
name: salehcard-flutter-mobile
description: New direction (2026-06-24) — build a Flutter mobile app on the existing Go API to replace the customer-facing /web storefront
metadata: 
  node_type: memory
  type: project
  originSessionId: 209e700a-92c7-4ee7-87b5-18458ee73f74
---

As of 2026-06-24, the plan changed: the customer-facing experience will be a
**Flutter mobile application** that consumes the existing Go + MongoDB backend
(`/api`). This **replaces the React storefront** (`/web`) as the customer client.

Scope/constraints stated by the user:
- Reuse the current backend (same API the web storefront calls — catalog, orders,
  wallet, auth). The admin console (`/admin`) is unaffected.
- The user has an existing design that is "not that complicated."

Decided (2026-06-24):
- **Location**: new top-level `/app` folder in the monorepo.
- **Architecture**: feature-first clean architecture — per feature `data/`
  (DTO↔JSON, remote datasource via Dio, repo impl) → `domain/` (entities, repo
  interface, usecases) → `presentation/` (Riverpod controllers, screens). Shared
  `core/` (network, storage, error, router, theme, i18n).
- **State management**: **Riverpod**.
- **Networking**: Dio + interceptors (Bearer header, 401→refresh→retry,
  error→Failure). Repos return Either<Failure,Entity>.
- **Auth transport**: **refresh token in JSON body** (NOT the httpOnly cookie).
  Requires a SMALL backend change in the `user` module: also return `refreshToken`
  in `AuthResponse` body, and have `POST /api/v1/auth/refresh` accept the refresh
  token from the body (fall back to cookie for web/admin). Mobile stores access +
  refresh tokens in `flutter_secure_storage`.
- **First version**: **walking skeleton** — one thin vertical slice end-to-end
  (login → product list → product detail) to prove the architecture, then expand.

Backend facts for the client (verified 2026-06-24 from api/internal/modules):
- Base path `/api/v1/`. Envelope: `{success, data?, error:{code,message}, meta?}`.
- Auth: `POST /auth/{register,login,refresh,logout}`; `GET/PATCH /users/me`.
  AuthResponse = `{accessToken, user}`. Bearer header on protected routes.
- Catalog: `GET /products` (filters: category, rootDomain, available, page, limit;
  meta pagination), `GET /products/{id}`, `GET /categories`.
  i18n fields (`title`, `name`, `description`) are `{en, ar, tr}` → Arabic = RTL.
- Orders: `POST /orders` (server re-prices; optional `Idempotency-Key`),
  `GET /orders`, `GET /orders/{id}`. Products carry dynamic `inputFields` schema
  (text/amount/quantity/select) → checkout renders a dynamic form.
- Wallet: `GET /wallet`, `POST /wallet/topups`.
- Order error codes to handle: OUT_OF_STOCK(409), INSUFFICIENT_FUNDS(402).
- CORS AllowedHeaders include `Idempotency-Key` (web only; native ignores CORS).

Toolchain (set up 2026-06-24):
- Flutter **3.44.3** / Dart **3.12.2** cloned to **C:\src\flutter** (on user PATH).
- Android Studio SDK at **%LOCALAPPDATA%\Android\Sdk** (platform android-36,
  build-tools 36.x, platform-tools); JDK21 (Adoptium) present. cmdline-tools is
  missing but not required to build. ANDROID_HOME persisted as user env var.
- Run target = **physical Android phone** (debug APK sideloaded). Base URL must be
  the dev machine's **LAN IP**: `http://192.168.10.170:8090/api/v1` (NOT localhost/
  10.0.2.2). Overridable via `--dart-define=API_BASE_URL=...`. Android manifest sets
  `usesCleartextTraffic=true` + INTERNET for dev HTTP over LAN. Windows Firewall must
  allow inbound 8090.

**Corporate-proxy Gradle/Java TLS fix (the build blocker):** the TLS-intercepting
proxy CA is trusted by the Windows store (so git/docker/flutter clone work) but NOT
by the JVM. `-Djavax.net.ssl.trustStoreType=Windows-ROOT` FAILS ("problem accessing
trust store" / "Could not initialize SSL context"). Fix that works: built an explicit
JKS truststore at **C:\Users\user\.android\flutter-cacerts.jks** (pass `changeit`)
seeded from JDK cacerts + all 56 certs in `C:\Users\user\.azure\win-ca-bundle.pem`,
and pointed the global **C:\Users\user\.gradle\gradle.properties** `org.gradle.jvmargs`
at it (that global file REPLACES, not merges, project jvmargs). Also set
`GRADLE_OPTS=-Djavax.net.ssl.trustStore=...flutter-cacerts.jks -D...trustStorePassword=changeit`
for the wrapper-download JVM. Project `app/android/gradle.properties` stays portable
(no machine paths). First build is slow (~Gradle dist + Maven through proxy).

**Device connectivity (IMPORTANT):** the dev PC is on wired **Ethernet** (Public
profile, 192.168.10.170) and the test phone (Samsung SM-S908N) on **WiFi**
(192.168.10.38). Despite the same /24, the WiFi and wired are on **isolated L2
segments/VLANs** — the phone CANNOT reach the PC's API over the LAN (firewall rules
are a red herring; opening 8090 changes nothing). **Solution that works: `adb reverse
tcp:8090 tcp:8090`** over USB — tunnels the phone's localhost:8090 to the PC. Build
the APK with `--dart-define=API_BASE_URL=http://127.0.0.1:8090/api/v1` and the app
reaches the API through the cable (re-run `adb reverse` after each reinstall; it
drops when USB disconnects). adb at `%LOCALAPPDATA%\Android\Sdk\platform-tools\adb.exe`.
Drive the UI headless via `adb shell input tap/text` + `screencap`/`pull`.

**Skeleton VERIFIED end-to-end on the physical phone (2026-06-25):** login
(POST /auth/login 200) → catalog (GET /products 200, real seeded products) →
detail (GET /products/{id} 200), auth guard redirect, and full RTL/LTR toggle
(العربية ⇄ English mirrors layout + Arabic strings; English fallback for products
without Arabic titles). Walking skeleton COMPLETE.

Note: some seed variant denominations look malformed (e.g. "40000-4e+12") — backend
data quality, not an app bug; revisit when wiring checkout.

**Design-module rollout (Claude Design → 6 modules; plan: `~/.claude/plans/use-the-claude-design-mcp-gentle-hartmanis.md`).**
Pattern: one general-purpose subagent builds a module in fresh context (analyze+test,
no commit/device); parent device-verifies on emulator/phone then commits. Auth bridge:
`signInWithPassword`/`signInDemo` maps the phone-login UI to seeded
`customer@salehcard.local`/`password123` → real JWT (form still validates phone+pwd
fields before calling demo login). Device verify caught 3 runtime bugs static checks
missed. Emulator `Medium_Phone_API_36.1` works too (boot + `adb reverse 8090`).
Progress on branch `feat/flutter-mobile-skeleton`:
- ✅ Foundation (a71fd1b): 5-tab StatefulShellRoute shell, cart Riverpod state, Home, core widgets.
- ✅ S1 Shop (13f1206 + 9d97a14): product detail/cart/checkout/success, dynamic inputFields form, Order types live in `features/checkout/` (entity+DTO+repo+`listOrders`).
- ✅ S2 Orders (0b4afbb): orders list (filter tabs) + full detail (completed/processing/refunded display-only + timeline); reuses checkout Order layer. Device-verified EN+AR.
- ✅ S3 Wallet (de7572b): `features/wallet/` — balance + ledger (GET /wallet), top-up (POST /wallet/topups; both card & USDT mock-approve INSTANTLY, no poll), send-money STUB behind TransferRepository (`// TODO(backend)`, one-file swap). Home wallet card + Add Money + account-menu Wallet entry wired. Device-verified: real top-up $0→$50 with live refresh, EN+AR.
- ✅ S4 Account (0e60f36): `features/account/` — profile (GET /users/me, PATCH /users/me with locale+savedPlayerIds only; email read-only, NO name field) + view/edit states (add/remove saved player IDs, sticky save). Extended shared auth `User`/`UserDto` with `savedPlayerIds`. Account-menu gained Profile entry + localized the hardcoded "Dark mode" → `darkMode`. Device verify caught a render bug: the Add button as a `SizedBox(height:50)` non-flex Row child threw "BoxConstraints forces an infinite width" → editor invisible; fixed by dropping the SizedBox and using `minimumSize: Size(72,50)` on the FilledButton. Verified real PATCH/GET round-trip, EN dark + AR dark(edit) + AR light(view), RTL correct.
- ✅ S5 Browse (7290e7e): `features/browse/` — categories grid (GET /categories?depth=0&withCounts=true → 8 root domains w/ live counts), client-side product search over the loaded catalog page (category tap prefills query; `// TODO(backend)` text-search param), notifications screen behind a stub repo (sample data; one-file swap at `notificationRepositoryProvider`). Repointed `/browse` shell tab to CategoriesScreen, deleted the now-unused foundation ProductListScreen. ICU plural for item counts (AR forms verified). Device-verified: real counts, search results/no-results/prefill, notifications populated, AR RTL grid (light+dark). Benign: category `image` URLs fail TLS handshake on emulator → errorBuilder falls back to initials chip (by design).
- ✅ S6 KYC (a6c75df): `features/kyc/` — FULLY STUBBED (no backend) behind a real `KycRepository`. Status screen renders all 4 states (unverified/pending/verified/rejected) via StatusBadge; form (name + doc-type radios + number + faux upload, submit-validates) flips the in-memory stub to pending on submit → status card. Account menu gained Verification entry. Stub state in memory; single initial-status line `_kInitialStatus` in kyc_repository_stub.dart (flip to preview cards); one swap point `kycRepositoryProvider`. Device-verified all 4 cards (flipped initial status for verified/rejected) + form→pending flow + AR RTL.
- ✅✅ ALL 6 DESIGN MODULES COMPLETE on branch `feat/flutter-mobile-skeleton` (Foundation a71fd1b → Shop → Orders → Wallet → Account → Browse → KYC a6c75df). Pattern held throughout: subagent builds in fresh context, parent device-verifies on emulator-5554 + commits app/ only (api/ working-tree changes left unstaged the whole time).
- ✅ MERGED TO MAIN (2026-06-26): pushed branch, then PR #6 (`Flutter mobile app: 6 design modules on the Go API`) merged to `main` (merge commit 9589675). The known main conflict (main's superseded standalone Home `a2a72f2` + product_chip/router/ARB) was resolved by taking the branch version ("ours") for all 8 conflict files; main's CI/CD `.github/` workflows preserved. Merged tree verified: build_runner/gen-l10n/analyze clean, 15 tests pass. NOTE: git remote `origin` still points at lowercase `salehcard`; the repo MOVED to `https://github.com/AliSleiman0/SalehCard.git` (push works via redirect; update with `git remote set-url` when convenient).
- Stubs still needing a backend later: KYC, send-money, notifications, customer text-search, order refund (display-only), USDT verify, profile name (each behind a real repo interface, one-file swap).
Deferred: push commits to remote; resolve main-merge conflict (main has a superseded standalone Home that clashes with the foundation shell Home).

Related: [[salehcard-design-port]] (the web storefront being replaced),
[[salehcard-dev-env]] (backend run instructions), [[salehcard-prod-deploy]].
