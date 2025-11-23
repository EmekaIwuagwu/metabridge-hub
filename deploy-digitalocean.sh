#!/bin/bash

set -e

# Detect project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "╔════════════════════════════════════════════════════════════╗"
echo "║     Articium DigitalOcean Automated Deployment          ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""
echo "Project Directory: $PROJECT_ROOT"
echo ""
echo "This script will:"
echo "  1. Install all dependencies (PostgreSQL, NATS, Redis)"
echo "  2. Configure PostgreSQL for password authentication"
echo "  3. Create database and user"
echo "  4. Install Go 1.21"
echo "  5. Build all services"
echo "  6. Run database migrations"
echo "  7. Install and start systemd services"
echo "  8. Verify deployment"
echo ""

read -p "Continue with automated deployment? (yes/no): " confirm
if [ "$confirm" != "yes" ]; then
    echo "Deployment cancelled."
    exit 0
fi

echo ""
echo "Starting deployment..."
echo ""

# ==============================================================================
# Step 1: Install PostgreSQL 16
# ==============================================================================

echo "1️⃣  Installing PostgreSQL 16..."

if ! command -v psql &> /dev/null; then
    sudo apt update
    sudo apt install -y postgresql-16 postgresql-contrib-16 postgresql-client
    echo "   ✓ PostgreSQL installed"
else
    echo "   ✓ PostgreSQL already installed"
fi

# Start PostgreSQL cluster
sudo systemctl start postgresql@16-main
sudo systemctl enable postgresql@16-main

# Verify cluster is running
pg_lsclusters | grep "16" | grep "online" || {
    echo "   ❌ PostgreSQL cluster not online"
    exit 1
}

echo "   ✓ PostgreSQL cluster running on port 5433"
echo ""

# ==============================================================================
# Step 2: Install NATS Server
# ==============================================================================

echo "2️⃣  Installing NATS Server..."

if ! command -v nats-server &> /dev/null; then
    cd /tmp
    wget -q https://github.com/nats-io/nats-server/releases/download/v2.10.0/nats-server-v2.10.0-linux-amd64.tar.gz
    tar -xzf nats-server-v2.10.0-linux-amd64.tar.gz
    sudo mv nats-server-v2.10.0-linux-amd64/nats-server /usr/local/bin/
    rm -rf nats-server-*
    echo "   ✓ NATS binary installed"
else
    echo "   ✓ NATS already installed"
fi

# Create NATS config
sudo mkdir -p /etc/nats
sudo mkdir -p /var/lib/nats
sudo chown nobody:nogroup /var/lib/nats

sudo tee /etc/nats/nats-server.conf > /dev/null << 'EOF'
port: 4222

jetstream {
  store_dir: /var/lib/nats
  max_mem: 1G
  max_file: 10G
}

http_port: 8222
EOF

# Create NATS systemd service
sudo tee /etc/systemd/system/nats.service > /dev/null << 'EOF'
[Unit]
Description=NATS Server
Documentation=https://docs.nats.io
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/nats-server -js -c /etc/nats/nats-server.conf
Restart=always
RestartSec=5s
User=nobody
Group=nogroup

[Install]
WantedBy=multi-user.target
EOF

# Start NATS
sudo systemctl daemon-reload
sudo systemctl start nats
sudo systemctl enable nats

# Verify NATS is running
sleep 2
curl -s http://localhost:8222/varz > /dev/null && echo "   ✓ NATS server running" || {
    echo "   ❌ NATS failed to start"
    exit 1
}

cd "$PROJECT_ROOT"
echo ""

# ==============================================================================
# Step 3: Install Redis
# ==============================================================================

echo "3️⃣  Installing Redis..."

if ! command -v redis-cli &> /dev/null; then
    sudo apt install -y redis-server
    echo "   ✓ Redis installed"
else
    echo "   ✓ Redis already installed"
fi

# Configure and start Redis
sudo sed -i 's/^supervised no/supervised systemd/' /etc/redis/redis.conf 2>/dev/null || true
sudo systemctl restart redis-server
sudo systemctl enable redis-server

# Verify Redis
redis-cli ping | grep -q "PONG" && echo "   ✓ Redis server running" || {
    echo "   ❌ Redis failed to start"
    exit 1
}

