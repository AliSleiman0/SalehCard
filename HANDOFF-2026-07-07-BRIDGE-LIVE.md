# HANDOFF — 2026-07-07 — Bridge live-SIM bring-up + product-save fixes

Session goal drifted from "configure the ALFA bridge product" into a deep debug of the
whole product-edit → bridge-recharge path. Net: **4 fixes across 3 PRs**, the bridge flipped
**ON in prod**, and the **root cause of every "config doesn't apply" bridge symptom found and
fixed** (needs one reinstall + retest to confirm).

## Start here (next-session priority order)

1. **Merge #53 and #54.** Then **reinstall the newest bridge APK** (built with the deviceId
   fix) to the device, restart the app, and confirm server config now applies.
2. **Retry the Touch transfer.** With config applying, `min-balance=0.5` takes effect →
   pre-check should pass (`balance 2.5 ≥ 1 + 0.16 + 0.5 = 1.66`) → the app actually sends the
   transfer SMS. Then **calibrate the transfer shortcode/template** (next knob).
3. **Rotate the exposed prod secrets** (see Security below).
4. Address device reliability (poll loop stalls) + the `reportResult` 404 / UUID-command bug.
5. If ALFA is needed, put an **Alfa SIM** in the device (it's currently Touch-only).

## PRs from this session

- **#52 — MERGED + deployed** — `fix(products)`: admin product-save fixes.
  - Validation errors returned **400 (was 500)** — `writeProductError` in `product/handler.go`.
  - Editor **round-trips variant `id`** so saves stop rotating variant `_id`s (was orphaning
    carts → "unknown variant" — and reseller per-variant price overrides).
  - Bridge phone field **reconciled** (`admin/.../lib/bridgeFields.ts`): one `phone`-keyed text
    field, positioned first (the app derives the recharge number from the first field).
  - Verified E2E against Mongo. **Requires re-saving each bridge product once** to self-heal.
- **#53 — open** — `fix(bridge)`: gosimple `S1016` in `bridge/handler.go` heartbeat handler.
  This is the ONLY thing failing CI on `main` (pre-existing; build/vet/test pass). Merge to green CI.
- **#54 — open** — `fix(bridge)`: the Android app. THREE fixes:
  - **Balance parse**: regex required a 4-digit year; live SMS use 2-digit (`Exp:10-06-27`) →
    parse returned 0.00 → transfers blocked. Now `\d{2,4}`. **Confirmed working** (device reads $2.50).
  - **Missing-SIM guard**: `touchSim/alfaSim` were `lateinit` → crash on a single-SIM phone.
    Now nullable + `hasSim()` + dispatcher guard → clean `SIM_NOT_AVAILABLE` (9004). **Confirmed.**
  - **`deviceId` config-parse fix (THE BIG ONE, added last)**: `ConfigurationDTO.deviceId` was
    `Int`, but the server sends the ObjectID hex **String** → Gson threw on **every** `getConfig`
    → the app silently ran on **all hardcoded defaults** (min-balance 20, default
    templates/fees/shortcodes). This is why `BRIDGE_TOUCH_MIN_BALANCE=0.5` had **no effect** and
    the transfer kept failing `2020` at a $2.50 balance. Fixed `deviceId` → `String`.
    **⚠️ APK rebuilt with this — MUST reinstall to confirm.**

## Bridge state — confirmed vs blocked

**Confirmed working** (from live device + `GET /api/admin/bridge/{devices,commands}`):
- Touch balance parses → device reports `touchBalance: 2.5`, `touchValidity: 10-06-27`.
- `CHECK_BALANCE` (touch) → `5000 succeeded`. `CHECK_BALANCE` (alfa) → `9004 No alfa SIM detected`.
- Order → `TRANSFER_CREDIT` command created, leased, executed.

**Why the transfer still fails (2020 "Not enough Touch prepaid credit")**: the device was
running on defaults (min-balance **20**), so required = `1 + 0.16 + 20 = 21.16 > 2.5`. **Root
cause = the `deviceId` parse bug above.** After reinstalling #54's APK, config applies →
min-balance 0.5 → required 1.66 → should pass. (If it fails *then*, it's the transfer template.)

**Still open after the deviceId fix:**
1. **Transfer shortcode/template calibration** — `BRIDGE_TOUCH_TRANSFER_TEMPLATE` (`{phone}T{amount}`)
   + `BRIDGE_TOUCH_TRANSFER_DEST` (`1199`) are unverified defaults. Confirm the real Touch
   credit-transfer syntax on a live SIM. Same for fees + the real prod min-balance.
2. **Single-SIM device** — `R3CT90MEVFM` (Galaxy S22) has **only a Touch SIM** (phone telephony:
   `isMultiSimInserted: false`). Alfa recharges are impossible until a second (Alfa) line is added.
3. **Device reliability** — the app's poll/heartbeat loop **stalls after a burst** (device shows
   "offline"; commands sit `queued` with `attempts: 0`). Foreground service stays up but the loop
   dies. Needs hardening (supervise + restart the poll loop; keep-alive).
4. **`reportResult` 404 / UUID commands** — `BridgeReporter` repeatedly fails to POST results for
   commands whose id is a **UUID** (server route needs an ObjectID → 404). Find where UUID-id
   commands originate; treat a 404 as permanent (drop, don't retry).

## Prod config (Azure app settings → `salehcard-prod` / `salehcard-api`)

Set this session:
- `BRIDGE_ENABLED=true`
- `BRIDGE_TOUCH_MIN_BALANCE=0.5`, `BRIDGE_ALFA_MIN_BALANCE=0.5` — **TEST values**, calibrate for prod.

Unset → using code defaults (need calibration once config applies): `BRIDGE_TOUCH_TRANSFER_*`,
`BRIDGE_*_MESSAGE_FEE`, `BRIDGE_*_BALANCE_USSD`, `BRIDGE_SUCCESS/FAILURE_PATTERNS`. All are
**API env vars, no APK rebuild** — but they only take effect once #54 (deviceId) is installed.

## ALFA product

`fulfillmentMode=bridge_device`, `bridge={alfa, transfer_credit}`, variant `faceValue=1`, single
`phone` field first (reconciled). Saves cleanly now (#52). But ALFA = Alfa network → needs an
**Alfa SIM** in the device (see blocked #2). The Touch transfers being tested go to a Touch line.

## Customer app

Rebuilt release APK pointed at prod
(`--dart-define=API_BASE_URL=https://salehcard-api.azurewebsites.net/api/v1`), installed +
shared over WhatsApp. Default in code is a dev LAN IP — always build with the prod dart-define.

## SECURITY — rotate (leaked into this session's transcript)

An errant `az ... appsettings list` printed **all** app-setting values. Rotate:
`JWT_SECRET`, `MONGO_URI` (Cosmos password), `MONTY_ACCESS_TOKEN`, `RAPIDAPI_KEY`,
`AZURE_STORAGE_CONNECTION_STRING` (storage AccountKey), `FCM_CREDENTIALS_JSON` (private key).
Also an admin Bearer token was pasted by the user (short-lived, already expired).

## Worktrees / branches / build

- Worktree `C:\Users\user\salehcard-fix` holds all three fix branches. (Main dev worktree
  `C:\Users\user\salehcard` still on `feat/usdt-onchain-payments` with uncommitted USDT work —
  untouched.)
- Newest bridge APK: `C:\Users\user\salehcard-fix\bridge\app\build\outputs\apk\debug\app-debug.apk`
  (debug-signed, ~28.7 MB; installs over the existing one, keeps token/config).
- Bridge build: `JAVA_HOME="C:/Program Files/Eclipse Adoptium/jdk-21.0.11.10-hotspot"`,
  `cd bridge && ./gradlew.bat :app:assembleDebug`. `bridge/local.properties` needs
  `sdk.dir=C:/Users/user/AppData/Local/Android/Sdk` (gitignored — recreate per worktree).
- Install: `adb install -r <WINDOWS path>`; accept the USB-debugging prompt on the phone (it
  kept dropping to `unauthorized`/`offline`).
