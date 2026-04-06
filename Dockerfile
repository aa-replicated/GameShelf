# ---- Build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Download dependencies (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and assets
COPY . .

# Compile static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gameshelf ./cmd/gameshelf

# ---- Runtime stage ----
FROM alpine:latest

# Install CA certificates for HTTPS (if ever needed) and timezone data
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /gameshelf /app/gameshelf

# All templates/static/migrations are embedded in the binary — no extra COPY needed

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/app/gameshelf"]
