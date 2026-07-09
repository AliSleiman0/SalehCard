# Memory Index

- [SalehCard RBAC](salehcard-rbac.md) — admin-console RBAC SHIPPED to main + prod 2026-07-08 (PR #64, custom roles, domain×view/manage, implicit Super Admin via nil adminRoleId, perms in JWT, +10 review fixes)
- [SalehCard Alfa debug handoff](salehcard-alfa-debug-handoff.md) — RESOLVED two bridge bugs: (1) "Alfa balance always 0" = balance-reply parser only knew Touch's `*220#` shape not Alfa's `*11#`; (2) "Touch transfer works but shows FAILED" = SMS sent-ack receiver was `RECEIVER_NOT_EXPORTED` (never fires on Android 13+) → aborts before reading operator reply. Both Android-APK-side; need `./gradlew installDebug` on prod bridge phone, no API deploy. Not committed.
- [SalehCard Flutter APK prod URL](salehcard-flutter-apk-prod-url.md) — customer APK defaults to a hardcoded dev LAN IP → hangs on infinite loading on real phones; always build release APKs with --dart-define=API_BASE_URL=https://salehcard-api.azurewebsites.net/api/v1
- [SalehCard Whish payments](salehcard-whish-payments.md) — Whish Pay (redirect+callback) code-complete & stub-verified (2026-07-06), not committed; BLOCKED on receiving Whish test/sandbox API creds; resume via HANDOFF-WHISH-IMPL.md
- [SalehCard Whish handoff](salehcard-whish-handoff.md) — SUPERSEDED by salehcard-whish-payments (implementation done); original planning note pinned the spec PDF + LACPA blueprint + redirect-vs-watcher decision
- [SalehCard USDT on-chain payments](salehcard-usdt-onchain-payments.md) — TRC20 auto-confirming payments (2026-07-06): new payment module + tron platform + watcher + Flutter/admin UI; gated on USDT_XPUB; 4 PRs code-complete, verified stub-mode, not committed
- [SalehCard mobile bridge](salehcard-mobile-bridge.md) — CONFIRMED direction: bridge automates Touch/Alfa recharge from owner's SIMs; code imported to /bridge + BRIDGE-PLAN.md shipped on branch feat/mobile-bridge (2026-07-06, PR still to open); 6-PR implementation plan, Go bridge module next
- [SalehCard dbtool](salehcard-dbtool.md) — api/cmd/dbtool: reusable guarded CLI for DB inspect/maintenance (product/codes/codes-clear/order); use instead of one-off scripts
- [SalehCard direct-recharge migration](salehcard-direct-recharge-migration.md) — alfa-direct/mtc-direct products were misconfigured (no phone box); fixed to bridge recharge_line via cmd/bridgedirect on 2026-07-08; follow-up: need PINs uploaded + bridge online
- [SalehCard prod DB direct access](salehcard-prod-db-direct-access.md) — how to run a script directly against prod Cosmos vCore from the dev box: add a firewall rule for the drifting 213.204.66.x egress IP (out of auto mode), feed MONGO_URI from the creds file, delete the rule after

- [SalehCard GTM PR stack](salehcard-gtm-pr-stack.md) — go-to-market hardening PRs #15→#20 (2026-07-02): stacked merge order, wallet-only + KYC-gate decisions, launch checklist
- [SalehCard Flutter mobile](salehcard-flutter-mobile.md) — NEW direction (2026-06-24): Flutter mobile app on existing Go API replaces the /web storefront
- [SalehCard Flutter emulator run](salehcard-flutter-emulator-run.md) — exact recipe to run /app on the emulator vs local API (emulator.exe direct + adb reverse + dart-define 127.0.0.1); startup-lock & 0-byte-output gotchas
- [SalehCard design port](salehcard-design-port.md) — /web ported from Claude Design prototype; CSS-var design system, catalog wired to API, rest mocked
- [SalehCard dev env](salehcard-dev-env.md) — run the stack: pnpm at ~/.local/bin, API on :8090, reuse existing mongo :27017
- [SalehCard admin port](salehcard-admin-port.md) — /admin console (Vite :5174); products+inventory+dashboard wired, rest stubbed; AdminOnly dev-bypass
- [SalehCard prod deploy](salehcard-prod-deploy.md) — LIVE on Azure; CI/CD auto-deploys on push to main (OIDC managed identity); DB on Cosmos vCore (Atlas retiring); read DEPLOYMENT.md + .github/CICD-SETUP.md; corporate-proxy TLS gotcha; secrets in DEPLOY-CREDS.local.md
- [SalehCard az TLS/Norton](salehcard-az-tls-norton.md) — `az` CLI fails TLS because Norton "Web/Mail Shield" MITMs HTTPS (not the proxy); OpenSSL rejects it, bundle fix won't work; use Azure Portal browser / Cloud Shell instead
- [SalehCard phone-OTP auth](salehcard-phone-otp-auth.md) — phone-OTP login shipped (Go+Flutter); SMS via hexagonal adapters (Monty/Twilio/log), pick with SMS_PROVIDER
- [SalehCard BUSINESS_TZ todo](salehcard-business-tz-todo.md) — deferred prod config: set BUSINESS_TZ=Asia/Beirut app setting so dashboard "today" buckets on Beirut not UTC
- [SalehCard ID-check verify](salehcard-idcheck-verify.md) — game-ID nickname verification (PR #47); deferred prod config: set IDCHECK_PROVIDER=rapidapi + RAPIDAPI_KEY (rotate the chat-exposed key first)