echo ""

# ==============================================================================
# Step 4: Configure PostgreSQL Authentication
# ==============================================================================

echo "4️⃣  Configuring PostgreSQL authentication..."

# Backup pg_hba.conf
sudo cp /etc/postgresql/16/main/pg_hba.conf /etc/postgresql/16/main/pg_hba.conf.backup

# Update pg_hba.conf for md5 authentication
sudo tee /etc/postgresql/16/main/pg_hba.conf > /dev/null << 'EOF'
# PostgreSQL Client Authentication Configuration File

# TYPE  DATABASE        USER            ADDRESS                 METHOD

# "local" is for Unix domain socket connections only
local   all             postgres                                peer
local   all             all                                     md5

# IPv4 local connections:
host    all             all             127.0.0.1/32            md5

# IPv6 local connections:
host    all             all             ::1/128                 md5

# Allow replication connections from localhost
local   replication     all                                     peer
host    replication     all             127.0.0.1/32            md5
host    replication     all             ::1/128                 md5
EOF

# Restart PostgreSQL
sudo systemctl restart postgresql@16-main
sleep 2

echo "   ✓ PostgreSQL configured for md5 authentication"
echo ""

# ==============================================================================
# Step 5: Create Database and User
# ==============================================================================

echo "5️⃣  Creating database and user..."

# Create articium user and database
sudo -u postgres psql << 'EOF' 2>&1 | grep -v "NOTICE" || true
-- Drop and recreate user
DROP USER IF EXISTS articium;
CREATE USER articium WITH PASSWORD 'articium' SUPERUSER;

-- Drop and recreate database
DROP DATABASE IF EXISTS articium_prod;
CREATE DATABASE articium_prod OWNER articium;

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE articium_prod TO articium;
EOF

echo "   ✓ Database 'articium_prod' created"
echo "   ✓ User 'articium' created with SUPERUSER privileges"

# Grant schema permissions
sudo -u postgres psql -d articium_prod << 'EOF' 2>&1 | grep -v "GRANT" || true
GRANT ALL ON SCHEMA public TO articium;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO articium;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO articium;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO articium;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO articium;
EOF

echo "   ✓ Schema permissions granted"

# Test connection
PGPASSWORD=articium psql -h /var/run/postgresql -p 5433 -U articium -d articium_prod -c "SELECT 1;" > /dev/null && {
    echo "   ✓ Database connection test successful"
} || {
    echo "   ❌ Database connection failed"
    exit 1
}

echo ""

# ==============================================================================
# Step 6: Install Go
# ==============================================================================

echo "6️⃣  Installing Go 1.21..."

if ! command -v go &> /dev/null; then
    GO_VERSION="1.21.5"

    # Download and install Go
    cd /tmp
    wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
    rm go${GO_VERSION}.linux-amd64.tar.gz

    # Add Go to PATH
    echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a /etc/profile
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    export PATH=$PATH:/usr/local/go/bin

    echo "   ✓ Go ${GO_VERSION} installed"
else
    echo "   ✓ Go already installed ($(go version | awk '{print $3}'))"
fi

# Verify Go installation
if ! command -v go &> /dev/null; then
    echo "   ❌ Go installation failed"
    exit 1
fi

echo ""

# ==============================================================================
# Step 7: Build All Services
# ==============================================================================

echo "7️⃣  Building all services..."

cd "$PROJECT_ROOT"
mkdir -p bin

echo "   Building API..."
CGO_ENABLED=0 go build -o bin/api ./cmd/api
echo "   ✓ API built ($(du -h bin/api | awk '{print $1}'))"

echo "   Building Relayer..."
CGO_ENABLED=0 go build -o bin/relayer ./cmd/relayer
echo "   ✓ Relayer built ($(du -h bin/relayer | awk '{print $1}'))"

echo "   Building Listener..."
CGO_ENABLED=0 go build -o bin/listener ./cmd/listener
echo "   ✓ Listener built ($(du -h bin/listener | awk '{print $1}'))"

echo "   Building Batcher..."
CGO_ENABLED=0 go build -o bin/batcher ./cmd/batcher
echo "   ✓ Batcher built ($(du -h bin/batcher | awk '{print $1}'))"

