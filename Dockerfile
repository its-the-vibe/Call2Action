# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.26.2-alpine AS builder

WORKDIR /src

# Download dependencies first (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a fully static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" \
    -o /call2action ./cmd/call2action

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM scratch

# Copy CA certificates so TLS connections work if needed
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the static binary
COPY --from=builder /call2action /call2action

ENTRYPOINT ["/call2action"]
