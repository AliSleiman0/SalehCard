# HANDOFF — Admin "make it real" (2026-06-28)

This session removed the last big chunks of admin mock data. Use this as the **recipe**
to finish the remaining pages the same way the dashboard was done. Read
`CONVENTIONS.md` → "What is wired vs mock" alongside this.

## What shipped this session (branch `feat/admin-reviews`)

- `02bee9e` — **Dashboard 100% real**: active-users (new `user.lastSeen`), wallet top-ups,
  revenue/orders sparklines, real System Health (`GET /health`), greeting/date, CSV export.
- `2596a9e` — **Products categories**, **Settings admin-users table**, **Inventory upload
  history** wired; `CATS`/`admins`/`uploadHistory` deleted from `demo.ts`.
- `38e60ad` — fix: never-logged-in admin showed "Jan 1" instead of "—" (zero-time bug, see Gotchas).

> **`admin/src/lib/mock/demo.ts` is now fully unused** (grep `lib/mock/demo` across `src/` → no
> importers). It's dead code — safe to delete in the next session as a cleanup commit.

## The recipe (mock page → real data)

Pattern used everywhere. Backend module layout is `model / repository / service / handler /
routes` (see CONVENTIONS.md). Frontend feature layout is `api/<x>.ts` (calls) + `hooks/use<X>.ts`
(React-Query) + optional `lib/adapt<X>.ts` (shape mapper).

1. **Find the mock**: `grep "from '@/lib/mock/demo'"` and `ComingSoonNote` in `admin/src/features`.
   Note the exact fields the page renders — that's your target API shape.
2. **Reuse before you build.** Check for an existing endpoint/hook first. The Settings admins
   table needed *zero* backend — it reused `useUsers({role:'admin'})` + `adaptUser`. Look in
   `features/*/hooks` and `features/*/api` before writing anything.
3. **Backend (only if no endpoint exists):**
   - Add the aggregation/query as a **concrete repo method** (mirror an existing one, e.g.
     `order.RevenueSeries`, `product.CountByRootDomain`).
   - **Wire read-only endpoints to the repo directly in `routes.go`/`admin.go`** (a closure or
     a tiny handler holding `*MongoRepository`) — do **NOT** add to the `Service` interface unless
     you must. See `product.categoryFacetsHandler` and `code.uploadHistoryHandler`. Reason: the
     `order` module's test fakes implement `code.Service`/`product.Service`; changing those
     interfaces breaks `order_test.go`.
   - If you *do* add to a `Repository` interface, update that module's test fake (`*_test.go`).
   - Add any new collection's index in the module's `EnsureIndexes`.
4. **Frontend:** add `api/<x>.ts` (`apiClient.get/post`, typed), a `hooks/use<X>.ts` query,
   swap the mock import in the page for the hook, add a **loading + empty state**, and
   `invalidateQueries` on related mutations (e.g. upload history invalidates on upload).
5. **Delete the mock export** from `demo.ts`; grep to confirm nothing else imports it.
6. **Update docs**: `CONVENTIONS.md` (wired vs mock), `admin/BACKLOG.md` (tick the items),
   `CLAUDE.md` if the data note changes, and the `salehcard-admin-port` memory.
7. **Verify in a real browser** (see "Run & verify"). curl proves the API; the browser proves
   CORS + the render.

## Gotchas (these bit us — save yourself the time)

- **Go zero `time.Time` serializes as `"0001-01-01T00:00:00Z"`, and `json:"...,omitempty"` does
  NOT drop it** (omitempty only drops nil/zero of basic types, not structs). That string is
  *truthy*, so a `field ? fmt(field) : '—'` guard fails. Use a real guard — see
  `lastActive()` in `admin/src/lib/utils.ts` (treats missing/zero/pre-epoch → "—"). Watch for
  this on any new time field. (Alternative: make the Go field `*time.Time`.)
- **Don't widen `Service` interfaces casually** — `order_test.go` has `fakeCodeSvc`/`fakeProductSvc`
  implementing them. Read endpoints → hit the repo directly (dashboard precedent).
