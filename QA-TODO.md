# QA TODO — deferred manual / e2e verification

Everything in BL-1…BL-11 shipped on **green static gates only** (standing user
choice: defer manual testing to the end of dev). This file is the single
collection point for those deferred checks — it supersedes the "deferred /
not yet tested" sections scattered across the handoff files. Run these before
calling the backlog items closed. `MANUAL-TEST.md` (the full purchase-funnel
script) still applies on top of this list.

## Test environment

- Mongo on `:27017`, seeded: `cd api && go run ./cmd/seed`
- API: `cd api && PORT=8090 go run ./cmd/server` (`api/.env` currently has
  `PUSH_PROVIDER=fcm` + the local key file — set `PUSH_PROVIDER=log` for the
  log-provider checks below)
- Admin console: `cd admin && pnpm dev` → `:5174` (dev admin bypass: run API
  with empty `JWT_SECRET`)
- Flutter app on the Android emulator: see memory/handoff recipe —
  `adb reverse tcp:8090 tcp:8090` + `--dart-define=API_BASE_URL=http://127.0.0.1:8090`
- Accounts: `customer@salehcard.local` / `admin@salehcard.local`, password `password123`
- **Every Flutter check: verify in English AND Arabic (RTL).**

## BL-1 / BL-2 — Wallet refresh loop + pending top-ups banner (Flutter)

1. File a top-up → Wallet shows pending banner (count + total) → approve in
   admin → pull-to-refresh / 45s poll clears banner and updates balance.
2. Second pending request → banner switches to plural + summed total.
3. Order `processing → completed` reflected via pull-to-refresh on Order detail.
4. Background the app >45s → no polling; resume → one immediate refresh.
5. Arabic: both new strings + RTL plural render correctly.

## BL-3 — Admin work-queue signals

1. Seed a pending top-up + pending KYC → dashboard shows non-zero
   "Pending top-ups" / "Pending KYC" tiles that navigate to `/topups` / `/kyc`.
2. Sidebar shows amber count badges that clear when the queues empty.

## BL-4 — Home search + bell (Flutter)

1. From Home, tap the search bar → `/search` opens.
2. Tap the bell → `/notifications` opens (bell now also carries the BL-11 badge).

## BL-5 — Stale "instant" copy purge (Flutter)

1. Sweep Home/product/checkout copy EN + AR: no "delivered instantly"-style
   claims remain for flows that go to manual processing.

## BL-6 — Web order detail reflects status (`/web`, legacy)

1. Refund a completed code order → web order detail shows a refunded
   status badge + notice; the code vault is suppressed/annotated (no green
   "Completed" on a refunded order).

## BL-7 — Mock data removed from live web pages (`/web`, legacy)

1. Dashboard, product detail, header, saved-IDs: no fake cashback /
   reviews / related-products / saved-players content; reseller dashboard
   route hidden or wired.

## BL-8 — Admin order-detail edges

1. Account-credit order: inline "Mark completed" button on the credit card works.
2. Refunded order: fulfillment card is overlaid/annotated (code no longer
   plainly reads "Delivered").
3. A `pending`-status order (crash mid-placement) offers the decided
   cleanup action instead of "No actions available".

## BL-9 — Reseller funding coherence

1. A reseller's top-up request shows an amber **reseller** chip in `/topups`
   vs a customer's muted chip; approval still writes a `topup` ledger row.

## BL-10 — Backend small inconsistencies

1. Set `BUSINESS_TZ=Asia/Beirut`, place a top-up + completed order near the
   UTC/Beirut date line → both "today" KPIs count them in the same day.
2. Approve a top-up as a **phone-only admin** → `decidedBy` + audit actor show
   the phone, not blank.
3. Tier editor no longer shows a Balance-limit field.

## BL-11 — Notifications (inbox + FCM push)

With `PUSH_PROVIDER=log` (API :8090 + emulator):
1. Sign in → `device_tokens` row appears (or a caught "Firebase not
   configured" log if options were placeholders).
2. Place an instant code order → bell badge shows 1 + LogSender line in API
   output; open `/notifications` → row with formatted amount; badge clears
   (read-all fired).
3. Admin: approve + reject a top-up, refund + manually complete an order,
   approve + reject KYC → six inbox rows, localized titles/amounts EN + AR.
4. `unread-count` returns 0 after read-all; a new event → 1.
5. Logout → device token row deleted; re-login → new row.

With real FCM (after `DEVOPS-TODO.md` item 1, on a device/emulator with Play services):
6. App **backgrounded** → tray notification arrives; **foregrounded** → badge
   increments live, no tray; **killed** → tray arrives; tap → app opens
   (deep-link routing is out of scope v1).
7. Stale-token prune: uninstall/reinstall, trigger an event → API logs
   UNREGISTERED and deletes the old token row.
8. Known v1 limitation to confirm acceptable: FCM tray text is the English
   server fallback (in-app inbox is localized).
