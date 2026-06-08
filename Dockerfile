# Build stage
FROM golang:1.23-bookworm AS builder

WORKDIR /app

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 go build -o /opt/opanel/bin/opaneld ./cmd/opaneld

# Runtime stage
FROM debian:trixie-slim

RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /opt/opanel/bin/opaneld /opt/opanel/bin/opaneld

RUN mkdir -p /opt/opanel/db /etc/opanel

COPY config.example.yaml /etc/opanel/config.yaml

EXPOSE 8443

CMD ["/opt/opanel/bin/opaneld", "server", "--config", "/etc/opanel/config.yaml"]
