---
name: salehcard-gtm-pr-stack
description: "Go-to-market hardening PR stack (#15→#20, 2026-07-02) — merge order, locked product decisions, launch checklist"
metadata: 
  node_type: memory
  type: project
  originSessionId: f3195803-c2ac-4031-b9f9-3f1069b8c66e
---

On 2026-07-02 a go-to-market audit + fix pass produced a **stacked PR chain** (each based on the previous; merge bottom-up, retarget each to main after its base merges; every merge to main auto-deploys prod):

1. **#15** `fix/security-hardening` — removes UNAUTHENTICATED `POST/PATCH/DELETE /api/v1/products` (was live in prod!), adds `config.Validate()` (JWT_SECRET + ALLOWED_ORIGINS required outside development), flips `ENV` default to `production` (fail-closed), gates the AdminOnly bypass on `devBypass` (ENV=development). **Merge ASAP.**
2. **#16** `feat/audit-log` — audit module (`admin_audit_log`, Recorder into user/product/kyc/review/reseller/order/wallet admin handlers), `/audit` admin page, last-admin 409 guard, mandatory money ledgers (reverse balance on ledger insert failure; wallet Debit/Refund compensate internally so retries are safe).
3. **#17** `feat/order-admin-actions` — `TransitionStatus` guarded FindOneAndUpdate (double-refund lock, returns pre-image); refund (processing|completed→refunded, wallet credit-back, **codes never re-pooled**) + complete (processing→completed only) endpoints + admin UI modals.
4. **#18** `feat/wallet-only-kyc-gate` — PlaceOrder accepts ONLY wallet (card/usdt were mock-approved = free inventory); `kyc.Gate.IsApproved` gates ALL purchases (403 KYC_REQUIRED); Flutter checkout wallet-only + KYC panel; legacy /web patched too.
5. **#19** `feat/topup-requests` — instant mock top-up replaced by admin-approved request queue (`topup_requests`; channels usdt/whish/omt/cash/other; claim→credit→mandatory-ledger with compensation); Flutter request form + history; admin `/topups` queue page.
6. **#20** `chore/admin-polish` — removes false "Dev mode" login note, prefilled email, fake badges/ping/"Super admin" label, mock Settings tabs, no-op Export; deletes dead mock/demo.ts; rewrites BACKLOG.md.

**Locked decisions (user-confirmed):** wallet-only launch (no gateway yet); KYC gates all purchases; audit+guardrails required. Assumed w/ user AFK: top-up request queue (vs disabling), web minimal patch (vs retiring).

**Launch checklist before/at merge:** Azure app settings `ENV=production` (new fail-fast will refuse boot if JWT_SECRET/ALLOWED_ORIGINS missing — intended), [[salehcard-business-tz-todo]] `BUSINESS_TZ=Asia/Beirut`; verify `admin_audit_log` + `topup_requests` indexes on Cosmos vCore; seed a SECOND admin account (last-admin guard); dress rehearsal: signup → KYC approve → top-up request approve → wallet order (code product instant; manual product admin-complete) → refund one → check ledger/audit/no code re-pool.

**SHIPPED 2026-07-02:** all six PRs merged to main (merge commit `9594cbf`, + `13dc43b` adding `workflow_dispatch` to deploy.yml). Home-screen KYC banner added (KYC skippable at signup; banner → /kyc; checkout gate enforces). CLAUDE.md updated to shipped reality.

**DEPLOYED 2026-07-02** after the user fixed GitHub Actions billing (all runs had been `startup_failure` — account-level block); deploy verified live (405 on unauth product POST, 401 on admin routes, health ok). Two post-ship regressions found by audit and fixed same day (`a1cf2b6`): Flutter checkout balance source (returning users locked out) and web top-up sending `method` instead of `channel` (every attempt 400'd). **Post-launch product backlog lives in root `BACKLOG.md` (BL-1…BL-15)** — the user plans each item as its own task/fresh session; start from the BL entry. Historical record of the original blocker:

**DEPLOY BLOCKED (resolved — kept for context):** every GitHub Actions run since 09:24 that day — all branches/workflows — is `startup_failure` with a phantom "BuildFailed" workflow. GitHub status healthy, workflows valid/unchanged, repo Actions enabled → account-level Actions block, almost certainly **billing** (private repo; runs worked 2026-06-27). Fix at github.com/settings/billing, then `gh workflow run "Deploy (prod)"` (workflow_dispatch deploys all 3 components regardless of path filters). Local az fallback fails on the corporate-proxy TLS gotcha even with AZURE_CLI_DISABLE_CONNECTION_VERIFICATION (Python strict-CA error); use Azure Cloud Shell if billing can't be fixed. **Prod is still running pre-#15 code with the unauthenticated product-mutation hole until this deploys.**

Plan file: `C:\Users\user\.claude\plans\yes-procced-plan-ahead-lazy-corbato.md`. Commit trailer used: the CLAUDE.md-specified Opus one.
