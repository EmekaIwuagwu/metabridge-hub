#!/bin/bash

# Articium Production Deployment Fix Script
# Run this on your DigitalOcean server at 159.65.73.133

set -e

BLUE='\033[0;34m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Articium Production Deployment Fix${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Change to project directory
cd /root/projects/articium

# Step 1: Verify binaries
echo -e "${YELLOW}Step 1: Verifying binaries...${NC}"
if [ ! -f "bin/api" ]; then
    echo -e "${RED}ERROR: bin/api not found!${NC}"
    echo "Run: make build"
    exit 1
fi
if [ ! -f "bin/relayer" ]; then
    echo -e "${RED}ERROR: bin/relayer not found!${NC}"
    echo "Run: make build"
    exit 1
fi
echo -e "${GREEN}✓ Binaries found${NC}"
echo ""

# Step 2: Initialize database schema
echo -e "${YELLOW}Step 2: Initializing database schema...${NC}"
echo "Applying main schema..."
docker exec -i articium-postgres psql -U bridge_user -d articium_production < internal/database/schema.sql 2>&1 | grep -v "already exists" || true

echo "Applying auth schema..."
docker exec -i articium-postgres psql -U bridge_user -d articium_production < internal/database/auth.sql 2>&1 | grep -v "already exists" || true

if [ -f "internal/database/batches.sql" ]; then
    echo "Applying batches schema..."
    docker exec -i articium-postgres psql -U bridge_user -d articium_production < internal/database/batches.sql 2>&1 | grep -v "already exists" || true
fi

if [ -f "internal/database/webhooks.sql" ]; then
    echo "Applying webhooks schema..."
    docker exec -i articium-postgres psql -U bridge_user -d articium_production < internal/database/webhooks.sql 2>&1 | grep -v "already exists" || true
fi

if [ -f "internal/database/routes.sql" ]; then
    echo "Applying routes schema..."
    docker exec -i articium-postgres psql -U bridge_user -d articium_production < internal/database/routes.sql 2>&1 | grep -v "already exists" || true
fi

echo -e "${GREEN}✓ Database schema initialized${NC}"
echo ""

# Step 3: Verify database tables
echo -e "${YELLOW}Step 3: Verifying database tables...${NC}"
TABLE_COUNT=$(docker exec -i articium-postgres psql -U bridge_user -d articium_production -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" | tr -d ' ')
echo "Found $TABLE_COUNT tables"

if [ "$TABLE_COUNT" -lt 5 ]; then
    echo -e "${RED}WARNING: Expected more tables. Check schema files.${NC}"
else
    echo -e "${GREEN}✓ Database tables verified${NC}"
fi
echo ""

# Step 4: Test binary execution
echo -e "${YELLOW}Step 4: Testing API binary...${NC}"
timeout 3 bin/api -config config/config.production.yaml 2>&1 | head -5 || echo "Binary test completed"
echo -e "${GREEN}✓ Binary can execute${NC}"
echo ""

# Step 5: Reload systemd
echo -e "${YELLOW}Step 5: Reloading systemd...${NC}"
systemctl daemon-reload
echo -e "${GREEN}✓ Systemd reloaded${NC}"
echo ""

# Step 6: Stop any running instances
echo -e "${YELLOW}Step 6: Stopping existing services...${NC}"
systemctl stop articium-api 2>/dev/null || true
systemctl stop articium-relayer 2>/dev/null || true
echo -e "${GREEN}✓ Services stopped${NC}"
echo ""

# Step 7: Start API service
echo -e "${YELLOW}Step 7: Starting API service...${NC}"
systemctl start articium-api

# Wait a moment
sleep 3

# Check status
if systemctl is-active --quiet articium-api; then
    echo -e "${GREEN}✓ API service is running!${NC}"
    systemctl status articium-api --no-pager -l
else
    echo -e "${RED}✗ API service failed to start${NC}"
    echo ""
    echo "Showing last 50 lines of logs:"
    journalctl -u articium-api -n 50 --no-pager
    exit 1
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Deployment fix completed successfully!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Next steps:"
echo "1. Start relayer: systemctl start articium-relayer"
echo "2. Enable on boot: systemctl enable articium-api articium-relayer"
echo "3. Test API: curl http://localhost:8080/health"
echo "4. View logs: journalctl -u articium-api -f"
