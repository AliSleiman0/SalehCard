# Translation Assessment

_Assessed 2026-07-05. No code changed — this is a findings/triage doc._

Covers the three translation surfaces in the monorepo:

| Surface | Locales | Keys | Verdict |
|---|---|---|---|
| `/app` (Flutter mobile — active) | **en, ar only** | 228 | Structurally perfect; high-quality AR; no Turkish |
| `/admin` console | en, ar, tr | 116 | Structurally perfect; high-quality AR+TR |
| `/web` storefront (legacy, retiring) | en, ar, tr | 204 | Complete; ~45 dead keys |

## What's solid

- **No missing / extra / empty keys** in any locale of any app. Every `en` key exists in `ar`/`tr`.
- **Placeholders match** everywhere (`{phone}`, `{count}`, `{amount}`, `{field}`). The one scan-flagged
  "mismatch" (`categoryItemCount`) is the Arabic being *better* — it adds correct `=2`/`few`/`many`
  plural categories English doesn't need.
- **Arabic and Turkish are genuinely good** — idiomatic, consistent terminology
  (المحفظة/Cüzdan, التحقق من الهوية, قيد المراجعة). Not machine-dumped.
- **Mobile app has zero hardcoded user-facing strings** — every label goes through `AppLocalizations`.

## Real gaps (ranked, need a decision)

### 1. Mobile app has no Turkish
Web + admin ship en/ar/tr; the Flutter app is **en/ar only** (`app/lib/core/i18n/arb/` has
`app_en.arb` + `app_ar.arb`, no `app_tr.arb`). If Turkish is a mobile target market this is a whole
missing translation (new `app_tr.arb` + register the `tr` locale). If mobile is Lebanon-only (ar/en)
by design, this is fine. **Decision needed.**

### 2. Language resets to English on every reload (web + admin)
Both `web/src/i18n/config.ts` and `admin/src/i18n/config.ts` force `lng: 'en'` on load (documented
handoff decision). The switcher works in-session but the choice doesn't persist. Making it stick is a
wiring change (localStorage detector). **Decision needed** — is the reset intentional?

### 3. Five dead keys in the mobile ARB
Defined in both `app_en.arb` / `app_ar.arb` but never referenced in app code:
- `requestPhysicalCard`
- `cardInfo`
- `kycUploadLabel`
- `kycUploadHint`
- `kycUploadSelected`

The three `kycUpload*` keys suggest the **KYC form's document-upload UI isn't wired**. Confirm that's
intended before deleting the keys (deleting the strings hides the gap; the missing UI may be the real
issue).

### 4. Hardcoded (non-localized) strings in admin
Would stay English in AR/TR mode:
- `admin/src/components/layout/Footer.tsx:49` — "Account"
- `admin/src/features/catalog/pages/ProductDetailPage.tsx:228-230` — country names
  (United States / United Arab Emirates / Saudi Arabia)
- `admin/src/features/reseller/pages/ResellerDashboardPage.tsx:133-134` — "Product" / "Retail" headers

Peripheral pages; core admin nav/actions/expenses are fully translated. (Scan was a narrow JSX
text-node regex — there may be a few more in the same peripheral pages.)

## Minor quality nits (cosmetic)

- **Inconsistent Arabic numerals** — mix of Western and Arabic-Indic digits:
  - app `newPasswordHint` = "٨ أحرف على الأقل" vs `passwordTooShort` = "8 أحرف على الأقل"
  - web `transfer_note` = "١–٣ ساعات" vs `trust_rated` = "240 ألف"
  - Pick one system and normalize.
- **`notifyMe`** — app AR "تنبيهي" is awkward; web AR uses the cleaner "أبلغني". Align (prefer "أبلغني"
  or "نبّهني").
- **`cartItemsCount`** (app) uses a fixed plural "عناصر" for all counts — grammatically wrong for 1–2
  items. `categoryItemCount` already handles this with proper plural forms; `cartItemsCount` should
  match that pattern.
- **`chooseAmount`** (app) = "اختر الفئة" ("choose category") for EN "Choose amount" — contextually
  it's a denomination so it's defensible, but slightly off.
- **admin `[of]`** in Turkish = `"/"` (a slash, not a word) → renders "Showing X / Y results".
  Intentional-looking but inconsistent with AR/EN which use words.

## Web unused keys (low priority — `/web` is being retired)

~45 keys in `/web` are unreferenced (reseller/agent, card-payment fields like `card_num`/`cvc`,
category tags). Leave alone unless doing a cleanup pass. Note: some `cat_*` may be dynamically
composed elsewhere; verify before deleting.

## Suggested fix order (when approved)

1. **#3** — remove dead ARB keys **after** confirming KYC upload wiring.
2. **Minor nits** — Arabic numeral normalization, `notifyMe`, `cartItemsCount` plural.
3. **#1 / #2** — bigger calls (mobile Turkish; language persistence) once direction is decided.
4. **#4** — localize the stray admin strings if AR/TR admin is a real use case.
