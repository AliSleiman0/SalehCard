# CI/CD setup (GitHub Actions → Azure)

Workflows: [`ci.yml`](workflows/ci.yml) (build/lint/test on PRs) and
[`deploy.yml`](workflows/deploy.yml) (auto-deploy to prod on push to `main`).
Builds run on GitHub cloud runners, so the corporate-proxy TLS workaround in
`DEPLOYMENT.md` does **not** apply here — `az`, `docker`, and the SWA tooling just work.

Auth is **OIDC federated credentials** — no long-lived Azure secrets stored in GitHub.

## One-time prerequisites

Do these in the **Azure Portal** (browser trusts the proxy CA, so no `az` SSL workaround).

### 1. App registration + service principal
Azure AD → App registrations → **New registration** (e.g. `salehcard-github-oidc`).
Record the **Application (client) ID** and **Directory (tenant) ID**.

### 2. Federated credentials
On that app → Certificates & secrets → **Federated credentials** → add one per subject
(Issuer `https://token.actions.githubusercontent.com`, Audience `api://AzureADTokenExchange`):

| Purpose | Entity / Subject |
| --- | --- |
| Prod deploys | Branch → `main` (`repo:AliSleiman0/salehcard:ref:refs/heads/main`) |
| PR CI (optional) | Pull request (`repo:AliSleiman0/salehcard:pull_request`) |
| Env gate (optional) | Environment → `production` |

> CI (`ci.yml`) does not log into Azure, so the PR credential is only needed if you later
> add Azure steps to PR runs.

### 3. Role assignments for the service principal
- **`AcrPush`** on registry `salehcardprodacr`.
- **`Contributor`** on resource group `salehcard-prod` (update the API web app's container
  tag + restart, and read each SWA deployment token at runtime).
  *Tighten later to `Website Contributor` + a custom role granting
  `Microsoft.Web/staticSites/listSecrets/action`.*

### 4. GitHub repo → Settings → Secrets and variables → Actions → **Variables**
These are not sensitive — add as **variables**, not secrets:

| Variable | Value |
| --- | --- |
| `AZURE_CLIENT_ID` | app (client) ID from step 1 |
| `AZURE_TENANT_ID` | tenant ID from step 1 |
| `AZURE_SUBSCRIPTION_ID` | `1adb4811-6234-4822-b7f2-8411ed2cb999` |
| `VITE_API_BASE_URL` | `https://salehcard-api.azurewebsites.net` |

`VITE_API_BASE_URL` is required because `web/.env.production.local` and
`admin/.env.production.local` are gitignored — CI must supply the prod API URL at build time.

### 5. Push the workflow files
Pushing under `.github/workflows/` needs a token with **`workflow` scope** (see
`DEPLOYMENT.md` open item #4).

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

## MongoDB move to Azure — Cosmos DB for MongoDB (vCore), Free tier

Independent of CI/CD; the pipelines never reference the connection string. Only the API App
Service `MONGO_URI` app setting changes.

1. Create a **Cosmos DB for MongoDB vCore** cluster in `salehcard-prod` (West Europe),
   **Free tier**. Set an admin user/password. Add firewall rules for the API App Service
   outbound IPs (`az webapp show -g salehcard-prod -n salehcard-api --query outboundIpAddresses`)
   and your IP for the one-time data load.
2. Copy the connection string (vCore SRV form). The Go driver trusts the Windows cert store,
   so the local proxy does not block this.
3. Load catalog data with the existing idempotent loader (15-min timeout, remote-friendly):
   ```
   cd api && MONGO_URI="<vcore-srv>" DB_NAME=salehcard go run ./cmd/loadseed
   ```
   Re-seed the admin user as needed.
4. Update the App Service `MONGO_URI` app setting to the vCore string, then restart. Verify
   `/health` and a catalog request, plus a test order + wallet debit.
5. Decommission the Atlas M0 cluster once verified (also clears Atlas open items #1/#5).
