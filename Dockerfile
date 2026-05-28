# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}" \
    -a -installsuffix cgo \
    -o router-sync ./cmd/router-sync

# Final stage — same image runs as either API (no privileges, port 18080) or
# Agent (host network, NET_ADMIN; needs ip + conntrack tools on the host
# namespace). Mode is selected via `--mode={api|agent}` at runtime.
FROM alpine:3.20

RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    iproute2 \
    conntrack-tools \
    wget

WORKDIR /app

COPY --from=builder /app/router-sync .

# Default config (Ansible overrides via bind-mount at /etc/router-sync/config.yaml).
COPY config.yaml /etc/router-sync/config.yaml

# API: 18080, Agent: 18082
EXPOSE 18080 18082

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:18080/health \
        || wget --no-verbose --tries=1 --spider http://127.0.0.1:18082/health \
        || exit 1

ENTRYPOINT ["./router-sync"]
CMD ["-config", "/etc/router-sync/config.yaml"]
