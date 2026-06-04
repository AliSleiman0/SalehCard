# SalehCard Admin — Implementation Backlog

Status of the admin console (`/admin`) and its backend (`/api/admin/*`).
Legend: ✅ done · 🟡 partial (UI built, backend mock/stub) · ⬜ not started

> Each area's UI shell is **fully ported** from the design. "🟡" means the
> screen renders against mock data (`admin/src/lib/mock/demo.ts`) and/or its
> backend route returns `501`. Wiring an area = replace the mock with the
> feature's `api/` + `hooks/` and implement the Go handler.

---

## Foundation

- [x] ✅ Scaffold `/admin` Vite + React + TS app (port 5174), mirrors `/web`
- [x] ✅ Design system ported (`styles/base.css` + `admin.css`), Tailwind tokens
- [x] ✅ Typed UI primitives (Icon, Art, Badges, Charts, Controls, Table, Modal, States)
- [x] ✅ Sidebar + Topbar shell, collapsible (persisted), grouped nav
- [x] ✅ Light/dark toggle (persisted) + EN/AR/TR switcher with full RTL (English forced on load)
- [x] ✅ Router (react-router-dom) + `RequireAdmin` guard → `/login`
- [x] ✅ `AdminOnly` Go middleware (JWT role=admin; dev-bypass when no `JWT_SECRET`)
- [x] ✅ `/api/admin` route group wired in `server.go`
- [x] ✅ Tooling: `make dev-admin` / `build-admin` / `lint-admin`, docker-compose, docs

## Auth

- [x] ✅ Login screen (design-faithful) + role-gated routing
- [ ] ⬜ **Real login / token issuance** — currently a mock admin session paired with
      the middleware dev-bypass. Wire to a real `POST /api/v1/auth/login`
      (`{ accessToken, user }`), keep token in memory only.
- [ ] ⬜ Token refresh handling for the admin client
- [ ] ⬜ "Account settings" / "Activity log" / "Sign out → revoke" menu actions

## Dashboard — 🟡 (KPIs + low-stock wired; revenue/orders mocked)

- [x] ✅ KPI cards — product/code-derived figures real; revenue/orders/users/top-ups mocked server-side
- [x] ✅ Low-stock alerts panel — real (`GET /api/admin/dashboard/low-stock`)
- [x] ✅ Fulfillment donut + system health (static/design)
- [ ] 🟡 Revenue area chart (daily/weekly/monthly) — client mock; `/dashboard/revenue-chart` returns mock series
- [ ] ⬜ Recent-orders feed — mock until orders module is wired
- [ ] ⬜ Replace mocked KPIs (revenue, orders, active users, wallet top-ups, pending transfers) with real data once orders/wallet land

## Products — ✅ wired (reference slice)

- [x] ✅ List: filter (category/fulfillment/search), bulk select, bulk activate/deactivate/delete, delete
- [x] ✅ Create/edit: EN/AR/TR title tabs, fulfillment-type selector swaps fields, variants/pricing builder, status toggle
- [x] ✅ Backend: `GET/POST/PUT/DELETE /api/admin/products`, `POST /products/bulk`
- [ ] ⬜ Server-side sorting + real pagination UI (table sorts client-side today)
- [ ] ⬜ Persist description, region, tags, badges (not in product model yet)
- [ ] ⬜ Real image upload (gradient box-art is a placeholder by design)

## Inventory / Codes — ✅ wired (reference slice)

- [x] ✅ Code stock list grouped by product, color-coded levels
- [x] ✅ Bulk upload (CSV/TXT) → parse → validate (dupes/format preview) → commit, with dedup
- [x] ✅ Low-stock threshold config per product
- [x] ✅ Code lookup/audit (exact + last-4 suffix) with order/delivery trail
- [x] ✅ Backend: `code` module (`POST/GET /products/:id/codes`, `GET /codes/:code`, `PUT /products/:id/stock-threshold`, `GET /inventory`)
- [ ] ⬜ Persist + read **real upload history** (currently mock)
- [ ] ⬜ Mark codes expired (lifecycle job) + reserve/deliver on order fulfillment
- [ ] ⬜ Export codes

