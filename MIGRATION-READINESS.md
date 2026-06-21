# SalehCard Migration — SPEC-vs-REALITY Readiness Report

*Lead architect's consolidated assessment. Synthesized from per-subsystem gap analyses and a codebase survey (8 area surveys + 9 subsystem gap analyses), grounded against the authoritative spec at `salehcard-migration-decisions.md` (Sections 1–14). Generated 2026-06-21.*

---

## 1. Executive summary

**The single most important reconciliation: `order` is the codebase's most complete module, yet it is the spec's least-implemented architecture.** The order module is genuinely *implemented* end-to-end — real placement, atomic code claim, wallet debit, compensation, idempotency, and a customer funnel wired browser-to-DB — but it implements the **wrong shape**. The spec's central architectural fact (Section 2.2) is that an order's behavior must fork on `fulfillment.mode` across four execution paths (`api` / `manual_operator` / `inventory` / `bridge_device`). What exists instead is the exact "single linear flow" the spec (line 49) explicitly calls *wrong*: a binary fork on product `FulfillmentType` where `code` auto-completes and everything else (`account_credit` + `transfer`) is parked in `processing`. There is no `fulfillment.mode` field at any layer.

The practical consequence is concrete and customer-visible: a standard PUBG top-up (`account_credit` + `api`, should auto-fulfill via a provider) and a PUBG special offer (`account_credit` + `manual_operator`) are indistinguishable today and **both wrongly sit in `processing`** awaiting a human — and the admin tooling to complete them is stubbed, so they would sit there forever. So the migration's core engine work is a **structural re-modeling of a working module**, not greenfield construction.

Outside the order engine, the picture splits cleanly: the **inventory/code pool** (Section 7) and **wallet ledger primitives** (Section 9) are substantially built and correct; **auth + user** are production-ready; the **storefront funnel** is fully wired. But four large subsystems the spec mandates — **provider abstraction** (5), **operator queue** (6), **Mobile Bridge** (10), and the **rich product/pricing schema** (3/4) — range from "schema-stub only" to "zero code." Three of the four order-dispatch branches currently have **no target subsystem to dispatch to**. The admin console is the visible tip of this: 3 of 11 screens are wired, and none of the twelve Section-12 admin gaps exist even as TODOs. Net: the funnel *demo* is convincing, but the *engine* the spec describes is roughly 25–30% built, concentrated in the reusable plumbing layers.

---

## 2. Current-state map

| Module / Area | Layer | Status | One-line |
|---|---|---|---|
| `order` (engine) | API | **partial** | Implemented but linear (forks on `FulfillmentType`, not `fulfillment.mode`); 3 of 4 dispatch modes absent |
| `order` idempotency | API | **implemented** | `Idempotency-Key` + partial-unique `(userId,key)` index, pre-check + conflict re-fetch |
| `order` compensation | API | **implemented** | Atomic all-or-nothing code claim + wallet refund on failure |
| `order` admin (refund / status) | API | **stub** | `response.Stub()` — no manual completion, no refund execution |
| `code` / inventory pool | API | **implemented** | Atomic dispense, bulk upload+dedupe, low-stock thresholds, stock mirroring |
| `wallet` ledger + atomic debit | API | **implemented** | `$gte`-guarded debit, append-only ledger (topup/purchase/refund) |
| `wallet` admin adjust / zero / sub-balance | API | **stub/absent** | No adjust-with-reason, no zeroing log, no reseller sub-balance ledger |
| `product` schema | API | **partial** | Flat `FulfillmentType` + Price/ResellerPrice only; no mode/provider/verification/inputFields/cost/FX |
| `user` + `auth` platform | API | **implemented** | JWT + rotating refresh cookie, AdminOnly/AuthRequired, dev-bypass |
| `user` admin (role/status/wallet-adjust) | API | **stub** | 501 stubs |
| `reseller` (tiers/pricing/sub-balance) | API | **stub** | Schema only; ApplyPricing/GetTier/Update all TODO |
| `promo` | API | **stub** | Schema complete; Validate/Apply/Create TODO |
| `settings` | API | **stub** | 501 |
| `dashboard` | API | **partial** | Product/code KPIs real; revenue/orders/users mocked |
| `payments` (card/usdt) | Platform | **stub** | `PaymentProvider` interface real; adapters "TODO implement", mock-approved |
| Provider abstraction (`Fulfill`/`Verify`) | Platform | **absent** | Does not exist (PaymentProvider is unrelated payment-charging) |
| `ProviderAccount` (system accounts) | API | **absent** | No model |
| Operator role / queue / executor | API | **absent** | No `operator` role, no queue, no claim/assign |
| Bridge module / WS gateway / device protocol | API | **absent** | Zero references; no WS dependency in go.mod |
| Storefront funnel (auth→catalog→cart→checkout→fulfillment→history) | web | **implemented** | Fully API-wired; CodeVault, TransferTimeline, wallet all live |
| Storefront multi-code (qty>1) display | web | **partial** | Backend claims all; UI shows only first code |
| Admin: Dashboard / Products / Inventory | admin | **implemented/partial** | Wired to API |
| Admin: Orders / Users / Resellers / Finance / Promos / Reviews / Settings / Auth | admin | **stub** | UI shells over `lib/mock/demo.ts`; explicit "wire to…" TODOs |

