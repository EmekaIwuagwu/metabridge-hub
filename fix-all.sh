#!/bin/bash

set -e

echo "╔════════════════════════════════════════════════════════════╗"
echo "║          FINAL FIX - Articium Setup                      ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# 1. Fix config.production.yaml
echo "1️⃣  Updating configuration..."
sed -i 's|host: "localhost"|host: "/var/run/postgresql"|' config/config.production.yaml
sed -i 's|host: "127.0.0.1"|host: "/var/run/postgresql"|' config/config.production.yaml
sed -i 's|host: ""|host: "/var/run/postgresql"|' config/config.production.yaml
sed -i 's|port: 5432|port: 5433|' config/config.production.yaml
sed -i 's|database: "articium_production"|database: "articium_prod"|' config/config.production.yaml
sed -i 's|username: "postgres"|username: "articium"|' config/config.production.yaml
sed -i 's|password: "postgres_admin_password"|password: "articium"|' config/config.production.yaml

echo "   ✓ Configuration updated"
echo ""

# 2. Verify configuration
echo "2️⃣  Verifying configuration..."
grep "database:" config/config.production.yaml | head -1
grep "host:" config/config.production.yaml | grep -v "0.0.0.0" | head -1
grep "port:" config/config.production.yaml | grep -v "8080" | head -1
echo ""

# 3. Ensure database exists
echo "3️⃣  Ensuring database exists..."
sudo -u postgres psql -c "CREATE DATABASE articium_prod;" 2>/dev/null || echo "   Database articium_prod already exists"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE articium_prod TO articium;" 2>/dev/null || echo "   Permissions already granted"
echo ""

# 4. Run migrations
echo "4️⃣  Running database migrations..."
./bin/migrator -config config/config.production.yaml
echo ""

# 5. Copy and reload systemd services
echo "5️⃣  Installing systemd services..."
sudo cp systemd/*.service /etc/systemd/system/
sudo systemctl daemon-reload
echo "   ✓ Services installed"
echo ""

# 6. Start all services
echo "6️⃣  Starting all services..."
sudo systemctl restart articium-api
sudo systemctl restart articium-relayer
sudo systemctl restart articium-listener
sudo systemctl restart articium-batcher
echo "   ✓ Services started"
echo ""

# 7. Wait for services to start
echo "7️⃣  Waiting for services to initialize..."
sleep 5
echo ""

# 8. Check service status
echo "8️⃣  Checking service status..."
echo ""
echo "API Status:"
systemctl is-active articium-api && echo "  ✅ articium-api: RUNNING" || echo "  ❌ articium-api: FAILED"
echo ""
echo "Relayer Status:"
systemctl is-active articium-relayer && echo "  ✅ articium-relayer: RUNNING" || echo "  ❌ articium-relayer: FAILED"
echo ""
echo "Listener Status:"
systemctl is-active articium-listener && echo "  ✅ articium-listener: RUNNING" || echo "  ❌ articium-listener: FAILED"
echo ""
echo "Batcher Status:"
systemctl is-active articium-batcher && echo "  ✅ articium-batcher: RUNNING" || echo "  ❌ articium-batcher: FAILED"
echo ""

# 9. Test API
echo "9️⃣  Testing API endpoint..."
if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "  ✅ API is responding!"
    echo ""
    echo "API Health Response:"
    curl -s http://localhost:8080/health | jq . 2>/dev/null || curl -s http://localhost:8080/health
else
    echo "  ⚠️  API is not responding yet"
    echo ""
    echo "Check logs with:"
    echo "  sudo journalctl -u articium-api -n 50 --no-pager"
fi

# 10. Check for failed services
echo ""
echo "🔍 Checking for failed services..."
if ! systemctl is-active articium-listener > /dev/null 2>&1; then
    echo ""
    echo "⚠️  Listener service failed. Last 20 log lines:"
    sudo journalctl -u articium-listener -n 20 --no-pager
fi

if ! systemctl is-active articium-batcher > /dev/null 2>&1; then
    echo ""
    echo "⚠️  Batcher service failed. Last 20 log lines:"
    sudo journalctl -u articium-batcher -n 20 --no-pager
fi

echo ""
echo "╔════════════════════════════════════════════════════════════╗"
echo "║                    ✅ SETUP COMPLETE!                      ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""
echo "Useful commands:"
echo "  systemctl status articium-api"
echo "  sudo journalctl -u articium-api -f"
echo "  curl http://localhost:8080/health"
echo ""
echo "Service logs:"
echo "  sudo journalctl -u articium-listener -f"
echo "  sudo journalctl -u articium-batcher -f"
echo ""
