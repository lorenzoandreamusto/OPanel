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
    php-mysql \
    mariadb-server \
    openssh-server \
    curl \
    unzip \
    bind9 \
    bind9utils \
    postfix \
    postfix-mysql \
    dovecot-core \
    dovecot-lmtpd \
    dovecot-mysql \
    rspamd \
    openssl \
    && rm -rf /var/lib/apt/lists/*

# Configure SSH
RUN mkdir -p /run/sshd \
    && sed -i 's/#PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config \
    && sed -i 's/#PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config \
    && echo "root:opanel" | chpasswd

# Install WP-CLI
RUN curl -sO https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar \
    && chmod +x wp-cli.phar \
    && mv wp-cli.phar /usr/local/bin/wp

# Create required directories
RUN mkdir -p /run/php /var/run/mysqld /var/www/vhosts /etc/nginx/sites-enabled /var/log/php8.4-fpm \
    /etc/bind/zones /var/vmail /var/lib/rspamd/dkim /run/dovecot \
    /var/log/dovecot /var/log/rspamd

# Remove default nginx site config and install OPanel default catch-all
RUN rm -f /etc/nginx/sites-enabled/default
COPY templates/nginx/default-server.conf /etc/nginx/sites-enabled/00-default.conf

# Set permissions for MariaDB runtime
RUN chown mysql:mysql /var/run/mysqld

# Create vmail user for Dovecot/Postfix (uid=8 is Debian default vmail)
RUN groupadd -g 5000 vmail 2>/dev/null || true \
    && useradd -u 5000 -g vmail -d /var/vmail -s /usr/sbin/nologin vmail 2>/dev/null || true \
    && mkdir -p /var/vmail && chown vmail:vmail /var/vmail && chmod 770 /var/vmail \
    && mkdir -p /var/lib/rspamd/dkim && chown _rspamd:_rspamd /var/lib/rspamd/dkim 2>/dev/null || true

COPY --from=backend-builder /opt/opanel/bin/opaneld /opt/opanel/bin/opaneld
RUN mkdir -p /opt/opanel/db /opt/opanel/templates /opt/opanel/static /etc/opanel

COPY --from=backend-builder /opt/opanel/static /opt/opanel/static
COPY --from=backend-builder /app/templates /opt/opanel/templates
COPY config.example.yaml /etc/opanel/config.yaml

EXPOSE 22 80 443 8443

# Start script that launches all services
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

CMD ["/entrypoint.sh"]
