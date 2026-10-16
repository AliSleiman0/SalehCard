# HANDOFF — 2026-10-16: per-unit pricing for live-app recharge

Review document for a fresh session. Work is **uncommitted** in worktree
`C:\dev\SalehCard-product-fields`, branch `fix/product-type-and-fields` off `origin/ios-first-release`.
The PR target is `ios-first-release` (the user chose option 2 in `HANDOFF-2026-10-16.md`). Plan file:
`C:\Users\user\.claude\plans\partitioned-cooking-falcon.md`.

## 1. What the client asked

The client sent screenshots and a recording of their hand-built **"soul star"** product.
- **Admin:** one package "90000" at $10 (reseller $9.70), plus an input field labelled "ID" with
  type **quantity** and min/max 9,000–900,000.
- **Website:** "Choose amount: 90000 – $10", a hardcoded "Account / Player ID" box and a quantity
  stepper (12 → $120, 99 → $990).
- **Complaint:** changing the field type doesn't change how the product behaves. They also asked
  why "Choose amount" is there.

**What they actually want:** live-app coin recharge (Soul Star, Up Live, Bigo, Poppo…). The customer
types their in-app ID and **the number of coins**, and pays proportionally. At $10 per 90,000 coins,
45,000 coins costs $5.

## 2. What I found

1. **The `quantity` field type means "number of packs".** The server applies its min/max to the
   pack count, so soul star demands at least 9,000 packs ($90,000). Every real order gets rejected at
   checkout.
2. **Website bugs:**
   - `web/.../adaptProduct.ts` drops `quantity` fields, then the page falls back to a hardcoded
     "Account / Player ID" box.
   - The stepper ignores min/max and caps at 99.
   - The Flutter app already respects the min/max.
3. **The bigger problem is already in prod.**
   - **Old site.** It sold ~221 `unit_balance` products ("charging-application"), each with an ID
     field, an "Enter Amount" field with min/max, and a per-unit cost. Up Live costs $0.0161 per
     diamond; the market rate is about $0.0175.
   - **Migration.** The catalog import never carried over the per-unit sell price; each product got a
     placeholder flat price.
   - **Result on prod.** 224 live charging-app products (the migration converts the 206 that are true
     per-unit products; the rest are pack-style or ambiguous) charge **one flat price whatever amount is
     typed** (read-only check of the prod public API on 2026-10-16). Examples:

     | Product | Flat price on prod | Typed range |
     | --- | --- | --- |
     | Up Live | $100 | 100–200,000 |
     | Bigo | $95.91 | |
     | Poppo, Xena | $100 | |
     | Soul Star (imported) | $10 | 10k–50M |
   - **The client's workaround.** The hand-built soul star was an attempt to get around this.

## 3. Decisions made with the user

- **Pricing approach:** **full per-unit pricing**, where the customer can type any exact amount. The
  user rejected the lighter "small packs × quantity" alternative.
- **Migration prices:** existing products are priced at **cost + markup %** (a CLI flag). The client
  fine-tunes them afterwards in admin.

## 4. What's being built

**Data model**
- New admin-owned `Product.unitPricing {fieldKey, blockUnits}`.
- The single variant price buys `blockUnits` units.
- `fieldKey` points at the existing `amount` input field, which supplies the min/max and the label.
- `{blockUnits: 0}` clears it on update.
- The importer-owned `amountConstraints` and `pricing.mode` fields are untouched.

**API (done, tested)**
- **Code:** `product/unit_pricing.go`, `product/service.go`, `product/repository.go`,
  `order/service.go`, `order/model.go`.
- **Validation:** the product must be `account_credit`, fulfilled manually (not api or bridge), have
  exactly one variant with price > 0, have a driving amount field with min > 0 and max ≥ min, and
  have no quantity field.
  - Updates are validated against the stored product with the incoming changes applied.
- **Order (`amount` required for these products):**
  - Qty must be 1 and the amount must be within bounds.
  - Price = `round2(blockPrice × amount ÷ blockUnits)`, calculated after reseller and offer
    pricing. Lines under $0.01 are rejected.
  - The driving field's value is overwritten with the real amount, and the denomination becomes
    "45,000".
- **Rejections:**
  - `amount` on a normal product is rejected.
  - An old app build that doesn't send `amount` gets an "update the app" error rather than being
    charged a price it never showed.
  - A product whose stored config has gone stale is refused.
