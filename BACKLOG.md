# SalehCard — Product Backlog

Source: post-launch business-logic/UX audit, 2026-07-02 (after the go-to-market
hardening series shipped to prod), trimmed 2026-10-16. Pick an item by its ID and
read its notes — they carry the context a fresh session needs. Ops work lives in
`DEVOPS-TODO.md`, deferred manual checks in `QA-TODO.md`.

---

## Open

### BL-15 Platform leftovers (post-launch growth work)
Most of the original BL-15 list has shipped: the settings module, on-chain USDT
(TRC20 + BEP20), upstream supplier adapters (`HANDOFF-SUPPLIERS.md`), auth/OTP
rate limiting, RBAC custom roles, code resend + expiry, and the per-reseller price
table. Still open:
- **Card payment gateway** — checkout is wallet-only (card is rejected with
  `PAYMENT_METHOD_UNAVAILABLE`). Whish Pay redirect flow is code-complete but
  blocked on Whish sandbox credentials.
- Deferred from BL-14: bulk order **refund**, bulk **email** (needs an active
  `platform/email` provider — dormant today), **PDF** export, an admin toast system.

### BL-16 Fee-free Binance-native USDT payments (enhancement)
**Why:** The on-chain USDT watcher only sees payments that actually touch the
chain. A Binance user paying to a Binance **deposit** address triggers a **free,
instant, OFF-CHAIN internal transfer** (Txid literally reads "Off-chain Transfer",
0 fee) that no chain watcher can ever see — this was the 2026-07-11 "payments not
received" incident (root cause proven; see memory `salehcard-usdt-onchain-payments`).
The current fix is a **non-custodial receiving address** so every payment is
on-chain — but that forces the customer to pay Binance's **~1.5 USDT withdrawal
fee + 5 USDT minimum**, which is costly for small purchases. This item is the
cheaper alternative: **accept the free internal transfer and confirm it by reading
Binance instead of the chain.** Bonus: internal transfers preserve the *exact*
amount (no fee deducted), so amount-salt matching is actually cleaner off-chain.
**Two build options (decide first):**
- **Read-only Binance deposit-history API** on the receiving account (Enable-Reading
  key only, IP-locked to NAT egress `52.157.69.89`; poll deposit history, match by
  exact salted amount). Keeps today's pay-to-address UX + a normal account. ⚠️ First
  verify Binance's deposit-history endpoint actually surfaces auto-routed internal
  transfers with a matchable amount + id — unconfirmed.
- **Binance Pay merchant** (purpose-built: free/instant for the customer, webhook
  auto-confirm, no API-key sharing). Needs a Binance **Merchant** account (business
  KYB) — heavier onboarding, best long-term.
**Open question carried forward:** whose Binance account is the receiver long-term
(client's vs a business account we control) — decides key custody + merchant feasibility.

---

## Done (details in git history)

- ✅ BL-1 Wallet refresh loop (Flutter)
- ✅ BL-2 Pending top-up requests visible on the Wallet screen (Flutter)
- ✅ BL-3 Admin work-queue signals (dashboard KPIs + nav badges)
- ✅ BL-4 Home screen search + bell wiring (Flutter)
- ✅ BL-5 Purge stale "instant" copy (Flutter l10n)
- ✅ BL-6 Web order detail must reflect order status
- ✅ BL-7 Remove remaining mock data from live web pages
- ✅ BL-8 Admin order-detail edges
- ✅ BL-9 Reseller funding coherence
- ✅ BL-10 Backend small inconsistencies
- ✅ BL-11 Notifications: build minimal or hide
- ✅ BL-12 Send Money: hide or implement
- ✅ BL-13 Cart: enable or remove
- ✅ BL-14 Admin "coming soon" set
