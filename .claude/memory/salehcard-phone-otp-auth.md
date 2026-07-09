---
name: salehcard-phone-otp-auth
description: Phone-OTP auth shipped (Go API + Flutter); SMS via hexagonal adapters (Monty/Twilio/log), select with SMS_PROVIDER
metadata: 
  node_type: memory
  type: project
  originSessionId: 643ed6d1-5194-4c5b-bb81-6b4a6ab5479a
---

Phone-OTP authentication was added across the Go API and the Flutter app (2026-06-26),
alongside the existing email/password and the new phone+password login — the login
screen lets the user choose code vs password.

- **Backend** (`api/internal/modules/user/` + `internal/platform/sms/`): endpoints
  `POST /auth/otp/request`, `/auth/otp/verify` (find-or-create by phone, optional
  password set), `/auth/login-phone`. `User` gained a sparse-unique `phone`; codes
  live in `otp_codes`. SMS goes through a `sms.Sender` interface.
- **SMS uses a hexagonal port/adapters layer** (`internal/platform/sms`): port `Sender`,
  adapters `monty.go` / `twilio.go` / `log.go`, and a `New(Config)` factory selected by
  **`SMS_PROVIDER`** (`monty` | `twilio` | `log`, default `log`). `monty` is the primary
  (Lebanon, Monty Mobile); `twilio` is the international option; `log` (default) logs the
  code to the server console. Add a provider = new adapter file + one case in `New`.
  OTP tunables: `OTP_LENGTH/TTL/RESEND_INTERVAL/MAX_ATTEMPTS`, `DEFAULT_COUNTRY_CODE=+961`.

**Status (2026-06-27): shipped to prod, finishing egress.** Merged to `main` (PR #7) and
auto-deployed to Azure; prod runs `SMS_PROVIDER=monty`, sender **`Arya`** (the approved
Source; "SalehCard" not yet registered). Remaining: prod sends fail with `-8 Invalid Source`
until the **Azure egress IP is whitelisted on Monty** — being fixed with a **NAT Gateway**
(one fixed IP; also add that IP to the Cosmos vCore firewall). Dev IP `213.204.66.98` is
already whitelisted. See `HANDOFF-2026-06-27.md`.

**Monty adapter verified against the live API.** Real endpoint is
`GET https://sms.montymobile.com/API/SendSMS` with header
`X-Access-Token` + query `username, apiId, json=True, destination(+E.164), source,
campaignname, text`; reply `{ErrorCode (0=ok), Description, Id, MessageCount}`. Env:
`SMS_PROVIDER=monty`, `MONTY_USERNAME`, `MONTY_API_ID`, `MONTY_ACCESS_TOKEN`,
`MONTY_SENDER_ID` (=Source), optional `MONTY_CAMPAIGN`, `MONTY_BASE_URL`
(default `https://sms.montymobile.com`). The Go adapter reaches Monty fine through the
corporate proxy (Go skips revocation; curl needs `--ssl-no-revoke`). **Blocker:** the test
account (`ozconSer`) rejects sources `SalehCard`/`ozconSer`/empty with `-8 Invalid Source` /
`-2 Empty Source` — a registered/approved Sender ID is required. Twilio path:
`SMS_PROVIDER=twilio` + `TWILIO_ACCOUNT_SID/AUTH_TOKEN/FROM`. The mip SMS gateway is **not**
used by SalehCard (its adapter + `SMS_GATEWAY_*` config were removed).

Known rough edge: an SMS send failure in `RequestOTP` currently surfaces as a generic
HTTP 500 `INTERNAL_ERROR` (the raw error isn't a typed `AppError`) — worth mapping to a
clearer code.

For OTP login to work **on the phone**, the app must point at an API that has these
endpoints — local via `adb reverse` (debug build), or Azure once the API is redeployed
(deploy was deferred; release APK currently targets Azure which lacks these endpoints).

Related: [[salehcard-flutter-mobile]] · [[salehcard-prod-deploy]] · [[salehcard-dev-env]]