- **Loader guard:** `loadseed` re-imports no longer overwrite a unit-priced product's variant.

**Migration CLI `api/cmd/unitpricing` (done, tested)**
- **Flags:** dry run by default; `--markup N` (required), `--report file.csv`, `--apply`,
  `--category`; prod writes also need `--confirm-prod`.
- **Block size:** the smallest power of 10 where cost × block ≥ $1. At a 10% markup, Up Live → $1.77
  per 100 and Poppo → $1.17 per 10,000.
- **Skips, with the reason in the CSV:**
  - Products that are already converted, not account_credit, not manually fulfilled, or have more
    than one variant.
  - Products with a quantity field, more than one amount field, no bounds, or no cost.
  - Products costing **≥ $1 per unit**. These are pack products (TAGS, WE PLAY GOLD, Gomeet
    Today) and need a human decision.
- **Re-run safety:** each update is guarded on `unitPricing` being absent, so re-running is safe.
- **Local seed dry run:** 206 would convert (204 charging apps + TikTok + Jawaker); 57 are skipped.

**Admin (subagent done; build, lint and test green)**
- **Fixed packages / Per unit switch** in Variants & pricing. Per-unit mode shows one row: price,
  reseller price, "per N units", the driving amount field, and a live preview.
- **Turning per-unit on** adds an "Enter Amount" field if needed and removes quantity fields, with a
  notice.
- **Quantity relabel:** shown as "quantity (number of packs)", with help text explaining quantity
  vs amount.
- **Order detail** shows "45,000 units".

**Website (subagent done, gates green, browser-checked)**
- **Fixes:** the stepper respects the quantity field's min/max (no 99 cap), and the phantom "Account
  / Player ID" box is gone.
- **Per-unit page:** an amount input with a range hint and "$X per N", an inline error and a live
  total. The package picker and stepper are hidden.
- **Cart and checkout:** cart lines are keyed by amount at qty 1, and checkout sends `amount`.
- **Browser check:** LAKA CHAT showed $11.52 for 45,000, matching the server.
- **Placeholder fixed:** the empty amount box used to read "Enter Enter Amount"; it now shows the
  range hint instead.

**Flutter app (subagent done; `flutter analyze` clean, `flutter test` 54/54)**
- Same behaviour as the website: amount input, live price, `amount` on the cart item and in the order
  request, driving field skipped at checkout, "from" price, l10n (en/ar), and DTO regen.
- The order detail screen hides "×qty" on amount lines.
- **Request body:** `placeOrderRequestJson` is now a public pure function, so it can be tested.
- A misconfigured unit-priced product shows "can't be bought right now".
- No iOS gate, since coins aren't regulated finance. No version bump.
- **Not run on a device or emulator yet.**
- **Agent's open questions:**
  - The range hints show ungrouped numbers ("Between 1000 and 1000000").
  - The amount box accepts whole numbers only.

## 5. Verified so far
- **API gates:** `go build`, `go vet` and `go test ./...` pass, and golangci-lint is clean.
- **Local end-to-end** (scratch DB `salehcard_unitpricing`, API :8099, web :5183, admin :5184):
  - The migration applies, and a re-run converts 0.
  - A re-import keeps the new prices.
  - A 45,000 wallet order charged exactly $11.52.
  - Missing amount, below-minimum amount and qty 2 are rejected with clear messages.

## 5b. Client confirmation + review fixes (later on 2026-10-16)

**Client media re-checked (voice notes transcribed with Whisper).** The Soul Star recording is *ours*,
not the client's. The client's evidence:
- Old site: Up Live (9,000 → $132.401) and Xena (135,000 → $14.798), where the customer types a count
  and the total updates live.
- Old admin video: cost and sell fields plus "من اجل كمية" (for quantity, e.g. 1,000).
- Voice note 9.ogg: "1,000 coins = $1 → put 1000 under, $1 above". That is exactly `blockUnits`.
- Video 1: "I put the number of coins, not the number of dollars."

The same voice note describes per-customer **level %** pricing (L1/L2/L3). The user **deferred it**.

**⚠️ Price source.** The old site's real prices are *below* the seed `pricing.cost`:

