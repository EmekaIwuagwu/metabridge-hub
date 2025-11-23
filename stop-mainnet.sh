#!/bin/bash
set -e

##############################################################################
# Articium - Mainnet Service Shutdown Script
#
# Gracefully stops all Articium mainnet services with automatic backup
##############################################################################

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
NC='\033[0m'

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_DIR="$PROJECT_ROOT/data"
BACKUP_DIR="$PROJECT_ROOT/backups"
LOG_DIR="$PROJECT_ROOT/logs"

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

echo ""
echo -e "${RED}======================================================================${NC}"
echo -e "${RED}  Stopping Articium Mainnet${NC}"
echo -e "${RED}======================================================================${NC}"
echo ""
echo -e "${YELLOW}⚠️  This will stop all mainnet bridge services${NC}"
echo ""
echo "Type 'STOP MAINNET' to confirm (anything else to cancel):"
read -r confirmation

if [ "$confirmation" != "STOP MAINNET" ]; then
    log_info "Shutdown cancelled"
    exit 0
fi

# Create automatic backup before stopping
log_info "Creating automatic backup before shutdown..."
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_PATH="$BACKUP_DIR/pre-shutdown-$TIMESTAMP"

mkdir -p "$BACKUP_PATH"

# Backup database if running
if docker ps | grep -q articium-postgres; then
    log_info "Backing up database..."
    docker exec articium-postgres pg_dump -U articium articium_mainnet > "$BACKUP_PATH/database.sql" 2>/dev/null || true
    if [ -f "$BACKUP_PATH/database.sql" ]; then
        log_success "Database backed up"
    else
        log_warning "Database backup failed"
    fi
fi

# Backup configuration
if [ -f "$PROJECT_ROOT/config/config.mainnet.yaml" ]; then
    cp "$PROJECT_ROOT/config/config.mainnet.yaml" "$BACKUP_PATH/"
    log_success "Configuration backed up"
fi

# Backup recent logs
if [ -d "$LOG_DIR" ]; then
    cp -r "$LOG_DIR" "$BACKUP_PATH/" 2>/dev/null || true
    log_success "Logs backed up"
fi

log_success "Backup completed: $BACKUP_PATH"

# Stop backend services gracefully
log_info "Stopping backend services gracefully..."

# Function to gracefully stop a service
stop_service() {
    local service_name=$1
    local pid_file="$DATA_DIR/${service_name}.pid"

    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p $pid > /dev/null 2>&1; then
            log_info "Stopping $service_name (PID: $pid)..."

            # Send SIGTERM for graceful shutdown
            kill -TERM $pid 2>/dev/null || true

            # Wait up to 30 seconds for graceful shutdown
            for i in {1..30}; do
                if ! ps -p $pid > /dev/null 2>&1; then
                    log_success "$service_name stopped gracefully"
                    rm "$pid_file"
                    return 0
                fi
                sleep 1
            done

            # Force kill if still running
            log_warning "$service_name did not stop gracefully, forcing..."
            kill -KILL $pid 2>/dev/null || true
            rm "$pid_file"
            log_success "$service_name stopped (forced)"
        else
            log_warning "$service_name not running (stale PID file)"
            rm "$pid_file"
        fi
    else
        log_info "$service_name not running (no PID file)"
    fi
}

# Stop services in reverse order of startup
stop_service "relayer"
stop_service "listener"
stop_service "api"

# Wait a moment for any cleanup
sleep 2

# Stop infrastructure
log_info "Stopping infrastructure services..."
cd "$PROJECT_ROOT/deployments/docker"

if [ -f "docker-compose.infrastructure.yaml" ]; then
    # Gracefully stop containers
    docker-compose -f docker-compose.infrastructure.yaml stop -t 30

    # Remove containers but keep volumes (data safety)
    docker-compose -f docker-compose.infrastructure.yaml down --remove-orphans

    log_success "Infrastructure stopped"
else
    log_warning "docker-compose.infrastructure.yaml not found"
fi

# Show summary
echo ""
echo -e "${GREEN}======================================================================${NC}"
echo -e "${GREEN}  All Mainnet Services Stopped${NC}"
echo -e "${GREEN}======================================================================${NC}"
echo ""
echo "📊 Summary:"
echo "  • All backend services stopped"
echo "  • Infrastructure containers stopped"
echo "  • Data volumes preserved"
echo "  • Backup created: $BACKUP_PATH"
echo ""
echo "📝 Service Status:"
for service in api listener relayer; do
    if [ -f "$DATA_DIR/${service}.pid" ]; then
        echo "  ✗ $service: PID file exists (unexpected)"
    else
        echo "  ✓ $service: stopped"
    fi
done
echo ""
echo "💾 Data Preservation:"
echo "  • Database data: preserved in Docker volume"
echo "  • Redis data: preserved in Docker volume"
echo "  • NATS data: preserved in Docker volume"
echo "  • Logs: $LOG_DIR"
echo ""
echo "🔄 To restart services:"
echo "  ./deploy-mainnet.sh"
echo ""
echo "📦 To completely remove all data (⚠️  DANGEROUS):"
echo "  docker-compose -f deployments/docker/docker-compose.infrastructure.yaml down -v"
echo ""
echo -e "${YELLOW}⚠️  Remember: Mainnet data is valuable. Backups in: $BACKUP_DIR${NC}"
echo ""