echo "   Building Migrator..."
CGO_ENABLED=0 go build -o bin/migrator ./cmd/migrator
echo "   ✓ Migrator built ($(du -h bin/migrator | awk '{print $1}'))"

echo ""
echo "   Total binary size: $(du -sh bin/ | awk '{print $1}')"
echo ""

# ==============================================================================
# Step 7: Run Database Migrations
# ==============================================================================

echo "8️⃣  Running database migrations..."

./bin/migrator -config config/config.production.yaml 2>&1 | grep -E "(Starting|loaded|established|Schema applied|All database)" || {
    echo "   ❌ Migrations failed"
    echo ""
    echo "Running migrator with full output:"
    ./bin/migrator -config config/config.production.yaml
    exit 1
}

echo "   ✓ All database migrations completed successfully"
echo ""

# Verify tables
TABLE_COUNT=$(PGPASSWORD=articium psql -h /var/run/postgresql -p 5433 -U articium -d articium_prod -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" | tr -d ' ')
echo "   ✓ Created $TABLE_COUNT database tables"
echo ""

# ==============================================================================
# Step 8: Install Systemd Services
# ==============================================================================

echo "9️⃣  Installing systemd services..."

# Copy service files
sudo cp systemd/articium-api.service /etc/systemd/system/
sudo cp systemd/articium-relayer.service /etc/systemd/system/
sudo cp systemd/articium-batcher.service /etc/systemd/system/
sudo cp systemd/articium-listener.service /etc/systemd/system/

# Reload systemd
sudo systemctl daemon-reload

echo "   ✓ Service files installed"
echo ""

# ==============================================================================
# Step 9: Start All Services
# ==============================================================================

echo "🔟 Starting all services..."

# Start services
sudo systemctl start articium-api
sleep 2
echo "   ✓ API service started"

sudo systemctl start articium-relayer
sleep 2
echo "   ✓ Relayer service started"

sudo systemctl start articium-batcher
sleep 2
echo "   ✓ Batcher service started"

sudo systemctl start articium-listener
sleep 2
echo "   ✓ Listener service started"

# Enable services for auto-start
sudo systemctl enable articium-api articium-relayer articium-batcher articium-listener > /dev/null 2>&1

echo "   ✓ All services enabled for auto-start"
echo ""

# ==============================================================================
# Step 10: Verify Deployment
# ==============================================================================

echo "🔟 Verifying deployment..."
echo ""

# Check service status
echo "Service Status:"
sudo systemctl is-active articium-api > /dev/null && echo "  ✅ API: Running" || echo "  ❌ API: Failed"
sudo systemctl is-active articium-relayer > /dev/null && echo "  ✅ Relayer: Running" || echo "  ❌ Relayer: Failed"
sudo systemctl is-active articium-batcher > /dev/null && echo "  ✅ Batcher: Running" || echo "  ❌ Batcher: Failed"
sudo systemctl is-active articium-listener > /dev/null && echo "  ✅ Listener: Running" || echo "  ❌ Listener: Failed"

echo ""

# Test API health endpoint
echo "API Health Check:"
sleep 5
if curl -s http://localhost:8080/health | jq -e '.status == "healthy"' > /dev/null 2>&1; then
    echo "  ✅ API health endpoint responding"
    curl -s http://localhost:8080/health | jq '.'
else
    echo "  ⚠️  API health check failed (may still be starting up)"
    echo "  Check logs: sudo journalctl -u articium-api -n 20"
fi

echo ""

# Check listening ports
echo "Listening Ports:"
ss -tlnp 2>/dev/null | grep -E "(8080|4222|6379|5433)" | while read line; do
    echo "  ✓ $line"
done

echo ""
echo "╔════════════════════════════════════════════════════════════╗"
echo "║             ✅ DEPLOYMENT COMPLETE!                        ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""
echo "Access your Articium API at:"
echo "  http://localhost:8080/health"
echo "  http://localhost:8080/ready"
echo ""
echo "View service logs:"
echo "  sudo journalctl -u articium-api -f"
echo "  sudo journalctl -u articium-listener -f"
echo ""
echo "Check all services:"
echo "  bash check-services.sh"
echo ""
echo "Restart all services:"
echo "  sudo systemctl restart articium-api articium-relayer articium-batcher articium-listener"
echo ""
