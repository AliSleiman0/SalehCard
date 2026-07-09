---
name: salehcard-whish-payments
description: Whish Pay (redirect+callback) implemented code-complete & stub-verified; blocked on receiving Whish test/sandbox API creds for live e2e; handoff at HANDOFF-WHISH-IMPL.md
metadata: 
  node_type: memory
  type: project
  originSessionId: 697c0b1d-3773-4e8a-84ff-bf1ff4867ea0
---

Whish Pay (Lebanese wallet, **redirect-hosted checkout + HMAC-authed server
callback**) was implemented across backend, Flutter app, and admin on 2026-07-06,
**generalizing the existing `payment` module** to host a second provider next to
USDT (rather than a parallel module). Settles both wallet top-ups and orders;
app opens the hosted page via `url_launcher` (external browser); dark until
`WHISH_*` configured. Ground truth = re-poll Whish's `GetStatus` in the callback
(never trust the browser redirect). Blueprint was LACPA `C:\Users\user\Lacpa\Backend\payments\`.

**Status:** code-complete, **verified in stub mode** (go build/vet/test green incl.
8 new Whish tests; live boot smoke test: enabled log, index migration clean,
webhook token-authed 400s, health 200; flutter analyze + admin build clean).
**NOT committed** — sits on top of the (also uncommitted) USDT work on branch
`feat/usdt-onchain-payments`.

**BLOCKED / next:** we don't yet have Whish's **test/sandbox API creds** — will
arrive later. When they do, follow the resume steps in
`salehcard/HANDOFF-WHISH-IMPL.md` §4: set `WHISH_PROVIDER=whish` + sandbox creds +
a public tunnel for `PAYMENTS_WEBHOOK_BASE_URL`, run the `QA-TODO.md` Whish
matrix (sandbox test values: phone `96170902894` / OTP `111111`; any other OTP =
failure), then get SalehCard's **own prod merchant account** per `DEVOPS-TODO.md` §15.

**Key files:** `api/internal/platform/whish/*`, `api/internal/modules/payment/{token,sweeper,whish_test}.go`
(+ edits to model/repository/service/handler/routes/admin), wallet index migration,
config `WHISH_*`/`PAYMENTS_*`, order `PaymentMethodWhish`, server wiring;
`app/.../whish_redirect_screen.dart`; admin PaymentsPage provider badge.

**Watch-outs:** LACPA reference creds (`channel 10200046`, `lacpa.academy`) are
sandbox-only + chat-exposed → ask LACPA to rotate; SalehCard needs its own account.
Whish may **IP-allowlist the caller** like Monty SMS → may need the prod NAT
Gateway IP registered. The wallet `(method,ref)` index auto-migrates
usdt_trc20-only → `{usdt_trc20, whish}` at boot.

Related: [[salehcard-usdt-onchain-payments]] (the template/precedent), the older
[[salehcard-whish-handoff]] planning note (now superseded by HANDOFF-WHISH-IMPL.md),
[[salehcard-flutter-emulator-run]] (how to run the app vs local API),
[[salehcard-phone-otp-auth]] (Monty SMS IP-allowlist precedent).
