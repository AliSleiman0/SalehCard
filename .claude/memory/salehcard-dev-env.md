---
name: salehcard-dev-env
description: How to run the SalehCard stack (web + api + mongo) on this machine — non-obvious ports and pnpm location
metadata: 
  node_type: memory
  type: project
  originSessionId: f6b0f2fb-ced9-4236-b5c3-ff82ebdd1a8a
---

Running the SalehCard monorepo at `/home/alis/salehcard` locally (not a git repo):

- **pnpm** is not on PATH by default. It's installed at `~/.local/bin` (pnpm 9). Prefix commands with `export PATH="$HOME/.local/bin:$PATH"`. (Corepack can't symlink into `/usr/bin` here; `npm i -g --prefix ~/.local pnpm@9` was used.)
- **Ports 8080 and 27017 are already taken** on this box (other projects' containers: constructiq_mongo, allwaytaxi). So:
  - Run the Go API on **8090**: `cd api && MONGO_URI="mongodb://localhost:27017" DB_NAME=salehcard PORT=8090 ALLOWED_ORIGINS="http://localhost:5173" go run ./cmd/server`
  - Reuse the **existing MongoDB on :27017** (databases are isolated; we use db `salehcard`). `make up`/docker-compose fails on the 27017 bind — don't bother, just use the running mongo.
  - Seed (idempotent): `cd api && MONGO_URI=... DB_NAME=salehcard go run ./cmd/seed` (inserts 3 products: Steam/PUBG/Bank Transfer).
- **web/.env** must point at the API: `VITE_API_BASE_URL=http://localhost:8090`. Then `cd web && pnpm dev` → http://localhost:5173.
- **`/admin`** is a second Vite app (admin console) on **:5174**: `admin/.env` → `VITE_API_BASE_URL=http://localhost:8090`, then `cd admin && pnpm dev` (or `make dev-admin`). See [[salehcard-admin-port]].
- **Admin auth dev-bypass:** `/api/admin/*` is guarded by `AdminOnly` (JWT role=admin). When `JWT_SECRET` is empty it BYPASSES auth (logs a one-time warning); when set it enforces 401/403. **Note `api/.env` ships a JWT_SECRET**, so to exercise admin endpoints by curl in dev, restart the server with `JWT_SECRET=` exported empty (godotenv won't override an already-set env var), or mint an HS256 admin token. The admin app's login screen is a mock (any creds) pending a real auth endpoint.
- Verify gates: `pnpm exec tsc -b`, `pnpm lint` (2 harmless react-refresh fast-refresh warnings expected in Toast.tsx + SavedIdCard.tsx), `pnpm build`, `pnpm test`.

Frontend design system is CSS-variable based (`src/styles/`), not Tailwind `dark:` — see [[salehcard-design-port]] and CONVENTIONS.md.
