# SalehCard

Digital-goods e-commerce platform — gaming gift cards, top-ups, and transfers.

## Stack
- **Frontend**: React 18 + TypeScript + Vite + Tailwind CSS (RTL, dark mode)
- **Backend**: Go 1.22 + chi + MongoDB
- **Auth**: JWT (access token in memory, refresh token in httpOnly cookie)
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

# 4. Start frontend (terminal 2)
cd web && cp .env.example .env && pnpm install && pnpm dev
```

Open http://localhost:5173 — the catalog page fetches products from the Go API.

## Project Structure
```
/api       Go backend (chi router, MongoDB, feature modules)
/web       React frontend (Vite, Tailwind, TanStack Query, Zustand)
/deploy    Docker Compose + env examples
```

## Commands
| Command | Description |
|---------|-------------|
| `make up` | Start MongoDB via Docker |
| `make down` | Stop MongoDB |
| `make seed` | Insert sample products |
| `make test` | Run all tests |
| `make lint` | Run all linters |

## Architecture
See [CONVENTIONS.md](./CONVENTIONS.md) for folder conventions, module patterns, and how to add new features.
