# SalehCard

Digital-goods e-commerce platform — gaming gift cards, top-ups, and transfers.

## Stack
- **Storefront** (`/web`): React 18 + TypeScript + Vite + Tailwind CSS (RTL, dark mode)
- **Admin console** (`/admin`): React 18 + TypeScript + Vite — sidebar shell, dense data tables, charts
- **Backend** (`/api`): Go 1.22 + chi + MongoDB
- **Auth**: JWT (access token in memory, refresh token in httpOnly cookie); admin routes gated by an `admin` role claim
- **i18n**: English (default), Arabic (RTL), Turkish

## Prerequisites
- Go 1.22+
- Node.js 20+ and pnpm 9+
- Docker + Docker Compose
- Make (optional but recommended)

## Quick Start
```bash
# 1. Start MongoDB
make up

# 2. Seed sample products (first time only)
make seed

# 3. Start backend (terminal 1)
cd api && cp .env.example .env && go mod tidy && go run ./cmd/server

# 4. Start storefront (terminal 2)
cd web && cp .env.example .env && pnpm install && pnpm dev

# 5. Start admin console (terminal 3)
cd admin && cp .env.example .env && pnpm install && pnpm dev
```

- Storefront → http://localhost:5173 (catalog fetches products from the Go API)
- Admin console → http://localhost:5174 (sign in with any credentials in dev — see note below)

> Make sure `web/.env` and `admin/.env` point `VITE_API_BASE_URL` at the API
> (e.g. `http://localhost:8090` if you run the API on 8090).

### Admin auth in development
Admin routes (`/api/admin/*`) require an `admin` role on the JWT. When the API runs
**without** a `JWT_SECRET` (the default local setup), the `AdminOnly` middleware
**bypasses** auth so the wired product/inventory pages work end-to-end, and the admin
app's login screen accepts any credentials (mock admin session). Set `JWT_SECRET` to
enforce real token validation. Wiring a real login/token-issuing endpoint is a TODO.

## Project Structure
```
/api       Go backend (chi router, MongoDB, feature modules)
/web       React storefront (Vite, Tailwind, TanStack Query, Zustand)
/admin     React admin console (Vite — sidebar shell, dense tables, charts)
/deploy    Docker Compose + env examples
```

## Commands
| Command | Description |
|---------|-------------|
| `make up` | Start MongoDB via Docker |
| `make down` | Stop MongoDB |
| `make seed` | Insert sample products |
| `make dev-admin` | Start the admin console (port 5174) |
| `make build-admin` | Build the admin console |
| `make lint-admin` | Lint the admin console |
| `make test` | Run all tests (api + web + admin) |
| `make lint` | Run all linters |

## Architecture
See [CONVENTIONS.md](./CONVENTIONS.md) for folder conventions, module patterns, and how to add new features.
