# HANDOFF — Supplier Integration (all 4 phases shipped, activation pending)

**Date:** 2026-07-15
**Branch/worktree:** `worktree-feat-supplier-phase1` (`.claude/worktrees/feat-supplier-phase1`)
**Status:** ✅ **All code merged + deployed to prod. Zero engineering work remains.** What's
left is credentials + supplier-side allowlisting + in-console config (canary).

---

## TL;DR for the next session

The 4-phase supplier integration (three white-label game panels + umanage telecom) is
**live in prod but inert/dark**. It fulfills nothing until credentials are set and the
prod NAT IP is allowlisted at each supplier. Do **not** re-implement anything — read this,
then work only the activation checklist below.

- **Merged:** PR #97, merge commit `8e16f9f`. CI + Deploy(prod) both green.
- **Design doc:** `DESIGN-SUPPLIERS.md` (repo root). **Rollout checklist:** `DEVOPS-TODO.md §20`.
- **Memory:** `salehcard-supplier-integration.md` (full technical record).

---

## Session addendum — 2026-07-16 (custom domains + CORS)

Not supplier code — infra work discovered while activating. Prod is moving to the
**`flashcashglobal.com`** custom domain (Cloudflare DNS, DNS-only). No repo code changed.

- **CORS (DONE, verified live).** The prod API's `ALLOWED_ORIGINS` (Azure app setting on
  `salehcard-api`) now includes `https://www.flashcashglobal.com` and
  `https://admin.flashcashglobal.com` alongside the existing `*.azurestaticapps.net` origins +
  `http://localhost:5174`. Config-only (env-driven, `config.go:300`); auto-restarted the API.
  Verified: preflight from both custom origins is echoed with `Allow-Credentials: true`; an
  unlisted origin is correctly rejected (no ACAO). Credentialed cookie/refresh flow works from
  the custom domains. No apex origin needed (apex only redirects — see below).
- **SWA custom domains.** `salehcard-web` → `www.flashcashglobal.com` = **Ready**;
  `salehcard-admin` → `admin.flashcashglobal.com` = **Ready**.
