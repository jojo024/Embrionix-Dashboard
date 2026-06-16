# ── Stage 1: Build frontend ───────────────────────────────────────────────────
FROM node:22-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ── Stage 2: Build backend ────────────────────────────────────────────────────
FROM golang:1.24-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o embrionix-dashboard \
    ./cmd/server/

# ── Stage 3: Runtime ──────────────────────────────────────────────────────────
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=backend  /app/embrionix-dashboard ./
COPY --from=frontend /app/web/dist            ./web/dist/
COPY configs/config.yaml                       ./configs/

RUN mkdir -p /app/data /app/logs && \
    addgroup -S embrionix && adduser -S embrionix -G embrionix && \
    chown -R embrionix:embrionix /app

USER embrionix

EXPOSE 8080
VOLUME ["/app/data", "/app/logs"]

ENTRYPOINT ["./embrionix-dashboard"]
CMD ["configs/config.yaml"]