---

## 3. Per-subsystem gap table

Requirement counts use the gap-analysis statuses (done / partial / missing).

| # | Subsystem | Readiness | done / partial / missing | Top missing items | Biggest risk |
|---|---|---|---|---|---|
| 1 | **Fulfillment model & order dispatcher** | early | 3 / 4 / 6 | `fulfillment.mode` field & mode dispatch; `api` path; `manual_operator` queue/executor | Working engine is the exact "single linear flow" spec calls wrong; re-model is structural, not additive; all non-code orders wrongly parked in `processing` |
| 2 | **API provider integration** | none | 0 / 0 / 5 | `Provider.Fulfill` interface; `Verify`/check_name hook; stub adapters; `ProviderAccount` | Name collision w/ existing `PaymentProvider`; building in isolation = dead code until dispatcher+schema land; sensitive inputs must not persist |
| 3 | **Manual operator queue** | none | 0 / 1 / 6 | `RoleOperator` + middleware; queue/inbox; atomic claim/assign + executor audit | Concurrency on claim (must be atomic FindOneAndUpdate); audit has no actor field; gated on absent `fulfillment.mode` |
| 4 | **Inventory / code pool** | substantial | 6 / 4 / 4 | format validation; preview/dry-run; damaged/requires-review states; mode-keyed dispatch | Dispatch keyed on `type=='code'` not `mode=='inventory'`; no quarantine state → known-bad code can be sold; poll-only low-stock alerts |
| 5 | **Transfer flow** | partial | 0 / 4 / 3 | `cancellable=false` enforcement + confirm; sensitive-input masking; embedded FX | **Hollow reference number** (`TransferRef` never set server-side); dead-end transfers (no completion path) |
| 6 | **Wallet** | partial | 5 / 3 / 5 | admin adjust-with-reason; account-zeroing + audit log; reseller sub-balance ledger | Zeroing concurrency (read-then-write can lose money); dead `adjustment` enum never written; money as `float64` |
| 7 | **Product schema & pricing** | early | 1 / 3 / 14 | `fulfillment.{mode,provider,confidence,cancellable}`; verification + inputFields; cost/margin/FX/tiers | Thin model already wired across api+web+admin → expansion is a breaking migration; sensitive inputs persist in plaintext today (Section 3.2 violation) |
| 8 | **Mobile Bridge** | none | 0 / 0 / 14 | entire `bridge` module; WS gateway (no lib in go.mod); device protocol + per-device auth | Real-SIM pilot is the single biggest project risk; backend must match an untested APK's exact contract (not in repo); double-execution/lost-result hazards |
| 9 | **Admin gaps (Section 12)** | none | 0 / 0 / 13 | operator mgmt; bridge device mgmt; zeroing log; provider accounts; custom pricing | "Admin gaps" is the UI tip of FOUR absent backend subsystems — treating as UI-only work massively under-scopes it |

---

## 4. What exists and is reusable

Concrete, correct building blocks the engine should reuse rather than rebuild:

