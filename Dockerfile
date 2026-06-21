# Builds the frontend; vite's outDir (web/vite.config.ts) writes straight
# into ../internal/webui/dist so the backend stage below can embed it.
FROM node:22-alpine AS frontend
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Builds the single Go binary with the real frontend build embedded
# (internal/webui/embed.go's go:embed picks up whatever is in dist/ at
# build time — the backend repo source only ships a placeholder there).
FROM golang:1.26-alpine AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY --from=frontend /src/internal/webui/dist ./internal/webui/dist
RUN CGO_ENABLED=0 go build -o /out/amethyst ./cmd/amethyst

# modernc.org/sqlite is pure Go, so the binary above is fully static and
# needs nothing but CA certs (for outbound HTTPS calls to the Telegram API).
FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend /out/amethyst ./amethyst

# VAULT_PATH must point at a mounted Obsidian vault; INDEX_PATH's directory
# must be writable (see docker-compose.yml for the expected volume layout).
ENV LISTEN_ADDR=:8080 \
    INDEX_PATH=/data/index.db
EXPOSE 8080
VOLUME ["/vault", "/data"]
ENTRYPOINT ["./amethyst"]
