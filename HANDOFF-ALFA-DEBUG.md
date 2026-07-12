# HANDOFF — Debugging Alfa recharge (2026-07-08)

Next session's job: **debug the Alfa mobile-recharge flow.** This doc maps the
whole Alfa path end-to-end so you can localize the bug fast. It does **not** yet
name the specific symptom — get that from the user first (see step 0).

---

## 0. Start here — capture the symptom

Ask the user what "Alfa is broken" actually means, because the fix location
depends entirely on the failure mode. Likely buckets:

- **Order won't place** → `400` at checkout (validation in `PlaceOrder`).
- **Order places but never completes** → parks in `processing` forever (bridge
  disabled, or device never picks it up, or recharge silently fails).
- **Order marked completed but line not actually recharged** → wrong
  template/destination, or reply-matching false-positive.
- **Recharge worked but order flagged failed** → reply-matching false-negative
  (Alfa's real reply text doesn't match `BRIDGE_SUCCESS_PATTERNS`).
- **Wrong amount / wrong number dialed** → phone normalization or FaceValue.
- **App shows nothing / wrong label** → NOT this; that was the i18n work below.

Get: a concrete order id (or reproduction steps), the operator reply text if
any, and whether this is **prod** (real SIM + Monty NAT) or **dev/stub**.

---

## 1. What shipped in the previous session (context, not the bug)

All done & verified — unlikely related to Alfa recharge, but here so you don't
re-investigate:

- **i18n label fix (prod DB, live):** many products/categories stored Arabic
  inside the English field (`en` held Arabic, not empty). Fixed via a new
  read-only-by-default tool **`api/cmd/inputlabels`** (`-distinct` to list
  unique Arabic-in-`en` strings, `-tr map.json -apply` to write English keyed by
  exact source string). Applied to prod: **4 categories + 91 input fields (77
  product docs)**; re-scan shows 0 remaining. `ar` untouched. This tool is
  **untracked** (not committed) along with stray `*.exe` build artifacts in
  `api/`. Note several **Alfa products** carry these input fields: `alfa`,
  `alfa-direct`, `alfa-gift`, `alfa-ushare`, `alfa-validity`, `airtime-alfa`
  (their `field_1` = "رقم" → now "Number").
