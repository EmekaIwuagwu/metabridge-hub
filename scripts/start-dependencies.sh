#!/bin/bash

# Start all required dependencies for Articium

set -e

echo "🚀 Starting Articium Dependencies..."
echo ""

# Start PostgreSQL
echo "1️⃣  Starting PostgreSQL..."

# Try to find PostgreSQL service
PG_SERVICE=""
for service in postgresql postgresql-14 postgresql-15 postgresql-16 postgresql@14-main postgresql@15-main; do
    if systemctl list-unit-files | grep -q "^${service}.service"; then
        PG_SERVICE=$service
        break
    fi
done

if [ -z "$PG_SERVICE" ]; then
    echo "   ✗ PostgreSQL service not found!"
    echo "   → Please install PostgreSQL first:"
    echo "     sudo bash scripts/install-dependencies.sh"
    exit 1
elif systemctl is-active --quiet "$PG_SERVICE"; then
    echo "   ✓ PostgreSQL already running ($PG_SERVICE)"
else
    sudo systemctl start "$PG_SERVICE"
    sleep 2
    echo "   ✓ PostgreSQL started ($PG_SERVICE)"
fi

# Start Redis
echo ""
echo "2️⃣  Starting Redis..."
if systemctl is-active --quiet redis; then
    echo "   ✓ Redis already running"
elif systemctl is-active --quiet redis-server; then
    echo "   ✓ Redis already running"
else
    if systemctl list-unit-files | grep -q redis.service; then
        sudo systemctl start redis
    elif systemctl list-unit-files | grep -q redis-server.service; then
        sudo systemctl start redis-server
    else
        echo "   ✗ Redis service not found. Please install Redis:"
        echo "     sudo apt-get install redis-server"
        exit 1
    fi
    sleep 2
    echo "   ✓ Redis started"
fi

# Start NATS (if installed as systemd service)
echo ""
echo "3️⃣  Starting NATS..."
if pgrep -x "nats-server" > /dev/null; then
    echo "   ✓ NATS already running"
elif systemctl is-active --quiet nats; then
    echo "   ✓ NATS already running"
else
    # Check if NATS is installed
    if command -v nats-server &> /dev/null; then
        # Start NATS in background
        nohup nats-server -js > /root/projects/articium/logs/nats.log 2>&1 &
        sleep 2
        echo "   ✓ NATS started (running in background)"
    else
        echo "   ⚠ NATS not found. Installing..."
        cd /tmp
        wget -q https://github.com/nats-io/nats-server/releases/download/v2.10.7/nats-server-v2.10.7-linux-amd64.tar.gz
        tar -xzf nats-server-v2.10.7-linux-amd64.tar.gz
        sudo mv nats-server-v2.10.7-linux-amd64/nats-server /usr/local/bin/
        rm -rf nats-server-v2.10.7-linux-amd64*

        # Start NATS
        nohup nats-server -js > /root/projects/articium/logs/nats.log 2>&1 &
        sleep 2
        echo "   ✓ NATS installed and started"
    fi
fi

echo ""
echo "✅ All dependencies are running!"
echo ""
echo "Next steps:"
echo "  1. Run the dependency checker: bash scripts/check-dependencies.sh"
echo "  2. Start the services:"
echo "     sudo systemctl start articium-api"
echo "     sudo systemctl start articium-relayer"
echo "     sudo systemctl start articium-listener"
echo "     sudo systemctl start articium-batcher"
echo ""
