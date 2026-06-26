# SalehCard Mobile — Design → Implementation Manifest

Source of truth for the 6 Claude Design modules being implemented in the Flutter app.
Full implementation plan: `C:\Users\user\.claude\plans\use-the-claude-design-mcp-gentle-hartmanis.md`.

## Design source
- Claude Design project (via `claude_design` MCP): **`9c570d94-584a-41c6-9827-b072048e9688`**
- Each module = one interactive component (`Phone<Module>.dc.html`) + one canvas (`SalehCard <Module>.dc.html`).
- The exact frame inventory (screen × EN/AR × light/dark × state) is the per-module tables below.
- To pull a full component body in a module session: `DesignSync get_file` with the project id + `Phone<Module>.dc.html`
  (interactive sessions only — the design MCP needs claude.ai auth).

## Shared design tokens (already implemented in the app)
`core/theme/app_tokens.dart` (gradient `#3B5BFF→#8A3BFF→#D633FF`, CTA `#A21CAF`, accent `#22E3C8`, danger `#FF4D6D`, radii),
`core/theme/app_colors.dart` (`AppColors` ThemeExtension, light+dark, `context.colors.*`). Fonts: Plus Jakarta Sans (EN) /
IBM Plex Sans Arabic (AR). All screens must support EN+AR (RTL) and light+dark.

## Wiring legend
✅ wire real endpoint · 🟡 stub repo / display-only (no backend) — keep behind a real repository interface with `// TODO(backend)`.

## Modules

### S1 — Shop / funnel (`PhoneShop`) → `features/checkout/` (+ extend `features/catalog/`)
| Screen | States | Target | Route | Endpoint | Status |
|---|---|---|---|---|---|
| product detail | default, out-of-stock | extend `catalog/.../product_detail_screen.dart` | `/product/:id` | `GET /products/{id}` (variants + `inputFields[]`) | ✅ |
| cart | default, empty | `features/checkout/presentation/screens/cart_screen.dart` | `/cart` (shell) | local cart (no API) | ✅ local |
| checkout | default, wallet-insufficient, submitting, payment-error | `.../checkout_screen.dart` | `/checkout` | `POST /orders` (Idempotency-Key; server re-prices; paymentMethod wallet\|card\|usdt; playerId/recipient from inputFields) | ✅ |
| order success | default, processing | `.../order_success_screen.dart` | `/order-success/:id` | order response (`fulfillment.deliveredCode` / processing) | ✅ |

Hardest piece: dynamic `inputFields[]` renderer (type text\|amount\|quantity\|select, `sensitive` masks).

### S2 — Orders (`PhoneOrders`) → `features/orders/`  *(reuse Order entity/DTO from S1 — do NOT duplicate)*
| Screen | States | Target | Route | Endpoint | Status |
|---|---|---|---|---|---|
| orders list | default, empty | `.../orders_list_screen.dart` | `/orders` | `GET /orders` | ✅ |
| order detail | completed, processing, refunded | `.../order_detail_screen.dart` | `/orders/:id` | `GET /orders/{id}` | ✅ (refunded = display-only 🟡; refund is admin/501) |

### S3 — Wallet (`PhoneWallet`) → `features/wallet/`
| Screen | States | Target | Route | Endpoint | Status |
|---|---|---|---|---|---|
| wallet | default, empty-ledger | `.../wallet_screen.dart` | `/wallet` | `GET /wallet` (balance + transactions) | ✅ |
| top-up | card, usdt-pending | `.../topup_screen.dart` | `/wallet/topup` | `POST /wallet/topups {amount,method,ref}` | ✅ card; 🟡 USDT pending/poll (verify admin/501) |
| send money | form, review, insufficient | `.../send_money_screen.dart` | `/wallet/send` | none | 🟡 stub (no peer-send endpoint) |

### S4 — Account (`PhoneAccount`) → `features/account/`
| Screen | States | Target | Route | Endpoint | Status |
|---|---|---|---|---|---|
| menu / settings | default | `.../account_menu_screen.dart` | `/account` (shell) | local (locale/theme/logout + links) | ✅ local |
| profile | view, editing | `.../profile_screen.dart` | `/profile` | `GET /users/me`, `PATCH /users/me` | ✅ (only `locale`+`savedPlayerIds`; **no name field** 🟡, email read-only) |

### S5 — Browse (`PhoneBrowse`) → `features/browse/`
| Screen | States | Target | Route | Endpoint | Status |
|---|---|---|---|---|---|
| categories | default | `.../categories_screen.dart` | `/browse` (shell) | `GET /categories?withCounts` | ✅ |
| search | default, no-results | `.../search_screen.dart` | `/search` | `GET /products` (no text param) | 🟡 client-side filter |
| notifications | default, empty | `.../notifications_screen.dart` | `/notifications` | none | 🟡 stub |

### S6 — KYC (`PhoneKYC`) → `features/kyc/`  *(fully stubbed — no endpoints)*
| Screen | States | Target | Route | Endpoint | Status |
|---|---|---|---|---|---|
| status | unverified, pending, verified, rejected | `.../kyc_status_screen.dart` | `/kyc` | none | 🟡 stub |
| form | default, filled, error | `.../kyc_form_screen.dart` | `/kyc/form` | none | 🟡 stub |
| pending | default | (form → pending state) | — | none | 🟡 stub |

## Build order
Foundation (Session 0) → **S1 Shop → S2 Orders** (shared Order types) → S3 Wallet · S4 Account · S5 Browse · S6 KYC (independent, parallelizable).

## Stubs to track in `app/HANDOFF-DESIGN.md`
KYC · send-money · notifications · customer search · order refund (display-only) · USDT verify (poll-only) · profile name.
