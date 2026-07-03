# SalehCard Conventions

## Repository Layout
```
/api      Go backend
/web      React + TypeScript storefront (customer-facing)
/admin    React + TypeScript admin console
/deploy   Docker Compose, env examples
```

## Backend Conventions

### Module Structure
Every feature lives in `internal/modules/<name>/`:
```
model.go       domain types + DTOs
repository.go  Repository interface + MongoRepository implementation
service.go     Service interface + implementation (business logic only)
handler.go     HTTP handlers (transport layer only)
routes.go      RegisterRoutes(r chi.Router, db *mongo.Database)
```

### Adding a New Backend Module
1. Create `internal/modules/<name>/` with the 5 files above
2. Call `<name>.RegisterRoutes(r, s.db)` in `internal/server/server.go` Routes()
3. Wire any new deps explicitly in `cmd/server/main.go` — no DI container

### Response Envelope
Always use `pkg/response` helpers:
- `response.OK(w, data)` → `{"success":true,"data":...}`
- `response.OKWithMeta(w, items, meta)` → paginated list
- `response.Error(w, 400, "bad_request", "msg")` → `{"success":false,"error":{...}}`

### Error Handling
- Service layer returns `pkg/errors` sentinels (ErrNotFound, ErrBadRequest, etc.)
- Handler maps errors to HTTP codes, never leaks internal details

### Pagination
Use `pkg/pagination`: ParseParams(r) → Params → pass to repo → CalcMeta(params, total) → OKWithMeta

### Wallet Funding & Ledger Types
`wallet_transactions` is the immutable ledger; the source-of-truth balance is
`users.walletBalance`. There are four ledger types (`wallet/model.go`): `topup`,
`purchase`, `refund`, `adjustment`.

**Convention: fund wallets through the top-up queue.** All money-in funding — for
**customers and resellers alike** — goes through the top-up request queue
(`POST /api/v1/wallet/topups` → admin approves at `/api/admin/wallet/topups`),
which writes ledger type **`topup`**. This is the only funding path counted by the
finance KPIs `TopUpsForDay` (`wallet/repository.go`) and `finance.walletTopups`
(`finance/finance.go`), so routing all deposits here keeps those figures coherent.
The admin top-up queue shows the requester's role, so reseller deposits are
visible without leaving the queue.

**Balance-adjust is for corrections only.** The direct balance-adjust endpoints
(`/api/admin/users/{id}/wallet-adjust`, `/api/admin/resellers/{id}/balance-adjust`)
write ledger type **`adjustment`** and are intentionally **excluded** from the
top-up KPIs. Use them to fix mistakes or make back-office ledger corrections, not
to fund a wallet as a matter of course.

### Reseller Pricing
Resellers are priced at checkout as the **lowest** of three candidates
(`order/service.go` `resellerPrice`):
1. **Retail** — `variant.price`.
2. **Tier margin** — `retail * (1 - ResellerTier.MarginPercent/100)`, resolved from
   the buyer's assigned tier via `reseller.MarginForUser`. A blank/unknown tier or a
   lookup error means no margin (retail), never a failed order.
3. **Per-variant override** — the optional `variant.resellerPrice` set on the product.

Sale offers do **not** stack on reseller pricing (they apply to retail buyers only).
`MarginPercent` was display-only before this; it is now enforced at order time.

## Frontend Conventions

### Feature Structure
```
src/features/<name>/
  api/         typed fetch functions (pure async, no hooks)
  hooks/       TanStack Query hooks wrapping api/
  components/  feature-specific UI
  pages/       route-level components (lazy-imported)
```

### Adding a New Frontend Feature
1. Create `src/features/<name>/` with above structure
2. Add route in `src/app/router.tsx` (React.lazy import)
3. Add translation keys to all 3 locale files (en/ar/tr)

### State Management Rules
| State type | Where |
|-----------|-------|
| Server data | TanStack Query (never Zustand) |
| Auth / user | useAuthStore (Zustand) |
| UI preferences | useLocaleStore, useThemeStore, useCurrencyStore |
| Local UI state | useState / useReducer |

### API Client
- All calls via `apiClient` from `@/lib/api-client`
- Access token in memory only — never localStorage, never sessionStorage
- Always type responses with `ApiResponse<T>`

### i18n
- All visible strings via `useTranslation()` and `t()`
- RTL: use Tailwind logical properties (`ms-`, `me-`, `ps-`, `pe-`, `start-`, `end-`) not physical (`ml-`, `mr-`, `left-`, `right-`)
- Locale switch: `useLocaleStore().setLocale(l)` — updates i18n, html lang, html dir automatically

### Design System
The visual identity ported from the Claude Design prototype is a **CSS-variable design system**, not Tailwind utilities. Tokens live in `src/styles/tokens.css` (`--brand-1`, `--surface`, `--text`, radii, gradients, glows; light/dark via `[data-theme]`), the class layer in `src/styles/components.css` (`.btn*`, `.card`, `.panel`, `.badge*`, `.field`, `.vault`, `.art`, `.seg`, `.tabs`, `.skel`, `.spinner`), and app layout in `src/styles/layout.css` (`.appheader`, `.hero`, `.pdp`, `.catgrid`, `.prodcard`, `.sidenav`, `.bottomnav`, responsive `@media`). All three are `@import`ed at the top of `src/index.css` before the `@tailwind` directives.

