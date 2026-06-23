# Production Deployment — Azure (2026-06-22)

SalehCard is **live in production on Azure**. This doc is the deployment handoff:
URLs, resources, config, redeploy recipes, the corporate-proxy workaround, and
open items. **Secrets are NOT here** — they live in `DEPLOY-CREDS.local.md`
(gitignored, local only).

> Code state: everything below ships from branch **`feat/catalog-migration-browse`**
> (PR #3, **not yet merged to `main`**).

## TL;DR — live URLs

| Component | URL |
|---|---|
| Storefront (web) | https://black-mud-01b8ea603.7.azurestaticapps.net |
| Admin console | https://purple-bay-0d1a52d03.7.azurestaticapps.net |
| API | https://salehcard-api.azurewebsites.net (`/health` → `{"status":"ok"}`) |
| Database | MongoDB Atlas `Cluster0` → `cluster0.b8ybow1.mongodb.net`, db `salehcard` |

Admin login email: **admin@salehcard.com** (password in `DEPLOY-CREDS.local.md`).
Data loaded: **86 categories + 635 products** + the admin user.

## Architecture

- **API** (`/api`, Go) → Docker image on **Azure App Service for Containers** (Linux).
- **Storefront** (`/web`) + **Admin** (`/admin`) (React/Vite SPAs) → two **Azure Static Web Apps** (Free tier), prod API URL baked at build time via `VITE_API_BASE_URL`.
- **DB** → MongoDB Atlas M0 (free). App is built for standalone Mongo (no multi-doc txns); the new `product.Upsert` uses an aggregation-pipeline update (Atlas-supported).
- **Image registry** → Azure Container Registry (Basic).

## Azure resources

- Subscription: **SalehCard** `1adb4811-6234-4822-b7f2-8411ed2cb999` (tenant *OZ Consultants*).
- Resource group: **`salehcard-prod`** (region **West Europe** — chosen because it also supports Static Web Apps).
- App Service plan: **`salehcard-plan`** (Linux, **B1**).
- Web App (API): **`salehcard-api`** — container `salehcardprodacr.azurecr.io/salehcard-api:v1`, `httpsOnly`, **health check `/health`**, alwaysOn.
- Container Registry: **`salehcardprodacr`** (Basic, admin-enabled) → `salehcardprodacr.azurecr.io`.
- Static Web Apps: **`salehcard-web`**, **`salehcard-admin`** (Free).

**Approx cost:** ~$18/mo (App Service B1 ~$13 + ACR Basic ~$5; SWA Free + Atlas M0 = $0).

## API configuration (App Service app settings)

`MONGO_URI` (Atlas srv), `DB_NAME=salehcard`, `JWT_SECRET` (strong, generated at
deploy — **set, so AdminOnly enforces in prod**), `ENV=production`,
`PORT=8080`, `WEBSITES_PORT=8080`, `ALLOWED_ORIGINS=<both SWA URLs>`,
`DOCKER_REGISTRY_SERVER_{URL,USERNAME,PASSWORD}`, `WEBSITES_ENABLE_APP_SERVICE_STORAGE=false`.

Auth notes baked into the code for prod:
- Refresh cookie is `SameSite=None; Secure` when `ENV!=development` (SPA and API are
  different domains → cross-site cookie required). See `user/handler.go`.
- Admin app does **real** login/refresh against `/api/v1/auth/*` (the old mock login
  was the cause of the prod 401 loop). See `admin/src/stores/auth.ts`.

## ⚠️ Corporate-proxy TLS workaround (READ before any `az`/deploy work)

The dev machine sits behind a **TLS-intercepting corporate proxy** whose CA is
rejected by OpenSSL (`Basic Constraints not marked critical`). Windows-based
tools (git, gh, Docker, the Go Atlas driver) trust it via the Windows cert store;
**Python (Azure CLI) and Node do not** by default. Established workarounds:

- `az` is at `C:\Program Files\Microsoft SDKs\Azure\CLI2\wbin\az.cmd` (not on bash PATH).
- Per shell, set: `$env:AZURE_CLI_DISABLE_CONNECTION_VERIFICATION="1"` (user-authorized,
  **process-scoped only — never persist it**) and `$env:REQUESTS_CA_BUNDLE="C:\Users\user\.azure\win-ca-bundle.pem"`
  (a dump of the Windows root+CA store; already persisted as a User env var — benign).
- **`az appservice` / `az webapp` / `az staticwebapp` fail SSL** through the proxy —
  their SDK ignores the disable flag. **Use `az rest` instead** (raw ARM calls go
  through the core client which honors it). Pass JSON bodies via `--body "@file.json"`
  (inline escaped JSON mangles the command line under cmd).
- **`az acr build` fails** (its source-upload uses a separate storage SDK). Build the
  image **locally and `docker push`** instead (Docker trusts the Windows store).
- For the **SWA CLI**, set `$env:NODE_EXTRA_CA_CERTS="C:\Users\user\.azure\win-ca-bundle.pem"`.

## Redeploy recipes

All commands assume the proxy env vars above are set in the PowerShell session.

### API (after changing Go code)
```powershell
# 1. Cross-compile a Linux binary on the host (works; in-container go build is flaky here).
cd C:\Users\user\salehcard\api
$env:CGO_ENABLED="0"; $env:GOOS="linux"; $env:GOARCH="amd64"
go build -trimpath -ldflags="-s -w" -o server-linux ./cmd/server
# 2. Minimal COPY-only image (write api/Dockerfile.prebuilt):
#    FROM gcr.io/distroless/static-debian12:nonroot
#    COPY server-linux /server
#    EXPOSE 8080
#    ENV PORT=8080
#    USER nonroot:nonroot
#    ENTRYPOINT ["/server"]
docker build -f api/Dockerfile.prebuilt -t salehcardprodacr.azurecr.io/salehcard-api:v2 api
$c = & $az acr credential show -n salehcardprodacr -o json | ConvertFrom-Json
docker login salehcardprodacr.azurecr.io -u $c.username -p $c.passwords[0].value
docker push salehcardprodacr.azurecr.io/salehcard-api:v2
# 3. Point the web app at the new tag (az rest PATCH config/web linuxFxVersion) then restart.
#    (On a NON-proxy network you can skip all this with: az acr build -r salehcardprodacr -t salehcard-api:v2 ./api)
```
Then clean up `server-linux` and `api/Dockerfile.prebuilt` (untracked, do not commit).

### Frontends (after changing web/ or admin/)
```powershell
# .env.production.local in each app already sets VITE_API_BASE_URL=https://salehcard-api.azurewebsites.net
cd C:\Users\user\salehcard\web ; pnpm build      # or admin
$tok = (& $az rest --method post --url "https://management.azure.com/subscriptions/1adb4811-6234-4822-b7f2-8411ed2cb999/resourceGroups/salehcard-prod/providers/Microsoft.Web/staticSites/salehcard-web/listSecrets?api-version=2023-12-01" -o json | ConvertFrom-Json).properties.apiKey
npx --yes @azure/static-web-apps-cli deploy ./dist --deployment-token $tok --env production
```
(For admin use `staticSites/salehcard-admin` and `./admin/dist`.)

### Reload catalog data into Atlas
```bash
cd api && MONGO_URI="<atlas-srv-uri>" DB_NAME=salehcard go run ./cmd/loadseed
# loadseed timeout is 15m (remote-cluster friendly). Idempotent; re-import refreshes price.
```

## Open items / follow-ups

1. **Rotate the Atlas DB password** — it was pasted in chat during setup. Change in Atlas →
   Database Access, then update the API's `MONGO_URI` app setting + restart.
2. **Change the admin password** — it was shown in chat. (Strong/random, but visible there.)
3. **Merge PR #3** (`feat/catalog-migration-browse`) into `main`.
4. **CI/CD** — deploys are currently manual; wire GitHub Actions (build+deploy cloud-side,
   which also sidesteps the local proxy). Needs a `workflow`-scope token to push workflow files.
5. **Tighten Atlas Network Access** — currently `0.0.0.0/0`; restrict to the App Service
   outbound IPs (`az webapp show ... outboundIpAddresses`) once stable.
6. **Custom domain** — currently on default `*.azurewebsites.net` / `*.azurestaticapps.net`.
7. **Remaining code-review findings** (fast-follows, see PR #3 description): fulfillAPI
   multi-item drop (latent until a real provider), `park()` CreditedToID, admin-created
   products need `rootDomain` derived from category, importer robustness (bool/int, en=ar).

## Useful pointers

- Earlier handoff (catalog migration): **`HANDOFF.md`**. Conventions: **`CONVENTIONS.md`**.
  Production-readiness notes: **`MIGRATION-READINESS.md`**.
- The `git status` shows persistent CRLF-only "modifications" on several `*.go` files
  (autocrlf, no content change) — ignore them; they never enter commits.
