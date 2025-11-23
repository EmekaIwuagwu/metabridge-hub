#!/bin/bash

echo "╔════════════════════════════════════════════════════════════╗"
echo "║          Articium Service Diagnostics                   ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# Function to check service
check_service() {
    local service=$1
    local name=$2

    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🔍 $name"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    if systemctl is-active $service > /dev/null 2>&1; then
        echo "✅ Status: RUNNING"
    else
        echo "❌ Status: FAILED"
    fi

    echo ""
    echo "Last 30 log lines:"
    echo "---"
    sudo journalctl -u $service -n 30 --no-pager | tail -30
    echo ""
}

# Check all services
check_service "articium-api" "API Server"
check_service "articium-relayer" "Relayer Service"
check_service "articium-listener" "Listener Service"
check_service "articium-batcher" "Batcher Service"

# Test API health
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔍 API Health Check"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Testing http://localhost:8080/health"
curl -s http://localhost:8080/health | jq . 2>/dev/null || curl -s http://localhost:8080/health
echo ""
echo ""

# Check dependencies
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔍 Dependencies Status"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# PostgreSQL
if systemctl is-active postgresql@16-main > /dev/null 2>&1; then
    echo "✅ PostgreSQL: RUNNING"
    PGPASSWORD=articium psql -h /var/run/postgresql -p 5433 -U articium -d articium_prod -c "SELECT COUNT(*) as message_count FROM messages;" 2>&1 | grep -v "^$"
else
    echo "❌ PostgreSQL: NOT RUNNING"
fi
echo ""

# Redis
if systemctl is-active redis-server > /dev/null 2>&1; then
    echo "✅ Redis: RUNNING"
    redis-cli ping 2>&1
else
    echo "❌ Redis: NOT RUNNING"
fi
echo ""

# NATS
if systemctl is-active nats > /dev/null 2>&1; then
    echo "✅ NATS: RUNNING"
else
    echo "❌ NATS: NOT RUNNING"
fi
echo ""

echo "╔════════════════════════════════════════════════════════════╗"
echo "║                  Diagnostics Complete                      ║"
echo "╚════════════════════════════════════════════════════════════╝"
