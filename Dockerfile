# Build stage - Frontend
FROM node:20-slim AS frontend-builder

WORKDIR /app/frontend

COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install

COPY frontend/ .
RUN npm run build

# Build stage - Go backend
FROM golang:1.24-bookworm AS backend-builder

WORKDIR /app

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend into the static directory
COPY --from=frontend-builder /app/static /opt/opanel/static

# Build the binary
RUN CGO_ENABLED=0 go build -o /opt/opanel/bin/opaneld ./cmd/opaneld

# Runtime stage
FROM debian:trixie-slim

RUN apt-get update && apt-get install -y \
    ca-certificates \
    nginx \
    php-fpm \
    mariadb-server \
    && rm -rf /var/lib/apt/lists/*

# Create required directories
RUN mkdir -p /run/php /var/run/mysqld /var/www/vhosts /etc/nginx/sites-enabled /var/log/php8.4-fpm

# Set permissions for MariaDB runtime
RUN chown mysql:mysql /var/run/mysqld

COPY --from=backend-builder /opt/opanel/bin/opaneld /opt/opanel/bin/opaneld
RUN mkdir -p /opt/opanel/db /opt/opanel/templates /opt/opanel/static /etc/opanel

COPY --from=backend-builder /opt/opanel/static /opt/opanel/static
COPY --from=backend-builder /app/templates /opt/opanel/templates
COPY config.example.yaml /etc/opanel/config.yaml

EXPOSE 8443

# Start script that launches all services
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

CMD ["/entrypoint.sh"]
