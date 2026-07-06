# SalehCard Mobile Bridge — Full Implementation Plan

## Context

The owner fulfills Lebanese mobile recharges (MTC Touch / Alfa) from his own SIM lines. An Android "bridge" app (now in this repo at **`/bridge`**, moved from a standalone prototype repo) automates this: it polls a backend for commands, executes them via SMS/USSD on a dual-SIM phone, and reports results. **The backend it polls does not exist** — this plan builds it into the SalehCard Go API and fixes every identified gap so the full loop works: customer buys a recharge product in the Flutter app → order parks `processing` → bridge command queued → device executes → order auto-completes (failures fall back to the existing manual admin queue).

**Decisions locked with the user**: both recharge methods (credit transfer by amount AND scratch-card codes from existing inventory); HTTP-only transport (delete the WebSocket on both sides — the Go API has zero WS infra and polling is its established pattern); minimal admin "Bridge" page.

**Verified ground truth**: `OrderItem.FulfillmentMode` with `bridge_device`, `dispatchFulfillment` → `fulfillBridge` (currently parks with "queued for bridge device"), guarded `TransitionStatus`, the `OrderSettler`/`SetOrderSettler` port pattern, `code.ClaimForOrder`, and ticker-worker lifecycle are **all committed on `origin/main`** (USDT PR #51 merged). The dirty working tree is the *separate uncommitted Whish work* — bridge work branches off `origin/main`; overlaps with Whish are additive blocks in 4 shared files (`config.go`, `server.go`, `cmd/server/main.go`, `order/service.go`), trivially rebased.

---

## Architecture

- A bridge product = `fulfillmentType: account_credit` + `fulfillmentMode: bridge_device` + new `Product.Bridge *BridgeSpec{Provider: touch|alfa, Method: transfer_credit|recharge_line}` (+ `Variant.FaceValue *float64` for transfer method).
- `PlaceOrder` validates the phone; `fulfillBridge` parks the order `processing`, (recharge_line) claims one scratch code from existing inventory, and creates **exactly one** `bridge_command`.
- Device polls with an atomic server-side **lease**, executes, POSTs the result idempotently. Bridge service completes the order through an `OrderSettler` port (same effects as admin manual completion: `TransitionStatus(processing→completed)` + `fulfillment.transferRef` + notify + loyalty) or **flags** it (timeline event only, order stays `processing` in the manual queue).
- **Money-out invariant preserved: bridge failures never auto-refund.** Admin completes manually or uses the existing refund endpoint.
- Dependency direction mirrors payment: `order` holds a narrow `bridgeDispatch` port; `bridge` defines `OrderSettler` locally; `server.go` closes the loop with `bridgeSvc.SetOrderSettler(orderSvc)`. `bridge` never imports `order`.
- A **reaper** worker (same shape as `payment.Watcher.Run`) requeues expired leases and fails exhausted/stale commands.

### Command state machine

```
queued --(poll: atomic lease)--> leased --(result success)--> succeeded  [terminal]
  |  ^                             \--(result failure/partial)--> failed --(flag order once)
  |  \--(reaper: lease expired, attempts < max)
  |--(reaper: attempts == max, or queued > BRIDGE_QUEUE_TIMEOUT)--> failed
  \--(admin cancel: queued|leased)--> cancelled;  admin retry: failed|cancelled --> queued (attempts reset)
```

- Lease = single `FindOneAndUpdate` (`status:queued`, provider ∈ device.Providers, oldest first) → sets `leased`, `deviceId`, `leaseExpiresAt = now+BRIDGE_LEASE_TTL`, `$inc attempts` — same idiom as `code.ClaimOne`/`TransitionStatus`.
- Result ingestion = `FindOneAndUpdate` on `{_id, status:leased, deviceId}`; already-terminal → `{"status":"duplicate"}` 200 so device retries converge.
- Partial transfer (code 2001, chunked SMS partially sent) is terminal-failed + flagged; **never auto-retried** (would double-send completed chunks).
- **Scratch-card conservatism**: code claimed (`available→delivered`, deliveredTo `bridge:<phone>`) BEFORE command creation; once a command exists the code is never auto-released (a timed-out USSD may have consumed it). Only the pre-dispatch window (claim ok, command insert failed) releases via `fulfillBridge` compensation. Admin **retry re-sends the SAME code — never claims a new one**.

### Device auth

Per-device opaque bearer tokens, stored hashed: admin registers a device → 32 random bytes → `bd_<base64url>` shown **once**; only `sha256(token)` persisted. `DeviceAuth` middleware hashes the presented bearer and does an indexed `tokenHash` lookup (device must be enabled), putting `*Device` in context. Chosen over role-JWT: instantly revocable per device, no expiry/refresh choreography a headless phone can't recover from interactively. Device audit entries set `ActorID: "bridge:"+deviceID` explicitly (no JWT claims in context).

### HTTP contract

Device routes at `/api/v1/bridge` (outside `AuthRequired`; `DeviceAuth` + rate limit keyed `bridge:<deviceID>` per `payment/routes.go` pattern). Admin routes under existing `AdminOnly` group. All JSON camelCase (matches Gson's default field naming — no annotations needed).

| Method | Path | Body → Response |
|---|---|---|
| GET | `/api/v1/bridge/config` | — → ConfigDTO |
| GET | `/api/v1/bridge/commands?max=N` | — → `[CommandDTO]` (atomic lease, default 1, cap 3) |
| POST | `/api/v1/bridge/commands/{id}/result` | ResultDTO → `{"status":"recorded"\|"duplicate"}` |
| POST | `/api/v1/bridge/heartbeat` | balances/SIM presence/appVersion → 204 |
| POST | `/api/v1/bridge/logs` | `{"entries":[...]}` → 204 |

- **ConfigDTO** = superset of the app's 16-field `ConfigurationDTO` (USSD/SMS templates with `{phone}`/`{amount}`/`{card}`/`{code}`, shortcodes, fees, min balances) **plus**: `pollIntervalSeconds`, `heartbeatIntervalSeconds`, `successMatchPatterns[]`, `failureMatchPatterns[]`, `maxSmsPerHalfHour`, and `alfaSimBalance`/`alfaSimValidityDate` alongside the touch pair (balances echo last heartbeat → reinstall restores state). Dates ISO `yyyy-MM-dd` end-to-end. `deviceId` = hex string (identity comes from the token; drop `?DeviceId=`).
- **CommandDTO**: `{commandId, provider, commandType: TRANSFER_CREDIT|RECHARGE_LINE|CHECK_BALANCE|SEND_SMS, recipientNumber, amount, cardCode, message, attempt, leaseExpiresAt}` (rename `CHECK-BALANCE` → `CHECK_BALANCE`).
- **ResultDTO**: existing `CommandResultDTO` fields + **`rawReply`** (always sent when an operator reply exists — persisted for audit), `transferredAmount`, `executedAt`. Server owns a Go mirror of `CommandResultCodes.kt`: `x000` = success, `2001` = partial, else failure.
- **Admin endpoints**: devices list/create(token once)/rotate-token/patch/delete; commands list (paged, filtered)/retry/cancel; enqueue CHECK_BALANCE; device logs. All mutations audited (`bridge.device_create`, `bridge.token_rotate`, `bridge.command_retry`, `bridge.command_cancel`, …). Admin JSON masks `cardCode` to last-4.

---

## Go bridge module — `api/internal/modules/bridge/`

Standard module split + `auth.go` + `reaper.go`. Collections: `bridge_devices`, `bridge_commands`, `bridge_logs` (TTL 30d).

- **model.go** — `Device{ID, Name, TokenHash, Providers []string, Enabled, LastSeenAt, TouchBalance/TouchValidity, AlfaBalance/AlfaValidity, AppVersion, timestamps}`; `Command{ID, OrderID *ObjectID, Provider, Type, Recipient, Amount *float64, CardCode (json:"-"), Message, Status, Attempts, DeviceID, LeaseExpiresAt, Result *CommandResult, FailReason, OrderFlagged, timestamps}`; `CommandResult{StatusCode, TransferredAmount, BillingAmount, Balance, ValidityDate, RawReply, ErrorMessage, ExecutedAt}`; `DispatchInput{OrderID, Provider, Method, Phone, Amount, CardCode}`; `IsSuccessCode`/`IsPartialCode`.
- **repository.go** — `EnsureIndexes` (unique `tokenHash`; commands `{status,provider,createdAt}`, `{orderId}`, `{deviceId,updatedAt}`; logs TTL). `DeviceRepo` (CRUD, `FindByTokenHash`, `Touch`); `CommandRepo` (`Create`, `LeaseNext(deviceID, providers, ttl, max)`, `CompleteLeased(id, deviceID, terminalStatus, res, failReason)` → `ErrConflict` when already terminal, `RequeueExpired(now, maxAttempts) ([]*Command,...)` returning newly-failed for flagging, `FailStaleQueued`, `Retry`, `Cancel`, `FindByOrder`, `ListAll(filter, pagination)`).
- **service.go** — `OrderSettler{CompleteBridgeOrder(ctx, orderID, transferRef) error; FlagBridgeOrder(ctx, orderID, reason) error}` (implemented by `*order.OrderService`); `Service`: `Enabled`, `DispatchOrder(ctx, DispatchInput)`, `Poll`, `IngestResult` (classify → `CompleteLeased` → settler complete/flag; settler errors logged, calls idempotent), `BuildConfig`, `Heartbeat`, `ReapTick`. **Stub mode** (`BRIDGE_STUB=true`, dev-only): ReapTick auto-succeeds queued commands after `BRIDGE_STUB_DELAY` through the same `IngestResult` path (mirrors tron/whish stub philosophy — exercises the full completion pipeline).
- **auth.go** — `GenerateToken`, `HashToken`, `DeviceAuth(repo)` middleware, `DeviceFromContext`.
- **handler.go / admin.go / routes.go** — `RegisterRoutes(r, db, cfg) *Service` (EnsureIndexes + mount `/api/v1/bridge`); `RegisterAdminRoutes(r, db, rec)`.
- **reaper.go** — `NewReaper(svc, interval)`, `Run(ctx)` identical in shape to `payment/watcher.go:30` (immediate first tick, `select ctx.Done()/ticker.C`).
- **Wiring** — `internal/server/server.go`: construct bridge before order, pass to `order.RegisterRoutes(...)`, `bridgeSvc.SetOrderSettler(orderSvc)`, `s.bridgeReaper` field + getter; `cmd/server/main.go:62`: third worker block beside Watcher/WhishSweeper (WaitGroup + workerCtx).

## Order module changes — `api/internal/modules/order/`

- New port on `OrderService`: `bridgeDispatch{Enabled() bool; DispatchOrder(ctx, bridge.DispatchInput) error}` (nil = today's park-only behavior); `routes.go` gains the param.
- **PlaceOrder validation** (mode == `bridge_device`): valid `Bridge` spec required; **single line, qty 1** (avoids multi-command partial ambiguity); phone from `PlayerID`/first field, normalized (`+961|961|00961` → local) and validated `^(03|70|71|76|78|79|81)\d{6}$` in new `order/phone.go`; transfer method requires variant `FaceValue > 0`.
- **`fulfillBridge`** (service.go:529): re-read product for spec+face value; recharge_line → `codes.ClaimForOrder(pid, orderID, "bridge:"+phone, 1)`; **park first** ("dispatched to bridge device") then `DispatchOrder` — a device result can never hit a still-`pending` order; claim/dispatch errors → compensate (release code + refund if charged) → `failed`.
- **New `order/bridge_settler.go`** (new file — keeps `service.go` churn away from the Whish diff): `CompleteBridgeOrder` = `TransitionStatus(processing→completed, event, bson.D{{"fulfillment.transferRef", ref}})`, `ErrConflict` → idempotent no-op, then notify (`orderCompletedNote`) + loyalty award — the exact admin-completion effects from `admin.go:339`. `FlagBridgeOrder` = new repo method `AppendTimelineEvent` (`$push` only, **no status change**) with `{status:"bridge_failed", note}`.

## Product model — `api/internal/modules/product/`

`BridgeSpec{Provider, Method}` as `Bridge *BridgeSpec` on `Product` + both inputs (inputs already carry `FulfillmentMode` — admin just never sent it); `FaceValue *float64` on `Variant`. Validation: bridge mode ⇒ spec required; transfer ⇒ every variant `FaceValue > 0`. Repository `$set`/`$unset` like `verification`. `DeriveMode` untouched.

## Admin console — `/admin` (payments-page pattern)

1. **Enums**: `FulfillmentMode` + `bridge`/`faceValue` in `src/types/index.ts`; `Badges.tsx` `FfKey` gains `'bridge'` (`ffKey(ft, mode?)`); update FF badge call sites in products/orders.
2. **ProductEditPage.tsx**: 4th `FF_OPTIONS` card "Mobile recharge (bridge)" ⇒ `account_credit` + `bridge_device`, provider/method selects, (transfer) Face-value column on variant rows; auto-seed a `phone` inputField; select card from `p.fulfillmentMode` on load; `buildInput()` sends `fulfillmentMode` + `bridge` + `faceValue`.
3. **Bridge page** (`features/bridge/{api,hooks,pages}`, lazy `/bridge` route in `app/router.tsx`, `nav_bridge` in `app/nav.ts` + en/ar/tr i18n): Devices panel (online dot from `lastSeenAt`, balances, enable toggle, register/rotate modals — token shown once) + Commands panel (status chips, order link, attempts, statusCode, rawReply tooltip, retry/cancel buttons), `refetchInterval: 10s`.

## Flutter — `/app` (near-zero)

Verified: checkout keys entirely off `fulfillmentType != 'code'` + `inputFields` and never reads `fulfillmentMode` — a bridge product works with **zero structural changes**. Only add: Lebanese phone validator on fields with `key == 'phone'` in `checkout_screen.dart` `_submit` (+ product-detail quick-buy if applicable), `TextInputType.phone`, en/ar/tr error strings.

## Android app — move to `salehcard/bridge` + full gap fix

Move the tree (raw-move commit first, then fixes; gradle wrapper IS committed and carries over; exclude `.idea/`, `local.properties`, `app/build/`). Rename applicationId → `com.salehcard.bridge`. Pin AGP to stable 8.x (9.0.1 is bleeding-edge) or document the toolchain. Fixes:

1. **Manifest/receiver bugs**: fix `.receiver.SmsReplyReceiver` → `.sms.SmsReplyReceiver`; implement missing `BootReceiver` (start service on boot when provisioned).
2. **Delete `websocket/` package** + the 9 `"hello world"` connect sites; results move to `POST commands/{id}/result` with a **persisted retry queue** (DataStore/Room, exponential backoff; `duplicate` response = success).
3. **Provisioning**: server URL + device token entered once, stored in EncryptedSharedPreferences; `RetrofitClient` reads them; "Test connection" via GET /config; battery-optimization exemption prompt; cleartext HTTP only in debug builds.
4. **Lease semantics**: honor `leaseExpiresAt`, defensively skip already-known commandIds.
5. **ProviderStore**: nullable SIMs (`hasTouchSim/hasAlfaSim` — no lateinit crash); missing-SIM commands report new code `PROVIDER_SIM_MISSING=9004`; state persisted + restored (config echo restores after reinstall).
6. **SmsReplyRouter**: suspend on a Mutex instead of throwing on concurrent waits (fixes balance-check collision).
7. **Alfa parity**: per-provider balance writes (bug: `checkBalance` always writes touch — CommandExecutor.kt:636), `alfaSimBalance/Validity` throughout, alfa insufficient-balance pre-check, `rechargeAlfa` returns 4xxx family (not 2023).
8. **ISO dates** end-to-end; parser converts operator `dd-MM-yyyy` replies.
9. **Success matching from server-config patterns**; `rawReply` always attached (including the touch "anything-without-fail" path).
10. Empty-queue log → local DEBUG (currently spams the wire every 1s); poll interval from config; **client-side SMS pacing** (rolling 30-min cap from config; saturation = retriable failure → lease requeue). Document Android's ~30 SMS/30min ceiling.
11. Heartbeat coroutine + batched log shipping.
12. Fees/min-balances from config only (drop hardcoded billing literals).

## Config vars (`api/internal/config/config.go` + `.env.example`)

`BRIDGE_LEASE_TTL` (3m), `BRIDGE_MAX_ATTEMPTS` (3), `BRIDGE_QUEUE_TIMEOUT` (30m), `BRIDGE_REAPER_INTERVAL` (30s), `BRIDGE_STUB` (false — `Validate()` rejects true outside development, mirroring USDT/Whish guards), `BRIDGE_STUB_DELAY` (10s), `BRIDGE_POLL_INTERVAL` (5s), `BRIDGE_MAX_SMS_PER_HALF_HOUR` (25), per-operator template/destination/fee/min-balance/USSD vars, `BRIDGE_SUCCESS_PATTERNS`/`BRIDGE_FAILURE_PATTERNS` (csv). All with Go defaults.

## Testing & verification

Repo test style = mocked stores + testify (see `order_usdt_test.go`, `payment/service_test.go` — no Mongo in tests).

- `bridge/service_test.go`: lease ordering/provider filter, settler-once via `OrderFlagged`, duplicate-result idempotency, partial→flag, reaper requeue/exhaust/stale, stub mode, token generate/hash roundtrip, `DeviceAuth` accept/reject.
- `order/order_bridge_test.go`: validation matrix (spec/qty/phone/face-value), claim-before-dispatch ordering, dispatch-failure compensation (code release + refund), settler idempotency + notify/loyalty effects, flag-without-status-change.
- `order/phone_test.go`; product bridge-spec validation tests.
- Android: unit tests for `splitAmount`, reply parsing, pacing window, retry queue.
- **Gates**: `cd api && go build ./... && go vet ./... && go test ./...`; `cd admin && pnpm build && pnpm lint`; `cd app && flutter analyze`.
- **E2E (stub)**: run API with `BRIDGE_STUB=true`, create a bridge product in admin, buy it in the Flutter app with wallet balance → order parks `processing` → stub auto-succeeds after delay → order `completed`, timeline shows bridge events, admin Bridge page shows the command lifecycle. Then failure path: `BRIDGE_STUB` off, let the command exhaust attempts → order flagged `bridge_failed`, still `processing`, admin completes manually.
- **QA-TODO.md**: add "Bridge recharge" section (real-device e2e, lease-expiry requeue, attempts-exhaustion → manual complete/refund, provisioning flow, token rotation, ar/RTL). **DEVOPS-TODO.md**: BRIDGE_* prod config, device provisioning runbook, live-SIM calibration session (templates/shortcodes/fees/reply wording), SMS throughput ceiling note.

## PR breakdown (stacked off `origin/main`)

1. **PR 1 — Go groundwork**: product `BridgeSpec`+`FaceValue`+validation; `order/phone.go` + PlaceOrder bridge validation; `fulfillBridge` still parks.
2. **PR 2 — Go bridge module** (biggest): the module + config + wiring + settler + `fulfillBridge` dispatch + code claim + audit + tests + stub mode.
3. **PR 3 — Admin console**: editor bridge card, Bridge page, enum plumbing, i18n.
4. **PR 4 — Flutter phone validation** (tiny).
5. **PR 5 — Android**: repo move + all 12 fixes + HTTP contract + provisioning.
6. **PR 6 — docs/QA** (or folded into 2/5).

PRs 1–2 touch the same 4 shared files as the uncommitted Whish work — additive blocks only; rebase whichever lands second.

## Risks / open items

1. Operator templates/shortcodes/fees/reply wording (incl. possible Arabic replies) only confirmable on real SIMs — everything is env-configurable; budget a live-SIM calibration session before go-live.
2. Transfer chunking (≤3 units/SMS) makes partial execution real; treated terminal + manual. Resume-from-chunk is a possible follow-up.
3. Face-value denomination (USD units on both networks?) needs owner confirmation.
4. No-auto-refund means a failure costs the admin two actions — accepted, preserves the single money-out path.
5. Park-before-dispatch crash window (processing order, no command) is visible on the admin Bridge page and manually completable.
6. Canonical `{phone}` form in templates (leading 0 or not) must be pinned during live-SIM testing.
