---
name: salehcard-admin-port
description: "SalehCard /admin console — what was ported from the admin design prototype, what's wired vs stubbed, key architecture decisions"
metadata: 
  node_type: memory
  type: project
  originSessionId: 0dfb3028-311b-4bd5-b10c-9881bbf8c5be
---

`/admin` is a **separate** Vite + React + TS app (port 5174) ported from the Claude Design admin prototype (bundle fetched from an api.anthropic.com/v1/design share link — links expire, ask for a fresh one). It sits alongside `/web`; do not modify `/web` when changing `/admin`. See [[salehcard-dev-env]] to run it and [[salehcard-design-port]] for the storefront's matching approach.

Key decisions (user's prompt left several as "my call"):
- **Design tokens duplicated, not a shared package** (pragmatic). `admin/src/styles/{base,admin}.css` are the prototype's CSS ported verbatim (base = brand tokens + primitives, admin = dense chrome: sidebar/topbar/tables/KPIs). Tailwind maps the same CSS vars. Design system is CSS classes (`.abtn .acard .tbl .bdg .st-* .ff-* .kpi .sidebar`), NOT Tailwind `dark:` — same as `/web`.
- **Routing** uses react-router-dom (not the prototype's localStorage view-state). `RequireAdmin` guard → `/login`. Theme persists; locale forced English on load (handoff rule). Sidebar collapse persisted in a `ui` zustand store.
- **Fulfillment color-coding** consistent via `<FfBadge>`: blue=code, green=account_credit(credit), orange=transfer.

**Wired to the Go API:** `products` (list/create/edit/delete/bulk), `inventory` (code stock, bulk upload w/ dedup, threshold config, code audit), dashboard KPIs + low-stock. **Mock-driven (TODO + ComingSoonNote banner, data in `lib/mock/demo.ts`):** orders, users, resellers, finance, promos, reviews, settings.

Backend (`/api`): `AdminOnly` middleware on `r.Route("/api/admin")` (dev-bypass when no JWT_SECRET). Fully implemented: `product/admin.go` (CRUD+bulk, flat routes so `code` can mount `/products/{id}/codes`), new `internal/modules/code/` module (inventory), `internal/modules/dashboard/`. Stubbed (501) with full route map via per-module `admin.go` `RegisterAdminRoutes` using `response.Stub(...)`: order, user, reseller, wallet(finance), promo, review, settings. Verified end-to-end against live mongo. Design ref stashed (gitignored) at `.design-ref/admin-prototype/`.
