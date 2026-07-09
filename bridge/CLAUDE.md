# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

An Android app (`com.example.mobilebridgev2`) that turns a dual-SIM phone into a remote-controlled
bridge for two Lebanese mobile operators, **touch** and **alfa**. A backend dispatches commands
(send SMS, transfer credit, recharge a line, check balance); the phone executes them over the
cellular network via SMS and USSD, then reports results back. The phone effectively acts as a
hardware gateway the backend drives.

## Build & run

```bash
./gradlew assembleDebug          # build debug APK
./gradlew installDebug           # build + install on connected device/emulator
./gradlew test                   # JVM unit tests (app/src/test)
./gradlew connectedAndroidTest   # instrumented tests (needs device/emulator)
./gradlew lint                   # Android lint
```

Run a single unit test class:
```bash
./gradlew test --tests "com.example.mobilebridgev2.ExampleUnitTest"
```

Gradle is Kotlin DSL with a version catalog (`gradle/libs.versions.toml`). Note that Retrofit/OkHttp/Gson
are pinned directly in `app/build.gradle.kts`, **not** the catalog. minSdk 29, targetSdk/compileSdk 36.

The backend is assumed to run locally during development: REST at `http://10.0.2.2:3000/` and WebSocket
at `ws://10.0.2.2:3000/ws/bridge` (`10.0.2.2` is the host loopback from the Android emulator). These are
hardcoded — REST in `retrofit/RetrofitClient.kt`, WebSocket in `websocket/BridgeWebSocketClient.kt`.

## Architecture

The whole runtime lives in `BridgeForegroundService` (a sticky foreground service). `MainActivity` only
requests the telecom permissions and starts the service; it has no other role. The service launches three
concurrent coroutines on `Dispatchers.IO`:

1. **Polling loop** — every 5s, `GET /commands/poll` and routes each `CommandDTO` into one of two
   per-provider channels (`touchCommandsQueue` / `alfaCommandsQueue`).
2. **Command queue worker** — pulls one command at a time, alternating between the two provider queues
   (the `turn` flag) so neither operator starves, and dispatches it. **Commands execute strictly serially**
   — this is deliberate, because `SmsReplyRouter` can only track one pending provider reply at a time.
3. **Balance checks** — every 30 min, enqueues a synthetic `CHECK-BALANCE` command on the touch queue.

### Command flow

`CommandDispatcher.dispatch()` is a `provider × commandType` switch returning a `CommandResultDTO`.
The real work is in `CommandExecutor`:

- **SEND_SMS** — fire-and-forget text via `SmsManager` on the provider's SIM.
- **TRANSFER_CREDIT** — splits the amount into chunks of ≤3 (`splitAmount`), sends one templated SMS per
  chunk to the operator's short code, and **waits for an SMS reply per chunk** to confirm. Financial state
  (`touchSimBalance`) is only mutated after a confirmed success. Partial completion is a first-class outcome.
- **RECHARGE_LINE** — `rechargeTouch` uses a single-shot `sendUssd`. `rechargeAlfa` is
  **interactive USSD**: Alfa's recharge menu ends on a confirmation dialog that `sendUssdRequest`
  can't answer, so it dials via `ACTION_CALL` (surfacing the system USSD dialog) and drives the
  confirm step with `AlfaUssdAccessibilityService` (package `com.example.mobilebridgev2.ussd`),
  bridged back to the coroutine via `UssdSessionCoordinator`. Requires the accessibility service +
  "Display over other apps" enabled on the device; captures the operator's reply into `rawReply`.
- **CHECK-BALANCE** — sends a USSD code, parses balance + validity date out of the reply with a regex
  (`getBalanceFromReply`, expects a `USD <amount> Exp: <dd-mm-yyyy>` line).

### SMS reply correlation

Many operations are request/reply over SMS. `CommandExecutor` calls `SmsReplyRouter.prepareReplyWait(...)`
to register a `CompletableDeferred` for the expected sender, sends the SMS, then awaits the reply with a
30s timeout. Inbound SMS arrive at the `SmsReplyReceiver` broadcast receiver, which forwards every message
to `SmsReplyRouter.onSmsReceived`, which completes the pending deferred if the sender matches. **Only one
reply can be awaited globally at a time** (`prepareReplyWait` throws if one is already pending) — this is
why the command worker is single-threaded.

### Config & state

`ProviderStore` is a global singleton holding all per-provider configuration (USSD codes, SMS templates,
short codes, fees, minimum balances) and live SIM state (`SubscriptionInfo` for each SIM, current balance,
validity date). It ships with hardcoded defaults that are overwritten at runtime from `GET /config`
(via `fillProviderStore`) and from live WebSocket config pushes. SIMs are matched to providers by scanning
`carrierName`/`displayName` for `"alfa"` vs `"touch"`/`"mtc"` in `readActiveSimSubscriptions`.

### Networking

- **REST** (`retrofit/BridgeApi.kt`) — pull-based: config + command polling. Note several declared
  endpoints (`reportCommandResult`, `configure`, `requestCreditsForDevice`) are defined but unused; results
  actually go back over the WebSocket.
- **WebSocket** (`BridgeWebSocketClient`, a singleton object) — the live channel. Pushes command results,
  logs (`WebSocketLogDTO` with `LogErrorCodes`), and balance/credit demands to the backend; receives config
  updates and top-up notifications (`ReceivedMessageDTO`). Every send auto-reconnects if `isConnected` is
  false. Auth is a placeholder (`"hello world"` token).

### Result codes

`CommandResultDTO.statusCode` uses the integer ranges defined in `dto/CommandResultCodes.kt` (1000s SMS,
2000s credit transfer, 3000s touch recharge, 4000s alfa recharge, 5000s USSD/balance, 9000s general errors).
When adding a new outcome, allocate a code in the matching range rather than inventing ad-hoc numbers.

## Known inconsistencies (verify before relying on them)

- `AndroidManifest.xml` registers `.receiver.SmsReplyReceiver` and `.receiver.BootReceiver`, but
  `SmsReplyReceiver` actually lives in the `...sms` package and **no `BootReceiver` class exists** in the
  source tree (despite the `RECEIVE_BOOT_COMPLETED` permission and comments about restart-after-reboot).
  The manifest references will not resolve as written.
- `RetrofitClient` uses cleartext HTTP and an empty device token; both are marked `TODO` for production.

## Conventions

- Code is exclusively Kotlin with coroutines; networking is `suspend`-based.
- Telecom operations (`SmsManager`/`TelephonyManager`) are created per-SIM via
  `createForSubscriptionId(...)` — always target the correct SIM's subscription ID, never the default.
- Provider strings are lowercased before comparison; keep `"touch"`/`"alfa"` as the canonical identifiers.
