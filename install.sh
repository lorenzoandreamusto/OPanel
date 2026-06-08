#!/bin/bash
set -e

# =============================================================================
# OPanel Installer
# Installs OPanel Control Panel on Debian/Ubuntu
# =============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

PANEL_USER="opanel"
PANEL_DIR="/opt/opanel"
CONFIG_DIR="/etc/opanel"
DB_DIR="/opt/opanel/db"
TEMPLATES_DIR="/opt/opanel/templates"
BIN_DIR="/opt/opanel/bin"
BACKUP_DIR="/opt/opanel/backups"
SSL_DIR="/opt/opanel/ssl"
EXT_DIR="/opt/opanel/extensions"
VHOSTS_DIR="/var/www/vhosts"
SERVICE_NAME="opanel"
ADMIN_PORT=8443
HAS_SYSTEMD=false

# Check if systemd is available
check_systemd() {
    if [ -d /run/systemd/system ] || pidof systemd >/dev/null 2>&1; then
        HAS_SYSTEMD=true
        log_info "Systemd detected"
    else
        log_warn "Systemd not detected (Docker/container environment). Skipping service management."
    fi
}

print_banner() {
    echo -e "${CYAN}"
    echo "  ____                    ____  "
    echo " / __ \                  |  _ \ "
    echo "| |  | |_ __   ___ _ __ | |_) |"
    echo "| |  | | '_ \ / _ \ '_ \|  _ < "
    echo "| |__| | |_) |  __/ | | | |_) |"
    echo " \____/| .__/ \___|_| |_|____/ "
    echo "       | |                     "
    echo "       |_|  Control Panel      "
    echo -e "${NC}"
}

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Detect OS
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        OS_VERSION=$VERSION_ID
        log_info "Detected OS: $PRETTY_NAME"
    else
        log_error "Cannot detect OS. /etc/os-release not found."
        exit 1
    fi

    case $OS in
        debian|ubuntu)
            log_info "Supported OS detected"
            ;;
        *)
            log_error "Unsupported OS: $OS. Only Debian and Ubuntu are supported."
            exit 1
            ;;
    esac
}

# Check hardware requirements
check_hardware() {
    log_info "Checking hardware requirements..."

    # Check RAM (minimum 512MB)
    TOTAL_RAM=$(free -m | awk '/^Mem:/{print $2}')
    if [ "$TOTAL_RAM" -lt 512 ]; then
        log_error "Insufficient RAM: ${TOTAL_RAM}MB (minimum: 512MB)"
        exit 1
    fi
    log_info "RAM: ${TOTAL_RAM}MB OK"

    # Check disk space (minimum 1GB free in /)
    FREE_DISK=$(df -BG / | awk 'NR==2{print $4}' | sed 's/G//')
    if [ "$FREE_DISK" -lt 1 ]; then
        log_error "Insufficient disk space: ${FREE_DISK}GB free (minimum: 1GB)"
        exit 1
    fi
    log_info "Disk: ${FREE_DISK}GB free OK"
}

# Update system and install dependencies
install_dependencies() {
    log_info "Updating package lists..."
    apt-get update -qq

    log_info "Installing system dependencies..."
    apt-get install -y -qq \
        nginx \
        php-fpm \
        mariadb-server \
        postfix \
        dovecot-imapd \
        rspamd \
        bind9 \
        ufw \
        fail2ban \
        openssh-server \
        git \
        tar \
        wget \
        curl \
        sudo \
        ca-certificates \
        gnupg \
        lsb-release \
        apt-transport-https \
        build-essential

    log_info "System dependencies installed"
}

# Install Go (for building from source)
install_go() {
    if command -v go &>/dev/null; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        log_info "Go already installed: $GO_VERSION"
        return
    fi

    log_info "Installing Go..."
    GO_VERSION="1.23.6"
    wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
    tar -C /usr/local -xzf /tmp/go.tar.gz
    rm /tmp/go.tar.gz
    
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile.d/golang.sh
    export PATH=$PATH:/usr/local/go/bin
    
    log_info "Go ${GO_VERSION} installed"
}

