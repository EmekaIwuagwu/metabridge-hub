#!/bin/bash
set -e

##############################################################################
# Metabridge Hub - Complete Testnet Deployment Script
#
# This script deploys the entire Metabridge infrastructure for testing:
# - Infrastructure (PostgreSQL, NATS, Redis, Monitoring)
# - Database schema and migrations
# - All backend services (API, Listener, Relayer)
#
# Requirements:
# - Docker and Docker Compose installed
# - Go 1.21+ installed
# - 8GB RAM minimum (16GB recommended)
# - 20GB disk space
##############################################################################

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="$PROJECT_ROOT/deployments/docker/docker-compose.testnet.yaml"
CONFIG_FILE="$PROJECT_ROOT/config/config.testnet.yaml"
LOG_DIR="$PROJECT_ROOT/logs"
DATA_DIR="$PROJECT_ROOT/data"

# Functions
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

check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    log_success "Docker found: $(docker --version)"

    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    log_success "Docker Compose found: $(docker-compose --version)"

    # Check Go
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed. Please install Go 1.21+ first."
        exit 1
    fi
    log_success "Go found: $(go version)"

    # Check system resources
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        TOTAL_RAM=$(free -g | awk '/^Mem:/{print $2}')
        if [ "$TOTAL_RAM" -lt 8 ]; then
            log_warning "System has ${TOTAL_RAM}GB RAM. 8GB+ recommended for testing."
        else
            log_success "System has ${TOTAL_RAM}GB RAM"
        fi
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        TOTAL_RAM=$(sysctl -n hw.memsize | awk '{print int($1/1024/1024/1024)}')
        if [ "$TOTAL_RAM" -lt 8 ]; then
            log_warning "System has ${TOTAL_RAM}GB RAM. 8GB+ recommended for testing."
        else
            log_success "System has ${TOTAL_RAM}GB RAM"
        fi
    fi

    # Check disk space
    AVAILABLE_SPACE=$(df -BG "$PROJECT_ROOT" | awk 'NR==2 {print $4}' | sed 's/G//')
    if [ "$AVAILABLE_SPACE" -lt 20 ]; then
        log_warning "Only ${AVAILABLE_SPACE}GB disk space available. 20GB+ recommended."
    else
        log_success "Sufficient disk space: ${AVAILABLE_SPACE}GB available"
    fi
}

create_directories() {
    log_info "Creating required directories..."

    mkdir -p "$LOG_DIR"
    mkdir -p "$DATA_DIR/postgres"
    mkdir -p "$DATA_DIR/redis"
    mkdir -p "$DATA_DIR/nats"
    mkdir -p "$DATA_DIR/prometheus"
    mkdir -p "$DATA_DIR/grafana"

    log_success "Directories created"
}

build_services() {
    log_info "Building Go services..."

    cd "$PROJECT_ROOT"

    # Build API
    log_info "Building API service..."
    go build -o bin/api cmd/api/main.go
    log_success "API built"

    # Build Listener
    log_info "Building Listener service..."
    go build -o bin/listener cmd/listener/main.go
    log_success "Listener built"

    # Build Relayer
    log_info "Building Relayer service..."
    go build -o bin/relayer cmd/relayer/main.go
    log_success "Relayer built"

    # Build Migrator
    log_info "Building Migrator..."
    go build -o bin/migrator cmd/migrator/main.go
    log_success "Migrator built"
}

start_infrastructure() {
    log_info "Starting infrastructure services..."

    cd "$PROJECT_ROOT/deployments/docker"

    # Start infrastructure
    docker-compose -f docker-compose.infrastructure.yaml up -d

    log_info "Waiting for services to be ready..."
    sleep 10

    # Wait for PostgreSQL
    log_info "Waiting for PostgreSQL..."
    until docker-compose -f docker-compose.infrastructure.yaml exec -T postgres pg_isready -U metabridge > /dev/null 2>&1; do
        echo -n "."
        sleep 2
    done
    log_success "PostgreSQL is ready"

    # Wait for NATS
    log_info "Waiting for NATS..."
    until docker-compose -f docker-compose.infrastructure.yaml exec -T nats nats-server --signal ping > /dev/null 2>&1; do
        echo -n "."
        sleep 2
    done
    log_success "NATS is ready"

    # Wait for Redis
    log_info "Waiting for Redis..."
    until docker-compose -f docker-compose.infrastructure.yaml exec -T redis redis-cli ping > /dev/null 2>&1; do
        echo -n "."
        sleep 2
    done
    log_success "Redis is ready"

    log_success "All infrastructure services are running"
}

run_migrations() {
    log_info "Running database migrations..."

    cd "$PROJECT_ROOT"

    # Run SQL schema
    docker exec metabridge-postgres psql -U metabridge -d metabridge_testnet -f /docker-entrypoint-initdb.d/schema.sql

    log_success "Database migrations completed"
}

