# Handoff — Hide admin bulk-email, repurpose it to bulk-SMS

> **Status:** planned, not yet implemented. Execute this in a fresh session. This doc is the
> complete spec; all decisions are locked. Branch off `main`.

## Context / why

SalehCard is **SMS-first**: auth is phone-OTP (Monty), customer notifications are push +
in-app inbox. Email has exactly **one** consumer — the admin *bulk-email* broadcast
(`user/admin.go bulkEmail`) shipped in BL-15 P3 — and prod runs `EMAIL_PROVIDER=log`, so
that button silently sends nothing in production. The client wants no email in the system;
mass outreach should go over SMS like everything else.

So: **remove the email broadcast from the admin UI and replace it with a bulk-SMS
broadcast** that reuses the existing Monty SMS adapter. The `platform/email` port stays in
the tree, dormant (unreferenced), for the day a real "email me my receipt" requirement
appears — we hide/replace the *feature*, we don't rip out the *seam*.

### ⚠️ Money warning (biggest risk)
Unlike email (which was log-only in prod), **prod already has Monty configured for OTP**, so
bulk-SMS goes **live and sends real, paid SMS the moment this deploys**. Every guardrail
below is load-bearing. Lebanon SMS is billed per 160-char segment.

## Decisions locked (from the user)
- **Guardrail = cap + confirm.** Backend hard-caps recipients per send at `BULK_SMS_MAX`
  (new env, default **200**) → `400` if exceeded. UI shows a `window.confirm` with the exact
  recipient count before firing.
- **Message = hard 160-char cap** (single SMS segment). `maxLength=160` textarea + live
  `n / 160` counter; backend also rejects `len(message) > 160` (belt-and-suspenders). **No
  subject** (SMS has none).
- Recipients come from `User.Phone` (skip nil/empty phone and `StatusDeleted`), mirroring how
  bulk-email skipped empty-email/deleted.
- `platform/email` package stays dormant (not deleted). Email config fields stay (harmless).

---

## Backend changes (`api/`)

Reuse the existing SMS port — **do not** reinvent. `sms.Sender` is
`Send(ctx context.Context, phoneE164, message string) error` (`platform/sms/sms.go:13-16`);
adapters monty/twilio/log; `sms.New(cfg)` switch. Contrast with email's `Send(ctx, Message)`.

### 1. `internal/server/server.go` — swap the mailer wiring for an SMS sender
The `mailer` block (lines **110-126**) and the `email` import (line 34) are now unused —
**replace** them with an SMS-sender block mirroring the same fail-open pattern:
```go
smsSender, err := sms.New(sms.Config{
    Provider: s.cfg.SMSProvider,
    Monty:    sms.MontyConfig{ /* BaseURL, Username, APIID, AccessToken, SenderID, Campaign from cfg */ },
    Twilio:   sms.TwilioConfig{ /* AccountSID, AuthToken, From from cfg */ },
})
if err != nil {
    slog.Warn("server: SMS provider misconfigured — falling back to log sender", "error", err)
    smsSender = sms.LogSender{}
}
```
(Copy the exact field mapping from `user/routes.go:30-51`, which already builds this for OTP.)
Then change the admin registration (line **150**):
```go
user.RegisterAdminRoutes(r, s.db, rec, smsSender, s.cfg.BulkSMSMax)
```
Swap import `platform/email` → `platform/sms`. Note: the OTP sender in `user.RegisterRoutes`
stays as-is; this is a second, admin-scoped construction. Minor duplication, zero blast radius
on customer routes — a future cleanup can DRY both into one hoisted sender if desired.

### 2. `internal/modules/user/admin.go` — `bulkEmail` → `bulkSMS`
- Struct (line 61): `mailer email.Sender` → `smsSender sms.Sender`; add `maxBulkSMS int`.
- Signature (line 68): `RegisterAdminRoutes(r chi.Router, db *mongo.Database, rec audit.Recorder, smsSender sms.Sender, maxBulkSMS int)`; assign both fields.
- Route (line 79): `r.Post("/users/bulk-email", a.bulkEmail)` → `r.Post("/users/bulk-sms", a.bulkSMS)`.
- Handler (lines 266-324) → `bulkSMS`:
  - Body: `{ IDs []string; Message string }` (drop `Subject`).
  - Validate: `Message` trimmed + required; reject `len(message) > 160` → `400`.
  - Resolve recipients via existing `a.repo.FindByIDs(ctx, ids)`, collecting `u.Phone`
    (**pointer** — nil-check then `strings.TrimSpace`) and skipping `u.Status == StatusDeleted`.
  - **Cap:** if `len(recipients) > a.maxBulkSMS` → `400` code `BULK_SMS_LIMIT`
    (message names the cap so the admin knows to narrow the selection).
  - Fan-out: keep the exact detached pattern (`context.WithoutCancel` + `context.WithTimeout(bg, bulkEmailTimeout)`
    — rename const to `bulkSMSTimeout`, keep 2 min; serial loop, log-and-continue) but call
    `a.smsSender.Send(ctx, phone, message)`. Serial is *correct* here — avoids hammering Monty.
  - Audit: `audit.ActionUserBulkSMS` (new const), `TargetType "user"`, `TargetID "bulk"`,
    summary `{"recipients": n}` (no subject).
  - Response: `response.OK(w, map[string]int{"queued": len(recipients)})` (unchanged shape).
- Imports: drop `platform/email`, add `platform/sms`. `strings`/`context`/`time`/`slog` stay.