# Create system user and group
create_system_user() {
    log_info "Creating system user and group..."

    # Create opanel_users group
    if ! getent group opanel_users >/dev/null 2>&1; then
        groupadd opanel_users
        log_info "Created group: opanel_users"
    fi

    log_info "System user setup complete"
}

# Create directory structure
create_directories() {
    log_info "Creating directory structure..."

    mkdir -p "$PANEL_DIR"
    mkdir -p "$BIN_DIR"
    mkdir -p "$DB_DIR"
    mkdir -p "$TEMPLATES_DIR"
    mkdir -p "$BACKUP_DIR"
    mkdir -p "$SSL_DIR"
    mkdir -p "$EXT_DIR"
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$VHOSTS_DIR"

    log_info "Directory structure created at $PANEL_DIR"
}

# Build opaneld from source
build_opaneld() {
    log_info "Building opaneld..."

    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    
    if [ ! -f "$SCRIPT_DIR/go.mod" ]; then
        log_error "go.mod not found. Run this script from the OPanel project directory."
        exit 1
    fi

    cd "$SCRIPT_DIR"
    
    export PATH=$PATH:/usr/local/go/bin
    export CGO_ENABLED=0
    
    go build -ldflags "-X main.version=1.0.0" -o "$BIN_DIR/opaneld" ./cmd/opaneld
    
    chmod +x "$BIN_DIR/opaneld"
    
    log_info "opaneld built successfully at $BIN_DIR/opaneld"
}

# Copy templates
copy_templates() {
    log_info "Copying templates..."
    
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    
    if [ -d "$SCRIPT_DIR/templates" ]; then
        cp -r "$SCRIPT_DIR/templates/"* "$TEMPLATES_DIR/"
        log_info "Templates copied to $TEMPLATES_DIR"
    else
        log_warn "Templates directory not found, skipping"
    fi
}

# Generate admin password
generate_password() {
    ADMIN_PASSWORD=$(openssl rand -base64 16 | tr -d '=/+' | head -c 20)
    log_info "Generated admin password"
}

# Create config file
create_config() {
    log_info "Creating configuration file..."

    cat > "$CONFIG_DIR/config.yaml" << EOF
# OPanel Configuration
# Generated by install.sh on $(date -Iseconds)

server:
  host: "0.0.0.0"
  port: $ADMIN_PORT

database:
  path: "$DB_DIR/opanel.db"

jwt:
  secret: "$(openssl rand -hex 32)"
  expiry_hours: 24
  refresh_days: 30

admin:
  username: "admin"
  password: "$ADMIN_PASSWORD"
  email: "admin@localhost"

paths:
  vhosts_dir: "$VHOSTS_DIR"
  templates_dir: "$TEMPLATES_DIR"
  sshd_config: "/etc/ssh/sshd_config"
  nginx_conf_dir: "/etc/nginx/sites-enabled"

system:
  opanel_group: "opanel_users"
  php_version: "8.2"
EOF

    chmod 600 "$CONFIG_DIR/config.yaml"
    log_info "Configuration created at $CONFIG_DIR/config.yaml"
}

# Create systemd service
create_service() {
    if [ "$HAS_SYSTEMD" = false ]; then
        log_warn "Skipping systemd service creation (no systemd)"
        return
    fi

    log_info "Creating systemd service..."

    cat > /etc/systemd/system/${SERVICE_NAME}.service << EOF
[Unit]
Description=OPanel Control Panel Daemon
After=network.target mysql.service mariadb.service

[Service]
Type=simple
ExecStart=$BIN_DIR/opaneld server --config $CONFIG_DIR/config.yaml
Restart=on-failure
RestartSec=5
User=root
WorkingDirectory=$PANEL_DIR

# Security hardening
NoNewPrivileges=false
ProtectSystem=false
ProtectHome=false

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable ${SERVICE_NAME} >/dev/null 2>&1

    log_info "Systemd service created and enabled"
}