- **Atomic, race-safe consistency primitives** — the document-atomic `FindOneAndUpdate` discipline mandated by CONVENTIONS is already proven in three places: code claim (`code/repository.go:306-325`, `available→delivered`), wallet debit (`wallet/repository.go:104-114`, `$gte`-guarded), and generic balance `$inc` (`wallet/repository.go:116-132`). Operator claim and account-zeroing should be built on this exact pattern.
- **Idempotency infrastructure** — `Idempotency-Key` header + partial-unique `(userId, idempotencyKey)` index + pre-check + conflict re-fetch (`order/handler.go:49`, `repository.go:44-48`, `service.go:64-69,139-142`). A precedent (not directly reusable) for the bridge's needed `(deviceId, commandId)` dedup.
- **Compensation pattern** — `order/service.go:214-222` + `code ReleaseByOrder` + `wallet Refund`: all-or-nothing claim with code release + wallet refund on partial failure. The `api`/`operator`/`bridge` paths will each need their own compensation but can follow this template.
- **Inventory engine (Section 7, largely done)** — bulk upload w/ dedupe, atomic dispense, per-product low-stock thresholds + level computation, product-stock mirroring, code-audit lookup endpoint. The `inventory` dispatch branch reuses `code.ClaimForOrder`/`ReleaseByOrder` as-is; the only change is **re-keying the trigger from `type=='code'` to `mode=='inventory'`**.
- **Wallet ledger (Section 9 core)** — immutable append-only `wallet_transactions` with topup/purchase/refund fully exercised. Admin adjust/zero/sub-balance should extend this (add `ActorID` + `Reason` fields) rather than start fresh.
- **Auth/RBAC platform** — JWT HS256 (`user_id/email/role`), rotating refresh cookie, `AdminOnly`/`AuthRequired` middleware with dev-bypass. Adding `RoleOperator` + an `OperatorOnly` gate, and a per-device JWT principal for the bridge, layers cleanly onto this.
- **Order fulfillment timeline** — `Fulfillment.StatusTimeline` is an immutable append-only event log (`order/model.go:55-67`). A solid foundation; it needs an **actor/operator field** added to support executor audit.
- **Storefront funnel + design system** — auth, catalog, cart, checkout, CodeVault (masked reveal+copy), TransferTimeline shell, wallet top-up, order history — all wired to the API. The success-UX per fulfillment *type* already exists; it just needs real data behind the transfer reference and multi-code display.
- **Admin wired screens + adapter pattern** — Dashboard/Products/Inventory wired via React-Query hooks + `adapt*` mappers; the same pattern extends to the eight stubbed screens.

---

## 5. The hard gaps (genuinely new subsystems)

These have little or no code and represent the bulk of net-new build effort. Effort labels are rough order-of-magnitude from the gap analyses.

| Subsystem | Spec § | State | Rough effort | Why it's hard |
|---|---|---|---|---|
| **`fulfillment.mode` dispatcher** | 2.2 / 11 | engine exists, wrong shape | **L** (re-model) | Untangle the `type==code` conflation without breaking the working inventory path + its compensation; the load-bearing prerequisite for 5/6/10 |
| **Provider abstraction** (`Fulfill`/`Verify` + adapters + `ProviderAccount`) | 5 | absent | **M–L** | New platform package distinct from `PaymentProvider`; `Verify` must run *before* charge; sensitive inputs must not persist |
| **Operator queue** (role, inbox, atomic claim/assign, executor audit) | 6 | absent | **L** | Atomic claim under concurrency; retrofit actor identity onto timeline; real fulfillment still happens out-of-band (messaging integrations absent) |
| **Mobile Bridge** (module, WS gateway, device protocol, per-device auth, durable outbox/acks) | 10 | absent | **XL** | No WS library in go.mod; must match an untested Kotlin APK's exact contract (not in repo); minor-units money (codebase uses `float64`); gated on dispatcher + operator queue; real-SIM pilot is the top project risk |
| **Rich product/pricing schema** (mode/provider/verification/inputFields/cost/margin/FX/tiers/per-customer/legacyIds/flags) | 3 / 4 | flat stub | **L** (breaking) | Already wired across api+web+admin → expanding is a cross-cutting migration; must agree shape with the catalog importer; sensitive-input masking is a settled security requirement currently violated |
| **Admin gap features** (operator mgmt, bridge mgmt, KYC, custom pricing, zeroing log, provider accounts, messaging) | 12 | absent | **L+** (aggregate) | Each screen is gated on one of the four backend subsystems above |

---

## 6. Build sequence, annotated (spec Section 13 + readiness)

1. **The spec** — *done.*
2. **Fulfillment-engine build prompt(s)**, split as the spec does:
   - **(a) Dispatcher + provider abstraction + order flow** — *order flow partially done (linear); dispatcher is greenfield re-model; provider abstraction is greenfield.* **Next step:** add `fulfillment.mode` to the product schema, convert the order fork from `FulfillmentType` to a four-way `mode` dispatch, and re-key the existing inventory path to `mode=='inventory'` (preserving its compensation). Land the `Provider.Fulfill/Verify` interface + one stub adapter so the `api` branch has a target.
   - **(b) Operator queue + inventory engine** — *inventory engine substantially done; operator queue greenfield.* **Next step:** add `RoleOperator` + middleware, an order queue/inbox for `mode=='manual_operator'`, atomic claim/assign, and an executor field on the order/timeline. Inventory only needs format-validation, quarantine states, and the mode re-key.
   - **(c) Wallet + pricing/tiers** — *wallet ledger done; admin adjust/zero/sub-balance + tiers/pricing greenfield/stubbed.* **Next step:** implement admin adjust-with-reason + account-zeroing (atomic, with audit log extending the ledger), then reseller tier persistence + price-resolution service.
