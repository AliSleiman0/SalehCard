# SalehCard — Product Backlog

Source: post-launch business-logic/UX audit, 2026-07-02 (after the go-to-market
hardening series shipped to prod). Each item is scoped to be planned and executed
on its own. Pick an item by its ID and read its notes — they carry the context a
fresh session needs. Admin-console-specific engineering notes also live in
`admin/BACKLOG.md`.

Legend: **P1** do first (customer/operator pain) · **P2** soon (trust/coherence) ·
**P3** decide-then-do (build or hide) · ✅ done

---

## P1 — Customer & operator pain

### ✅ BL-1 Wallet refresh loop (Flutter)
The funding journey ends with "admin approves → balance appears", but the app
only refetches providers when a screen is disposed and re-entered. No
pull-to-refresh, no polling. A user watching the Wallet or Home screen never
sees their approved credit or a completed order without navigating away.
- Add `RefreshIndicator` (pull-to-refresh) to Wallet, Orders list, Order detail,
  and the top-up screen's request list.
- Consider a lightweight periodic refresh (e.g. re-fetch wallet + requests every
  30–60s while the Wallet screen is visible).
- Files: `app/lib/features/wallet/presentation/screens/wallet_screen.dart`,
  `wallet/presentation/providers.dart` (walletProvider, topUpRequestsProvider),
  `checkout/presentation/providers.dart` (ordersProvider), orders screens.

### ✅ BL-2 Pending top-up requests visible on the Wallet screen (Flutter)
The pending/approved/rejected request list renders only inside the top-up
screen. Surface a compact "pending request" banner/row on the Wallet screen
(`topUpRequestsProvider` already exists) so users understand why their balance
hasn't moved.

### ✅ BL-3 Admin work-queue signals (dashboard KPIs + nav badges)
Admins have no signal that top-up requests or KYC submissions are waiting —
they must remember to open `/topups` and `/kyc`. Money is blocked on these.
- Backend: add pending counts to `GET /api/admin/dashboard/stats`
  (`api/internal/modules/dashboard/dashboard.go`) — `kyc.Repository.CountPending`
  already exists; add an admin-wide pending count to `wallet.TopUpStore`.
- Admin UI: two dashboard KPI tiles (pending top-ups, pending KYC) linking to
  the queues, and live nav badges (`admin/src/app/nav.ts` supports `badge`;
  feed it from a small query instead of the removed hardcoded values).

### ✅ BL-4 Home screen search + bell wiring (Flutter)
`home_screen.dart` wires the search bar and the notification bell to a
"coming soon" snackbar, while fully working `/search` and `/notifications`
screens exist and are already wired from the Browse tab
(`categories_screen.dart:37,64`). Two one-line fixes.

### ✅ BL-5 Purge stale "instant" copy (Flutter l10n)
Copy contradicting the manual top-up / processing model, all still rendered:
- `promoSubtitle` ("land in your wallet in seconds") — Home promo card.
- `deliveredInstantly` ("Credit is delivered to this account instantly") —
  shown on product detail for exactly the account_credit/transfer products
  that now go to manual processing.
- Notification stub bodies ("credited to your wallet", "instant delivery").
Also delete orphaned keys: `payCardTitle`, `payCardSub`, `payUsdtTitle`,
`payUsdtSub`, `topUpSuccess` (en + ar arb + the three generated
app_localizations files).

## P2 — Trust & coherence

### ✅ BL-6 Web order detail must reflect order status
`web/src/features/orders/pages/OrderDetailPage.tsx` branches only on
fulfillment kind, never on `o.status`: a refunded code order still shows the
delivered code in the vault, and `CreditConfirm`
(`components/OrderParts.tsx:49-52`) hardcodes a green "Completed" badge. Now
that refunds are real, add a status badge to the order head, a refunded
notice, and suppress/annotate the code vault + credit card for refunded
orders (`adaptOrder.ts` already computes the status label).

### ✅ BL-7 Remove remaining mock data from live web pages
`web/src/lib/mock/demo.ts` is still imported by: `DashboardPage.tsx` (fake
cashback tile + saved-players grid), `ResellerDashboardPage.tsx` (entire page
mock, "Bulk order" CTA is a toast no-op — hide the route or wire it),
`SavedIDsPage.tsx`, `ProductDetailPage.tsx` (fake reviews/related products),
`Header.tsx`. Wire to real endpoints where they exist (saved player IDs and
reviews exist in the API) or remove the sections.

### ✅ BL-8 Admin order-detail edges
- Add an inline "Mark completed" button on the account-credit card
  (`OrderDetailPage.tsx` ff==='credit' branch) — today only the sidebar quick
  action covers it.
- Refunded orders: overlay/annotate the fulfillment card (code still reads
  "Delivered", credit reads its normal state).
- Decide the `pending`-status story: such orders (crash mid-placement) show
  "No actions available" — probably allow refund→failed or a cleanup action.

### ✅ BL-9 Reseller funding coherence
- Reseller top-up requests land in the same admin queue as customers' with no
  role indicator — add the user's role to `adminTopUpView`
  (`api/internal/modules/wallet/admin.go`) and a chip in `TopupsPage.tsx`.
