# SalehCard Admin — Status & Backlog

Status of the admin console (`/admin`) and its backend (`/api/admin/*`).
Updated 2026-07-02 as part of the go-to-market hardening pass.

## Wired end-to-end (real API, no mock data)

- Auth: real `POST /api/v1/auth/login`, role-gated client routing, in-memory
  access token, 401/403 auto-logout. Server enforcement via `auth.AdminOnly`
  (dev-bypass only when `ENV=development` AND `JWT_SECRET` is empty).
- Dashboard (KPIs, revenue chart, fulfillment breakdown, low stock, recents,
  system health, CSV export).
- Products (list/filter/bulk/create/edit/delete + category facets).
- Inventory (stock, bulk code upload with dedup, thresholds, code lookup,
  upload history).
- Orders (list/detail, **refund**, **manual completion** of processing orders).
- Users (list/detail, role + status changes with last-admin guard, wallet
  adjust with mandatory ledger).
- Resellers (list/detail, tier assignment, balance adjust, tier CRUD).
- Finance (transactions feed, revenue summary).
- Top-up requests (pending queue, approve-and-credit, reject-with-reason).
- Promos, Offers, Expenses (full CRUD).
- Reviews + KYC moderation queues.
- Activity log (`/audit` — role changes, money adjustments, refunds,
  moderation decisions; backed by `admin_audit_log`).
- Settings → Admin accounts (read-only list; role changes happen on the
  user profile).

## Not built yet (honest gaps)

- [ ] Store/general settings, payment-gateway config, notification thresholds
      (`GET/PUT /api/admin/settings` is still a 501 stub — the former mock
      tabs were removed from the UI).
- [ ] Granular admin permission model (Super admin / Editor / Viewer) — one
      flat `admin` role today, protected by the last-admin guard.
- [ ] CSV/PDF exports (orders, users, resellers, finance, codes) — buttons
      are disabled "coming soon".
- [ ] Bulk order actions, bulk user email/suspend, user account deletion.
- [ ] Reseller custom pricing rules tab; add-reseller flow (promote a user
      to the reseller role from their profile instead).
- [ ] USDT verification queue tab in Finance.
- [ ] Notifications feed (topbar bell removed until one exists) + toast
      system for mutation feedback.
- [ ] Global top-bar search across entities (routes into Orders today).
- [ ] Code lifecycle: mark expired, re-deliver/resend a lost delivered code.
- [ ] Server-side sorting/pagination on the products table; product
      description/region/tags persistence; real image upload.
- [ ] Admin-client token refresh handling.
- [ ] Component/integration tests for wired pages (utils + adapters only).
- [ ] Real card/crypto payment gateway (checkout is wallet-only; wallet is
      funded via the admin-approved top-up request queue).