3. **Bridge subsystem** — *greenfield (zero code).* Gated behind 2(a) dispatcher and 2(b) operator queue ("device = operator type"). **Next step (when unblocked):** obtain the interns' protocol reference + APK source, add a WS library, model `BridgeDevice` in minor-units, build config/poll/result endpoints with `(deviceId,commandId)` idempotency. Do **not** start before the dispatcher lands.
4. **Catalog importer** — *separate prompt; engine schema must be agreed with its output before the product schema hardens.*
5. **Seed review + load** — *blocked on the product schema carrying `legacyId`/`legacyCategoryId`/`flags` (all absent) for re-runnable import.*
6. **Admin gap features** — *all greenfield; gated on their backend subsystems (2/3/5/10).* Operator + bridge management first per spec, then KYC/messaging per owner priority.
7. **Real-SIM bridge pilot** — *not started; the launch-gating validation and top project risk.*

---

## 7. Open [OWNER] questions

Consolidated and deduped against spec Section 14, annotated with the subsystem(s) each one blocks.

| # | Decision needed (Section 14) | Blocks |
|---|---|---|
| 1 | **Provider names** behind the numeric IDs (e.g. provider 5 = PUBG verifier), or access to live provider config. Interface + stubs buildable without it; real adapters are not. | Provider integration (5), product `verification.provider`/`fulfillment.provider` (3), provider/system accounts admin (12) |
| 2 | **Crypto sends**: automatic-with-fees vs. manual review. | Whether crypto routes to the `api` branch or the `manual_operator` queue — fulfillment dispatcher (2), transfer flow (8), operator queue (6) |
| 3 | **Money transfers**: keep manual WhatsApp flow vs. customer-facing automation. | Default dispatch target for `transfer+manual_operator` — transfer flow (8), operator queue (6) |
| 4 | **FX rates**: live API vs. manual updates. | Transfer embedded FX + `fx_rate_review` flag — product pricing (4), transfer flow (8), multi-currency wallet (9) |
| 5 | **KYC / identity verification**: required at launch or later? | KYC admin module scope (12); overlaps the `check_name`/`Verify` surface (5) |
| 6 | **Operator pay model**: confirm salaried (default) vs. commission/profit-share. | What the operator/order audit records per order — operator queue (6); commission stays stubbed if salaried |
| 7 | **Message-app integrations** (WhatsApp/Telegram/Sham Cash): required in v1? | Operator fulfillment channel + messaging admin (6, 12) |
| 8 | **Articles + Themes store**: confirm drop (assumed yes). | Documentation only — confirm omission is intentional (12) |
| 9 | **Acknowledge** the "automatic recharge" launch claim depends on a successful real-SIM Touch/Alfa pilot of an as-yet-untested APK. | Launch gating for Bridge (10) — acknowledgement, not a build decision |
| impl. | Provision the **interns' protocol reference doc + APK V1 source** (not in this repo). | Bridge protocol must match the existing APK contract (10) |
| impl. | **Two order surfaces**: are wallet top-ups (طلبات الشحن) a separate stream from product orders (سجل الطلبات) for migration/reporting? | Admin Finance/Orders data model (9, 12) |
| impl. | **Custom per-customer pricing** scope: in v1? Precedence vs reseller tiers? | Pricing model (4), admin custom-pricing (12) |
| impl. | **Adjustment/zeroing policy**: may admin debit drive balance negative or clamp at zero? Is a reason mandatory? Which role may zero? | Wallet admin adjust/zero + RBAC (9, 12) |

---

## 8. Recommended next action

**Build the `fulfillment.mode` dispatcher: add `fulfillment.{type, mode, provider, confidence, cancellable}` to the product schema and convert the order engine's binary `FulfillmentType` fork into a four-way `mode`-based dispatch — re-keying the existing, working inventory path to `mode=='inventory'` while preserving its atomic claim + compensation.**

This is the highest-value move because it is the spec's "central architectural fact" (Section 2.2) and the **hard prerequisite gating three of the four other major tracks**: the `api` provider path (5), the `manual_operator` queue (6), and the `bridge_device` path (10) all have nothing to fork to until `mode` exists. It also directly retires the most damaging current defect — that every `account_credit`/`transfer` order is wrongly parked in `processing` with no way to tell auto-fulfillable orders from human-fulfillable ones. Critically, it is **low-risk to start** despite touching the most complete module: the inventory mechanism, compensation, and idempotency it depends on are already built and correct, so the work is a contained re-keying plus a new dispatch table, not a from-scratch engine. Landing the `Provider.Fulfill/Verify` interface with one stub adapter in the same track (spec sequence 2a) gives the `api` branch a target and unblocks parallel work on the operator and provider subsystems. The two inputs to line up first are the **catalog importer's output shape** (so the schema is agreed once) and **[OWNER] question 1** (provider IDs) — though the interface and stubs can proceed without the latter.