- **Apex `flashcashglobal.com` (HANDED TO DEVOPS — Cloudflare-only, no Azure change).** Bare
  domain currently fails TLS (SWA serves a default infra cert it doesn't cover). Fix = 2 steps in
  Cloudflare zone `flashcashglobal.com`: (1) set the apex DNS record to **Proxied** (orange cloud)
  so Cloudflare's edge cert covers it; (2) add a **Redirect Rule** — Hostname eq
  `flashcashglobal.com` → 301 `concat("https://www.flashcashglobal.com", http.request.uri.path)`,
  preserve query. Leave `www`/`admin` records DNS-only. This makes the SWA apex registration
  unnecessary — no TXT record needed.
- **Cleanup note:** a stale SWA apex custom-domain registration on `salehcard-web`
  (`flashcashglobal.com`, status `Validating`) was started then abandoned in favor of the
  Cloudflare-redirect plan. Harmless; delete with
  `az staticwebapp hostname delete -n salehcard-web -g salehcard-prod --hostname flashcashglobal.com`.

---

## What's DONE (do not redo)

| Phase | What | Commit |
|---|---|---|
| 1 | Panel adapter (`panel.go`) + api-mode fulfillment + provider port (`CheckStatus`, `ErrPending`) | `fe9e00b` |
| 2 | Async supplier settler (wait-reconcile, re-dispatch backoff, stuck flagging) | `6307f1d` |
| 3 | `/suppliers` admin page + `suppliers` RBAC domain + dashboard signals (backend + frontend) | `0008ddf`, `a6550e6` |
| 4 | umanage telecom adapter (id 13, LBP, search-by-ref reconciliation) | `0008ddf` |
| fix | Sync must never auto-publish a hidden imported product (only hide withdrawn ones) | `0435fe6` |

All gates green (go build/vet/test 36 pkgs; golangci-lint against `.golangci.yml`; admin
pnpm build+lint). Full live e2e verified against a **scripted fake panel** — balance probe,
catalog browse, import-with-markup (→ hidden api-mode product), sync (hide-only, price-drift
report), per-supplier settings threshold→low_balance, recent-orders trace, dashboard
parked/stuck counters, unknown-supplier 404. **umanage was stub/unit-tested only** — no live
creds yet.

### Architecture you need to know

- **Suppliers only appear on the page when `Configured()`** (`config.go:270`). Panel needs
  `SUPPLIER_<name>_TOKEN`; umanage (telecom) needs `SUPPLIER_UMANAGE_KEY` **and**
  `SUPPLIER_UMANAGE_SECRET`. Unconfigured → dropped from `EnabledSuppliers()` → not rendered.
  **This is why the prod page shows only 3 cards, not 4** — umanage creds aren't set. Expected.
- **Error taxonomy drives parking:** environmental errors (balance/throttle/auth/IP) PARK +
  re-dispatch; order-specific errors (bad player, bad qty) fail + compensate; `ErrPending`
  parks with a ref the settler polls.
- **The guarded `TransitionStatus` FindOneAndUpdate is the ONLY double-refund lock.** The
  settler never uses unguarded `UpdateFulfillment`/`compensate`.
- **umanage has no upstream idempotency key** (panels dedupe on `order_uuid`). An ambiguous
  create → `ErrPending` with `search:{family}:{orderUUID}` ref, reconciled by `CheckStatus`
  matching `app_user_reference`. Never a blind re-dispatch → no double-charge.
- **Sync never publishes.** Import creates a **hidden** (`available=false`) api-mode product;
  publishing stays an explicit admin decision. Sync only ever hides withdrawn/unavailable
  upstream products, never `false→true`.

---

## Prod facts (Azure)

- App Service **`salehcard-api`**, RG **`salehcard-prod`**, sub `1adb4811-6234-4822-b7f2-8411ed2cb999`, West Europe.
- **NAT egress IP = `52.157.69.89`** (single stable IP; `salehcard-nat` → `salehcard-nat-pip`).
  Verified authoritative: `vnetRouteAllEnabled=true` forces ALL outbound (incl. supplier API
  calls) through `snet-appsvc` → NAT. Same IP Monty SMS already uses. **This is the IP to give
  every supplier for allowlisting.**
- **Already set** (last session): `SUPPLIER_JENTEL_TOKEN`, `SUPPLIER_SPEEDCARD_TOKEN`,
  `SUPPLIER_GIFT4CARD_TOKEN`. Verified health 200 after restart.
- **Current prod page state** (screenshot 2026-07-15): jentel `● OK` ($502.19),
  speedcard `● OK` ($603.47), gift4card `🔴 IP blocked` (NAT IP not yet allowlisted their side).

---

## REMAINING WORK — the activation checklist

### 1. Azure (Claude can do — BLOCKED on user input)
- [ ] Set `SUPPLIER_UMANAGE_KEY` + `SUPPLIER_UMANAGE_SECRET` on `salehcard-api` app settings.
      **Blocked: user has not provided umanage creds yet.** Once set + restart, a 4th
      "umanage · telecom" card (LBP) appears automatically.

### 2. Supplier-side / out-of-band (USER only — Claude cannot do)
- [ ] Email each supplier: *"Please allowlist `52.157.69.89` for our API account."* — jentel,
      speedcard, **gift4card (currently blocking)**, umanage. gift4card flips 🔴→`● OK` on confirm.
- [ ] Fund prepaid balances at each supplier (jentel/speedcard already funded per balances shown).

### 3. In-console config (USER, admin console — no code)
- [ ] Grant the **`suppliers`** RBAC domain to any non-super-admin operators (super admins
      already have it implicitly via `nil adminRoleId → ["*"]`).
- [ ] Per supplier: **Browse & import** products → set **markup %** + **low-balance threshold**
      in Settings.
- [ ] **CANARY** (do before widening): import ONE product from ONE healthy panel (jentel or
      speedcard), place a REAL api-mode order end-to-end, confirm it fulfills.

### ⚠️ Canary caveat (important)
The panel adapters are verified against a **scripted fake panel, NOT the suppliers' real
APIs.** The first real import + order per supplier is where any real-world response-shape
drift surfaces. Treat the canary as the real integration test. If a real panel returns an
unexpected shape, the fix is in `api/internal/platform/provider/panel.go` (parsing) — check
`Profile()`, `ListProducts()` (`panelQty.UnmarshalJSON` handles null/array/{min,max}), and
`classifyProbe()` (120-122→auth, 123→ip_blocked, 130→maintenance).

---

## Security constraints (must persist)

- The 3 panel tokens are **chat-exposed**; owner **accepted this and is NOT rotating them**.
- Secrets live in Azure app settings / gitignored `DEPLOY-CREDS.local.md` — **NEVER** hardcoded in repo.
- Commit trailer: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- Do **NOT** `cd` to the main repo root or edit files outside the worktree.

## Key files (if a real-panel fix is needed)
- `api/internal/platform/provider/{provider.go, panel.go, umanage.go, build.go}`
- `api/internal/modules/supplier/{service.go, admin.go, repository.go, model.go}`
- `api/internal/modules/order/{supplier_settler.go, service.go, routes.go}`
- `api/internal/config/config.go` (`SupplierConfig.Configured`, `EnabledSuppliers`)
- `admin/src/features/suppliers/` (page, hooks, api)