- **Product `category` is an exact-match filter string** (slug like `giftcards`), not a display
  name. Dropdowns are populated from **catalog-distinct facets** (`GET /api/admin/products/categories`)
  so the value is guaranteed to match the filter. Don't wire to `/api/v1/categories` slugs
  blindly — alignment isn't guaranteed.
- **`lastSeen` only stamps on login/refresh** (`user.issueTokens`). Seeded users have no
  `lastSeen`, so "active users" reads low and "last active" shows "—" until someone actually
  logs in. Not a bug — it's an empty signal.
- **Seed data is monotonous**: every seeded order belongs to `customer@salehcard.local`, so the
  Recent-Orders "Customer" column shows "customer" (email local-part) on every row. Real, just
  one customer. Don't "fix" it.
- **CORS**: new root endpoints (like `/health`) and any custom request header must sit under the
  CORS middleware in `internal/server/server.go` (it's applied before route registration, so
  you're fine — just don't move routes above it). curl won't catch preflight failures; the
  browser will.
- **Admin auth in dev**: run the API with `JWT_SECRET=` empty → `AdminOnly` injects a synthetic
  `dev@salehcard.local` admin (that's why uploaded batches show `by dev`). With a real secret it
  enforces tokens.
- **Windows**: `go test` may print `unlinkat ... being used by another process` on cleanup — it's
  a harmless file-lock artifact, not a test failure. Check the `ok`/`FAIL` lines.

## Remaining work (what's still not real)

1. **Settings store-config** — the real remaining wire-up. `SettingsPage.tsx` has
   `<ComingSoonNote mock />`; the General / Payment-gateways / Notifications tabs are local React
   state with no backend. Needs a new `settings` backend module (`model/repo/handler` +
   `GET/PUT /api/admin/settings`, a single settings doc) then wire the three forms. The
   **admin-users tab is already real.** (`CONVENTIONS.md` still lists `settings` backend as
   stubbed-501.)
2. **Finance → USDT queue tab** — `ComingSoonNote`: there's no pending queue because USDT
   auto-confirms today. This is a *feature gap*, not a mock swap — needs the USDT-pending payment
   flow first. Revenue tab is already real.
3. **Resellers → pricing tab** — `ComingSoonNote`: per-product reseller price overrides not
   built yet (today reseller pricing uses the per-variant reseller price). Feature gap; rest of
   reseller detail is wired.
4. **Cleanup** — delete the now-dead `admin/src/lib/mock/demo.ts`.

Everything else in `/admin` (dashboard, products incl. categories, inventory incl. upload
history, orders, users, resellers list/detail, finance revenue, promos, reviews, settings
admins) is API-wired.

## Run & verify (exact recipe)

```bash
# Mongo already on :27017 (or `make up`). Seed is idempotent.
cd api && JWT_SECRET= go run ./cmd/seed
cd api && JWT_SECRET= PORT=8090 go run ./cmd/server      # admin auth bypassed in dev
cd admin && pnpm dev                                      # http://localhost:5174 (any creds log in)
```

Gates: `cd api && go build ./... && go vet ./... && go test ./...` · `cd admin && pnpm build && pnpm lint`.

Verify an endpoint is real before trusting the page (this is how the bug above was caught):
```bash
curl -s "http://localhost:8090/api/admin/<your-endpoint>" | node -e 'let d="";process.stdin.on("data",c=>d+=c).on("end",()=>console.log(d))'
```
Then open the page in a **real browser** (CORS preflight) and confirm the render + empty state.
For anything showing a timestamp, eyeball it against a seeded-but-never-touched row → it should
read "—", not a 1-AD date.

## Reference (wired examples to copy from)

- Read endpoint off the repo (no Service change): `api/internal/modules/product/admin.go`
  (`categoryFacetsHandler`), `api/internal/modules/code/routes.go` (`uploadHistoryHandler`).
- New collection written on an action + history read: `code` module `upload_batches`
  (`model.go` `UploadBatch`, `repository.go` `RecordBatch`/`ListUploadHistory`, `service.go`
  `Upload`, `handler.go` stamps `UploadedBy` from `auth.ClaimsFromContext`).
- Reuse a hook + adapter: `SettingsPage.tsx` admins table → `useUsers` + `adaptUser`.
- Frontend api/hook pair: `features/products/api/categories.ts` + `hooks/useCategories.ts`.
