# HANDOFF — Mobile Bridge (Lebanese recharge automation)

Last session: 2026-07-07. Status: **fully implemented, merged to `main`, one bridge
device installed and provisioned on a real phone.** Read `BRIDGE-PLAN.md` (repo root)
for the full design; this is the "where things stand + what's next" layer.

## What it is

Customer buys a Touch/Alfa mobile-recharge product in the SalehCard app → the order
parks `processing` → a dual-SIM Android **bridge** phone (in `/bridge`) polls the API,
executes the recharge via SMS/USSD, and reports back → the order auto-completes.
Failures fall back to the existing manual admin queue (a bridge failure **never**
auto-refunds — single money-out path preserved).

## Shipped (all on `main`, commits 7da00a5 → f751aa8)

- **PR1** `product.BridgeSpec{provider,method}` + `Variant.FaceValue`; `order/phone.go`
  (Lebanese normalize/validate); `PlaceOrder` bridge validation (single line, qty 1,
  valid number, face value for transfer_credit).
- **PR2** Go `api/internal/modules/bridge/`: per-device hashed bearer-token auth
  (`DeviceAuth`), atomic command lease queue, idempotent result ingestion, reaper,
  dev stub. `order/bridge_settler.go` implements `bridge.OrderSettler`
  (`CompleteBridgeOrder`/`FlagBridgeOrder`). `fulfillBridge` claims a scratch code
  for recharge_line, parks, dispatches, compensates on failure. Wired in
  `server.go`/`main.go`; `BRIDGE_*` config in `config.go` + `.env.example`.
- **PR3** Admin **Bridge** page (`/bridge`): register devices (token shown once),
  enable/rotate/remove, live online + SIM balances, command log + retry/cancel.
  Product editor: "Deliver via mobile-recharge bridge" is a sub-option of Account
  credit (operator/method selects + per-variant Transfer amount + auto-seeded phone).
- **PR4** Flutter checkout Lebanese-mobile validation (en/ar) + phone keyboard.
- **PR5** Android app HTTP-only refactor: WebSocket deleted; `BridgeReporter`
  (results with retry, heartbeat); `DeviceConfigStore` provisioning (URL + token);
  manifest/`BootReceiver` fixes; alfa-balance parity; `CHECK_BALANCE`.

Verification done: `go build/vet/test` green; admin `pnpm build`+`lint` clean; Flutter
`analyze` clean; bridge APK assembles; **runtime smoke test passed** (BRIDGE_ENABLED +
BRIDGE_STUB: device create→token, `GET /config` correct DTO, poll/heartbeat/admin OK,
401 without token).

## Current live state

- **API on `main`** auto-deploys to Azure. Bridge is **OFF by default**
  (`BRIDGE_ENABLED=false`) — prod behavior is unchanged until you set it true. Enable
  by setting `BRIDGE_ENABLED=true` in the Azure app settings (see DEVOPS-TODO).
- **One device is installed + provisioned** on a Samsung Galaxy S22 (SM-S908N) via ADB
  (debug APK). It has the server URL + token entered.
- **Uncommitted Whish work** lives on the *other* worktree
  (`C:/Users/user/salehcard` @ `feat/usdt-onchain-payments`) — untouched by this work.

## Deferred — needed before real recharge orders flow (BRIDGE-PLAN.md §14)

1. **Live-SIM calibration** (the big one): confirm the operator SMS/USSD templates,
   shortcodes, fees, and reply wording on real Touch/Alfa SIMs. All env-configurable via
   `BRIDGE_TOUCH_*` / `BRIDGE_ALFA_*` / `BRIDGE_SUCCESS_PATTERNS` / `BRIDGE_FAILURE_PATTERNS`
   — no APK rebuild needed to recalibrate.
2. Face-value denomination units (USD on both networks?) — confirm with owner.
3. Android hardening (deferred, low risk on the one controlled device): nullable-SIM
   guards (avoid lateinit crash on a single-SIM/wrong-carrier phone), EncryptedSharedPreferences
   for the token, client-side SMS pacing (Android ~30 SMS/30min ceiling), Mutex reply
   router (balance-check vs transfer reply collision), `applicationId` rename off
   `com.example.mobilebridgev2`.
