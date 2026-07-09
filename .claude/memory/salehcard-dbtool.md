---
name: salehcard-dbtool
description: "api/cmd/dbtool — the committed, reusable CLI for inspecting/maintaining the SalehCard DB (use instead of one-off scripts)"
metadata: 
  node_type: memory
  type: reference
  originSessionId: 2b62a19c-06ce-454b-a26c-b79e2aafc4d3
---

**Use `api/cmd/dbtool` for prod/local DB inspection & maintenance** instead of hand-rolling
throwaway scripts (added 2026-07-08, PR #62). Reads `MONGO_URI`; read-only subcommands run
freely, destructive ones are a dry run unless `--apply`. Warns when connected to the managed
(Cosmos) cluster; masks code/PIN values; `--help` needs no DB.

Commands:
- `product <id>` — fulfillment config (type/mode/bridge/stock) + code counts by status
- `codes <productId>` — list a product's code rows (masked) + the order behind each
- `codes-clear <productId> [--apply]` — delete ALL a product's codes, reset stock (dry run default)
- `order <id>` — order summary (status/total/userId/items)

Run against prod via the [[salehcard-prod-db-direct-access]] firewall recipe:
```
MONGO_URI="$(grep -oE 'mongodb\+srv://[^[:space:]]*maxIdleTimeMS=120000' DEPLOY-CREDS.local.md | head -1)" \
  go run ./cmd/dbtool <command> [args]
```
Also has `code <value>` (find one code by exact value → product/status/order) and
`code-clear <value> [--apply]` (delete a burned/duplicate PIN everywhere it appears,
re-mirrors each product's stock).

Gotcha: Go's `flag` stops at the first positional, so put `--apply` BEFORE the id
(`codes-clear --apply <id>`, `code-clear --apply <value>`), not after.

Gotcha (2026-07-08): the Go mongo driver's TLS cert verification can start HANGING
mid-session (connect times out `~4s`, "context deadline exceeded ... total connections: 1")
even though the firewall + cluster are healthy and a raw Python TCP+TLS test to the
SRV node (`fc-…-000.mongocluster.cosmos.azure.com:10260`) succeeds — likely a slow
OCSP/CRL check or a TLS-intercepting security product on the box. Workaround: append
`&tlsInsecure=true` to MONGO_URI for the maintenance connection. NOT a firewall issue
(widening to the whole 213.204.66.0/24 did NOT help; tlsInsecure did). Extend by adding a case in the `switch` +
a `cmd*` function. Related migration tool: `cmd/bridgedirect` (see [[salehcard-direct-recharge-migration]]).
