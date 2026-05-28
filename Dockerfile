# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o router-sync main.go

# Final stage — host network + NET_ADMIN; needs ip/conntrack on the host namespace
FROM alpine:3.20

RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    iproute2 \
    conntrack-tools \
    wget

WORKDIR /app

COPY --from=builder /app/router-sync .

# Default config is overridden by Ansible bind-mount at /etc/router-sync/config.yaml
COPY config.yaml /etc/router-sync/config.yaml

EXPOSE 18080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:18080/health || exit 1

ENTRYPOINT ["./router-sync"]
CMD ["-config", "/etc/router-sync/config.yaml"]
