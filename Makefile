.PHONY: dev dev-server dev-client build test lint docker-build docker-up setup clean

# ── Development ──────────────────────────────────────────────────────────────
dev:
	@bash scripts/dev.sh

dev-server:
	@go run ./cmd/server

dev-client:
	@cd client && npm run dev

# ── Build ─────────────────────────────────────────────────────────────────────
build: build-client build-server

build-server:
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/server ./cmd/server
	@echo "✓ server binary at bin/server"

build-client:
	@cd client && npm run build
	@echo "✓ client assets at client/dist"

# ── Testing ──────────────────────────────────────────────────────────────────
test: test-go test-e2e

test-go:
	@go test ./... -v -count=1

test-e2e:
	@cd test && pip install -q -r requirements.txt && pytest -v

# ── Code quality ─────────────────────────────────────────────────────────────
lint:
	@go vet ./...
	@cd client && npm run type-check

# ── Docker ───────────────────────────────────────────────────────────────────
docker-build:
	@docker compose build

docker-up:
	@docker compose up --build

docker-down:
	@docker compose down

# ── Setup ────────────────────────────────────────────────────────────────────
setup:
	@bash scripts/setup.sh

# ── Clean ────────────────────────────────────────────────────────────────────
clean:
	@rm -rf bin/ client/dist/ data/