4. `rawReply` is wired through the DTO but the CommandExecutor only populates it on some
   paths — thread the raw operator reply everywhere for full audit.

## How to operate / test

**Enable in prod:** set `BRIDGE_ENABLED=true` (Azure app settings). Register a device in
admin → Bridge, copy the one-time token.

**Dev stub e2e (no phone):** run the API with
`ENV=development JWT_SECRET= BRIDGE_ENABLED=true BRIDGE_STUB=true` — the reaper
auto-succeeds queued commands after `BRIDGE_STUB_DELAY`, so a bridge order created in the
app completes on its own. (`BRIDGE_STUB` is refused outside development by
`config.Validate`.)

**Provision the device app:** open MobileBridge → enter Server URL (`https://<api-host>`)
+ Device token (`bd_…`) → Save → Start → grant SMS/phone/notification perms. Disable
battery optimization for it (Samsung kills foreground services otherwise).

## Delivery / install gotchas (learned this session)

- The bridge app **can never go on the Play Store** (SEND_SMS/RECEIVE_SMS auto-rejected).
  It's always a sideload to the one operator phone. Firebase App Distribution does **not**
  remove the Play Protect prompt (SMS perms trigger it regardless) — not worth the setup.
- **Samsung Auto Blocker** (One UI 6.1+) hard-blocks *all* sideloads with no "install
  anyway" — the client must turn it OFF: Settings → Security and privacy → Auto Blocker.
  Then allow "Install unknown apps" for the source (e.g. WhatsApp) and dismiss Play Protect
  (More details → Install anyway). This was the actual blocker, not debug mode.
- Current APK is **debug-signed** → a future update with a different key forces
  uninstall (wipes the saved token). If ongoing updates matter, build a **release-signed**
  APK once (add a keystore + signing config) so updates install over the top. Not yet done.

## Build recipes (Android SDK + Adoptium JDK 21 are installed on this box)

```
# Bridge APK  (JAVA_HOME must point at a valid JDK 21; the env default was broken)
export JAVA_HOME="C:/Program Files/Eclipse Adoptium/jdk-21.0.11.10-hotspot"
cd bridge && ./gradlew.bat :app:assembleDebug
# → bridge/app/build/outputs/apk/debug/*.apk  (~28 MB; a local build got renamed to admin.Saleh.apk)
# local.properties (gitignored) needs: sdk.dir=C:/Users/user/AppData/Local/Android/Sdk

# SalehCard customer APK
cd app && flutter build apk --release
# → app/build/app/outputs/flutter-apk/app-release.apk  (~58 MB, debug-key signed)

# Install to a connected phone (adb.exe is native Windows — pass a WINDOWS path, not a
# git-bash C:/... path, or it "fails to stat"). Use PowerShell:
#   & "$env:ANDROID_HOME\platform-tools\adb.exe" install -r "C:\...\app-debug.apk"
```

## Key files

- Go: `api/internal/modules/bridge/{model,repository,service,auth,handler,admin,routes,reaper}.go`,
  `api/internal/modules/order/{bridge_settler.go,service.go(fulfillBridge),phone.go}`,
  `api/internal/config/config.go` (`BridgeConfig`), `api/internal/server/server.go`, `api/cmd/server/main.go`.
- Admin: `admin/src/features/bridge/`, `admin/src/features/products/pages/ProductEditPage.tsx`.
- Flutter: `app/lib/features/checkout/presentation/screens/checkout_screen.dart`.
- Android: `bridge/app/src/main/java/com/example/mobilebridgev2/` (net/BridgeReporter,
  config/DeviceConfigStore, retrofit/, service/, receiver/BootReceiver).

## Open items for next session

- [ ] Live-SIM calibration session (templates/fees/reply patterns) — blocks real orders.
- [ ] Flip `BRIDGE_ENABLED=true` in Azure once calibrated; create the first prod device.
- [ ] (Optional) release-signed bridge APK + keystore for clean remote updates.
- [ ] (Optional) Android hardening items in §3 above.
- [ ] Open/annotate the PR if a review trail is wanted (already merged to main directly).
