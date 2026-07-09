---
name: salehcard-az-tls-norton
description: "RESOLVED 2026-07-05 — Norton removed, local `az` works; historical TLS-MITM notes kept for reference"
metadata:
  node_type: memory
  type: reference
  originSessionId: 607d6d73-89aa-4b92-b75f-4f6e8155c2a4
---

**RESOLVED 2026-07-05: Norton was removed from the machine.** Local `az` (2.87.0) now mints
tokens and makes live calls fine, logged in as `Ali.Sleiman@oz-consultants.com` on
subscription **SalehCard** (`1adb4811-…`), tenant OZ Consultants. Azure work can be done
from the local CLI — no Portal/Cloud Shell detour needed. CLAUDE.md + DEPLOYMENT.md still
carry the old "use Cloud Shell" warning; treat that as stale. Gotcha found post-Norton:
`Microsoft.Storage` provider was NotRegistered on the subscription (first storage account) —
`SubscriptionNotFound` from `az storage account check-name` means register the RP, not a bad sub.

**Historical (pre-2026-07-05):** `az` failed with `SSLCertVerificationError ... Basic
Constraints of CA cert not marked critical` because **Norton "Web/Mail Shield"** MITMed all
TLS (leaf issued by `CN=Norton Web/Mail Shield Root`); OpenSSL 3.x rejected its root
structurally, so `REQUESTS_CA_BUNDLE` fixes couldn't work. `git`/`gh`/browsers used schannel
and were unaffected. Workaround was Azure Portal / Cloud Shell.

Note: the auto-mode safety classifier may still block `az ... appsettings set` writes to the
live prod API — if denied, the user runs the exact command via `!` or grants a Bash
allow-rule. See [[salehcard-prod-deploy]].
