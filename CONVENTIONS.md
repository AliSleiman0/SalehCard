# SalehCard Conventions

## Repository Layout
```
/api      Go backend
/web      React + TypeScript frontend
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
