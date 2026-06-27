# CI/CD setup (GitHub Actions → Azure)

Workflows: [`ci.yml`](workflows/ci.yml) (build/lint/test on PRs) and
[`deploy.yml`](workflows/deploy.yml) (auto-deploy to prod on push to `main`).
Builds run on GitHub cloud runners, so the corporate-proxy TLS workaround in
`DEPLOYMENT.md` does **not** apply here — `az`, `docker`, and the SWA tooling just work.

Auth is **OIDC federated credentials** — no long-lived Azure secrets stored in GitHub.

> **What we actually used (2026-06-27):** a **user-assigned managed identity**, *not* an app
> registration. App registration requires directory-level rights in the **OZ Consultants**
> tenant that we don't have; a managed identity is a plain Azure *resource* that only needs
> Contributor on the resource group, lives in the subscription's tenant automatically, and
> supports GitHub OIDC federated credentials identically. `azure/login@v2` works the same.
> Identity in use: **`salehcard-github-oidc`** in RG `salehcard-prod`.

## One-time prerequisites

Do these in the **Azure Portal** (browser trusts the proxy CA, so no `az` SSL workaround).
**First switch the portal directory to OZ Consultants** (tenant `ce8b637b-0d06-4e80-8c99-031b7902f0ad`) —
the subscription/registry live there, and the identity must be created in that same tenant.

### 1. User-assigned managed identity
Portal → **Managed Identities → Create** → RG `salehcard-prod`, West Europe, name
`salehcard-github-oidc`. Record its **Client ID** (Overview).

### 2. Federated credential(s)
On the identity → **Settings → Federated credentials → + Add** → scenario *GitHub Actions deploying
Azure resources* (Issuer `https://token.actions.githubusercontent.com`, Audience `api://AzureADTokenExchange`).
**The repo name is case-sensitive and must match GitHub exactly — `AliSleiman0/SalehCard`** (capital S/C):

| Purpose | Entity / Subject |
| --- | --- |
| Prod deploys | Branch → `main` (`repo:AliSleiman0/SalehCard:ref:refs/heads/main`) |
| PR CI (optional) | Pull request (`repo:AliSleiman0/SalehCard:pull_request`) |
| Env gate (optional) | Environment → `production` |

> The form's **Name** field is required — the **Add** button stays disabled until it's filled.
> CI (`ci.yml`) does not log into Azure, so the PR credential is only needed if you later add
> Azure steps to PR runs.

### 3. Role assignments for the managed identity
In each IAM blade, the picker → "Assign access to" → choose the **Managed identity** tab
(User-assigned), then select `salehcard-github-oidc`:
- **`AcrPush`** on registry `salehcardprodacr` (push images).
- **`Contributor`** on resource group `salehcard-prod` (set the API container tag + restart, and
  read each SWA deployment token at runtime). Contributor covers `Microsoft.Web/sites/*` and
  `staticSites/listSecrets` — **both roles are required**; AcrPush alone fails the App Service steps.
  *Tighten later to `Website Contributor` + a custom role granting `staticSites/listSecrets/action`.*

> "Add role assignment" greyed out? You're either in the wrong tenant/registry, or you lack
> **Owner / User Access Administrator** at that scope (plain Contributor can't assign roles) —
> the subscription owner must grant the roles or grant you User Access Administrator.

### 4. GitHub repo → Settings → Secrets and variables → Actions → **Variables**
These are not sensitive — add as **variables**, not secrets:

| Variable | Value |
| --- | --- |
| `AZURE_CLIENT_ID` | the **managed identity's** Client ID from step 1 |
| `AZURE_TENANT_ID` | `ce8b637b-0d06-4e80-8c99-031b7902f0ad` (OZ Consultants — **not** Ezonic) |
| `AZURE_SUBSCRIPTION_ID` | `1adb4811-6234-4822-b7f2-8411ed2cb999` |
| `VITE_API_BASE_URL` | `https://salehcard-api.azurewebsites.net` |

`VITE_API_BASE_URL` is required because `web/.env.production.local` and
`admin/.env.production.local` are gitignored — CI must supply the prod API URL at build time.

### 5. Push the workflow files
Pushing under `.github/workflows/` needs a token with **`workflow` scope**.

## Troubleshooting (issues hit during first setup)
- **`AADSTS70021` / "No matching federated identity record … case-insensitive match, NOT
  case-sensitive"** → the FIC subject case doesn't match the token. GitHub stamps the repo's
  canonical name (`SalehCard`); set the FIC Repository to that exact case. If the FIC already
  matches, then `AZURE_CLIENT_ID` is pointing at the *wrong* identity (e.g. a leftover app
  registration in another tenant) — set it to the managed identity's Client ID.
- **`AuthorizationFailed … 'Microsoft.Web/sites/config/list/action'`** → OIDC works but RBAC is
  missing; add **Contributor** on `salehcard-prod` (AcrPush alone isn't enough).
- **Identity not found in the role-assignment member search** → wrong tenant (created in Ezonic
  instead of OZ Consultants), or you searched the name with a typo — search by Client-ID GUID.

## Optional: manual approval before prod
Create a GitHub **Environment** named `production` with a required reviewer, then add
`environment: production` to the `deploy-*` jobs in `deploy.yml`. No other change needed.

## Rollback
Every image is tagged with the commit SHA. To roll back:
```
az webapp config container set -g salehcard-prod -n salehcard-api \
  --container-image-name salehcardprodacr.azurecr.io/salehcard-api:<previous-sha>
az webapp restart -g salehcard-prod -n salehcard-api
```

---

## MongoDB move to Azure — Cosmos DB for MongoDB (vCore), Free tier ✅ DONE 2026-06-27

Independent of CI/CD; the pipelines never reference the connection string — only the API App
Service `MONGO_URI` app setting changed. **Cluster: `docdb-cluster-20260626-2230`** (Free tier,
`salehcard-prod`, West Europe).

What was done (for reference / repeating):
1. Created the **vCore Free-tier** cluster. Networking = public access + "Allow Azure services"
   + dev IP `213.204.66.98` (no `0.0.0.0/0`). Encryption = service-managed key.
2. Connection string (vCore SRV): the password **must be percent-encoded** (`^`→`%5E`, `@`→`%40`)
   and the URI **must include `retrywrites=false`** for vCore. Go driver trusts the Windows store,
   so the proxy doesn't block local connects.
3. Catalog loaded with `cd api && go run ./cmd/loadseed` (env `MONGO_URI`/`DB_NAME`) → 86 cats + 635 products.
4. Non-catalog data migrated `mongodump` (Atlas) → `mongorestore --noIndexRestore` (vCore) for
   `users` + `wallet_transactions`; `codes`/`orders` were empty. Pre-migration backup:
   `C:\Users\user\.salehcard-atlas-dump`.
5. Set the App Service `MONGO_URI` to the encoded vCore string + restart; verify `/health`, catalog, login.
6. **TODO:** decommission Atlas M0 once verified (clears its leaked password + open network rule).