## Orders — 🟡 (UI built, backend stubbed 501)

- [ ] 🟡 Order list — filter by status/type/payment/date, search by id/email/code (mock UI)
- [ ] 🟡 Order detail adapts by fulfillment type — code reveal (mask/unmask), credit confirmation, transfer timeline (mock UI)
- [ ] 🟡 Manual transfer status update control (UI only)
- [ ] 🟡 Refund modal (wallet / original method) (UI only)
- [ ] ⬜ Backend: `GET /orders`, `GET /orders/:id`, `POST /orders/:id/refund`, `PUT /orders/:id/status`

## Users — 🟡 (UI built, backend stubbed 501)

- [ ] 🟡 List — filter by role/status, search (mock UI)
- [ ] 🟡 Detail tabs: profile, orders, wallet & cashback, saved IDs, role & access (mock UI)
- [ ] 🟡 Wallet adjust modal (credit/debit + reason) (UI only)
- [ ] ⬜ Backend: `GET /users`, `GET /users/:id`, `PUT /users/:id/role`, `PUT /users/:id/status`, `POST /users/:id/wallet-adjust`

## Resellers — 🟡 (UI built, backend stubbed 501)

- [ ] 🟡 List with tier/sub-balance/margin/performance (mock UI)
- [ ] 🟡 Detail: tier assignment, sub-balance adjust + history, per-product pricing overrides (mock UI)
- [ ] 🟡 Tier config cards (Bronze/Silver/Gold) (mock UI)
- [ ] ⬜ Backend: `GET /resellers`, `GET /resellers/:id`, `PUT /resellers/:id/tier`, `POST /resellers/:id/balance-adjust`, tier CRUD `GET/POST/PUT/DELETE /reseller-tiers`

## Finance — 🟡 (UI built, backend stubbed 501)

- [ ] 🟡 Unified transactions feed (filter by type/method) (mock UI)
- [ ] 🟡 Revenue summary (by category/method/currency, daily/weekly/monthly) (mock UI)
- [ ] 🟡 USDT verification queue (verify/reject) — optional per design (mock UI)
- [ ] ⬜ Backend: `GET /transactions`, `GET /revenue-summary`, `GET/PUT /usdt-verifications`
- [ ] ⬜ Real CSV/PDF export

## Promos — 🟡 (UI built, backend stubbed 501)

- [ ] 🟡 List (code/type/usage/dates/status) (mock UI)
- [ ] 🟡 Create/edit (type, value, min order, max uses, per-user limit, date range, categories) (mock UI)
- [ ] ⬜ Backend: `GET/POST/PUT/DELETE /promos`
- [ ] ⬜ Auto-generate code action

## Reviews — 🟡 (UI built, backend stubbed 501)

- [ ] 🟡 Moderation queue, expand, bulk approve/reject (mock UI)
- [ ] ⬜ Backend: `GET /reviews`, `PUT /reviews/:id`

## Settings — 🟡 (UI built, backend stubbed 501)

- [ ] 🟡 General (store name/contact, default language/currency) (mock UI)
- [ ] 🟡 Payment gateways (Visa/USDT toggles, masked keys, webhook URLs) (mock UI)
- [ ] 🟡 Notification thresholds (mock UI)
- [ ] 🟡 Admin accounts list + invite (mock UI)
- [ ] ⬜ Backend: `GET/PUT /settings`
- [ ] ⬜ **Granular per-role permission model** (Super admin / Editor / Viewer) — open item flagged in the design

## Cross-cutting

- [ ] ⬜ Global top-bar search → real cross-entity results (orders/users/products/codes); today it routes the query into Orders
- [ ] ⬜ Toast/notification system for mutation success/error (currently inline + `window.confirm`)
- [ ] ⬜ Notifications bell → real feed
- [ ] ⬜ Component/integration tests for wired pages (only `lib/utils` unit-tested so far)
