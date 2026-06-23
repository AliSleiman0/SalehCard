# SalehCard catalog migration

Two-step, re-runnable migration of the legacy storefront catalog into the new system.

- **Importer** (`/migration`, this module) — fetches `https://api.salehcard.com`, applies the business rules (spec §5/§6), and writes JSON seeds. No DB access. Standalone Go module (stdlib only).
- **Loader** (`api/cmd/loadseed`, in the `api` module) — reads the seeds and upserts them into MongoDB via the product + category repositories. Idempotent (keyed on `legacyId`).

## 1. Import (produce seeds)

```bash
cd migration
go run ./cmd/import                 # all 8 roots; uses the _raw/ cache when present
go run ./cmd/import --refresh       # re-fetch from the live API
go run ./cmd/import --root 63       # a single root (debugging)
```

Outputs under `migration/`:
- `seed/categories.json`, `seed/products.json` — the migrated catalog (target schema).
- `seed/_review.json` — the human worklist: every product/category with a decision flag, plus per-flag counts and a per-root summary.
- `run-summary.md` — totals by root / fulfillment type / mode, flag counts, and the owner decisions still open (incl. the category-107 split/rename recommendation).
- `_raw/category_{id}_{lang}.json` — cached raw responses (also serve as offline fixtures).

### Corrections that survive re-runs — `overrides.json`

Hand-edit `overrides.json` (keyed by `legacyId`) to force fields and clear flags; the importer applies it **last**, so corrections persist across re-imports:

```json
{
  "18": {
    "fulfillment": { "type": "account_credit", "mode": "api", "provider": 5 },
    "pricing": { "mode": "currency_value" },
    "clearFlags": ["fulfillment_review"]
  }
}
```

Use `"clearFlags": ["*"]` to clear all flags on a product.

## 2. Load (after reviewing the seeds)

```bash
cd api
go run ./cmd/loadseed --dry-run     # report inserts/updates, write nothing
go run ./cmd/loadseed               # upsert into MongoDB (idempotent; re-run = updates, no duplicates)
```

The loader keeps `stock` and the synthesized `variants[]._id` insert-only, so re-importing never clobbers admin-managed inventory or churns variant ids referenced by order history.

## Classification confidence

When fulfillment/pricing classification is uncertain the importer applies the **safer default** (`manual_operator` over `api`, `cancellable=false` for money movement) **and flags it** — it never guesses silently. Raw `mainPrice`/`base_price` are preserved exactly; only the *interpretation* (pricing `mode`) is classified and flagged.
