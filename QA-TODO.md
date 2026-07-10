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

## Product images on prod Azure Blob (DEVOPS-TODO item 13, provisioned 2026-07-05)

Infra verified (public container serves anonymously; API booted with no
`storage provider misconfigured` warning). Deferred e2e:

1. Prod admin console → edit a product → upload an image → the returned URL is
   `https://salehcardassets.blob.core.windows.net/product-images/…` (not `/uploads/…`),
   the display JPEG + 256px thumbnail both exist, and the image renders in the
   admin list + Flutter app (catalog + product detail).
2. Replace the image → new blob referenced (v1 leaves the old blob orphaned — expected).

## Admin SMS 2FA (setting-gated second factor)

Backend + admin console shipped; deferred manual/e2e checks:

Dev (API :8090, `SMS_PROVIDER=log` → the code prints to the API log):
1. Seed admin (no phone) logs in as before — 2FA is off by default.
2. Settings → Security → toggle **Admin SMS 2FA** on. With no phone on the acting
   admin it must be **refused** with the `ADMIN_2FA_NO_PHONE` message and the toggle
   reverts (nothing persisted). Set a phone (and re-login) to enable.
3. Admin with a phone, 2FA on: log out → log in → login response is a **challenge**
   (no token yet), a code appears in the API log; enter it → signed in. Wrong code →
   error + attempt count; wait >5 min → `OTP_EXPIRED`; **Resend** within 60 s →
   throttled, after 60 s → new code. **Back** returns to the password form.
4. A **customer** login (2FA on) still signs in directly — no challenge.
5. Toggle off → admin login is password-only again.

Prod (real, billed Monty SMS — see `DEVOPS-TODO.md`):
6. Confirm `admin@salehcard.com` phone is `+961 78991778`, enable the toggle, then
   log out/in → a real SMS arrives; enter it → signed in.

## KYC document-photo upload (mandatory front + back)

Backend + Flutter + admin shipped on green static gates; deferred manual/e2e checks
(dev: API :8090 with default `STORAGE_PROVIDER=local` → files land in `api/uploads/kyc/`):

1. App KYC form: submit with no photos → both tiles show "This photo is required.";
   passport selected → back tile shows "Optional for passports" (and a Remove link once
   uploaded); switching to ID card/license makes back required again.
2. Pick via **gallery** and via **camera** → tile shows preview + spinner, then check
   badge; Submit lands the submission as pending; API log/`uploads/kyc/` has the JPEGs.
3. Kill the API mid-upload → tile shows "Upload failed — tap to retry."; retry works.
4. `curl POST /api/v1/kyc` (Bearer) without `documentFrontUrl` → 400; with an
   off-keyspace URL (`https://x/y.jpg`) → 400.
5. Upload an 11 MB file and a `.txt` renamed `.jpg` → both 400; 11 rapid uploads from
   one IP → the 11th is 429 (`RATE_LIMIT_KYC_UPLOAD_MAX` default 10/min).
6. Admin `/kyc`: row shows Front/Back thumbnails; click opens the full image in a new
   tab; legacy (pre-photo) submissions still render and can be approved/rejected.
7. Arabic: all new strings render correctly in RTL.
8. **If iOS ever ships**: add `NSCameraUsageDescription` +
   `NSPhotoLibraryUsageDescription` to `app/ios/Runner/Info.plist` (image_picker) —
   Android needs no manifest change (system photo picker / camera intent).

## On-chain USDT payments — real-chain test matrix (before prod enable)

Stub-mode e2e is covered by automated tests + dev manual checks; the following
need REAL small-value TRC20 transfers against a staging/prod config (each costs
real USDT + TRX fees):

1. Exact payment: top-up intent $1 → send exactly 1 USDT → confirmed within
   ~1 min, wallet +$1, ledger row `type=topup method=usdt_trc20`, push received.
2. Order flow: place a $1 usdt order (inventory product) → pay exactly → order
   completes with delivered code.
3. Underpay an order intent by $0.50 → order fails, wallet credited $0.50,
   "underpaid" notification.
4. Overpay a top-up by $0.30 → full received amount credited (top-ups credit
   actual). Overpay an ORDER by $0.30 → order fulfilled + $0.30 excess credited.
5. Pay AFTER expiry (set `USDT_INTENT_EXPIRY=2m` on staging): order fails at
   expiry; the late transfer lands as a wallet credit within the grace window.
6. Send a NON-USDT TRC20 token to a deposit address → ignored (no credit).
7. Send the same-address twice: second transfer to an already-confirmed intent's
   address is NOT credited (only the first transfer per intent counts) — funds
   recoverable by the owner via sweep; verify support can find it via
   /api/admin/payments + tronscan.
8. Rate limits: 11 rapid intent creations from one IP → 429 on the 11th.
9. Watcher resilience: restart the API mid-pending-intent → intent still
   confirms after boot (first tick runs immediately).

Items 1–9 above assume DERIVED mode (`USDT_XPUB`). For the shared-address
launch mode (`USDT_ADDRESS`, one fixed address + salted-amount matching), the
matrix changes:

10. Shared exact payment: two top-up intents with the SAME base amount ($1)
    from two accounts → both show the shared address with two DIFFERENT
    amounts (e.g. 1.000001 / 1.000002); send each exactly → both confirm and
    credit their exact amounts. The app's amount field must show/copy the full
    salted value (not "1.00").
11. Shared wrong amount: send the base $1.00 (salt stripped) → intent stays
    pending until expiry; the transfer appears in admin → Payments →
    Unmatched deposits; "Attribute" it to the customer → wallet credited
    (ledger ref `deposit:<txHash>`), customer push received, audit row
    `payment.deposit_attribute` written. "Ignore" a dust transfer → status
    ignored, audit row written.
12. Shared duplicate amount: after intent A confirms, send the SAME amount
    again → second transfer lands in Unmatched deposits (never double-credits).
13. Mode flip: with an open shared intent, flip staging to `USDT_XPUB` (unset
    `USDT_ADDRESS`) and restart → the open shared intent still confirms when
    paid (per-intent mode stamp), new intents get unique derived addresses.
14. Both modes set (`USDT_XPUB` + `USDT_ADDRESS`) → the API refuses to boot
    with a clear error, in dev too.
