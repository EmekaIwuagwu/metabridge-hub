#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Project root
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_ROOT"

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}   Articium Hub - Startup Script${NC}"
echo -e "${BLUE}======================================${NC}\n"

# Function to check if a service is running
check_service() {
    if pgrep -f "$1" > /dev/null; then
        echo -e "${GREEN}✓${NC} $2 is running"
        return 0
    else
        echo -e "${RED}✗${NC} $2 is not running"
        return 1
    fi
}

# Function to wait for service
wait_for_service() {
    local url=$1
    local name=$2
    local max_attempts=30
    local attempt=1

    echo -e "${YELLOW}⏳ Waiting for $name to start...${NC}"

    while [ $attempt -le $max_attempts ]; do
        if curl -s "$url" > /dev/null 2>&1; then
            echo -e "${GREEN}✓${NC} $name is ready!"
            return 0
        fi
        sleep 1
        attempt=$((attempt + 1))
    done

    echo -e "${RED}✗${NC} $name failed to start"
    return 1
}

# Stop function
stop_all() {
    echo -e "\n${YELLOW}Stopping all services...${NC}\n"

    # Stop frontend
    pkill -f "vite" 2>/dev/null && echo -e "${GREEN}✓${NC} Frontend stopped" || echo -e "${YELLOW}ℹ${NC} Frontend was not running"

    # Stop API
    pkill -f "bin/api" 2>/dev/null && echo -e "${GREEN}✓${NC} API stopped" || echo -e "${YELLOW}ℹ${NC} API was not running"

    echo -e "\n${GREEN}All services stopped${NC}"
}

# Check if user wants to stop
if [ "$1" == "stop" ]; then
    stop_all
    exit 0
fi

# Check if user wants to restart
if [ "$1" == "restart" ]; then
    stop_all
    sleep 2
    echo -e "\n${BLUE}Restarting services...${NC}\n"
fi

# Check if user wants to see status
if [ "$1" == "status" ]; then
    echo -e "${BLUE}Service Status:${NC}\n"
    check_service "postgresql" "PostgreSQL"
    check_service "bin/api" "API Server"
    check_service "vite" "Frontend"
    echo -e "\n${BLUE}Ports:${NC}"
    echo -e "  API:      http://localhost:8080"
    echo -e "  Frontend: http://localhost:3000"
    exit 0
fi

echo -e "${BLUE}1. Checking PostgreSQL...${NC}"
if ! pg_isready -p 5433 > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠${NC} PostgreSQL is not running, starting..."
    service postgresql start
    sleep 2
    if pg_isready -p 5433 > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} PostgreSQL started"
    else
        echo -e "${RED}✗${NC} Failed to start PostgreSQL"
        exit 1
    fi
else
    echo -e "${GREEN}✓${NC} PostgreSQL is running"
fi

echo -e "\n${BLUE}2. Checking Database Setup...${NC}"
if ! PGPASSWORD=bridge_password psql -U bridge_user -d articium_testnet -h localhost -p 5433 -c "SELECT 1" > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠${NC} Setting up database..."
    psql -U postgres -p 5433 -c "CREATE DATABASE articium_testnet;" 2>/dev/null || echo "Database exists"
    psql -U postgres -p 5433 -c "CREATE USER bridge_user WITH PASSWORD 'bridge_password';" 2>/dev/null || echo "User exists"
    psql -U postgres -p 5433 -c "GRANT ALL PRIVILEGES ON DATABASE articium_testnet TO bridge_user;" 2>/dev/null
    psql -U postgres -p 5433 -c "ALTER USER bridge_user CREATEDB;" 2>/dev/null
    echo -e "${GREEN}✓${NC} Database configured"
else
    echo -e "${GREEN}✓${NC} Database is ready"
fi

echo -e "\n${BLUE}3. Starting Backend API...${NC}"

# Kill old API if running
pkill -f "bin/api" 2>/dev/null && echo -e "${YELLOW}ℹ${NC} Killed old API process"

# Check if binary exists
if [ ! -f "$PROJECT_ROOT/bin/api" ]; then
    echo -e "${YELLOW}⚠${NC} API binary not found, building..."
    make build
fi

# Start API
export DB_PASSWORD=bridge_password
export REQUIRE_AUTH=false
export BRIDGE_ENVIRONMENT=testnet

nohup "$PROJECT_ROOT/bin/api" -config "$PROJECT_ROOT/config/config.testnet.yaml" > /tmp/articium-api.log 2>&1 &
API_PID=$!

# Wait for API to start
if wait_for_service "http://localhost:8080/health" "API"; then
    echo -e "${GREEN}✓${NC} API Server running (PID: $API_PID)"
    echo -e "   Logs: tail -f /tmp/articium-api.log"
else
    echo -e "${RED}✗${NC} API failed to start. Check logs:"
    echo -e "   ${YELLOW}tail -50 /tmp/articium-api.log${NC}"
    exit 1
fi

echo -e "\n${BLUE}4. Starting Frontend...${NC}"

# Kill old frontend if running
pkill -f "vite" 2>/dev/null && echo -e "${YELLOW}ℹ${NC} Killed old frontend process"

# Check if node_modules exists
if [ ! -d "$PROJECT_ROOT/frontend/node_modules" ]; then
    echo -e "${YELLOW}⚠${NC} Installing frontend dependencies..."
    cd "$PROJECT_ROOT/frontend"
    npm install
fi

# Start frontend
cd "$PROJECT_ROOT/frontend"
nohup npm run dev > /tmp/articium-frontend.log 2>&1 &
FRONTEND_PID=$!

# Wait for frontend to start
if wait_for_service "http://localhost:3000" "Frontend"; then
    echo -e "${GREEN}✓${NC} Frontend running (PID: $FRONTEND_PID)"
    echo -e "   Logs: tail -f /tmp/articium-frontend.log"
else
    echo -e "${RED}✗${NC} Frontend failed to start. Check logs:"
    echo -e "   ${YELLOW}tail -50 /tmp/articium-frontend.log${NC}"
    exit 1
fi

echo -e "\n${GREEN}======================================${NC}"
echo -e "${GREEN}   All Services Started Successfully!${NC}"
echo -e "${GREEN}======================================${NC}\n"

echo -e "${BLUE}Service URLs:${NC}"
echo -e "  🌐 Frontend:  ${GREEN}http://localhost:3000${NC}"
echo -e "  🔧 API:       ${GREEN}http://localhost:8080${NC}"
echo -e "  📊 Health:    ${GREEN}http://localhost:8080/health${NC}"

echo -e "\n${BLUE}Useful Commands:${NC}"
echo -e "  Stop all:     ${YELLOW}./start-all.sh stop${NC}"
echo -e "  Restart all:  ${YELLOW}./start-all.sh restart${NC}"
echo -e "  Check status: ${YELLOW}./start-all.sh status${NC}"
echo -e "  API logs:     ${YELLOW}tail -f /tmp/articium-api.log${NC}"
echo -e "  Frontend logs:${YELLOW}tail -f /tmp/articium-frontend.log${NC}"

echo -e "\n${BLUE}Process IDs:${NC}"
echo -e "  API:      $API_PID"
echo -e "  Frontend: $FRONTEND_PID"

echo -e "\n${GREEN}🚀 Ready for testing!${NC}\n"