- **KYC back-button fix:** committed to `main` (`6999939`) — app-only, no deploy.
- **Client APK built** (release, prod API): `app/build/app/outputs/flutter-apk/app-release.apk`.
- **Auth 90-day tokens:** shipped earlier (PR #56, live).

**The `inputlabels` tool + its `main_test.go` are a reusable way to inspect prod
data** — reuse the same DB-connection pattern for any Alfa data probe (§6).

---

## 2. Alfa recharge architecture (the bridge_device path)

Alfa/Touch recharges are fulfilled by the **Mobile Bridge**: an Android device
holding the owner's SIMs sends operator SMS/USSD. Server enqueues a command; the
device polls, executes, posts the result back. Feature is **off by default**
(`BRIDGE_ENABLED=false` → orders park for manual completion, no device contacted).

### Product configuration (what makes a product "Alfa")
- `product.FulfillmentMode == "bridge_device"` (`product/model.go:29`).
- `product.Bridge` = `BridgeSpec{Provider, Method}` (`product/model.go:92-108`):
  - `Provider`: `"touch"` | `"alfa"` (`BridgeProviderAlfa`, `model.go:79`).
  - `Method`: `"transfer_credit"` (move credit from SIM balance, needs a
    positive `Variant.FaceValue`) | `"recharge_line"` (apply a scratch-card code
    claimed from inventory) (`model.go:87-90`).
  - `BridgeSpec.Valid()` gates both (`model.go:100`).
- **First thing to verify in prod:** are the Alfa products actually
  `bridge_device` with a valid `Bridge` spec and (for transfer_credit) a
  `FaceValue` on each variant? If not, orders park or 400.

### Order placement — `PlaceOrder` (`order/service.go`)
- Bridge lines have hard constraints (`service.go:245-260`): **exactly one line,
  quantity 1**, a **dialable Lebanese mobile**, and for `transfer_credit` a
  **positive FaceValue**. Any miss → `400 badRequest`.
- Recharge target number: `bridgePhone` (`order/phone.go:51`) — the `PlayerID`
  (Flutter snapshots the first input field) or a field keyed `"phone"`.
- Number normalization: `normalizeLebanesePhone` + `validLebaneseMobile`
  (`order/phone.go:11-46`). Accepts `+961 71…`, `0096171…`, `03 123 456`,
  `71123456` → 8-digit local. **Valid prefixes: `03 70 71 76 78 79 81`.**
  ⚠️ If Alfa has a range not in this regex, valid numbers get rejected — check
  `lebaneseMobile` (`phone.go:11`) against current Alfa MSISDN ranges.

### Fulfillment — `fulfillBridge` (`order/service.go:556`)
- If `s.bridge == nil || !Enabled()` → **park** (processing, "queued for bridge
  device") — the pre-bridge/manual behavior. **If `BRIDGE_ENABLED` isn't set in
  prod, every Alfa order lands here and never auto-completes** — a very likely
  "Alfa doesn't work" report.
- Else: for `recharge_line`, claim one card code first; **park processing BEFORE
  enqueuing** (so a device result can't race a not-yet-persisted order); then
  `DispatchOrder` → bridge (`bridge_settler.go:19-22`, `DispatchInput`).
- Misconfigured-after-purchase (product lost its Bridge spec) → parks for a human
  rather than dispatching an ambiguous command (`service.go:563`).

### Command build + settle (`bridge/service.go`)
- `DispatchOrder` → `Command{Provider:"alfa", Type: TransferCredit|RechargeLine}`
  (`service.go:116-132`).
- Device pulls config via the device-config payload (`service.go:280-294`),
  which carries the **Alfa templates/destinations** and the **success/failure
  match patterns**. The actual SMS string is built device-side from these.
- Result comes back → `IngestResult` → `settle` (`service.go:161-214`):
  - success → `OrderSettler.CompleteBridgeOrder` (guarded processing→completed +
    transferRef + notify + loyalty) (`order/bridge_settler.go:29`).
  - failure → `OrderSettler.FlagBridgeOrder` — **stays processing**, appends a
    `bridge_failed` timeline note, **never auto-refunds** (single money-out path
    preserved) (`bridge_settler.go:58`).
- ⚠️ Success/failure is decided by **text matching** on the operator reply
  (`SuccessPatterns`/`FailurePatterns`, `service.go:62-63`). Wrong patterns =
  correct recharge flagged failed, or a failure marked complete. **Prime suspect
  for "line recharged but order stuck / order completed but no credit."**

---

## 3. Alfa config knobs (env → `OperatorConfig`)

Defaults in `config/config.go:310-321` (all overridable; **CLAUDE.md warns these
are guesses — confirm on a live Alfa SIM before trusting them**):

| Env var | Default | Meaning |
|---|---|---|
| `BRIDGE_ENABLED` | `false` | master switch; false → park for manual |
| `BRIDGE_ALFA_BALANCE_USSD` | `*11#` | balance check |
| `BRIDGE_ALFA_RECHARGE_TEMPLATE` | `{phone}R{code}` | recharge_line SMS body |
| `BRIDGE_ALFA_RECHARGE_DEST` | `1313` | recharge_line destination |
| `BRIDGE_ALFA_TRANSFER_TEMPLATE` | `{phone}T{amount}` | transfer_credit SMS body |
| `BRIDGE_ALFA_TRANSFER_DEST` | `1399` | transfer_credit destination |
| `BRIDGE_ALFA_MIN_BALANCE` | `20` | refuse transfer below this SIM balance |
| `BRIDGE_ALFA_MESSAGE_FEE` | `0.14` | per-message fee accounting |
| `BRIDGE_SUCCESS_PATTERNS` | `success,transferred` | reply → succeeded |
| `BRIDGE_FAILURE_PATTERNS` | `fail,do not have,insufficient` | reply → failed |
| `BRIDGE_STUB` / `BRIDGE_STUB_DELAY` | `false` / `10s` | dev auto-succeed (refused outside dev) |

Placeholders `{phone} {amount} {code}` are substituted per command. **Verify the
real Alfa third-party recharge shortcode + SMS syntax** — the transfer/recharge
templates and destinations (1313/1399) are the most likely wrong values.

---

## 4. Prod facts to confirm first (don't assume)

1. **Is `BRIDGE_ENABLED=true` in the prod App Service settings?** If not, that's
   the whole "Alfa never completes" story (everything parks manual). Check Azure
   app settings for `salehcard-api` (WebApp). Auto mode may block `az` writes;
   reads should be OK, else ask the user.
2. **Is a bridge device registered + polling?** See the admin Bridge page /
   `bridge/admin.go` (device token issuance) and the `devices` collection
   (`bridge/model.go:56` heartbeat fields incl. `alfaBalance`/`alfaValidity`).
   No live device ⇒ commands sit `queued`.
3. **Are the Alfa products `bridge_device` + valid `Bridge` + `FaceValue`?**
   (§2). Query prod (§6).
4. **Prod SMS egress:** Monty IP-allowlists; prod egresses via NAT
   (`52.157.69.89`). Bridge uses the **device's** SIM, not Monty — so Monty NAT
   is about OTP/bulk-SMS, not recharge. Don't conflate.

---

## 5. Reproduce / debug locally

- Run API with the bridge in stub mode so orders auto-complete without a device:
  `ENV=development BRIDGE_ENABLED=true BRIDGE_STUB=true BRIDGE_STUB_DELAY=5s PORT=8090 go run ./cmd/server`
- Seed/create an Alfa `bridge_device` product (Provider `alfa`, Method
  `transfer_credit`, a variant with `FaceValue`). Product seed is
  `product/seed/seed.go` (no bridge product seeded there today — add one or
  create via admin).
- Place an order for it with a valid `71xxxxxx` number; watch it go
  processing → completed (stub) and confirm the command/`transferRef`.
- Tests already covering this path — **read these first, they encode the
  intended behavior**:
  - `order/order_bridge_test.go` (dispatch + settle wiring)
  - `bridge/service_test.go` (command build, ingest, patterns)
  - `product/product_test.go` (BridgeSpec validation)

---

## 6. Inspecting prod data (reuse the label-tool pattern)

Prod is **Cosmos vCore** — no direct access without a firewall rule. Recipe
(needs `az` logged in AND **out of auto mode**; the firewall write + inline URI
are blocked in auto mode):

1. `curl -s https://api.ipify.org` → current egress IP (drifts across
   `213.204.66.x`).
2. `az rest --method put --url ".../mongoClusters/docdb-cluster-20260626-2230/firewallRules/localpc-<date>?api-version=2024-07-01" --body '{"properties":{"startIpAddress":"<ip>","endIpAddress":"<ip>"}}'` (poll ~30s → `Succeeded`). Sub `1adb4811-6234-4822-b7f2-8411ed2cb999`, RG `salehcard-prod`.
3. `MONGO_URI="$(grep -oE 'mongodb\+srv://[^[:space:]]*maxIdleTimeMS=120000' DEPLOY-CREDS.local.md | head -1)" DB_NAME=salehcard go run ./cmd/<probe>`
4. **Delete the firewall rule when done** (`az rest --method delete ...`).

A quick Alfa probe is a 20-line Go tool modeled on `cmd/inputlabels` /
`cmd/dupcheck`: dump each Alfa product's `fulfillmentMode` + `bridge` +
variants' `faceValue`, and recent `orders` where an item is an Alfa product with
their `status` + timeline (look for `bridge_failed` notes). See
memory `salehcard-prod-db-direct-access` for the full runbook.

---

## 7. Likely suspects, ranked

1. `BRIDGE_ENABLED` not set in prod → all Alfa orders park, never complete.
2. **Reply-pattern mismatch** — real Alfa SMS reply text doesn't contain
   `success`/`transferred` (or a real failure contains none of the failure
   patterns) → wrong terminal state.
3. **Wrong recharge/transfer template or destination** (1313/1399, `{phone}R{code}`)
   vs. actual Alfa third-party recharge syntax.
4. **Phone regex** missing a live Alfa prefix (`phone.go:11`).
5. Product misconfig — not `bridge_device`, or `transfer_credit` variant with no
   `FaceValue` → 400 at checkout.
6. No registered/polling bridge device → commands stuck `queued`.

---

## 8. Verify gates

```bash
cd api && go build ./... && go vet ./... && go test ./...
```
App changes (if any): `cd app && flutter analyze`. Bridge/order changes touch
`api/**` → **pushing to main auto-deploys the API to prod** (path-filtered).
Rebuild the client APK only if the Flutter app itself changes:
`cd app && flutter build apk --release --dart-define=API_BASE_URL=https://salehcard-api.azurewebsites.net/api/v1`

---

## Key files

- `api/internal/modules/order/service.go` — `PlaceOrder` bridge validation
  (~245), `fulfillBridge` (556), `park` (497)
- `api/internal/modules/order/phone.go` — number normalize/validate + `bridgePhone`
- `api/internal/modules/order/bridge_settler.go` — Complete/Flag on device result
- `api/internal/modules/bridge/{model,service,handler,admin,routes}.go` — command
  lifecycle, device poll/ingest, device config payload, admin device tokens
- `api/internal/modules/product/model.go` — `BridgeSpec`, `FaceValue`, modes
- `api/internal/config/config.go:31-47, 305-323` — `OperatorConfig` Alfa knobs
- Tests: `order/order_bridge_test.go`, `bridge/service_test.go`
- (Prior branch, may not be on `main`) `BRIDGE-PLAN.md` on `feat/mobile-bridge`
  — see memory `salehcard-mobile-bridge`.
