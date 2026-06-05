# Handoff — Purchase Funnel (Steps 1–2 DONE, Step 3 remainder)

> For the next session. Read alongside [`PURCHASE-FUNNEL.md`](PURCHASE-FUNNEL.md)
> (overall funnel assessment). Step 2 design lives in
> `~/.claude/plans/ask-me-questions-with-resilient-kitten.md`.

## Where we are

**Step 1 (customer auth) and Step 2 (orders + fulfillment + wallet) are DONE and
verified** on branch `feat/admin-dashboard`. The purchase funnel is real
end-to-end: register/login → catalog → cart → checkout → server-priced order →
payment (wallet ledger / card mock / usdt mock) → code delivered from real
inventory → order history; admin Orders renders real data.

### What Step 2 shipped (verified by `go test`, `pnpm build/lint/test`, and a full curl E2E)

Backend (`api/internal/modules/`):
- **order**: real `PlaceOrder` — server re-prices from the catalog (never trusts
  client price/fulfillment), idempotency via `Idempotency-Key` header + a
  partial-unique `(userId, idempotencyKey)` index, order-first then atomic
  code-claim then flip `completed`, compensation (release codes + wallet refund)
  on failure. `account_credit`/`transfer` → `processing` (manual fulfillment).
  Customer routes `POST/GET /api/v1/orders`, `GET /api/v1/orders/{id}` (ownership
  enforced). Admin **read** endpoints (`GET /api/admin/orders`, `/{id}`) live;
  refund + manual status updates still `response.Stub`. Order line items snapshot
  `title/denomination/category` for stable history. Unit tests in `order_test.go`.
- **code**: new `ClaimOne` (atomic `FindOneAndUpdate available→delivered`) +
  `ReleaseByOrder`; service `ClaimForOrder`/`ReleaseForOrder`/`CountAvailable`
  with stock re-mirror. `ErrOutOfStock` sentinel.
- **wallet**: full ledger — `wallet_transactions` collection, balance authoritative
  on `users.walletBalance` (atomic `$inc` debit with `$gte` guard),
  `GET /api/v1/wallet`, `POST /api/v1/wallet/topups`, `Debit`/`Refund` used by the
  order service. `ErrInsufficientFunds` sentinel.
- server.go wires `order.RegisterRoutes` + `wallet.RegisterRoutes` behind `AuthRequired`.

Frontend (`web/src/`):
- `variantId` now flows product→cart (was discarded in `adaptProduct`).
- orders/wallet API + React-Query hooks; `adaptOrder` maps API→`OrderView`;
  `fulfillment.ts` maps `account_credit`↔`credit`.
- CheckoutPage posts a real order (per-attempt idempotency key in a ref, button
  disabled while pending); Orders/OrderDetail/OrderSuccess/Wallet/Dashboard/Header
  all read real endpoints; dead `stores/wallet.ts` removed.

### Decisions locked in Step 2 (for context)
1. Server re-prices (never trusts client). 2. All three pay methods: card mock,
wallet **real ledger**, usdt **auto-approve mock**. 3. Standalone Mongo → no
multi-doc txns; doc-atomic claim + compensation. 4. Idempotency-Key header.
5. credit/transfer → `processing`, admin read-only (no manual-complete yet).

## What still remains (Step 3 / polish)

- **Refunds**: implement `order/admin.go` `POST /orders/{id}/refund` → `wallet.Refund`
  + status `refunded` (wallet plumbing already exists).
- **Admin manual-complete** of `processing` (credit/transfer) orders:
  `PUT /orders/{id}/status` (set delivered code / credited / transfer ref).
- **USDT "pending until confirmed"** flow if desired (currently auto-approve mock).
- **Multi-code / multi-qty delivery**: `Fulfillment.DeliveredCode` stores only the
  first claimed code; codes for qty>1 live on the `codes` docs. Surface all if needed.
- **Mixed-fulfillment carts**: any non-code item makes the whole order `processing`.
- **Admin Finance** screen against the real `wallet_transactions` feed (currently stub).
- **Loyalty/cashback + saved IDs** on the dashboard are still mock (`DEMO`).

## Run / verify
- Stack: Mongo on `:27017` (reuse existing). API: `JWT_SECRET=devsecret PORT=8090
  go run ./cmd/server` (seed first: `go run ./cmd/seed` — seeds Steam Wallet (code),
  PUBG (credit), Bank Transfer, + `customer@salehcard.local`/`admin@…`, pw `password123`).
  Note: code products start with **no codes** — upload via
  `POST /api/admin/products/{id}/codes` before buying.
- Tests: `cd api && go test ./...`; `cd web && pnpm build && pnpm lint && pnpm test`.
- The one un-automated check left from the Step-2 plan is the **browser
  click-through** (login → checkout → real code on success → reload /orders →
  wallet top-up updates balance). The API was fully curl-verified.
