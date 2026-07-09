---
name: salehcard-usdt-onchain-payments
description: "On-chain USDT (TRC20) auto-confirming payments built for SalehCard (2026-07-06) — new payment module + tron platform + Flutter/admin UI; how it's structured, gated, and rolled out"
metadata: 
  node_type: memory
  type: project
  originSessionId: 594e57fa-c223-462c-a3f3-4ad9a4e19007
---

Built on-chain USDT (TRC20) payments with automatic confirmation for SalehCard
(2026-07-06), settling BOTH wallet top-ups and direct order checkout. Structure
was ported from LACPA's hexagonal payments module (`C:\Users\user\Lacpa\Backend\payments`)
but adapted for on-chain (no redirect/callback — a background watcher polling
TronGrid is the ground truth). Wallet stays float64 USD; the payment module is
int64 micro-USDT internally. See [[salehcard-gtm-pr-stack]], [[salehcard-dev-env]].

**Locked decisions**: unique deposit address per intent (watch-only HD derivation
from an xpub, BIP44 m/44'/195'/0'/0/i — no keys on server); TRC20 only; no stock
reservation (stockout → wallet credit); Flutter polls; 1 USDT = 1 USD.

**Backend** (`api/`):
- `internal/platform/tron/` — Reader port + trongrid/stub adapters + `DeriveAddress`
  (btcsuite hdkeychain+base58 + keccak). `USDT_PROVIDER` selects (default stub).
- `internal/modules/payment/` — Intent state machine (pending→confirming→confirmed/
  expired), Mongo store (unique partial index on `(network,txHash)` = double-credit
  guard), Service (settle with wallet-compensation), Watcher (first long-running
  worker in the codebase — started in `cmd/server/main.go` graceful shutdown).
- `wallet.WalletService.TopUp` (credit+ledger, `method:"usdt_trc20"`, unique partial
  `(method,ref)` index makes settlement retries idempotent).
- order integration: `order → payment` import; `paySvc.SetOrderSettler(orderSvc)`
  closes the loop in server.go. usdt order = pending + intent, watcher confirms →
  `FulfillPaidOrder` (returns `payment.ErrOrderNotPayable` on stockout/expiry →
  wallet credit). Order status reuses `pending` (no new awaiting_payment).

**Clients**: Flutter `app/lib/features/payments/` slice + `usdt_deposit_screen`
(QR via qr_flutter, copy, countdown, 7s lifecycle poll); entry points in
topup_screen (usdt channel) + checkout `_PaymentSelector`. Admin
`admin/src/features/payments/` read-only page (grp_finance nav, tronscan links).

**Feature gate**: enabled iff `USDT_XPUB` set. `GET /api/v1/payments/config`
answers `usdtEnabled`. `config.Validate` refuses stub+xpub outside dev.

**Prod rollout** deferred — see DEVOPS-TODO.md §14 (generate xpub OFFLINE, TronGrid
key, Always On, sweep runbook) + QA-TODO.md real-chain matrix. All 4 PRs
code-complete + verified stub-mode e2e (top-up AND order both confirmed, wallet
untouched by usdt orders); go build/vet/test (28 pkgs), flutter analyze, admin
build+lint all green. Committed on branch `feat/usdt-onchain-payments` as 3
commits (backend / flutter / admin), NOT pushed, no PR opened yet. Plan file:
`~/.claude/plans/plan-full-implementation-stateful-stream.md`.
