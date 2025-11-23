# Multi-stage Dockerfile for Articium services
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make build-base

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build all binaries
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /build/bin/listener ./cmd/listener
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /build/bin/relayer ./cmd/relayer
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /build/bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /build/bin/migrator ./cmd/migrator

# ============================================================
# Listener Service
# ============================================================
FROM alpine:latest AS listener

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/bin/listener /app/listener

EXPOSE 9090

ENTRYPOINT ["/app/listener"]
CMD ["--config", "/app/config/config.testnet.yaml"]

# ============================================================
# Relayer Service
# ============================================================
FROM alpine:latest AS relayer

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/bin/relayer /app/relayer

EXPOSE 9091

ENTRYPOINT ["/app/relayer"]
CMD ["--config", "/app/config/config.testnet.yaml"]

# ============================================================
# API Service
# ============================================================
FROM alpine:latest AS api

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/bin/api /app/api

EXPOSE 8080
EXPOSE 9092

ENTRYPOINT ["/app/api"]
CMD ["--config", "/app/config/config.testnet.yaml"]

# ============================================================
# Migrator Service
# ============================================================
FROM alpine:latest AS migrator

RUN apk --no-cache add ca-certificates tzdata postgresql-client

WORKDIR /app

COPY --from=builder /build/bin/migrator /app/migrator
COPY internal/database/schema.sql /app/schema.sql

ENTRYPOINT ["/app/migrator"]
CMD ["--config", "/app/config/config.testnet.yaml"]
