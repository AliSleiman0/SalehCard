# Plan: Product Image Upload + Rendering (with server-side compression)

_Planned 2026-07-05. Not yet implemented — this doc is the implementation spec for a future session.
Exploration findings verified against the codebase on that date; re-verify line numbers before editing._

## Context

Products currently render as initials/gradient placeholders (Flutter `ProductChip`, admin gradient
`Art`). The backend already has `Product.Images []string` (URLs) and the Flutter browse/search
screens already render `Image.network` with an initials fallback — but there is **no upload path, no
storage integration, and no image field in the admin editor**. This plan adds the full path: admin
picks a file → Go API validates + compresses it into two sizes (**256px thumbnail** + **1024px
display**, JPEG q80) → stores in **Azure Blob** (connection-string auth, public-read container;
**local-disk adapter in dev**) → URLs saved on the product → rendered in the **Flutter app + admin
console** (legacy `/web` skipped — being retired).

**Decisions already made (with the product owner):** render scope = Flutter + admin; blob auth =
connection string; compression = two sizes (thumb + display).

## Design decisions

- **Endpoint is product-agnostic**: `POST /api/admin/products/images` (multipart, admin-only)
  returns `{imageUrl, thumbnailUrl}`. The editor uploads first, then includes the URLs in the normal
  create/update `ProductInput`. (A per-product endpoint can't serve the create flow — no id yet.)
- **Two sizes stored explicitly**: keep `images[0]` = display URL (backward compatible — Flutter
  already reads `images.first`; seed data unchanged) and add a new **`Product.Thumbnail string`**
  field for the 256px URL. Lists/chips prefer `thumbnail`, falling back to `images.first`.
- **New `platform/blob` port** mirroring `platform/sms` (port + `New()` switch + one file per
  adapter): `azure` and `local` adapters, selected by `STORAGE_PROVIDER` (default `local` → dev
  works with zero Azure config).
- **Compression is pure-Go stdlib** (`image`, `image/jpeg`, `image/png`) + `golang.org/x/image/draw`
  (scaling) + `golang.org/x/image/webp` (webp decode). Always output JPEG q80; composite
  transparency onto white; never upscale. Matches the repo's lean-deps posture (no cgo).
- **Old blobs are not deleted on replace** in v1 (orphans are pennies; avoids delete races). Future work.

---

## Phase 1 — Backend: blob port, compression, upload endpoint

### 1. New port: `api/internal/platform/blob/`

Mirror `api/internal/platform/sms/sms.go` structure:

- `blob.go` — port + factory:
  ```go
  type Storage interface {
      Upload(ctx context.Context, key, contentType string, data []byte) (url string, err error)
  }
  type Config struct { Provider string; Azure AzureConfig; Local LocalConfig }
  func New(cfg Config) (Storage, error)   // switch: ""|"local" → local, "azure" → azure, else error
  ```
- `azure.go` — `github.com/Azure/azure-sdk-for-go/sdk/storage/azblob`: client from
  `AzureConfig{ConnectionString, Container}` (constructor errors if connection string empty — creds
  validated lazily in the adapter, like `monty.go`). Upload with content-type header; return
  `https://<account>.blob.core.windows.net/<container>/<key>`.
- `local.go` — dev adapter: writes to `LocalConfig.Dir` (default `./uploads`), returns
  `LocalConfig.PublicBase + "/uploads/" + key`.

### 2. Compression helper: `api/internal/platform/imaging/imaging.go`

```go
// Process decodes (jpeg/png/webp), validates, and re-encodes two JPEGs.
func Process(data []byte) (display []byte, thumb []byte, err error)
```

- Sniff real type via `http.DetectContentType` (first 512 bytes) — allowlist
  `image/jpeg|png|webp`; reject by content, not filename.
- Decode-bomb guard: `image.DecodeConfig` first, reject dimensions > 12000px either side.
- Resize with `x/image/draw` `CatmullRom` to max-edge 1024 (display) and 256 (thumb); skip resize if
  already smaller. Composite onto white RGBA before JPEG encode (kills alpha). `jpeg.Encode(q=80)`.
- Unit tests with small generated fixtures: png-with-alpha → JPEG; oversized dims rejected;
  non-image bytes rejected.

### 3. Config: `api/internal/config/config.go`

Add a `// Storage (product image uploads)` block (like the SMS block):

- `StorageProvider` — `getEnv("STORAGE_PROVIDER", "local")`
- `AzureStorageConnectionString` — `os.Getenv("AZURE_STORAGE_CONNECTION_STRING")`
- `AzureStorageContainer` — `getEnv("AZURE_STORAGE_CONTAINER", "product-images")`
- `UploadsDir` — `getEnv("UPLOADS_DIR", "./uploads")`
- `PublicBaseURL` — `getEnv("PUBLIC_BASE_URL", "http://localhost:8090")` (local adapter URL prefix)

No `Validate()` change (adapters validate lazily, per the existing convention). Update
`api/.env.example` with the new vars + comments.

### 4. Product model: `api/internal/modules/product/model.go`

- Add `Thumbnail string` (`bson:"thumbnail,omitempty" json:"thumbnail"`) to `Product`,
  `UpsertProductInput`, `CreateProductInput`, `UpdateProductInput`.
- Repository: write `thumbnail` alongside `images` in create/update.

### 5. Upload endpoint: product module

- `product/handler.go` — new `UploadImage(w, r)`:
  - `r.Body = http.MaxBytesReader(w, r.Body, 10<<20)` (10 MB — **no size-limit middleware exists
    in the repo**, this is the cap).
  - `r.ParseMultipartForm(10<<20)` + `r.FormFile("image")` → read → `imaging.Process` → two
    `blob.Upload` calls (keys: `products/<uuid>.jpg` and `products/<uuid>_thumb.jpg`, content-type
    `image/jpeg`).
  - Errors: parse/validation → `response.BadRequest`; storage → `response.InternalError`.
    Success → `response.OK(w, map[string]string{"imageUrl":…, "thumbnailUrl":…})`.
  - Audit (best-effort after success, same pattern as product delete): new
    `audit.ActionProductImageUpload = "product.image_upload"` in `audit/model.go`, summary
    `{"imageUrl": …}`.
- `product/admin.go` — add `r.Post("/products/images", h.UploadImage)` (flat, like siblings).
- Handler gets the `blob.Storage` dep: extend `RegisterAdminRoutes(r, db, rec, store blob.Storage)`;
  construct the adapter in `server.go Routes()` next to sms/push (fall open to `local` + log warning
  on Azure misconfig, mirroring the sms/push fallback).

### 6. Static serving for the local adapter: `api/internal/server/server.go`

Mount a chi static file server for `/uploads/*` from `cfg.UploadsDir` **only when**
`StorageProvider == "local"`. Add `uploads/` to `api/.gitignore`.

### 7. Deps: `api/go.mod`

`go get github.com/Azure/azure-sdk-for-go/sdk/storage/azblob golang.org/x/image`

## Phase 2 — Admin console

### 8. Multipart support: `admin/src/lib/api-client.ts`

Add `upload(path, formData)` (or teach `request()` to detect `FormData`): skip the JSON
`Content-Type` header (the browser sets the multipart boundary) and skip `JSON.stringify`; keep the
Bearer header and `credentials:'include'`.

### 9. Product editor: `admin/src/features/products/pages/ProductEditPage.tsx`

- New state: `images: string[]`, `thumbnail: string` (populate in the load `useEffect`; replace the
  blind `images: data?.data?.images ?? []` round-trip in `buildInput()` with the state values).
- New "Image" `.acard pad` (place near Status in the side fieldset): copy the file-input pattern
  from `InventoryPage.tsx` (~lines 320–347: hidden
  `<input type="file" accept="image/jpeg,image/png,image/webp">` + `.abtn` trigger + `.dropzone`);
  on select → `POST /api/admin/products/images` (react-query mutation in
  `features/products/api/products.ts` + `hooks/useProducts.ts`) → set `images=[imageUrl]`,
  `thumbnail`; preview via the unused `.imgslot` class (`admin/src/styles/admin.css`) rendering
  `<img>`; add a Remove button (clears both); disable Save while an upload is in flight.
- Client-side pre-checks (UX only, server is authoritative): file size ≤ 10 MB, type allowlist.
- Add `thumbnail?: string` to `ProductInput` (`api/products.ts`) and to the admin `Product` type
  (`admin/src/types/index.ts`).

### 10. Product list thumbnail: `admin/src/features/products/pages/ProductListPage.tsx`

In the row cell that renders `<Art art={artForCategory(p.category)} size={36}/>`: render
`<img src={p.thumbnail || p.images[0]}>` (36px, rounded) when present, keeping the gradient `<Art>`
as fallback (and as `onError` fallback).

## Phase 3 — Flutter app

### 11. Entity + DTO

- `app/lib/features/catalog/domain/entities/product.dart` — add `final String? thumbnail;` + getters:
  `String? get imageUrl => images.isNotEmpty ? images.first : null;`
  `String? get thumbUrl => thumbnail ?? imageUrl;`
- `product_dto.dart` — add `thumbnail` field; regenerate `product_dto.g.dart` via
  `dart run build_runner build --delete-conflicting-outputs`.

### 12. `ProductChip`: `app/lib/core/widgets/product_chip.dart`

Add optional `String? imageUrl`; in the 60×60 tile render
`Image.network(imageUrl, fit: BoxFit.cover, errorBuilder: → initials)` when non-null, else the
existing initials. (Canonical pattern: `search_screen.dart` list tiles.)

### 13. Call sites

- `home_screen.dart` chip builder — pass `imageUrl: p.thumbUrl`.
- `product_detail_screen.dart` `_Banner` — gains an optional image: show
  `Image.network(p.imageUrl)` (full 1024px display URL) with the existing tint+initials as
  fallback/error.
- `search_screen.dart` / `category_products_screen.dart` — switch their inline `images.first` dance
  to `p.thumbUrl` (lists should load the small file).

## Phase 4 — Ops (deferred to deploy time; add to DEVOPS-TODO.md, don't block dev)

In **Azure Cloud Shell** (local `az` broken by the Norton TLS MITM — see CLAUDE.md):

```bash
az storage account create -n salehcardassets -g <prod-rg> -l <region> --sku Standard_LRS --allow-blob-public-access true
az storage container create --account-name salehcardassets -n product-images --public-access blob
az storage account show-connection-string -n salehcardassets -g <prod-rg>
# then set on the API Container App: STORAGE_PROVIDER=azure, AZURE_STORAGE_CONNECTION_STRING=<value>
```

Record the connection string in `DEPLOY-CREDS.local.md` (gitignored). Future work: swap to managed
identity; optional best-effort old-blob delete on replace.

## Files touched (summary)

| Area | Files |
|---|---|
| New | `api/internal/platform/blob/{blob,azure,local}.go`, `api/internal/platform/imaging/imaging.go` (+ tests) |
| Backend edits | `config/config.go`, `product/{model,repository,handler,admin}.go`, `server/server.go`, `audit/model.go`, `api/.env.example`, `api/go.mod`, `api/.gitignore` |
| Admin | `lib/api-client.ts`, `features/products/{pages/ProductEditPage.tsx, pages/ProductListPage.tsx, api/products.ts, hooks/useProducts.ts}`, `types/index.ts` |
| Flutter | `entities/product.dart`, `data/…/product_dto.dart` (+ regen `.g.dart`), `core/widgets/product_chip.dart`, `home_screen.dart`, `product_detail_screen.dart`, `search_screen.dart`, `category_products_screen.dart` |
| Docs | `DEVOPS-TODO.md` (Azure setup), `CLAUDE.md` (storage port note) |

## Verification

1. **Gates**: `cd api && go build ./... && go vet ./... && go test ./...` (imaging tests included);
   `cd admin && pnpm build && pnpm lint`; `cd app && flutter analyze`.
2. **API e2e (dev, local adapter)**: run API on :8090;
   `curl -F "image=@test.png" -H "Authorization: Bearer <admin JWT>" http://localhost:8090/api/admin/products/images`
   → expect `{imageUrl, thumbnailUrl}`; confirm two JPEGs land in `api/uploads/` and both URLs serve
   via `GET /uploads/...`; verify a 15 MB file → 400, a `.txt` renamed `.png` → 400.
3. **Admin e2e (real browser — CORS preflight)**: edit a product → upload image → preview appears →
   Save → reload shows image; product list shows the thumbnail.
4. **Flutter e2e**: run app on emulator (adb reverse recipe); home chips + product detail show the
   image; a product with no image still shows initials; kill the API mid-scroll → error fallback
   shows initials (no crash).
5. Add any deferred manual checks to `QA-TODO.md`.