- Use `@/components` primitives: Icon, Logo, ImageArt, Stars, Price, Button, Input, Card, Panel, Badge, Modal, Toast (`ToastProvider`/`useToast`), CodeVault, Stepper, Segmented, Tabs, Skeleton, LoadingSpinner, EmptyState, ErrorState. These emit the design-system classNames above — **do not** re-style them with Tailwind.
- Tailwind remains installed; its theme maps the same CSS variables (`colors.brand1 = var(--brand-1)`, `fontFamily.display`, etc.) so utilities resolve to themed tokens for incidental one-off layout. Prefer the `.row`/`.col`/`.wrap`/`.grid` helpers and component primitives over ad-hoc Tailwind for anything reusable.
- RTL is handled in the ported CSS via logical properties (`inset-inline`, `padding-inline`, `margin-inline-start`). When adding new CSS prefer logical properties; for Tailwind, prefer `ms-`/`me-`/`ps-`/`pe-`/`start-`/`end-`.
- Box-art product imagery is generated gradient tiles via `<ImageArt art={...}/>` (palettes in `src/lib/art.ts`) — the design deliberately uses no real brand logos.

### Dark Mode
Driven by the `[data-theme]` attribute on `<html>` (the CSS keys on it). `useThemeStore().setTheme('dark'|'light')` sets `data-theme` **and** toggles the `dark` class (so Tailwind `dark:` still works). Theme persists; **locale does not** — the app forces English on every load (handoff requirement), with Arabic/Turkish available via the language switcher.

## Admin App (`/admin`)

A **separate** Vite + React + TS app (port **5174**) for the internal team — ported from the Claude Design admin prototype. Same brand DNA as `/web`, adapted for a dense control-panel context (sidebar shell, compact tables, tighter radii). **Do not modify `/web` when changing `/admin`.**

### Structure
```
/admin/src
  app/          entry, providers, router (react-router-dom), RequireAdmin guard, nav config
  components/   typed design-system primitives (Icon, Art, Badges, Charts, Controls,
                Table, PageHead, Modal, States) + layout/ (Sidebar, Topbar, AdminLayout)
  features/<area>/   one folder per admin area: pages/ (+ api/ hooks/ where wired)
  lib/          api-client, query-client, utils (money/stockLevel), mock/demo.ts
  i18n/         config + en (default) / ar / tr — chrome strings only (table data stays English)
  stores/       zustand: auth, theme, locale, ui (sidebar collapsed)
  styles/       base.css (brand tokens + primitives) + admin.css (admin chrome), ported verbatim
  types/        shared API + domain types
```

### Key decisions (this slice)
- **Design tokens are duplicated, not shared.** Chosen the pragmatic option over a `/packages/tokens` package: `admin/src/styles/base.css` + `admin/src/tailwind.config.ts` carry the same brand tokens as `/web`, mapped to the same CSS variables, plus admin-only fulfillment accents (`--ff-code/credit/transfer`). If a third consumer appears, promote these to a shared package.
- **Auth logic is duplicated** (`admin/src/lib/api-client.ts`) rather than imported from `/web` — same in-memory-token rule.
- **Design system = ported CSS** (`base.css` + `admin.css`), NOT Tailwind `dark:` utilities — same approach as `/web`. Components emit the design classNames (`.abtn`, `.acard`, `.tbl`, `.bdg`, `.st-*`, `.ff-*`, `.kpi`, `.sidebar`, …); don't restyle them with Tailwind.
- **Routing** uses `react-router-dom` (CONVENTIONS feature pattern), not the prototype's localStorage view-state. `RequireAdmin` redirects unauthenticated/non-admin users to `/login`. Theme persists; locale forced to English on load (same handoff rule as `/web`).
- **Fulfillment color-coding** is consistent everywhere via `<FfBadge>`: blue = `code`, green = `account_credit` (`credit`), orange = `transfer`.

### What is wired vs mock
- **Wired to the Go API:** `products` (list/create/edit/delete/bulk + category dropdowns from catalog-distinct facets), `inventory` (code stock, bulk upload, threshold config, code audit, **upload history**), the **dashboard** (every figure — revenue/orders/active-users/wallet KPIs + sparklines, charts, fulfillment, low-stock, system health), and **settings → admin-users** table (`/api/admin/users?role=admin`).
- **Mock-driven (with `// TODO` + a `ComingSoonNote` banner):** orders, resellers, finance, promos, reviews, and the remaining settings store-config forms. Mock data lives in `lib/mock/demo.ts` (ported from the prototype's `data.js`).

### Admin backend (`/api`)
- `internal/platform/auth/middleware.go` → `AdminOnly(secret)` guards the `/api/admin` group: requires `role == "admin"` on the JWT; **bypasses with a synthetic admin in dev when `JWT_SECRET` is empty** (logs a one-time warning), enforces when a secret is set.
- **Fully implemented:** product admin CRUD + bulk + category facets (`internal/modules/product/admin.go`), the inventory/code module incl. upload-history (`internal/modules/code/`, `GET /api/admin/upload-history` backed by the `upload_batches` collection), dashboard (`internal/modules/dashboard/`), and the admin user list (`internal/modules/user/admin.go`, `GET /api/admin/users`).
- **Stubbed (501) with the full route map:** the remaining areas register `RegisterAdminRoutes(r, db)` returning `response.Stub("…")` — `reseller`, `settings`, and any others not yet filled in. Fill these in following the existing module pattern (model → repository → service → handler).
- All admin routes are wired in `internal/server/server.go` under `r.Route("/api/admin", …)`.
