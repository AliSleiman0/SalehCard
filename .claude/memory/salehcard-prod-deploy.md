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

**Critical gotcha:** the dev machine is behind a TLS-intercepting corporate proxy that
breaks Azure CLI/Node SSL. Use `az rest` for Microsoft.Web ops, build+`docker push`
locally (not `az acr build`), and set `NODE_EXTRA_CA_CERTS`/`REQUESTS_CA_BUNDLE` +
process-scoped `AZURE_CLI_DISABLE_CONNECTION_VERIFICATION`. Full details in DEPLOYMENT.md.

Open: rotate Atlas+admin passwords (exposed in chat), merge PR #3, add CI/CD. See also
[[salehcard-dev-env]] for the local stack.
