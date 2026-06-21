BINARY := bin/amethyst

.PHONY: build build-frontend build-backend test test-go test-web lint lint-go lint-web run clean docker-build

# Builds the frontend first so it's embedded in the binary (internal/webui) —
# see internal/webui/embed.go and web/vite.config.ts's build.outDir.
build: build-frontend build-backend

build-frontend:
	cd web && npm ci && npm run build

build-backend:
	go build -o $(BINARY) ./cmd/amethyst

test: test-go test-web

test-go:
	go test ./...

test-web:
	cd web && npm test

lint: lint-go lint-web

lint-go:
	go vet ./...

lint-web:
	cd web && npm run lint

run: build
	./$(BINARY)

clean:
	rm -rf bin

docker-build:
	docker build -t amethyst .
