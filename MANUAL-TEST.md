# SalehCard — Manual Test Guide (A→Z)

A full click-through of everything implemented in the purchase funnel:
**auth → catalog → cart → checkout → payment → coded fulfillment → order history →
wallet → dashboard**, plus access-control guards and the manual-fulfillment
(`processing`) path.

> **Run the phases in order, top to bottom** — they build on each other (e.g. you
> must fund the wallet before paying with it, and the logged-out checks must come
> before you log in). Each step lists the route/action and what to expect.
>
> Scope: this covers what's wired to the **real backend**. Items still mocked or
> deferred are under [Known limitations](#known-limitations).

---

## 1. Prerequisites

- **MongoDB** running on `localhost:27017` (dev default; reuse your existing one).
- **Go** (API) and **pnpm** (web — installed at `~/.local/bin`).
- A browser.

## 2. Start the stack

Run each in its own terminal from the repo root (`/home/alis/salehcard`).
Tip: in this session you can prefix a command with `!` to run it inline.

```bash
# 2a. Seed dev data (idempotent — safe to re-run; seeds products + users)
cd api && PORT=8090 go run ./cmd/seed

# 2b. API on :8090  (api/.env supplies JWT_SECRET + ALLOWED_ORIGINS=http://localhost:5173)
cd api && PORT=8090 go run ./cmd/server

# 2c. Web on :5173  (web/.env already points VITE_API_BASE_URL at http://localhost:8090)
cd web && pnpm dev
```

Open **http://localhost:5173**. Health check: `curl localhost:8090/health` → `{"status":"ok"}`.

### 2d. Load codes onto the code product (required for the code flow)

Code products ship with **no codes**. Steam Wallet is the code product — upload a
batch (admin token required):

```bash
B=localhost:8090
ATOK=$(curl -s $B/api/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"admin@salehcard.local","password":"password123"}' | jq -r '.data.accessToken')
PID=$(curl -s "$B/api/v1/products?limit=50" | jq -r '.data[]|select(.title.en=="Steam Wallet")|.id')
curl -s "$B/api/admin/products/$PID/codes" -H "Authorization: Bearer $ATOK" \
  -H 'Content-Type: application/json' \
  -d '{"codes":[{"code":"TEST-1111-2222","pin":"4471"},{"code":"TEST-3333-4444"},{"code":"TEST-5555-6666"}],"batch":"manual-test"}'
curl -s "$B/api/v1/products/$PID" | jq '.data.stock'   # confirm stock > 0
```

## 3. Seeded accounts

| Email | Role | Password |
|---|---|---|
| `customer@salehcard.local` | customer | `password123` |
| `admin@salehcard.local` | admin | `password123` |

You'll **register a fresh account** in Phase 2. The seeded `customer@…` account is
used in Phase 1 for the "wrong password" / "duplicate email" checks.

### Seeded products (open by name in the catalog; IDs regenerate on a fresh re-seed)
| Product | Fulfillment | Behavior |
|---|---|---|
| **Steam Wallet** | `code` | instant coded delivery — the code flow |
| **PUBG Mobile UC** | `account_credit` | order goes **Processing**, asks for a Player ID, no code |
| **Bank Transfer** | `transfer` | order goes **Processing**, asks for recipient, no code |

---

# Test phases (run in order)

## Phase 1 — While logged OUT

Do these *before* creating your account (you start logged out).

| # | Route / action | Expected |
|---|---|---|
| 1.1 | visit `/orders` | redirected to `/login` |
| 1.2 | visit `/wallet` | redirected to `/login` |
| 1.3 | visit `/dashboard` | redirected to `/login` |
| 1.4 | visit `/checkout` | redirected to `/login` |
| 1.5 | visit `/order-success/abc` | redirected to `/login` |
| 1.6 | `/login` → `customer@salehcard.local` + **wrong** password | "invalid credentials" error, stays on `/login` |
| 1.7 | `/register` → password shorter than 8 chars | inline validation error, no submit |
| 1.8 | `/register` → email `customer@salehcard.local` (already exists) + valid password | "email already exists" error |

## Phase 2 — Register & session

| # | Route / action | Expected |
|---|---|---|
| 2.1 | `/register` → a **fresh** email + password (≥8) → **Create account** | redirected to `/dashboard`, balance **$0.00** |
| 2.2 | now logged in, visit `/login` (or `/register`) | auto-redirected to `/dashboard` |
| 2.3 | reload the page | still logged in (session restored via refresh cookie) |

## Phase 3 — Fund the wallet (do this before paying by wallet)

| # | Route / action | Expected |
|---|---|---|
| 3.1 | `/wallet` | balance **$0.00**, empty transaction history |
| 3.2 | pick an amount (default $50) → **Top up** | balance → **$50.00**; ledger row **Top up +$50.00** |
| 3.3 | look at the header wallet pill | shows the live balance ($50.00) |

## Phase 4 — Catalog

| # | Route / action | Expected |
|---|---|---|
| 4.1 | `/` (home) | landing page loads: hero + **category tiles (mock)** + the **Best sellers** and **Featured** sections, whose product cards come from the **API** (3 seeded products → 3 cards) |
| 4.2 | `/category/:slug` (e.g. `/category/games`) | products filtered by category from the API (e.g. Steam Wallet + PUBG under `games`) |
| 4.3 | open **Steam Wallet** (from a product card) | product detail page |
| 4.4 | pick a denomination ($5 / $10 / $20) | variant highlights; total updates |

> Note: the home **category tiles** and the category **taxonomy** are still mock
> (`lib/mock/demo.ts`); only the **product cards/lists** are API-backed.

## Phase 5 — Cart

| # | Route / action | Expected |
|---|---|---|
| 5.1 | on detail → **Add to cart** | toast; header cart count increments |
| 5.2 | `/cart` | line item with correct price/qty; adjust qty |
| 5.3 | **Checkout** | lands on `/checkout` with matching total |

## Phase 6 — Checkout & payment

> You have ~$50 in the wallet from Phase 3.

| # | Route / action | Expected |
|---|---|---|
| 6.1 | method = **Wallet** (balance ≥ total) → **Place order** | `/order-success/:id`, "Payment confirmed", a delivered code |
| 6.2 | (repeat a checkout) double-click **Place order** fast | only **one** order created (button disables in-flight + idempotency key) |
| 6.3 | (repeat) select **Visa** → button reads **Pay $X** → click | mock-approved → success page with code |
| 6.4 | (repeat) select **USDT** → **Pay $X** | mock-approved → success page with code |
| 6.5 | (repeat) **Wallet**, choose qty/denomination so **total > current balance** | inline **insufficient** notice + **Top up** CTA; order blocked; no charge |

## Phase 7 — Fulfillment & order history

| # | Route / action | Expected |
|---|---|---|
| 7.1 | on a code success page → **Reveal** | the real code unmasks; **Copy** works |
| 7.2 | `/orders` | your orders listed as **Completed** with id + timestamp |
| 7.3 | **reload** `/orders` | orders still there (persisted server-side) |
| 7.4 | click an order → `/orders/:id` | detail shows the same code |
| 7.5 | buy **PUBG Mobile UC** (enter a Player ID) → checkout | order = **Processing**, no code (awaits admin) |
| 7.6 | buy **Bank Transfer** (enter recipient) → checkout | order = **Processing**, reference + recipient shown |

## Phase 8 — Wallet & dashboard recheck

| # | Route / action | Expected |
|---|---|---|
| 8.1 | `/wallet` | balance **decreased** by your wallet purchases; ledger shows **Top up +$X** and **Checkout −$X** rows |
| 8.2 | `/dashboard` | live balance + **Recent orders** (real) + loyalty points |

## Phase 9 — Sign out & re-confirm guard

| # | Route / action | Expected |
|---|---|---|
| 9.1 | **Sign out** (left sidebar on `/dashboard`) | guest view; header wallet pill gone |
| 9.2 | visit `/orders` | redirected to `/login` again |

## Phase 10 — Admin reads (optional, via API)

The admin **console** is a separate app, but you can confirm the read endpoints
the funnel lit up:

```bash
B=localhost:8090
ATOK=$(curl -s $B/api/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"admin@salehcard.local","password":"password123"}' | jq -r '.data.accessToken')
curl -s "$B/api/admin/orders?limit=10" -H "Authorization: Bearer $ATOK" | jq '.data | length, .meta'
```
Expected: a paginated list of all orders (was `501` before this work).

---

## Reset / troubleshooting

- **Logs:** API → `/tmp/sc-api.log`, web → `/tmp/sc-web.log`.
- **Out of codes** (one consumed per code order): re-run step **2d** to upload more.
- **Stop servers:** `kill` the PIDs on ports 8090 / 5173 (`ss -ltnp | grep -E ':8090|:5173'`).
- **Fresh data:** orders/wallet/codes accumulate. Ask before dropping the DB — it
  removes seeded products/users too (re-run the seed afterward).

## Known limitations

Expected to be incomplete (next steps, not bugs):

- **Refunds** and **admin manual-completion** of `processing` (credit/transfer)
  orders aren't built yet — those orders stay in `processing`.
- **USDT** is an auto-approve mock (no real on-chain settlement).
- A multi-quantity code order reveals the **first** code on the success page (the
  rest are recorded on the inventory side).
- **Loyalty/cashback** figures and **Saved IDs** on the dashboard are still mock.
- **Admin Finance** screen is still a stub (the wallet ledger feed exists in the API).
