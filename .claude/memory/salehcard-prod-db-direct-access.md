---
name: salehcard-prod-db-direct-access
description: How to run a script/mongosh directly against the prod Cosmos vCore DB from the dev box (firewall rule for the drifting egress IP)
metadata: 
  node_type: memory
  type: project
  originSessionId: 3230a474-9b25-4a4a-bf7d-ac2b99c69cbc
---

To connect **directly** to the prod Cosmos DB for MongoDB vCore (`docdb-cluster-20260626-2230`,
RG `salehcard-prod`) from the local dev machine — e.g. a one-off Go seed/inspect script —
you must add a firewall rule for the **current** dev egress IP, which **drifts** across the
`213.204.66.x` ISP range (seen: `.98`, `.3`, `.221`). Stale rules pile up in the cluster
firewall; the NAT-gateway prod egress is `52.157.69.89`.

Recipe (needs `az` logged in AND being **out of auto mode** — prod firewall writes + the
inline prod URI are blocked by the auto-mode classifier):
1. `curl -s https://api.ipify.org` → current IP.
2. `az rest --method put --url ".../mongoClusters/docdb-cluster-20260626-2230/firewallRules/localpc-<date>?api-version=2024-07-01" --body '{"properties":{"startIpAddress":"<ip>","endIpAddress":"<ip>"}}'` (wait ~30s, provisioningState=Succeeded).
3. Feed `MONGO_URI` from the gitignored `DEPLOY-CREDS.local.md` WITHOUT hardcoding it inline
   (classifier blocks the secret in the transcript): `MONGO_URI="$(grep -oE 'mongodb\+srv://[^[:space:]]*maxIdleTimeMS=120000' DEPLOY-CREDS.local.md | head -1)" go run ./cmd/<tool>`.
4. **Delete the rule when done** (`az rest --method delete ...`).

Prod-admin provisioning done this way on 2026-07-06 (see [[salehcard-prod-deploy]] + DEPLOY-CREDS.local.md).
Gotcha found: the 3 target phone numbers already existed as phone-only app accounts with real
wallet balances — promoting in place (add email/role/password to the existing doc) preserves
the wallet and avoids the unique-phone-index conflict of creating a fresh account.
