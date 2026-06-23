# Catalog import — run summary

- Categories: **86**
- Products: **635**

## Products by root domain

- app_topups: 255
- games: 178
- giftcards: 46
- gsm_tools: 52
- money_transfers: 14
- software: 27
- telecom: 52
- wallets_crypto: 11

## Products by fulfillment type

- account_credit: 570
- code: 46
- transfer: 19

## Products by fulfillment mode

- api: 11
- bridge_device: 35
- inventory: 46
- manual_operator: 543

## Review flags (counts)

- amount_corruption: 1
- fulfillment_review: 524
- fx_rate_review: 31
- markup_stripped: 106
- needs_translation: 721
- pricing_review: 311
- sensitive_input: 12

## Top decisions still needed (owner)

- Category 107 "Cryptocurrency Section" is mislabeled — recommend splitting/renaming into Crypto and E-Wallets, and moving IMO under app top-ups. Emitted with rootDomain=wallets_crypto; do not silently rename.
- Provider IDs behind check_name (e.g. provider 5 = PUBG verifier) need owner confirmation before real api adapters are wired.
- Money-transfer and Syrian-telecom pricing carries embedded/stale FX (flag fx_rate_review) — owner decision: live FX vs manual updates.
- Products flagged fulfillment_review default to manual_operator (safer) — confirm or set fulfillment.mode in overrides.json.

## overrides.json

Hand-edit `overrides.json` (keyed by legacyId) to force `fulfillment.{type,mode,provider}` / `pricing.mode` and clear flags; the importer applies it last so corrections survive re-runs. Example:

```json
{
  "18": {
    "fulfillment": { "type": "account_credit", "mode": "api", "provider": 5 },
    "clearFlags": ["fulfillment_review"]
  }
}
```
