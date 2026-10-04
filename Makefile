SHELL := /bin/bash
.DEFAULT_GOAL := help

BIN   := bin/homebase
EMBED := server/internal/webui/dist

# Shown in Settings, About and by `homebase version`. A v* tag gives
# "v1.2.0"; commits after it "v1.2.0-3-gabc1234"; no tag yet, the commit.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)

.PHONY: help dev build test lint fmt ci clean

help:
	@echo "make dev    Go server on 127.0.0.1:8080 and Vite on :5173, API proxied"
	@echo "make build  web app, then $(BIN) with the web app embedded"
	@echo "make test   lint, go vet, go test, web type check, web unit tests"
	@echo "make lint   gofmt check and Biome"
	@echo "make fmt    rewrite Go and web files in place"
	@echo "make ci     clean dependency install, then test and build"
	@echo "make clean  remove build output"

web/node_modules: web/package.json web/package-lock.json
	cd web && npm ci
	@touch $@

# Both processes share the shell's process group, so Ctrl-C stops both.
# A .env file (never committed) can hold settings such as HOMEBASE_DATA.
dev: web/node_modules
	@set -a; [ -f .env ] && . ./.env; set +a; \
	trap 'trap - INT TERM EXIT; kill 0' INT TERM EXIT; \
	(cd server && \
		HOMEBASE_DATA="$${HOMEBASE_DATA:-$(CURDIR)/data}" \
		HOMEBASE_ADDR="$${HOMEBASE_ADDR:-127.0.0.1:8080}" \
		go run ./cmd/homebase serve) & \
	(cd web && npm run dev) & \
	wait

build: web/node_modules
	cd web && VITE_VERSION=$(VERSION) VITE_COMMIT=$(COMMIT) npm run build
	find $(EMBED) -mindepth 1 ! -name .gitkeep -delete
	cp -R web/dist/. $(EMBED)/
	cd server && CGO_ENABLED=0 go build -trimpath \
		-ldflags='-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)' -o ../$(BIN) ./cmd/homebase
	@echo "built $(BIN)"

test: lint
	cd server && go vet ./...
	cd server && go test ./...
	cd web && npm run typecheck
	cd web && npm test

lint: web/node_modules
	@out="$$(gofmt -l server)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi
	cd web && npm run lint

fmt: web/node_modules
	gofmt -w server
	cd web && npm run format

ci:
	cd server && go mod verify
	cd web && npm ci
	$(MAKE) test build

clean:
	rm -rf bin web/dist
	find $(EMBED) -mindepth 1 ! -name .gitkeep -delete
