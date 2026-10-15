# SalehCard / FlashCash Global

Digital-goods e-commerce platform — gaming gift cards, game credit, mobile recharge, money
transfers. The customer app ships as **FlashCash Global** on Android and iOS.

Live: **https://www.flashcashglobal.com** · admin **https://admin.flashcashglobal.com** ·
API `https://salehcard-api.azurewebsites.net/api/v1` (Azure; CI/CD deploys on push to `main`).

## Stack
- **Mobile app** (`/app`): Flutter (Riverpod, go_router) — the customer-facing client.
- **Backend** (`/api`): Go 1.22 + chi + MongoDB (prod: Azure Cosmos DB for MongoDB vCore).
- **Admin console** (`/admin`): React 18 + TypeScript + Vite.
- **Bridge** (`/bridge`): Android app that runs Touch/Alfa recharges from owned SIMs.
- **Storefront** (`/web`): React 18 + Vite — legacy, being retired.
- **Auth**: JWT access token + rotating refresh token (httpOnly cookie on web, in-body for
  mobile); phone-OTP login; RBAC on the admin console.

## Prerequisites
- Go 1.22+, Node.js 20+ and pnpm 9+, Flutter (see `BUILD-MACHINE.md` for the Android toolchain)
- Docker + Docker Compose (for local MongoDB), Make (optional)

## Quick start
```bash
make up                                   # MongoDB on :27017
cd api && go run ./cmd/seed               # seed products + dev users (idempotent)
cd api && PORT=8090 go run ./cmd/server   # API (use 8090 if 8080 is taken)
cd admin && pnpm install && pnpm dev      # admin console → http://localhost:5174
cd web && pnpm install && pnpm dev        # legacy storefront → http://localhost:5173
```

Copy each `.env.example` to `.env` first, and point `VITE_API_BASE_URL` at the API
(e.g. `http://localhost:8090`). Dev accounts (password `password123`):
`customer@salehcard.local`, `admin@salehcard.local` — local only, not in prod.

Release builds of the app must pass the prod API URL, or the app hangs on a dev LAN IP:
```bash
cd app && flutter build appbundle --dart-define=API_BASE_URL=https://salehcard-api.azurewebsites.net/api/v1
```

## Docs
- `CLAUDE.md` — orientation, architecture rules, where everything lives. Start here.
- `CONVENTIONS.md` — module layout and design-system rules.
- `DEPLOYMENT.md` — Azure resources and redeploys.
- `BACKLOG.md`, `DEVOPS-TODO.md`, `QA-TODO.md` — open product, ops, and QA work.
- Store releases: `HANDOFF-2026-08-07-PLAYSTORE.md`, `HANDOFF-2026-10-14-IOS-APPSTORE.md`.
