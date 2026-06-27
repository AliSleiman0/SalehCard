# Handoff — Catalog migration + storefront browse (2026-06-21)

> **Read this first.** This is the live state. The Purchase-Funnel handoff (Steps
> 1–2) is preserved below as earlier context. Plans referenced live under
> `~/.claude/plans/` (latest: `a-bubbly-volcano.md` — categories API + rootDomain).

## TL;DR — where we are

The legacy SalehCard catalog is **migrated, loaded, and browsable**. Three increments
sit **uncommitted on `main`** (git constraint: commit only when asked). Everything
builds and is verified (`go build/vet/test`, `pnpm build/lint/test`, curl/live).

**Servers may still be running from last session:** Mongo `:27017`, API `:8090`,
storefront **`:5174`** (5173 is held by another project `zakkerni`; CORS allows 5174).

## The three uncommitted increments (newest first)

### 3. Category API + storefront browse-by-domain (this session)
- **`GET /api/v1/categories`** — new read surface on the `category` module
  (`service.go`/`handler.go`/`routes.go` + `FindAll`; wired in `server.go`).
  Params `?depth=0` (8 root domains), `?rootDomain=`, `?parentLegacyId=`,
  `?withCounts=true`. Counts via new `product` `CountByRootDomain` aggregation.
- **`rootDomain` on products** — denormalized field + `ListFilter`/`buildFilter`
  + index; `GET /api/v1/products?rootDomain=games` returns a whole domain. The
  loader stamps it from each product's category; **re-ran `loadseed` (0 ins / 635
  upd)** to backfill.
- **Storefront** now DB-driven: `useCategories` + `fetchCategories` +
  `adaptRootCategory` + `lib/categoryPresentation.ts` (8-entry curated art/label/
  tagline keyed on the stable root domains). Tiles, header nav, footer, and the
  product-page breadcrumb browse by root domain. Names: curated English / **Arabic
  from the DB**; counts live. New `art` gradient `tools` for gsm_tools.
- Result: all 8 domains browsable incl. **GSM Tools (52)** that the old 7 mock
  tiles couldn't reach.

### 2. Catalog importer + full loader (prior session)
- **`/migration`** — standalone Go module (stdlib-only) that fetches the legacy
  API (`api.salehcard.com`, **import-time only — not a runtime dependency**),
  transforms per the spec rules, and writes `migration/seed/{categories,products,
  _review}.json` + `run-summary.md`. Re-runnable (`_raw/` cache; `--refresh`).
- **`api/cmd/loadseed` + `api/internal/migration/loadseed`** — idempotent upsert
  (keyed on `legacyId`) into Mongo. **`api/internal/modules/category`** (new) +
  rich schema on `product` (pricing, inputFields, verification, etc.) with a
  synthetic single Variant so the order engine/frontends are untouched.
- Dataset now in Mongo: **86 categories + 635 products** (coexisting with 3 dev-seed
  products). `_review.json` worklist: **553 flagged products** (biggest: 524
  `fulfillment_review`, 311 `pricing_review`) — resolve via `migration/overrides.json`,
  then re-run importer + loadseed.

### 1. Fulfillment-mode dispatcher + Provider seam (prior session)
- `product` gains `FulfillmentMode` (`api`/`manual_operator`/`inventory`/
  `bridge_device`) + `DeriveMode` + `FulfillmentProvider`; the order engine
  dispatches on mode. `api/internal/platform/provider/` is a minimal seam
  (`StubProvider` → `ErrNotImplemented` → order parks in the manual queue).
  Documented in **`MIGRATION-READINESS.md`** (untracked).

## Pick up here — immediate next steps

1. **Commit the three increments** (offered, awaiting the user's go-ahead + whether
   one branch/PR or split). Commit trailer: `Co-Authored-By: Claude Opus 4.8 (1M
   context) <noreply@anthropic.com>`.
2. **Triage `migration/seed/_review.json`** → fill `migration/overrides.json`
   (force `fulfillment.{type,mode,provider}` / `pricing.mode`, clear flags) → re-run
   `cd migration && go run ./cmd/import` then `cd api && go run ./cmd/loadseed`. The
   524 `fulfillment_review` products (which "manual" top-ups are really `api`) are
   the key owner decision.
3. ~~**Admin parity**: wire the admin's hardcoded `CATS` to real categories.~~ **Done** —
   the admin category dropdowns now load from catalog-distinct facets
   (`GET /api/admin/products/categories`); the `CATS` mock was deleted from `demo.ts`.
4. **Subcategory drill-down** (depth 1/2) — today's storefront browse is flat
   (root → all products under it).
5. Optional: surface rich fields (`description`, `inputFields`, `pricing.cost`) in
   the PDP / admin editor.
6. Still open from the funnel work below: refunds, admin manual-complete of
   `processing` orders, admin Finance feed.

## Run / verify (this increment)

```bash
# Mongo on :27017 (reuse). Catalog already loaded; re-load is idempotent:
cd api && go run ./cmd/loadseed            # 0 inserts / 635 updates
cd api && PORT=8090 go run ./cmd/server
cd web && pnpm dev                          # → :5173 (or :5174 if taken)

# Gates:
cd api && go build ./... && go vet ./... && go test ./...
cd web && pnpm build && pnpm lint && pnpm test   # 2 harmless react-refresh warnings

# Smoke:
curl 'http://localhost:8090/api/v1/categories?depth=0&withCounts=true'   # 8 roots + counts
curl 'http://localhost:8090/api/v1/products?rootDomain=gsm_tools&limit=1' # total=52
```
Browser check still un-automated: open the storefront, confirm 8 tiles with counts,
click GSM Tools → 52 products, toggle locale to Arabic → tile names switch to DB Arabic.

---

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
