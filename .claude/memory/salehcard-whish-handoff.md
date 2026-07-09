---
name: salehcard-whish-handoff
description: "Whish Pay is NOT yet implemented in SalehCard — next-session handoff written; where the spec, the LACPA blueprint, and the key decision live"
metadata: 
  node_type: memory
  type: project
  originSessionId: 594e57fa-c223-462c-a3f3-4ad9a4e19007
---

**Whish Pay is NOT implemented in SalehCard** (only exists as a manual out-of-band
top-up channel LABEL). The next session should implement it. A self-contained
handoff is at **`salehcard/HANDOFF-WHISH.md`** (created 2026-07-06).

Key facts it pins:
- **Spec PDF:** `C:\Users\user\Downloads\WHISH PAY Web Service - Technical Specification - v1.4.2.pdf`
  (endpoints transcribed in the handoff §3: `POST /payment/whish` → collectUrl;
  `POST /payment/collect/status`; unsigned GET callbacks; sandbox test phone
  96170902894 / OTP 111111).
- **LACPA blueprint (code to port):** `C:\Users\user\Lacpa\Backend\payments\providers\whish\`
  + `provider.go` / `service.go` (HandleCallback) / `token.go` (HMAC for unsigned
  callbacks) / `ports/http.go` (webhook routes before the auth group).
- **Architectural crux:** Whish is redirect+callback (fiat), which does NOT fit the
  watcher-poll shape of the USDT module shipped in PR #51 (see
  [[salehcard-usdt-onchain-payments]]). Recommended: generalize `modules/payment`
  with a `Provider` port (Initiate→redirectURL, GetStatus) + add an Intent
  `redirectUrl`/`externalId` + a public webhook route (net-new for SalehCard) +
  HMAC tokens. Reuse the existing wallet/order settler ports.
- **Creds shared in chat (channel 10200046 / secret 9bdaee6… / lacpa.academy) are
  LACPA's**, reference/sandbox only — SalehCard needs its OWN Whish merchant
  account; flag LACPA to rotate the chat-exposed secret. Load via `WHISH_*` env.
- Lebanon rail (same audience as Monty SMS, [[salehcard-phone-otp-auth]]) — check
  whether Whish IP-allowlists the caller like Monty does.

HANDOFF-WHISH.md is untracked (not committed) as of writing.