start_services() {
    log_info "Starting Metabridge services..."

    cd "$PROJECT_ROOT"

    # Set environment
    export BRIDGE_ENVIRONMENT=testnet
    export DATABASE_HOST=localhost
    export DATABASE_PORT=5432
    export DATABASE_USER=metabridge
    export DATABASE_PASSWORD=metabridge_password
    export DATABASE_NAME=metabridge_testnet
    export NATS_URL=nats://localhost:4222
    export REDIS_URL=redis://localhost:6379

    # Start API in background
    log_info "Starting API service..."
    nohup ./bin/api --config "$CONFIG_FILE" > "$LOG_DIR/api.log" 2>&1 &
    echo $! > "$DATA_DIR/api.pid"
    log_success "API service started (PID: $(cat $DATA_DIR/api.pid))"

    # Start Listener in background
    log_info "Starting Listener service..."
    nohup ./bin/listener --config "$CONFIG_FILE" > "$LOG_DIR/listener.log" 2>&1 &
    echo $! > "$DATA_DIR/listener.pid"
    log_success "Listener service started (PID: $(cat $DATA_DIR/listener.pid))"

    # Start Relayer in background
    log_info "Starting Relayer service..."
    nohup ./bin/relayer --config "$CONFIG_FILE" > "$LOG_DIR/relayer.log" 2>&1 &
    echo $! > "$DATA_DIR/relayer.pid"
    log_success "Relayer service started (PID: $(cat $DATA_DIR/relayer.pid))"

    # Wait for services to start
    sleep 5
}

verify_deployment() {
    log_info "Verifying deployment..."

    # Check API health
    log_info "Checking API health..."
    if curl -f http://localhost:8080/health > /dev/null 2>&1; then
        log_success "API is healthy"
    else
        log_error "API health check failed"
        return 1
    fi

    # Check database connection
    log_info "Checking database connection..."
    if docker exec metabridge-postgres psql -U metabridge -d metabridge_testnet -c "SELECT 1;" > /dev/null 2>&1; then
        log_success "Database is accessible"
    else
        log_error "Database connection failed"
        return 1
    fi

    # Check NATS
    log_info "Checking NATS connection..."
    if docker exec metabridge-nats nats-server --signal ping > /dev/null 2>&1; then
        log_success "NATS is running"
    else
        log_error "NATS check failed"
        return 1
    fi

    log_success "All health checks passed!"
}

show_status() {
    echo ""
    echo "======================================================================"
    echo -e "${GREEN}Metabridge Hub Testnet - Deployment Complete!${NC}"
    echo "======================================================================"
    echo ""
    echo "📊 Service URLs:"
    echo "  API:        http://localhost:8080"
    echo "  Prometheus: http://localhost:9090"
    echo "  Grafana:    http://localhost:3000 (admin/admin)"
    echo ""
    echo "📝 Logs:"
    echo "  API:      tail -f $LOG_DIR/api.log"
    echo "  Listener: tail -f $LOG_DIR/listener.log"
    echo "  Relayer:  tail -f $LOG_DIR/relayer.log"
    echo ""
    echo "🔧 Quick Commands:"
    echo "  Health:   curl http://localhost:8080/health"
    echo "  Chains:   curl http://localhost:8080/v1/chains"
    echo "  Status:   curl http://localhost:8080/v1/status"
    echo ""
    echo "🛑 To stop all services:"
    echo "  ./stop-testnet.sh"
    echo ""
    echo "📖 Documentation:"
    echo "  Deployment:  docs/runbooks/DEPLOYMENT.md"
    echo "  Monitoring:  docs/runbooks/MONITORING.md"
    echo "  Emergency:   docs/runbooks/EMERGENCY_PROCEDURES.md"
    echo ""
    echo "======================================================================"
}

cleanup_on_error() {
    log_error "Deployment failed. Cleaning up..."

    # Stop services
    if [ -f "$DATA_DIR/api.pid" ]; then
        kill $(cat "$DATA_DIR/api.pid") 2>/dev/null || true
    fi
    if [ -f "$DATA_DIR/listener.pid" ]; then
        kill $(cat "$DATA_DIR/listener.pid") 2>/dev/null || true
    fi
    if [ -f "$DATA_DIR/relayer.pid" ]; then
        kill $(cat "$DATA_DIR/relayer.pid") 2>/dev/null || true
    fi

    # Stop Docker containers
    cd "$PROJECT_ROOT/deployments/docker"
    docker-compose -f docker-compose.infrastructure.yaml down

    exit 1
}

# Main execution
main() {
    echo ""
    echo "======================================================================"
    echo "  Metabridge Hub - Testnet Deployment"
    echo "======================================================================"
    echo ""

    # Set error handler
    trap cleanup_on_error ERR

    # Execute deployment steps
    check_prerequisites
    create_directories
    build_services
    start_infrastructure
    run_migrations
    start_services
    verify_deployment
    show_status

    log_success "Deployment completed successfully!"
}

# Run main function
main "$@"
