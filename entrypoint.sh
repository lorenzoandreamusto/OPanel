#!/bin/bash
set -e

echo "Starting OPanel services..."

# Start MariaDB
echo "Starting MariaDB..."
mkdir -p /var/run/mysqld
chown mysql:mysql /var/run/mysqld
# Initialize MariaDB data directory if needed
if [ ! -d /var/lib/mysql/mysql ]; then
    mysql_install_db --user=mysql --datadir=/var/lib/mysql
fi
mysqld_safe --skip-grant-tables &
sleep 3
# Stop skip-grant-tables and restart properly
mysqladmin shutdown 2>/dev/null || true
sleep 1
mysqld_safe &
sleep 3

# Wait for MariaDB to be ready
for i in $(seq 1 30); do
    if mysqladmin ping --socket=/var/run/mysqld/mysqld.sock 2>/dev/null; then
        echo "MariaDB is ready"
        break
    fi
    sleep 1
done

# Start PHP-FPM
echo "Starting PHP-FPM..."
php-fpm8.4 --nodaemonize &

# Start Nginx
echo "Starting Nginx..."
nginx

# Start OPanel
echo "Starting OPanel..."
exec /opt/opanel/bin/opaneld server --config /etc/opanel/config.yaml