| Product | Old site | Seed cost |
| --- | --- | --- |
| Up Live | $0.01471 | $0.01611 |
| Xena | $0.00010961 | $0.00011195 |

So `--markup 10` would raise prices about 12–20%. **Don't `--apply` on prod until the client confirms
the price per block** (or signs off the dry-run CSV).

**Review fixes (all gates green):**
- **api**
  - Fixed-amount offers are ignored on unit-priced products, and admin create/update returns
    `400 OFFER_FIXED_UNIT_PRICED`.
  - Server "from" prices are rescaled to the minimum amount.
  - On conversion, `reseller_prices` entries for the variant are deleted (service and CLI; CSV column
    `reseller_custom_prices_removed`).
  - `loadseed` now also preserves inputFields, fulfillment type/mode and pricing on unit-priced products.
  - Non-whole amounts are rejected.
  - CLI: skips admin opt-outs (`unitPricing: null`); requires `pricing.mode=unit_balance` and USD or
    empty currency; adds a `pricing_mode` column.
- **web**
  - The ID box is decided after filtering, so per-unit products always get an ID box.
  - The cart key includes the player ID, so different IDs never merge or replace.
  - Amount parsing accepts Arabic/Persian digits and "45,000.00", and rejects fractions.
  - A $0.01 minimum is enforced on the client.
- **admin**
  - Warns which packages a switch to Per unit deletes.
  - Switching back to Fixed before saving restores the removed quantity fields and the package label.
- **app**
  - Arabic/Persian digit formatter on every numeric box.
  - The cart key includes the player ID.
  - Range hints are grouped.

**Decisions to review:**
- Reseller custom prices are deleted *before* the product write. If the conversion write then fails,
  those prices are lost (rare).
- On the web, "45.000" parses as 45 (no European-style grouping).
- Gates: api build, vet, `test -count=1` and golangci-lint all pass. web 64/64, admin 92/92, flutter
  65/65; `flutter analyze` is clean.
- **Not done:** browser re-check of the fixes, and any device or emulator run of the app.

## 5c. Browser QA (Claude in Chrome, local stack, 2026-10-16)

**Setup:** API :8099 against scratch DB `salehcard_unitpricing`, web :5183, admin :5184. All cases
passed.

**Website**
- **Product page (LAKA CHAT):** amount box with range hint and "$2.56 per 10,000", plus the ID box.
  45,000 → $11.52.
- **Amount input:** Arabic and Persian digits work; "45,000.00" and "45 000" → 45,000. "45.5" shows a
  whole-number error; out-of-range and empty values disable the buttons; a total under $0.01 is
  blocked (SAHI).
- **Cart:** two IDs → two lines; re-adding the same ID gives "Cart updated".
- **Checkout:** request has `amount: 45000`, `qty: 1`, per-line `playerId`. The wallet was charged
  exactly $23.04.
- **Regressions:** the quantity-field product (DD LIVE) and the code product (Steam) still show the
  picker and stepper; the stepper clamps to its max of 5,000 (no 99 cap).
- **Arabic:** RTL layout and labels are fine.

**Admin**
- Migrated product opens in Per unit mode, with the correct preview.
- Switching to Per unit warns "Saving will delete 2 other packages" and shows the quantity-field
  notice. Switching back to Fixed before saving restores both.
- Per-unit save and back-to-Fixed save both persist, and the variant id is kept.
- A fixed $ offer is rejected with the per-unit message. A 10% offer: the website shows $10.37 and
  the API charges $10.37; the list from-price is rescaled ($2.05 → $1.84).
- Converting a product deletes the reseller's custom price.
- Order detail shows "45,000 units" plus each line's ID.

**API rejections:** fractional amount, missing amount ("update the app"), qty 2, below minimum, and
`amount` on a normal product.

**No console errors.**

**Follow-ups 1–2 FIXED (admin only, no API change, Chrome-verified):**
1. **Order detail credit card.**
   - It lists every account-credit line (ID, units, line price) plus the order total.
   - While Processing, multi-account orders show a warning: "credits N different accounts".
   - The completion dialog has a tick box per credited line, and confirm stays disabled until all
     are ticked. Single-line orders are unchanged.
   - Code: `creditLines()` in `adaptOrder.ts`, tested.
