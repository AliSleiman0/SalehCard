# Handoff — 2026-07-02 (P1 + P2 backlog complete → only P3 remains)

The prioritized backlog's **entire P1 + P2 set (BL-1 … BL-10) is now code-complete
and shipped to prod.** This session finished the last three (BL-9, BL-10, BL-3),
bundled them into **one** PR to avoid multiple prod deploys, and merged. What's
left in `BACKLOG.md` is **only P3 (BL-11 … BL-15)** — all decision-gated.

Read `BACKLOG.md` for the item list and `CONVENTIONS.md` for the rules. This file
is the bridge. The earlier `HANDOFF-2026-07-02-backlog.md` covers the Flutter
BL-1/BL-2 session and its l10n/analyze gotchas — still useful for `/app` work.

---

## Shipped & deployed today

- **main** is at merge `adad9dc` (PR #26). Push to `main` auto-deploys api + admin
  to Azure. **The PR #26 deploy was triggered on merge (~19:11 UTC) — confirm it
  went green:** `gh run list --repo AliSleiman0/SalehCard --workflow=deploy.yml`.
- Earlier P1/P2 items landed in PRs #21–#25 (BL-1,2,4,5,6,7,8). Flutter items
  (BL-1/2/4/5) deploy on the **separate** mobile path, not this api/web/admin CI/CD.

### Final bundle — PR #26 (three commits, one deploy)
- **BL-9 — Reseller funding coherence.** Role added to the admin top-up queue
  (`wallet/admin.go` `adminTopUpView`) + role chip on `TopupsPage`. Convention set
  in `CONVENTIONS.md`: **fund all wallets via the top-up queue** (ledger `topup`,
  counted by `TopUpsForDay` + `finance.walletTopups`); **balance-adjust
  (`adjustment`) is corrections-only**, excluded from those KPIs.
- **BL-10 — Backend small inconsistencies.**
  - Wallet `TopUpsForDay` now buckets by **business timezone** (`BUSINESS_TZ`),
    matching `order.DayStats`. Shared loader extracted to **`api/pkg/timeutil`**
    (`BusinessLocation()`); `order` and `wallet` both consume it.
  - **Never-blank admin attribution:** `Phone` added to `auth.Claims`, populated at
    the single mint site (`user/service.go issueTokens`); new helper
    **`auth.ActorLabel(claims)`** = email → phone → user-id. Applied to top-up
    `DecidedBy` (`wallet/admin.go`), the audit recorder (`audit/repository.go`,
    fills only when email blank), KYC `reviewedBy`, and code `UploadedBy`.
  - **Removed the dead `ResellerTier.BalanceLimit`** (editable but enforced
    nowhere) from backend model/CRUD/seed/tests and the admin tier editor. Old
    Mongo docs keep a stray `balanceLimit` key — schemaless, ignored, no migration.
- **BL-3 — Admin work-queue signals.** New admin-wide
  `wallet.TopUpStore.CountPending`; `pendingTopups` + `pendingKyc` (reusing
  `kyc.CountPending`) on `GET /api/admin/dashboard/stats`. Two clickable dashboard
  KPI tiles (→ `/topups`, → `/kyc`) and **live amber sidebar badges** fed from the
  shared `useDashboardStats()` query (no extra request).

**Static gates (all green at merge):** api `go build`/`go vet`/`go test ./...` no
failures; admin `pnpm build` ✓, `pnpm lint` 0 errors (3 pre-existing react-refresh
warnings in `Art.tsx`/`Badges.tsx` are harmless), `pnpm test` 36/36.

---

## Deferred: manual / e2e verification (standing user choice)

Everything shipped on green static gates only. Before considering these "closed",
run against API :8090 + admin :5174 + seeded data:
- **BL-3:** seed a pending top-up + pending KYC → dashboard shows non-zero
  "Pending top-ups"/"Pending KYC" tiles that navigate to `/topups`/`/kyc`; sidebar
  shows amber count badges that clear when the queues empty.
- **BL-9:** a reseller's top-up request shows an amber **reseller** chip in
  `/topups` vs a customer's muted chip; approval still writes a `topup` ledger row.
- **BL-10:** (1) set `BUSINESS_TZ=Asia/Beirut`, place a top-up + completed order
  near the UTC/Beirut date line → both "today" KPIs count them in the same day;
  (2) approve a top-up as a **phone-only admin** → `decidedBy` + audit actor show
  the phone, not blank; (3) tier editor no longer shows a Balance-limit field.

---

## What remains: P3 only (BL-11 … BL-15) — decide, then build or hide

None are started. All are decision-gated (see `BACKLOG.md:110-149`):
- **BL-11 Notifications** — build a minimal backend module (recommended) OR hide
  the app screen/bells. *Build* touches **api** (+ Flutter); *hide* is Flutter-only.
- **BL-12 Send Money** / **BL-13 Cart** — **Flutter-only** hide-or-implement
  decisions; **do not** trigger this api/admin CI/CD.
- **BL-14 Admin "coming soon" set** — CSV/PDF exports, bulk actions, add-reseller
  flow, `/audit` nav entry, etc. (admin, larger).
- **BL-15 Platform stubs** — settings API (501), real payment gateway, upstream
  fulfillment providers, code re-deliver/expire, auth/OTP rate limiting, granular
  admin roles. Post-launch growth work.

---

## Workflow notes / gotchas (carry forward)

- **Minimize prod deploys — batch.** Every merge to `main` = one path-filtered
  Azure deploy. The user explicitly prefers **one deploy**: stack independent
  api/admin items on **one branch/PR** and merge once (that's how BL-9+BL-10+BL-3
  shipped together). Only branch separately when items must deploy independently.
- **Per-item loop:** plan from `BACKLOG.md` in Plan Mode → `ExitPlanMode` approval
  → implement → static gates → commit to a feature branch off `main` → PR →
  **merge only on explicit command** (merging ships prod).
- **Gates:** `cd api && go build ./... && go vet ./... && go test ./...`;
  `export PATH="$HOME/.local/bin:$PATH"; cd admin && pnpm build && pnpm lint &&
  pnpm test`. The Windows `unlinkat …test.exe … used by another process` line is a
  harmless cleanup artifact, not a failure.
- **Git:** commit trailer `Co-Authored-By: Claude Opus 4.8 (1M context)
  <noreply@anthropic.com>`. GitHub repo is **`AliSleiman0/SalehCard`** (capital
  S/C; `origin` still points at the lowercase URL and redirects on push).
  golangci-lint on PRs is stricter than `go vet`.
- **Manual/e2e testing is deferred by standing user choice** across this whole
  series — ship on green static gates; log the deferred checks (above).
- New shared pkg **`api/pkg/timeutil`** is the single source for `BUSINESS_TZ`
  day-bucketing — use `timeutil.BusinessLocation()` for any new day-grained KPI
  instead of hard-coding UTC.
