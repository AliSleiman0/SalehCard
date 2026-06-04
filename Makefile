.PHONY: up down seed dev test lint

up:
	docker compose -f deploy/docker-compose.yml up -d mongodb

down:
	docker compose -f deploy/docker-compose.yml down

seed:
	cd api && go run ./cmd/seed

dev:
	@echo "Run in separate terminals:"
	@echo "  Backend:  cd api && go run ./cmd/server"
	@echo "  Frontend: cd web && pnpm dev"

test:
	cd api && go test ./... -v
	cd web && pnpm test

lint:
	cd api && golangci-lint run ./...
	cd web && pnpm lint
