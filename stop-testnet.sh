#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_DIR="$PROJECT_ROOT/data"

log_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

echo ""
echo "======================================================================"
echo "  Stopping Metabridge Hub Testnet"
echo "======================================================================"
echo ""

# Stop backend services
log_info "Stopping backend services..."

if [ -f "$DATA_DIR/api.pid" ]; then
    kill $(cat "$DATA_DIR/api.pid") 2>/dev/null || true
    rm "$DATA_DIR/api.pid"
    log_success "API service stopped"
fi

if [ -f "$DATA_DIR/listener.pid" ]; then
    kill $(cat "$DATA_DIR/listener.pid") 2>/dev/null || true
    rm "$DATA_DIR/listener.pid"
    log_success "Listener service stopped"
fi

if [ -f "$DATA_DIR/relayer.pid" ]; then
    kill $(cat "$DATA_DIR/relayer.pid") 2>/dev/null || true
    rm "$DATA_DIR/relayer.pid"
    log_success "Relayer service stopped"
fi

# Stop infrastructure
log_info "Stopping infrastructure services..."
cd "$PROJECT_ROOT/deployments/docker"
docker-compose -f docker-compose.infrastructure.yaml down

log_success "All services stopped"

echo ""
echo "To start again, run: ./deploy-testnet.sh"
echo ""