### 3. `internal/modules/audit/model.go`
Add `ActionUserBulkSMS = "user.bulk_sms"`. The old `ActionUserBulkEmail` const may be left
(unused consts are legal in Go) or removed for tidiness.

### 4. `internal/config/config.go`
Add `BulkSMSMax int` via the existing `getInt` helper — env `BULK_SMS_MAX`, default **200**
(mirror the `RATE_LIMIT_*` knobs). Existing `EMAIL_PROVIDER`/`SMTP_*`/`SENDGRID_*`/`EMAIL_FROM`
fields become dormant — leave them (harmless; removing is optional cleanup).

### 5. `internal/platform/email/*`
No change. Package stays in the tree, imported nowhere (dormant). Its tests keep passing.

---

## Frontend changes (`admin/`)

All three touch points are in the Users feature; the email flow is fully built (not a stub),
so this is a rename + field-swap, not new scaffolding. Clone the email path exactly.

### 1. `admin/src/features/users/pages/UserListPage.tsx`
- Bulk-bar button (lines 170-172): label `Email` → `SMS` (keep the `send` icon), state
  `emailIds`→`smsIds` (declared line 45), `setEmailIds(bulk.sel)`→`setSmsIds(bulk.sel)`.
- Modal render (lines 254-263): `BulkEmailModal` → `BulkSMSModal`.
- `BulkSMSModal` (rename `BulkEmailModal`, lines 268-325, same file):
  - **Drop the Subject `<input>`** (line 303). Keep one Message `<textarea className="afield">`
    with `maxLength={160}` + a live counter `{message.length} / 160` beneath it.
  - Heading `SMS {ids.length} user(s)`; subtext "Users without a phone number are skipped
    automatically."
  - `submit`: validate message non-empty → **`window.confirm(`Send SMS to ${ids.length} user(s)? This sends real messages.`)`**
    → `send.mutate({ ids, message })`. (Confirm at send-time, after composing.)
  - `onSuccess`: `toast.success(`SMS queued to ${res.data?.queued ?? 0} recipient(s).`)` then
    `onSent()`. Keep inline `error` for the failure path.
  - Note the count nuance: the confirm shows *selected* count; backend skips phone-less/deleted,
    so the toast's `queued` may be lower — that's expected and fine.

### 2. `admin/src/features/users/api/users.ts`
`bulkEmailUsers(ids, subject, body)` (lines 101-105) → `bulkSmsUsers(ids: string[], message: string)`:
`apiClient.post<{ queued: number }>(`${ADMIN}/bulk-sms`, { ids, message })`.

### 3. `admin/src/features/users/hooks/useUsers.ts`
`useBulkEmail` (lines 76-81) → `useBulkSms`, `mutationFn: ({ ids, message }) => bulkSmsUsers(ids, message)`.
No query invalidation (sending SMS doesn't change the user list), same as before.

---

## Config / DevOps

- **New env knob** `BULK_SMS_MAX` (default 200). Reuses the **existing** `SMS_PROVIDER` + Monty
  creds already set in prod — no new provider config needed.
- **`DEVOPS-TODO.md`:** replace **item 9 (email provider)** with a bulk-SMS note:
  *bulk-SMS reuses the live Monty provider → sends real SMS in prod on deploy; guarded by
  `BULK_SMS_MAX` (default 200) + an admin confirm dialog + 160-char cap. Tune the cap via the
  app setting. `EMAIL_*` settings are now unused and can be left unset.* Keep items 8, 10, 11.

## Verification

**Backend** (`cd api && go build ./... && go vet ./... && go test ./...`):
- Repurpose the existing bulk-email unit test if present (search `admin_test.go` for `bulkEmail`),
  else add one with a fake `sms.Sender` counting calls, asserting: (a) skips nil-`Phone` +
  `StatusDeleted`, (b) `400` when recipients exceed `maxBulkSMS`, (c) `400` when message > 160,
  (d) `Send` called once per valid recipient, (e) `queued` = valid-recipient count.

**Frontend** (`cd admin && pnpm build && pnpm lint && pnpm test`).

**E2e (dev: mongo :27017, API `ENV=development JWT_SECRET= PORT=8090 SMS_PROVIDER=log`, admin :5174):**
1. Select users → bulk bar → **SMS** → modal opens (no Subject, 160-char counter, textarea won't
   exceed 160).
2. Send → confirm dialog shows the selected count → on confirm, toast "SMS queued to N
   recipient(s)"; with `SMS_PROVIDER=log` each message is logged, not sent.
3. Users without a phone / deleted are excluded from `queued`.
4. Select > `BULK_SMS_MAX` users (or lower the cap) → backend `400 BULK_SMS_LIMIT`, surfaced inline.
5. Audit page shows a `user.bulk_sms` row (actor = admin, recipients count).

## File index
**Backend:** `internal/server/server.go` (110-126, 34, 150) · `internal/modules/user/admin.go`
(25, 56-85, 266-324) · `internal/modules/audit/model.go` · `internal/config/config.go` ·
`platform/email/*` (unchanged, dormant) · reuse `platform/sms/sms.go`.
**Frontend:** `admin/src/features/users/pages/UserListPage.tsx` (45, 170-172, 254-263, 268-325) ·
`admin/src/features/users/api/users.ts` (101-105) · `admin/src/features/users/hooks/useUsers.ts` (76-81).
**Docs:** `DEVOPS-TODO.md` (item 9).

## Out of scope / do not touch
- Do **not** delete `platform/email` or its config fields (dormant seam kept intentionally).
- Do **not** un-hide the reseller admin surface (deferred by the client).
- Do **not** build filter-based recipient selection — ids-only, mirroring the current flow.