2. **Fixed-price credit products that collect an amount.**
   - The editor shows a live warning in Fixed mode: "Customers type … but pay the package price —
     switch to Per unit / change the field type".
   - The product list shows a "Price ignores amount" badge (38 local products, all Gomeet-style
     credit packs). Transfer products are never flagged.
   - Code: `amountIgnoredByPrice()` in `unitPricing.ts`, tested.
- **Gates:** admin build, lint and test pass (102/102).

**Follow-ups found (none caused by this change), items 1–2 now fixed:**
- **Admin order "Account credit" card:** shows only the *first* line's player ID, with one "Mark
  credit as completed". A multi-ID order (now possible from the cart) risks the operator crediting
  only the first ID. Show every line's ID, or complete per line.
- **Fixed product with a leftover `amount` field:** the customer types an amount at checkout but pays
  the package price. This is the original client trap, and it still applies to the ~57 products the
  migration skips. Add an admin warning in Fixed mode, or hide such fields.
- **Order-success page:** says "Top-up credited" while the order is Processing (manual fulfillment).
- **Admin per-unit row:** shows an unlabelled "0%" badge. It is the reseller discount, not a margin;
  label it.
- **Offer editor preview:** still shows a fixed $ discount on a per-unit product before save. The
  save is rejected, so this is cosmetic.
- **Website:** a new cart line's toast says "Add to cart" (not "Added"); the "from" label on related
  products isn't translated in AR; `CartPage.tsx` has a hard-coded "cart.empty" eyebrow.

**Not covered:** the Flutter app on a device or emulator.

## 5d. QA round 2 (fresh DB `salehcard_qa2`, Chrome + CLI, 2026-10-16)

**A. Migration run the way prod will run it: all pass**
- **Fresh load:** `seed` + `loadseed` → 638 products, 0 unit-priced, flat prices (Up Live and Xena
  $100).
- **Dry run:** 263 candidates → 203 would convert / 60 skipped.
  - Skip reasons: 27 `currency_value`, 19 not credit, 8 packs ≥ $1/unit, 6 with several amount
    fields.
  - The dry run **wrote nothing** (snapshot hash identical).
  - It reported 1 reseller custom price to remove.
  - 203 rather than 206 is down to the new mode check.
- **Apply:** 203 converted and 1 custom price deleted. A second `--apply` converts 0.
- **Re-import:** `loadseed` changed **0** unit-priced rows. An admin-edited min (9,000) survived the
  re-import.
- **Prod guard:** a fake `*.mongocluster.cosmos.azure.com` URI with `--apply` and no `--confirm-prod`
  refuses instantly, without connecting.
- **CSV:** `C:\Users\user\AppData\Local\Temp\claude\C--dev-salehcard\f0e34d47-c7b2-4859-a5cb-3ea0afa80342\scratchpad\qa2-dry.csv`.

**B. New web/admin scenarios: all pass**
- **Category "from" price:** LAKA $2.05, BULALA $0.96.
- **Reseller price** (admin $2.30 per 10k; Silver tier 8%): the reseller sees $2.30 → 45,000 =
  $10.35 and is charged exactly $10.35.
- **Price edit** $2.56 → $3.00: the customer sees $3.00 and is charged $13.50.
- **Mixed cart** (per-unit + package + code): one order, correct per-line prices, one debit.
- **Insufficient funds:** the UI blocks with "Top up to cover the difference"; the API returns
  402 and the wallet is unchanged.
- **Idempotent retry:** same order, one debit.
- **Customer order history** shows "LAKA CHAT — 45,000".
- **Refund:** exact $3.00 back; a second refund returns 409.
- **Single-line completion:** no checklist; the customer is notified.
- **10% offer:** consistent across the admin list, website offers page, category card and product
  page ($2.70/10k → $12.15).
- **KYC gate:** blocks in the UI and returns 403 `KYC_REQUIRED`.
- **Console and server log:** no console errors and no 5xx.

**Bugs found and fixed (web, both older than this branch but made worse by it):**
1. **Stale data across sign-in.**
   - **Cause:** React Query caches catalog prices and the wallet per query key, not per user.
   - **Symptom:** after a reseller signed out and a customer signed in on the same browser, the
     customer saw the reseller's $2.30 price and $1,279 wallet until a reload.
   - **Fix:** `stores/auth.ts` resets the query cache when the signed-in identity changes or on
     logout (`stores/auth.test.ts`, 3 tests). Verified reseller → guest → customer.
