# Memory Index

- [SalehCard design port](salehcard-design-port.md) — /web ported from Claude Design prototype; CSS-var design system, catalog wired to API, rest mocked
- [SalehCard dev env](salehcard-dev-env.md) — run the stack: pnpm at ~/.local/bin, API on :8090, reuse existing mongo :27017
- [SalehCard admin port](salehcard-admin-port.md) — /admin console (Vite :5174); products+inventory(+upload history)+dashboard(fully real)+settings admin-users wired; orders/resellers/finance/promos/reviews still mock; AdminOnly dev-bypass
- [SalehCard prod deploy](salehcard-prod-deploy.md) — LIVE on Azure (2026-06-22); read DEPLOYMENT.md; corporate-proxy TLS gotcha; secrets in gitignored DEPLOY-CREDS.local.md