# Configure firewall
configure_firewall() {
    if [ "$HAS_SYSTEMD" = false ]; then
        log_warn "Skipping UFW configuration (no systemd)"
        return
    fi

    log_info "Configuring UFW firewall..."

    # Reset UFW to defaults
    ufw --force reset >/dev/null 2>&1

    # Set default policies
    ufw default deny incoming >/dev/null 2>&1
    ufw default allow outgoing >/dev/null 2>&1

    # Allow SSH
    ufw allow 22/tcp >/dev/null 2>&1

    # Allow HTTP/HTTPS
    ufw allow 80/tcp >/dev/null 2>&1
    ufw allow 443/tcp >/dev/null 2>&1

    # Allow OPanel
    ufw allow $ADMIN_PORT/tcp >/dev/null 2>&1

    # Allow mail ports
    ufw allow 25/tcp >/dev/null 2>&1
    ufw allow 143/tcp >/dev/null 2>&1
    ufw allow 465/tcp >/dev/null 2>&1
    ufw allow 587/tcp >/dev/null 2>&1
    ufw allow 993/tcp >/dev/null 2>&1

    # Enable UFW
    ufw --force enable >/dev/null 2>&1

    log_info "UFW configured: SSH(22), HTTP(80), HTTPS(443), OPanel($ADMIN_PORT), Mail(25,143,465,587,993)"
}

# Start OPanel
start_opanel() {
    if [ "$HAS_SYSTEMD" = false ]; then
        log_info "OPanel binary installed at $BIN_DIR/opaneld"
        log_info "Start manually: $BIN_DIR/opaneld server --config $CONFIG_DIR/config.yaml"
        return
    fi

    log_info "Starting OPanel..."
    systemctl start ${SERVICE_NAME}

    # Wait for service to be ready
    sleep 2

    if systemctl is-active --quiet ${SERVICE_NAME}; then
        log_info "OPanel is running"
    else
        log_warn "OPanel may not have started correctly. Check: systemctl status $SERVICE_NAME"
    fi
}

# Print summary
print_summary() {
    SERVER_IP=$(hostname -I | awk '{print $1}')
    if [ -z "$SERVER_IP" ]; then
        SERVER_IP="localhost"
    fi

    echo ""
    echo -e "${CYAN}============================================${NC}"
    echo -e "${GREEN}  OPanel Installation Complete!${NC}"
    echo -e "${CYAN}============================================${NC}"
    echo ""
    echo -e "  URL:      ${YELLOW}https://${SERVER_IP}:${ADMIN_PORT}${NC}"
    echo -e "  Username: ${YELLOW}admin${NC}"
    echo -e "  Password: ${YELLOW}${ADMIN_PASSWORD}${NC}"
    echo ""
    echo -e "  Config:   $CONFIG_DIR/config.yaml"
    echo -e "  Binary:   $BIN_DIR/opaneld"
    echo -e "  Database: $DB_DIR/opanel.db"
    if [ "$HAS_SYSTEMD" = true ]; then
        echo -e "  Logs:     journalctl -u $SERVICE_NAME -f"
    else
        echo -e "  Run:      $BIN_DIR/opaneld server --config $CONFIG_DIR/config.yaml"
    fi
    echo ""
    echo -e "${CYAN}============================================${NC}"
    echo ""
    echo -e "${YELLOW}IMPORTANT: Save this password somewhere safe!${NC}"
    echo ""
}

# Main installation flow
main() {
    print_banner
    
    check_root
    detect_os
    check_hardware
    check_systemd
    install_dependencies
    install_go
    create_system_user
    create_directories
    build_opaneld
    copy_templates
    generate_password
    create_config
    create_service
    configure_firewall
    start_opanel
    print_summary
}

main "$@"