2. **Product page state leaked between products.**
   - **Symptom:** `/product/:id` reused one component, so the typed player ID and amount carried
     over to the next product. Because the ID is now part of the cart key, a code product's cart
     line picked up "ID: MIX2".
   - **Fix:** `app/router.tsx` remounts the page per id. Verified.
- **Web gates:** build passes, lint has the 1 existing warning, tests 67/67.

**Mixed cart + code product: FIXED (2026-10-16, user chose "deliver codes now, park the rest").**
- **API changes:**
  - Inventory lines in a mixed cart are claimed immediately (`fulfillMixed`, shared
    `claimInventoryLines`).
  - Any claim failure triggers the existing `compensate` (release, full refund, failed).
  - Codes are stored **per line** in `items[].codes` (also for pure-code orders, so multi-line and
    qty>1 now show every code). `fulfillment.deliveredCode` still holds the first code for old
    clients.
  - The order stays **Processing** for the manual lines; timeline note "codes delivered for N
    line(s)". Admin search matches per-line codes.
  - Single atomic `UpdateOne` guarded on pending|processing (`UpdateFulfillmentWithCodes`).
- **Clients:**
  - **Web and app:** show a code vault per line even while Processing, plus the credit/transfer
    block. "No code needed" is never shown when codes exist.
  - **Admin:** shows the Code delivery card (per line, per-code reveal) alongside the credit
    checklist.
- **Gates:** api 37/37 packages + golangci-lint pass; web 72/72; admin 106/106; flutter 70/70 with
  analyze clean.
- **Chrome QA on `salehcard_qa2`:**
  - **Mixed cart:** the code is shown at once on the success and detail pages, stock goes 2→1, admin
    shows both cards, the checklist completes the order and the code is kept.
  - **Out of stock:** a code line out of stock is refused with 409 before any charge.
  - **Code-only, two lines:** both codes shown on web and admin.
- **Old app builds:** the detail screen shows `deliveredCode` (the first code). The success screen
  shows "processing" for a mixed order. Fixed in the next build.
- **⚠️ Prod remediation after deploy:** mixed orders placed **before** this fix never claimed their
  codes. They stay Processing with "No code needed". Query prod for processing orders with
  `items.fulfillmentType: "code"` and no codes, and deliver those by hand (or refund the code line).

**Previously "found, not fixed", now fixed (see above):**
- **Mixed cart with a code product never delivered the code.**
  - **What happens:** `resolveOrderMode` deliberately parks any mixed cart as manual and claims no
    code (stock stayed 2/0).
  - **What people see:** the customer's order page says "No code needed for this product". The
    admin's "Mark completed" delivers nothing.
  - **Options:** split mixed carts into separate orders; or claim codes for the code lines and park
    only the rest; or block mixing code + manual items at checkout.
- **Order-success page:** "Top-up credited / Amount credited" wording on a Processing order, from
  §5c, is still open.
- **Catalog prices with 3 decimals** (e.g. $0.656): the page shows $15.37, the wallet is debited
  $15.366. This is existing seed data.
- **Failed order in history:** an insufficient-funds attempt leaves a "Failed" order there
  (order-first design).

## 6. Open items for review
1. **Already done:**
   - All gates were re-run green at the end of the session.
   - The admin editor was browser-checked: it loads a migrated product in Per unit mode, saves, and
     keeps both `unitPricing` and the variant id.
   - **Still to do:** run the Flutter screens on an emulator or phone.
2. **Web decision to review:** adding the same amount twice replaces the cart line instead of adding
   a second one.
3. Commit, then open a PR into `ios-first-release` — **only when the user asks**.
4. **Rollout:**
   1. Merge to `main`, which deploys api, admin and web.
   2. The **user** runs the prod dry run (it needs the Cosmos firewall rule and the creds file), and
      the client reviews the CSV and markup.
   3. The user runs `--apply --confirm-prod`.
   4. The client fixes or deletes the hand-made "soul star" and reviews the 32 pack products by
      hand.
   5. New app builds: Android 1.0.4+5, and iOS once the 1.0.3 review is decided. Until then, old
      apps show "update the app" on these products only.
5. **Tell the client now:** live-app orders on prod are currently charged a flat price, so operators
   should check the typed amount against what was paid before crediting.
