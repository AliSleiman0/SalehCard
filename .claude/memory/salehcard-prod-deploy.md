---
name: salehcard-prod-deploy
description: SalehCard is live in production on Azure — read DEPLOYMENT.md before any deploy/infra work
metadata: 
  node_type: memory
  type: project
  originSessionId: bd1aac8e-5bea-4913-9a01-91e4f7a4cfe3
---

SalehCard was deployed to **Azure production on 2026-06-22** (subscription `SalehCard`,
RG `salehcard-prod`, West Europe): API on App Service (`salehcard-api`), storefront +
admin on Static Web Apps, MongoDB Atlas (M0). All live and verified.

**Before touching deploy/infra, read the repo's `DEPLOYMENT.md`** (URLs, resources,
redeploy recipes, open items). Secrets are in the gitignored `DEPLOY-CREDS.local.md`
(admin login, Atlas URI). Code ships from branch `feat/catalog-migration-browse` (PR #3,
not yet merged — see [[salehcard-design-port]], [[salehcard-admin-port]]).

**Update (2026-06-27):** CI/CD now exists — `.github/workflows/deploy.yml` auto-deploys
to Azure on push to `main` (path-filtered, OIDC via managed identity `salehcard-github-oidc`,
client `8e67393c-…`). **Prod DB migrated Atlas → Azure Cosmos DB for MongoDB (vCore)**
`docdb-cluster-20260626-2230` (`retrywrites=false`; Atlas pending decommission). Repo renamed
to **`AliSleiman0/SalehCard`** (capital S/C) — OIDC subjects are case-sensitive. Phone-OTP auth
+ Monty SMS shipped — see [[salehcard-phone-otp-auth]] and `HANDOFF-2026-06-27.md`.

**Critical gotcha:** the dev machine is behind a TLS-intercepting corporate proxy that
breaks Azure CLI/Node SSL. **Easiest: run `az` in Azure Cloud Shell** (browser, pre-auth,
no proxy — but it may launch in the wrong tenant; `az login --tenant ce8b637b-…` + set the
`SalehCard` sub). Locally: `az rest` for Microsoft.Web ops, build+`docker push` (not
`az acr build`), CA-bundle env vars. Full details in DEPLOYMENT.md.

Open: rotate exposed secrets (Atlas+admin+**Cosmos**+**Monty token**), decommission Atlas,
register a "SalehCard" Monty sender. See also [[salehcard-dev-env]] for the local stack.
