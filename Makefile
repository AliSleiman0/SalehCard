.PHONY: up down seed dev dev-admin build-admin lint-admin test lint

up:
	docker compose -f deploy/docker-compose.yml up -d mongodb

down:
	docker compose -f deploy/docker-compose.yml down

seed:
	cd api && go run ./cmd/seed

dev:
	@echo "Run in separate terminals:"
	@echo "  Backend:        cd api && go run ./cmd/server"
	@echo "  Storefront:     cd web && pnpm dev      # http://localhost:5173"
	@echo "  Admin console:  cd admin && pnpm dev    # http://localhost:5174"

# Admin dashboard (separate Vite + React app on port 5174)
dev-admin:
	cd admin && pnpm dev

build-admin:
	cd admin && pnpm build

lint-admin:
	cd admin && pnpm lint

test:
	cd api && go test ./... -v
	cd web && pnpm test
	cd admin && pnpm test

lint:
	cd api && golangci-lint run ./...
	cd web && pnpm lint
	cd admin && pnpm lint
