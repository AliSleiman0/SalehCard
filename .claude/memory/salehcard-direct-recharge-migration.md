---
name: salehcard-direct-recharge-migration
description: "Lebanese \"direct\" recharge products (alfa-direct/mtc-direct) were misconfigured; fixed to bridge recharge_line via cmd/bridgedirect on 2026-07-08"
metadata: 
  node_type: memory
  type: project
  originSessionId: 2b62a19c-06ce-454b-a26c-b79e2aafc4d3
---

The 15 `alfa-direct` + `mtc-direct` prod products (e.g. "ALFA 3.03 DIRECT") were
imported in inconsistent half-states — some `account_credit`+`manual_operator`, some
`bridge_device` with `bridge=null`, one left `fulfillmentType=code` — none with a bridge
spec, none using the `phone` input-field key, all with a `qty` field. Symptom the user
hit: the `code` one showed **no phone box** at checkout (code/pin products render zero
input fields), so there was nowhere to enter the number to charge.

Root model: a code/pin product hands the customer a scratch-card PIN to redeem themselves
(no phone). "Direct" = charge the entered line → must be `account_credit` + bridge. Two
bridge methods differ hugely: `transfer_credit` (bridge sends credit from the SIM's own
balance; needs per-variant `faceValue`; NO PINs — this is what the working `airtime-alfa`/
`airtime-touch` products use) vs `recharge_line` (redeems one uploaded scratch-card PIN
from inventory per order). User chose **recharge_line** for the direct products.

Fixed 2026-07-08 with **`api/cmd/bridgedirect`** (committed, PR #59): sets
`account_credit` + `bridge_device` + `bridge{provider, recharge_line}` + a single `phone`
field (reusing the localized label, dropping qty/duplicates). Provider derived from
category: `alfa*`→alfa, `mtc*`/`touch*`→touch. Dry-run default, `--apply`; applied to prod
via the [[salehcard-prod-db-direct-access]] firewall dance.

**Operational follow-up (NOT code):** these products have **zero PIN stock** — recharge_line
claims one PIN per order, so PINs must be uploaded (now visible in the Inventory bulk-upload
picker per [[salehcard-alfa-debug-handoff]]-era work) and the bridge phone kept online, else
orders fail `OUT_OF_STOCK`. Data-quality aside spotted: `airtime-alfa` "3$ ALFA" has
provider `touch` (wrong operator) — pre-existing, not touched by this migration.
