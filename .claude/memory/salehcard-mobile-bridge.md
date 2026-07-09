---
name: salehcard-mobile-bridge
description: "mobile_operator_apk_bridge — Android dual-SIM bridge app to auto-fulfill MTC-Touch/Alfa recharges from SalehCard orders; prototype, needs a Go-side command backend"
metadata: 
  node_type: memory
  type: project
  originSessionId: 69e9d3a3-d9a7-456b-88f3-88bcc3250e7a
---

`C:\Users\user\mobile_operator_apk_bridge` (separate repo, one commit "base version", 2026-07-06): Kotlin/Compose Android app that automates Lebanon mobile recharge fulfillment from an admin's dual-SIM phone (Touch + Alfa SIMs). Foreground service polls `http://10.0.2.2:3000/` for commands (Retrofit) + reports results over `ws://.../ws/bridge` (OkHttp WS). Commands: TRANSFER_CREDIT (SMS `{phone}T{amount}` → 1199/1399, chunked ≤3 USD), RECHARGE_LINE (Touch USSD `*300*{phone}#{card}`; Alfa SMS `{phone}R{code}` → 1313), CHECK-BALANCE (USSD `*220#`/`*11#`), SEND_SMS. Success detected by parsing incoming operator SMS replies.

Intended flow: customer buys recharge in the SalehCard Flutter app → Go API queues a command → bridge executes on SIM → result auto-completes the order (fits the existing `processing` → admin-complete model in [[salehcard-gtm-pr-stack]]).

**Live bring-up (2026-07-07)** — bridge code is MERGED to main + deployed; **`BRIDGE_ENABLED=true` set in prod**. First live-SIM test on a real Touch line. Full session detail in repo `HANDOFF-2026-07-07-BRIDGE-LIVE.md` (on branch `fix/bridge-balance-parse-and-sim-guard`, PR #54). Key findings/fixes this session:
- **THE root config bug (PR #54)**: Android `ConfigurationDTO.deviceId` was `Int` but the server sends the ObjectID hex **String** → Gson threw on EVERY `getConfig` → app silently ran on ALL hardcoded defaults (min-balance 20, default templates/fees), ignoring every server env knob. Fixed → `String`. This is why `BRIDGE_TOUCH_MIN_BALANCE=0.5` had no effect and transfers failed `2020` at a $2.50 balance. APK rebuilt + reinstalled; **retest pending**.
- Also PR #54: balance parser regex `\d{4}`→`\d{2,4}` (live SMS use 2-digit year `Exp:10-06-27`) — confirmed reads $2.50; and nullable-SIM guard (`lateinit alfaSim` crash → clean `9004 No alfa SIM`).
- **The test device (Galaxy S22, R3CT90MEVFM) is SINGLE-SIM (Touch only)** — Alfa impossible until a second Alfa line is inserted. ALFA product needs an Alfa SIM.
- Prod env set: `BRIDGE_ENABLED=true`, `BRIDGE_TOUCH/ALFA_MIN_BALANCE=0.5` (TEST values). Transfer template/shortcode (`{phone}T{amount}`→`1199`), fees, real min still uncalibrated defaults.
- Open: retry Touch transfer after the deviceId-fix reinstall (should pass pre-check now); calibrate transfer template; device poll-loop stalls (goes offline after bursts); `reportResult` 404s on UUID-id commands (server needs ObjectID). Product-save path fixed separately in **PR #52 (merged)** — see [[salehcard-flutter-apk-prod-url]]. **SECURITY: prod secrets (JWT_SECRET, MONGO_URI, MONTY/RAPIDAPI/Azure-storage/FCM keys) leaked into the session transcript by an errant `az appsettings list` — rotate.**

Status (2026-07-06): FULLY IMPLEMENTED on branch **`feat/mobile-bridge`** (pushed, 6 commits, off origin/main; PR still to open — `gh pr create` was permission-denied, open via GitHub UI). All 5 PRs done + verified:
- PR1 (7da00a5): product `BridgeSpec`{provider,method} + `Variant.FaceValue` + order `phone.go` (LB mobile normalize/validate) + PlaceOrder bridge validation.
- PR2 (9bad1f3): Go `api/internal/modules/bridge/` — per-device hashed bearer tokens (DeviceAuth), atomic command lease queue, idempotent result ingestion, reaper, dev stub; order `bridge_settler.go` (CompleteBridgeOrder/FlagBridgeOrder) + fulfillBridge dispatch/claim/compensate; wired in server.go/main.go; BRIDGE_* config.
- PR3 (219b1a6): admin Bridge page (devices+token+commands+retry) + product-editor bridge sub-option of account_credit.
- PR4 (752c1b2): Flutter checkout LB phone validation.
- PR5 (f751aa8): Android HTTP-only refactor (deleted websocket/, BridgeReporter, provisioning DeviceConfigStore, manifest+BootReceiver fixes, alfa balance, CHECK_BALANCE). Builds debug APK on AGP 9.0.1/Gradle 9.2.1/JDK21 (set JAVA_HOME=Adoptium jdk-21; local.properties sdk.dir needed, gitignored).
Verification: go build/vet/test green; admin pnpm build+lint clean; flutter analyze clean; APK assembles; **runtime smoke test passed** (BRIDGE_ENABLED=true BRIDGE_STUB=true: device create→token, GET /config returns correct camelCase DTO, poll/heartbeat/admin-list OK, 401 without token).
Deferred (BRIDGE-PLAN.md, need live SIMs): operator template/reply calibration, nullable-SIM hardening, EncryptedSharedPreferences, SMS pacing, Mutex reply router. Decisions locked: BOTH methods, HTTP-only, per-device hashed tokens. `bridge_device` FulfillmentMode + fulfillBridge park already on main via USDT PR #51.

Earlier status (2026-07-06): direction confirmed; code imported + BRIDGE-PLAN.md (051a2d3).

Earlier assessment (2026-07-06): the bridge is NOT strictly required to ship recharge products — SalehCard's existing manual flow (order → `processing` → admin recharges by hand → marks complete) works day one. The bridge is the automation layer. Its backend does NOT exist yet (the :3000 command/config/WS API is not in the salehcard repo — needs a Go `bridge` module or sidecar). Known gaps before production: auth is placeholder (`"hello world"` JWT, empty device token, plain ws/http), no command idempotency/ack (poll redelivery risk), in-memory balance state, single pending SMS-reply router (concurrency mis-route risk), Android SEND_SMS rate limits + flaky sendUssd API, sideload-only (Play Store won't accept SEND_SMS), operator-ToS/volume-flagging risk on consumer transfer lines.