- Convention decision: fund resellers via balance-adjust (ledger type
  `adjustment`) or the queue (type `topup`)? Finance metrics count these
  differently (`TopUpsForDay`, `finance.walletTopups`). Pick one and note it
  in CONVENTIONS.md, or make the finance rollup show both.

### ✅ BL-10 Backend small inconsistencies
- "Wallet top-ups today" uses UTC day boundaries
  (`wallet/repository.go TopUpsForDay`) while "orders today" uses
  `BUSINESS_TZ` (`order/repository.go DayStats`) — align on businessLocation.
- Phone-only admins produce blank actor attribution in the audit log and
  top-up decisions (JWT claims carry only email) — add phone to `auth.Claims`
  or fall back to UserID display.
- `ResellerTier.BalanceLimit` is editable but never enforced — either enforce
  it (balance-adjust + top-up approval for reseller users) or remove the field
  from the tier editor.

## P3 — Decide, then build or hide

### ✅ BL-11 Notifications: build minimal or hide
**Decided: build, with FCM push (PR #27, merged & deployed 2026-07-02).**
Go `notification` module (inbox rows on order completed/refunded, top-up
approved/rejected, KYC decided; list/unread-count/read-all/device-token
endpoints) + `platform/push` FCM adapter (`PUSH_PROVIDER`, log default) +
Flutter wiring (real repository, bell badges, mark-read, firebase_messaging
token lifecycle). Remaining ops steps (prod `PUSH_PROVIDER=fcm` app settings)
in `DEVOPS-TODO.md`; deferred manual checks in `QA-TODO.md`.

### ✅ BL-12 Send Money: hide or implement
**Decided: hide (2026-07-03).** The "Send money" action button is removed from
the Wallet screen; the `/wallet/send` route, `SendMoneyScreen`, and
`TransferRepositoryStub` stay dormant with a re-enable comment at the button
site. Revisit as a real peer-transfer feature post-launch (needs guarded
debit + credit + compensation, two ledger rows — see BL-15 territory).

### ✅ BL-13 Cart: enable or remove
**Decided: keep hidden, don't delete (2026-07-03).** Already the shipped
state: the cart nav tab and "Add to cart" button are commented out with
re-enable notes; Buy Now clears the cart and goes straight to checkout; no
user-reachable path into `/cart` remains. Cart machinery stays dormant for a
possible multi-item/bulk future (checkout + order model are single-product
today — enabling it is a backend feature, not a UI toggle).

### ✅ BL-14 Admin "coming soon" set
**Done (2026-07-04).** Shipped: client-side **CSV exports** (orders/users/
resellers/finance/codes — current-filter, `downloadCsv`); **bulk user
suspend/activate** (`POST /api/admin/users/bulk`, last-admin guarded) with bulk
order **export**; **soft-delete user** (`DELETE /api/admin/users/{id}` →
anonymize: status=deleted, email/phone freed, orders/ledger kept); **reseller
tier-margin pricing** now enforced at checkout (lowest of retail / tier margin /
per-variant override — `order/service.go resellerPrice`, see CONVENTIONS.md) with
an effective-price **pricing tab**; **add-reseller** flow (promote-from-Users
modal, reuses role+tier endpoints); **USDT verification queue** tab (top-up queue
filtered to `channel=usdt`); **`/audit` sidebar nav** item.
Deferred to **BL-15** (money-risk / needs new infra): bulk order **refund**, bulk
**email** + a `platform/email` provider, per-reseller price table, **PDF** export,
on-chain USDT `txHash` field, and a toast system.

### BL-15 Platform stubs (post-launch growth work)
- `GET/PUT /api/admin/settings` → 501 (store config, gateway keys,
  notification thresholds; the admin UI tabs were removed until this exists).
- Real card/crypto payment gateway (checkout is wallet-only by design until
  then; `platform/payments/{card,usdt}.go` are inert placeholders).
- Upstream fulfillment providers: the registry is empty so ALL api-mode and
  bridge-device orders park in `processing` for manual completion
  (`order/routes.go` provider.NewRegistry(); adapters go in
  `platform/provider/`).
- Code lifecycle: re-deliver/resend a lost code, mark codes expired.
- Rate limiting on auth/OTP endpoints (also protects Monty SMS spend).
- Granular admin roles (Super admin / Editor / Viewer) — flat `admin` today.

---

## Recently completed (context)

- ✅ Go-to-market hardening series (PRs #15–#20, merged & deployed
  2026-07-02): security fixes, audit log + last-admin guard + mandatory
  ledgers, order refund/manual completion, wallet-only payments + KYC gate on
  all purchases (skippable at signup; home banner + checkout panel), admin-
  approved top-up request queue, admin console de-mocking.
- ✅ Post-ship regression fixes (same day): Flutter checkout now reads the
  live wallet balance (returning users were locked out); web wallet top-up
  updated to the new request API (was 400-ing on every attempt) with honest
  pending-approval UX.
