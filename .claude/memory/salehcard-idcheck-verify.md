---
name: salehcard-idcheck-verify
description: Game-ID nickname verification (PR
metadata: 
  node_type: memory
  type: project
  originSessionId: 7e12f039-d8f3-4a19-bf45-d6783cf3c0b8
---

**PR #47** (`feat/game-id-verification`, off origin/main e46ac6c, opened 2026-07-05, 3 commits) does two things:
1. **Nickname verification** (`check_name`) for single-ID games: customer enters game ID on product-detail → `POST /api/v1/products/{id}/verify-account` → nickname → Buy Now unlocks. Hexagonal `internal/platform/idcheck` (Verifier port + `rapidapi`/`stub` adapters, mirrors `platform/sms`). Checkable single-ID slugs: `pubgm-global`, `dfm-garena`, `free-fire`.
2. **Per-game extra fields** attached to the order (collect-only games): Mobile Legends (id+zone), Genshin (id+server), Brawl Stars / Clash of Clans / Clash Royale (id+email). `OrderItem.Fields []{key,label,value}` — labels resolved server-side from the product's InputFields spec, `Sensitive` fields never persisted. Admin product editor gained a full **input-fields builder** (add/remove/key/type/label/sensitive). MLBB/Genshin are checkable in the API (`mobile-legends/{id}/{zone}`, `genshin/{id}/{server}`) but kept collect-only by decision → no multi-param verification built.

**Deferred prod config (do at deploy — not derivable from code):**
- Set Azure app settings **`IDCHECK_PROVIDER=rapidapi`** + **`RAPIDAPI_KEY=<rotated>`** on the API. Default is `stub` (safe no-op) until set. Optional `RATE_LIMIT_VERIFY_MAX`/`_WINDOW` (default 20/min).
- **ROTATE the RapidAPI key** `569a804d51msh…bcb33` — it was pasted in chat (compromised). Server-side only; never in the APK/admin bundle.
- Per-product: set `Verification` via the admin product editor's "Purchase-time ID verification" card (game slug e.g. `pubgm-global`, `dfm-garena`) — no DB migration.
- **App needs a manual APK rebuild** to show the verify UI (backend+admin auto-deploy on merge). See [[salehcard-flutter-emulator-run]] / prod build recipe.

Decisions: verify on product page before Buy · fail-open on outage · allow banned accounts (shown, not blocked) · admin config UI now. Fast-follows: persist nickname on order for operators, server-side re-verify at PlaceOrder, lookup caching.
