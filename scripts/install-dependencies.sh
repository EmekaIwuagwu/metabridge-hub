#!/bin/bash

# Install all required dependencies for Articium on Ubuntu/Debian

set -e

echo "╔════════════════════════════════════════════════════════════╗"
echo "║        Articium Dependency Installer                     ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "❌ Please run as root: sudo bash scripts/install-dependencies.sh"
    exit 1
fi

# Update package list
echo "📦 Updating package list..."
apt-get update -qq

# Install PostgreSQL
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "1️⃣  Installing PostgreSQL..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if command -v psql &> /dev/null; then
    echo "   ✓ PostgreSQL already installed"
    psql --version
else
    apt-get install -y postgresql postgresql-contrib
    echo "   ✓ PostgreSQL installed"
fi

# Start and enable PostgreSQL
systemctl start postgresql
systemctl enable postgresql
echo "   ✓ PostgreSQL started and enabled"

# Configure PostgreSQL
echo "   🔧 Configuring PostgreSQL..."

# Create database and user
sudo -u postgres psql -c "SELECT 1 FROM pg_database WHERE datname = 'articium_production'" | grep -q 1 || \
    sudo -u postgres psql -c "CREATE DATABASE articium_production;"

sudo -u postgres psql -c "ALTER USER postgres WITH PASSWORD 'postgres_admin_password';"

# Update pg_hba.conf to allow local connections with password
PG_VERSION=$(sudo -u postgres psql -c "SHOW server_version;" -t | cut -d' ' -f1 | cut -d'.' -f1)
PG_HBA="/etc/postgresql/${PG_VERSION}/main/pg_hba.conf"

if [ -f "$PG_HBA" ]; then
    # Backup original
    cp "$PG_HBA" "${PG_HBA}.backup"

    # Update local connections to use md5 (password authentication)
    sed -i 's/local\s*all\s*all\s*peer/local   all             all                                     md5/' "$PG_HBA"
    sed -i 's/host\s*all\s*all\s*127.0.0.1\/32\s*ident/host    all             all             127.0.0.1\/32            md5/' "$PG_HBA"

    # Reload PostgreSQL
    systemctl reload postgresql
    echo "   ✓ PostgreSQL configured for password authentication"
fi

# Install Redis
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "2️⃣  Installing Redis..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if command -v redis-cli &> /dev/null; then
    echo "   ✓ Redis already installed"
    redis-cli --version
else
    apt-get install -y redis-server
    echo "   ✓ Redis installed"
fi

# Configure Redis before starting
echo "   🔧 Configuring Redis..."

# Ensure Redis directories exist with correct permissions
mkdir -p /var/lib/redis /var/log/redis
chown -R redis:redis /var/lib/redis /var/log/redis 2>/dev/null || true
chmod 755 /var/lib/redis /var/log/redis

# Fix Redis configuration
if [ -f /etc/redis/redis.conf ]; then
    # Backup original config
    cp /etc/redis/redis.conf /etc/redis/redis.conf.backup

    # Set bind to localhost
    sed -i 's/^bind .*/bind 127.0.0.1/' /etc/redis/redis.conf
    sed -i 's/^# bind 127.0.0.1/bind 127.0.0.1/' /etc/redis/redis.conf

    # Set supervised to systemd
    sed -i 's/^supervised no/supervised systemd/' /etc/redis/redis.conf
    sed -i 's/^supervised auto/supervised systemd/' /etc/redis/redis.conf

    # Disable protected mode for local development
    sed -i 's/^protected-mode yes/protected-mode no/' /etc/redis/redis.conf
fi

# Start and enable Redis
systemctl enable redis-server
systemctl stop redis-server 2>/dev/null || true
sleep 2
systemctl start redis-server

# Check if Redis started successfully
if systemctl is-active --quiet redis-server; then
    echo "   ✓ Redis started and enabled"

    # Test Redis connection
    if redis-cli ping 2>/dev/null | grep -q "PONG"; then
        echo "   ✓ Redis is responding to PING"
    else
        echo "   ⚠️  Redis started but not responding to PING"
    fi
else
    echo "   ⚠️  Redis failed to start automatically"
    echo "   Running Redis troubleshooter..."
    bash scripts/fix-redis.sh
fi

# Install NATS
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "3️⃣  Installing NATS..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if command -v nats-server &> /dev/null; then
    echo "   ✓ NATS already installed"
    nats-server --version
else
    cd /tmp
    wget -q https://github.com/nats-io/nats-server/releases/download/v2.10.7/nats-server-v2.10.7-linux-amd64.tar.gz
    tar -xzf nats-server-v2.10.7-linux-amd64.tar.gz
    mv nats-server-v2.10.7-linux-amd64/nats-server /usr/local/bin/
    rm -rf nats-server-v2.10.7-linux-amd64*
    echo "   ✓ NATS installed"
fi

# Create NATS systemd service
echo "   🔧 Creating NATS systemd service..."
cat > /etc/systemd/system/nats.service << 'NATSEOF'
[Unit]
Description=NATS Server
After=network.target
Documentation=https://docs.nats.io

[Service]
Type=simple
ExecStart=/usr/local/bin/nats-server -js -c /etc/nats/nats-server.conf
Restart=always
RestartSec=5
User=root
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
NATSEOF

# Create NATS config directory and file
mkdir -p /etc/nats
cat > /etc/nats/nats-server.conf << 'CONFEOF'
# NATS Server Configuration
port: 4222
http_port: 8222

# JetStream
jetstream {
  store_dir: /var/lib/nats
  max_memory_store: 1GB
  max_file_store: 10GB
}

# Logging
log_file: "/var/log/nats/nats-server.log"
debug: false
trace: false
CONFEOF

# Create directories
mkdir -p /var/lib/nats
mkdir -p /var/log/nats
chmod 755 /var/lib/nats
chmod 755 /var/log/nats

# Enable and start NATS
systemctl daemon-reload
systemctl enable nats
systemctl start nats
echo "   ✓ NATS service created and started"

# Install other useful tools
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "4️⃣  Installing additional tools..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Install useful utilities
apt-get install -y curl jq net-tools

echo "   ✓ Additional tools installed"

# Create logs directory
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "5️⃣  Creating application directories..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

mkdir -p /root/projects/articium/logs
chmod 755 /root/projects/articium/logs
echo "   ✓ Logs directory created"

# Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Installation Complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Installed services:"
echo "  ✓ PostgreSQL $(psql --version | awk '{print $3}')"
echo "  ✓ Redis $(redis-cli --version | awk '{print $2}')"
echo "  ✓ NATS $(nats-server --version | awk '{print $3}')"
echo ""
echo "Service status:"
systemctl is-active postgresql && echo "  ✓ PostgreSQL: running" || echo "  ✗ PostgreSQL: not running"
systemctl is-active redis-server && echo "  ✓ Redis: running" || echo "  ✗ Redis: not running"
systemctl is-active nats && echo "  ✓ NATS: running" || echo "  ✗ NATS: not running"
echo ""
echo "Next steps:"
echo "  1. Run migrations: ./bin/migrator -config config/config.production.yaml"
echo "  2. Copy service files: sudo cp systemd/*.service /etc/systemd/system/"
echo "  3. Reload systemd: sudo systemctl daemon-reload"
echo "  4. Start services: sudo systemctl start articium-api"
echo ""
echo "Or run the quick test:"
echo "  bash scripts/check-dependencies.sh"
echo ""
