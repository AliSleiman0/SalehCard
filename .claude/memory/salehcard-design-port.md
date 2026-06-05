---
name: salehcard-design-port
description: "SalehCard /web frontend was ported from a Claude Design prototype — styling architecture and what's real vs mocked"
metadata: 
  node_type: memory
  type: project
  originSessionId: f6b0f2fb-ced9-4236-b5c3-ff82ebdd1a8a
---

The `/web` frontend is a port of a finished Claude Design prototype (bundle source: React/JSX + CSS, fetched from an api.anthropic.com/v1/design share link — links expire, ask the user for a fresh one).

Key architecture decisions (user-approved):
- **Styling is a ported CSS-variable design system**, NOT Tailwind `dark:` utilities. Tokens/classes live in `src/styles/{tokens,components,layout}.css`; `[data-theme]` swaps light/dark. Tailwind stays installed with tokens mapped to CSS vars for incidental layout. Components in `src/components/` emit these classNames.
- **Theme**: `useThemeStore.setTheme` sets `data-theme` on `<html>` AND toggles `.dark`. Theme persists.
- **Locale**: forced to **English on every load** (not persisted) — handoff requirement; ar/tr switchable at runtime. RTL via CSS logical properties.
- **Data**: catalog (Home/Category/ProductDetail) is wired to the **real Go product API** via `useProducts`/`useProduct` + `features/catalog/lib/adaptProduct.ts` (maps API `Product` → `ViewProduct`). Everything else (orders, wallet, saved-ids, reseller, category taxonomy, reviews) uses **mock data in `src/lib/mock/demo.ts`** with `// TODO` markers until those backend modules land.
- **Three fulfillment shapes** routed by product type: `code` (CodeVault reveal+copy), `credit` (account-credit confirmation), `transfer` (status timeline + reference). Built at checkout `pay()`, dispatched in OrderSuccessPage/OrderDetailPage.
- Client state in Zustand: cart, wallet (decrement on buy / topUp), ui (agent/reseller view toggle), plus locale/currency/theme/auth.

Box-art product images are generated gradient tiles (`<ImageArt>`, palettes in `src/lib/art.ts`) — no real brand logos by design.

See [[salehcard-dev-env]] for how to run it.
