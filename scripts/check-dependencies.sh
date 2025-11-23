#!/bin/bash

# Articium Dependency Checker
# This script checks if all required services are running

set -e

echo "╔════════════════════════════════════════════════════════════╗"
echo "║        Articium Dependency Checker                       ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check function
check_service() {
    local name=$1
    local host=$2
    local port=$3

    if timeout 2 bash -c "cat < /dev/null > /dev/tcp/$host/$port" 2>/dev/null; then
        echo -e "  ${GREEN}✓${NC} $name is running on $host:$port"
        return 0
    else
        echo -e "  ${RED}✗${NC} $name is NOT accessible on $host:$port"
        return 1
    fi
}

# Check systemd service
check_systemd() {
    local service=$1
    if systemctl is-active --quiet "$service" 2>/dev/null; then
        echo -e "  ${GREEN}✓${NC} $service is running"
        return 0
    else
        echo -e "  ${RED}✗${NC} $service is NOT running"
        return 1
    fi
}

# Track issues
ISSUES=0

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Checking Required Services"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check PostgreSQL
echo ""
echo "1️⃣  PostgreSQL (Database)"
if check_service "PostgreSQL" "localhost" "5432"; then
    psql -U postgres -c "SELECT version();" >/dev/null 2>&1 && \
        echo -e "     ${GREEN}✓${NC} PostgreSQL connection successful" || \
        echo -e "     ${YELLOW}⚠${NC}  Can connect but authentication may be required"
else
    ISSUES=$((ISSUES + 1))
    echo -e "     ${RED}→${NC} PostgreSQL is not running or not installed"

    # Try to detect if it's just not running or not installed
    if command -v psql >/dev/null 2>&1; then
        echo -e "     ${YELLOW}→${NC} PostgreSQL is installed but not running"
        echo -e "     ${YELLOW}→${NC} Try: sudo bash scripts/start-dependencies.sh"
    else
        echo -e "     ${RED}→${NC} PostgreSQL is not installed"
        echo -e "     ${RED}→${NC} Install with: sudo bash scripts/install-dependencies.sh"
    fi
fi

# Check NATS
echo ""
echo "2️⃣  NATS (Message Queue)"
if check_service "NATS" "localhost" "4222"; then
    :
else
    ISSUES=$((ISSUES + 1))
    echo -e "     ${RED}→${NC} Install and start NATS:"
    echo "       wget https://github.com/nats-io/nats-server/releases/download/v2.10.7/nats-server-v2.10.7-linux-amd64.tar.gz"
    echo "       tar -xzf nats-server-v2.10.7-linux-amd64.tar.gz"
    echo "       sudo mv nats-server-v2.10.7-linux-amd64/nats-server /usr/local/bin/"
    echo "       nats-server -js &"
fi

# Check Redis
echo ""
echo "3️⃣  Redis (Cache)"
if check_service "Redis" "localhost" "6379"; then
    redis-cli ping >/dev/null 2>&1 && \
        echo -e "     ${GREEN}✓${NC} Redis responding to PING" || \
        echo -e "     ${YELLOW}⚠${NC}  Redis is listening but may not be responding"
else
    ISSUES=$((ISSUES + 1))
    echo -e "     ${RED}→${NC} Start with: sudo systemctl start redis"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📂 Checking Files and Directories"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check binaries
echo ""
echo "4️⃣  Binary Files"
for binary in api relayer listener batcher migrator; do
    if [ -x "/root/projects/articium/bin/$binary" ]; then
        echo -e "  ${GREEN}✓${NC} bin/$binary exists and is executable"
    else
        echo -e "  ${RED}✗${NC} bin/$binary missing or not executable"
        ISSUES=$((ISSUES + 1))
    fi
done

# Check config
echo ""
echo "5️⃣  Configuration File"
if [ -f "/root/projects/articium/config/config.production.yaml" ]; then
    echo -e "  ${GREEN}✓${NC} config/config.production.yaml exists"
else
    echo -e "  ${RED}✗${NC} config/config.production.yaml missing"
    ISSUES=$((ISSUES + 1))
fi

# Check logs directory
echo ""
echo "6️⃣  Logs Directory"
if [ -d "/root/projects/articium/logs" ]; then
    echo -e "  ${GREEN}✓${NC} logs/ directory exists"
else
    echo -e "  ${YELLOW}⚠${NC}  logs/ directory doesn't exist (will be created)"
    mkdir -p /root/projects/articium/logs
    chmod 755 /root/projects/articium/logs
    echo -e "  ${GREEN}✓${NC} logs/ directory created"
fi

# Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [ $ISSUES -eq 0 ]; then
    echo -e "${GREEN}✅ All dependencies are ready!${NC}"
    echo ""
    echo "You can now start the services:"
    echo "  sudo systemctl start articium-api"
    echo "  sudo systemctl start articium-relayer"
    echo "  sudo systemctl start articium-listener"
    echo "  sudo systemctl start articium-batcher"
else
    echo -e "${RED}❌ Found $ISSUES issue(s) that need to be resolved${NC}"
    echo ""
    echo "Please fix the issues above before starting the services."
fi
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

exit $ISSUES
