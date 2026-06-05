# SalehCard — Manual Test Guide (A→Z)

A full click-through of everything implemented in the purchase funnel:
**auth → catalog → cart → checkout → payment → coded fulfillment → order history →
wallet → dashboard**, plus access-control guards and the manual-fulfillment
(`processing`) path.

> Scope: this covers what's wired to the **real backend**. Items still mocked or
> deferred are listed under [Known limitations](#known-limitations).

---

## 1. Prerequisites

- **MongoDB** running on `localhost:27017` (the dev default; reuse your existing one).
- **Go** (for the API) and **pnpm** (for the web app — installed at `~/.local/bin`).
- A browser.

---

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

### 2d. Make sure a code product has codes

Code products ship with **no codes** until you upload some. Steam Wallet is the
code product. Upload a batch (admin token required):

```bash
B=localhost:8090
ATOK=$(curl -s $B/api/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"admin@salehcard.local","password":"password123"}' | jq -r '.data.accessToken')

# find the Steam Wallet product id (IDs are regenerated on a fresh seed)
PID=$(curl -s "$B/api/v1/products?limit=50" | jq -r '.data[]|select(.title.en=="Steam Wallet")|.id')

curl -s "$B/api/admin/products/$PID/codes" -H "Authorization: Bearer $ATOK" \
  -H 'Content-Type: application/json' \
  -d '{"codes":[{"code":"TEST-1111-2222","pin":"4471"},{"code":"TEST-3333-4444"},{"code":"TEST-5555-6666"}],"batch":"manual-test"}'

curl -s "$B/api/v1/products/$PID" | jq '.data.stock'   # confirm stock > 0
```

---

## 3. Seeded accounts

| Email | Role | Password |
|---|---|---|
| `customer@salehcard.local` | customer | `password123` |
| `admin@salehcard.local` | admin | `password123` |

For a clean run, **register a fresh account** (also exercises registration and
starts you at a $0 wallet).

### Seeded products (pick by name in the UI; IDs change on a fresh re-seed)
| Product | Fulfillment | Behavior |
|---|---|---|
| **Steam Wallet** | `code` | instant coded delivery — use this for the code flow |
| **PUBG Mobile UC** | `account_credit` | order goes **processing**, asks for a Player ID, no code |
| **Bank Transfer** | `transfer` | order goes **processing**, asks for recipient, no code |

---

## A. Authentication

| Step | Route / action | Expected |
|---|---|---|
| A1 | `/register` → email + password (≥8 chars) → **Create account** | redirected to `/dashboard` |
| A2 | short password (<8) | inline validation error, no submit |
| A3 | register an email that already exists | error message ("email already exists") |
| A4 | **Sign out** (left sidebar on `/dashboard`) | back to a guest view (no wallet pill in header) |
| A5 | `/login` → seeded `customer@salehcard.local` / `password123` | redirected to `/dashboard` |
| A6 | wrong password at `/login` | "invalid credentials" error |
| A7 | while logged in, visit `/login` or `/register` | auto-redirected to `/dashboard` |
| A8 | reload the page while logged in | session restored (still logged in — refresh cookie) |

## B. Catalog

| Step | Route / action | Expected |
|---|---|---|
| B1 | `/` (home) | product grid renders from the API |
| B2 | click a category / `/category/:slug` | filtered list |
| B3 | click **Steam Wallet** | product detail page |
| B4 | on detail, pick a denomination ($5 / $10 / $20) | selected variant highlights; total updates |

## C. Cart & Checkout

| Step | Route / action | Expected |
|---|---|---|
| C1 | product detail → **Add to cart** | toast; cart count in header increments |
| C2 | `/cart` | line item(s) with correct price/qty; adjust qty |
| C3 | `/cart` → **Checkout** (or **Buy now** on detail) | `/checkout` with matching total |
| C4 | checkout shows payment methods | **Wallet**, **Visa**, **USDT** tiles |

## D. Payment methods (each places a real order)

| Step | Route / action | Expected |
|---|---|---|
| D1 | **Wallet** selected, balance ≥ total → **Place order** | success page with a delivered code |
| D2 | **Wallet** selected, balance < total | inline **insufficient** notice + **Top up** CTA; button blocked; no charge |
| D3 | **Visa** tile → button reads **Pay $X** → click | mock-approved → success page with code |
| D4 | **USDT** tile → **Pay $X** | mock-approved → success page with code |
| D5 | double-click **Place order** quickly | only **one** order is created (button disables while in-flight; idempotency key dedupes) |

## E. Fulfillment & order history

| Step | Route / action | Expected |
|---|---|---|
| E1 | after a code order → `/order-success/:id` | "Payment confirmed"; product, total, method |
| E2 | click **Reveal** in the code vault | the real delivered code unmasks; **Copy** works |
| E3 | `/orders` | the order listed as **Completed** with id + timestamp |
| E4 | **reload** `/orders` | order still there (persisted server-side) |
| E5 | click an order row → `/orders/:id` | detail shows the same code |
| E6 | buy **PUBG Mobile UC** (enter a Player ID) | order = **Processing**, no code (awaits admin) |
| E7 | buy **Bank Transfer** (enter recipient) | order = **Processing**, reference + recipient shown |

## F. Wallet

| Step | Route / action | Expected |
|---|---|---|
| F1 | `/wallet` | current balance + transaction history |
| F2 | choose an amount → **Top up** | balance increases; a **Top up +$X** ledger row appears |
| F3 | after a wallet purchase | balance **decreased** by the price; a **Checkout −$X** ledger row appears |
| F4 | header wallet pill | reflects the live balance on every page |

## G. Dashboard

| Step | Route / action | Expected |
|---|---|---|
| G1 | `/dashboard` | greeting, live wallet balance, **Recent orders** (real), loyalty points |
| G2 | a fresh account before any order | "No orders yet" |

## H. Access control (guards)

| Step | Route / action | Expected |
|---|---|---|
| H1 | logged **out**, visit `/orders` | redirected to `/login` |
| H2 | logged **out**, visit `/wallet` | redirected to `/login` |
| H3 | logged **out**, visit `/dashboard` | redirected to `/login` |
| H4 | logged **out**, visit `/checkout` | redirected to `/login` |
| H5 | logged **out**, visit `/order-success/:id` | redirected to `/login` |

## I. Admin reads (optional — via API)

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
- **Stop servers:** `kill` the PIDs holding ports 8090 / 5173 (e.g.
  `ss -ltnp | grep -E ':8090|:5173'`).
- **Fresh data:** orders/wallet/codes accumulate. Ask before dropping the DB —
  it removes seeded products/users too (re-run the seed afterward).

---

## Known limitations

These are **expected** to be incomplete (next steps, not bugs):

- **Refunds** and **admin manual-completion** of `processing` (credit/transfer)
  orders are not built yet — those orders sit in `processing`.
- **USDT** is an auto-approve mock (no real on-chain settlement).
- For a multi-quantity code order, the success page reveals the **first** code
  (the rest are recorded on the inventory side).
- **Loyalty/cashback** figures and **Saved IDs** on the dashboard are still mock.
- **Admin Finance** screen is still a stub (the wallet ledger feed exists in the API).
