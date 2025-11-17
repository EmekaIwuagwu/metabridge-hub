#!/bin/bash
set -e

##############################################################################
# Metabridge Engine - Production Mainnet Deployment Script
#
# ⚠️  CRITICAL: This script deploys to MAINNET with REAL FUNDS
#
# This script deploys the entire Metabridge infrastructure for production:
# - Infrastructure (PostgreSQL, NATS, Redis, Monitoring)
# - Database schema and migrations
# - All backend services (API, Listener, Relayer)
# - Enhanced security configurations (3-of-5 multisig)
#
# Requirements:
# - Docker and Docker Compose installed
# - Go 1.21+ installed
# - 32GB RAM minimum (64GB recommended)
# - 1TB SSD disk space
# - AWS KMS or HSM for validator key management
# - Security audit completed and verified
# - Multi-signature wallet configured
# - SSL/TLS certificates installed
# - Production monitoring configured
##############################################################################

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="$PROJECT_ROOT/deployments/docker/docker-compose.mainnet.yaml"
CONFIG_FILE="$PROJECT_ROOT/config/config.mainnet.yaml"
LOG_DIR="$PROJECT_ROOT/logs"
DATA_DIR="$PROJECT_ROOT/data"
BACKUP_DIR="$PROJECT_ROOT/backups"
AUDIT_FILE="$PROJECT_ROOT/security-audit.verified"

# Production Security Thresholds
MIN_RAM_GB=32
RECOMMENDED_RAM_GB=64
MIN_DISK_GB=1000
MIN_VALIDATORS=5
REQUIRED_SIGNATURES=3

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

log_critical() {
    echo -e "${MAGENTA}[CRITICAL]${NC} $1"
}

print_banner() {
    echo ""
    echo -e "${RED}======================================================================${NC}"
    echo -e "${RED}  ⚠️  MAINNET DEPLOYMENT - REAL FUNDS AT RISK ⚠️${NC}"
    echo -e "${RED}======================================================================${NC}"
    echo ""
    echo -e "${YELLOW}This script will deploy Metabridge Engine to PRODUCTION MAINNET.${NC}"
    echo -e "${YELLOW}All transactions will use REAL cryptocurrency.${NC}"
    echo ""
    echo -e "${CYAN}Prerequisites Checklist:${NC}"
    echo "  ✓ Security audit completed and signed off"
    echo "  ✓ Multi-signature wallets configured (3-of-5)"
    echo "  ✓ AWS KMS or HSM for validator key management"
    echo "  ✓ SSL/TLS certificates installed and verified"
    echo "  ✓ Monitoring and alerting configured"
    echo "  ✓ Backup and disaster recovery procedures tested"
    echo "  ✓ Team on standby for immediate support"
    echo "  ✓ Insurance or security fund in place"
    echo ""
}

confirm_deployment() {
    echo -e "${RED}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║                    FINAL WARNING                               ║${NC}"
    echo -e "${RED}║                                                                ║${NC}"
    echo -e "${RED}║  You are about to deploy to MAINNET with REAL FUNDS.          ║${NC}"
    echo -e "${RED}║  This action cannot be easily reversed.                       ║${NC}"
    echo -e "${RED}║                                                                ║${NC}"
    echo -e "${RED}║  Have you completed ALL security requirements?                ║${NC}"
    echo -e "${RED}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${YELLOW}Type 'DEPLOY TO MAINNET' to proceed (anything else to cancel):${NC}"
    read -r confirmation

    if [ "$confirmation" != "DEPLOY TO MAINNET" ]; then
        log_error "Deployment cancelled by user"
        exit 1
    fi

    log_warning "Proceeding with mainnet deployment..."
    sleep 3
}

verify_security_audit() {
    log_info "Verifying security audit..."

    if [ ! -f "$AUDIT_FILE" ]; then
        log_critical "Security audit verification file not found!"
        echo ""
        echo "Required: $AUDIT_FILE"
        echo ""
        echo "To create this file, you must:"
        echo "1. Complete a professional security audit"
        echo "2. Address all critical and high-severity findings"
        echo "3. Create the verification file with audit details:"
        echo ""
        echo "cat > $AUDIT_FILE << 'EOF'"
        echo "AUDIT_COMPANY: [Company Name]"
        echo "AUDIT_DATE: [YYYY-MM-DD]"
        echo "AUDIT_REPORT: [Path to full report]"
        echo "CRITICAL_ISSUES: 0"
        echo "HIGH_ISSUES: 0"
        echo "VERIFIED_BY: [Your Name]"
        echo "SIGNATURE: [GPG Signature]"
        echo "EOF"
        echo ""
        exit 1
    fi

    # Verify audit file contents
    if ! grep -q "CRITICAL_ISSUES: 0" "$AUDIT_FILE"; then
        log_critical "Security audit shows unresolved critical issues!"
        cat "$AUDIT_FILE"
        exit 1
    fi

    if ! grep -q "HIGH_ISSUES: 0" "$AUDIT_FILE"; then
        log_critical "Security audit shows unresolved high-severity issues!"
        cat "$AUDIT_FILE"
        exit 1
    fi

    log_success "Security audit verified"
    log_info "Audit details:"
    cat "$AUDIT_FILE" | grep -E "AUDIT_COMPANY|AUDIT_DATE|VERIFIED_BY"
}

check_prerequisites() {
    log_info "Checking production prerequisites..."

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

    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_success "Go found: $GO_VERSION"

    # Check system resources - STRICT for mainnet
    log_info "Checking system resources (production requirements)..."

    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        TOTAL_RAM=$(free -g | awk '/^Mem:/{print $2}')
        if [ "$TOTAL_RAM" -lt "$MIN_RAM_GB" ]; then
            log_critical "Insufficient RAM: ${TOTAL_RAM}GB (minimum ${MIN_RAM_GB}GB required)"
            exit 1
        elif [ "$TOTAL_RAM" -lt "$RECOMMENDED_RAM_GB" ]; then
            log_warning "RAM: ${TOTAL_RAM}GB (${RECOMMENDED_RAM_GB}GB recommended for optimal performance)"
        else
            log_success "RAM: ${TOTAL_RAM}GB"
        fi
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        TOTAL_RAM=$(sysctl -n hw.memsize | awk '{print int($1/1024/1024/1024)}')
        if [ "$TOTAL_RAM" -lt "$MIN_RAM_GB" ]; then
            log_critical "Insufficient RAM: ${TOTAL_RAM}GB (minimum ${MIN_RAM_GB}GB required)"
            exit 1
        elif [ "$TOTAL_RAM" -lt "$RECOMMENDED_RAM_GB" ]; then
            log_warning "RAM: ${TOTAL_RAM}GB (${RECOMMENDED_RAM_GB}GB recommended for optimal performance)"
        else
            log_success "RAM: ${TOTAL_RAM}GB"
        fi
    fi

    # Check disk space - STRICT for mainnet
    AVAILABLE_SPACE=$(df -BG "$PROJECT_ROOT" | awk 'NR==2 {print $4}' | sed 's/G//')
    if [ "$AVAILABLE_SPACE" -lt "$MIN_DISK_GB" ]; then
        log_critical "Insufficient disk space: ${AVAILABLE_SPACE}GB (minimum ${MIN_DISK_GB}GB required)"
        exit 1
    else
        log_success "Disk space: ${AVAILABLE_SPACE}GB available"
    fi

    # Check for SSL certificates
    log_info "Checking SSL/TLS configuration..."
    if [ ! -d "/etc/letsencrypt/live" ] && [ ! -f "$PROJECT_ROOT/certs/tls.crt" ]; then
        log_critical "SSL/TLS certificates not found!"
        echo "Please configure SSL certificates before mainnet deployment."
        echo "See: docs/guides/AZURE_DEPLOYMENT.md#setup-ssl--https"
        exit 1
    fi
    log_success "SSL/TLS certificates configured"

    # Check for required environment variables
    log_info "Checking environment variables..."
    REQUIRED_VARS=(
        "AWS_KMS_KEY_ID"
        "DATABASE_PASSWORD"
        "VALIDATOR_ADDRESSES"
        "MULTI_SIG_WALLET"
    )

    for var in "${REQUIRED_VARS[@]}"; do
        if [ -z "${!var}" ]; then
            log_error "Required environment variable not set: $var"
            echo "Please set all required environment variables in .env.mainnet"
            exit 1
        fi
    done
    log_success "Required environment variables configured"
}

verify_validator_configuration() {
    log_info "Verifying validator configuration..."

    # Check validator addresses from config
    if [ -z "$VALIDATOR_ADDRESSES" ]; then
        log_critical "VALIDATOR_ADDRESSES not set in environment"
        exit 1
    fi

    # Count validators
    VALIDATOR_COUNT=$(echo "$VALIDATOR_ADDRESSES" | tr ',' '\n' | wc -l)

    if [ "$VALIDATOR_COUNT" -lt "$MIN_VALIDATORS" ]; then
        log_critical "Insufficient validators: $VALIDATOR_COUNT (minimum $MIN_VALIDATORS required for 3-of-5 multisig)"
        exit 1
    fi

    log_success "Validator configuration verified: $VALIDATOR_COUNT validators"

    # Verify AWS KMS integration
    if [ -z "$AWS_KMS_KEY_ID" ]; then
        log_critical "AWS_KMS_KEY_ID not configured"
        log_error "Mainnet requires HSM or AWS KMS for validator key management"
        exit 1
    fi

    # Test AWS KMS access
    if command -v aws &> /dev/null; then
        if aws kms describe-key --key-id "$AWS_KMS_KEY_ID" &> /dev/null; then
            log_success "AWS KMS key accessible: $AWS_KMS_KEY_ID"
        else
            log_critical "Cannot access AWS KMS key: $AWS_KMS_KEY_ID"
            exit 1
        fi
    else
        log_warning "AWS CLI not found - skipping KMS verification"
    fi
}

verify_multisig_wallet() {
    log_info "Verifying multi-signature wallet configuration..."

    if [ -z "$MULTI_SIG_WALLET" ]; then
        log_critical "MULTI_SIG_WALLET not configured"
        echo "Mainnet deployment requires a multi-signature wallet for contract ownership"
        exit 1
    fi

    log_success "Multi-sig wallet configured: $MULTI_SIG_WALLET"
    log_info "Ensure this wallet has $REQUIRED_SIGNATURES-of-$MIN_VALIDATORS signature requirement"
}

create_directories() {
    log_info "Creating required directories..."

    mkdir -p "$LOG_DIR"
    mkdir -p "$DATA_DIR/postgres"
    mkdir -p "$DATA_DIR/redis"
    mkdir -p "$DATA_DIR/nats"
    mkdir -p "$DATA_DIR/prometheus"
    mkdir -p "$DATA_DIR/grafana"
    mkdir -p "$BACKUP_DIR"
    mkdir -p "$PROJECT_ROOT/certs"

    # Set strict permissions for production
    chmod 700 "$DATA_DIR"
    chmod 700 "$BACKUP_DIR"
    chmod 700 "$PROJECT_ROOT/certs"

    log_success "Directories created with secure permissions"
}

build_services() {
    log_info "Building Go services with production optimizations..."

    cd "$PROJECT_ROOT"

    # Build with optimizations and stripped debug symbols
    BUILD_FLAGS="-ldflags='-w -s' -trimpath"

    # Build API
    log_info "Building API service..."
    go build $BUILD_FLAGS -o bin/api cmd/api/main.go
    log_success "API built"

    # Build Listener
    log_info "Building Listener service..."
    go build $BUILD_FLAGS -o bin/listener cmd/listener/main.go
    log_success "Listener built"

    # Build Relayer
    log_info "Building Relayer service..."
    go build $BUILD_FLAGS -o bin/relayer cmd/relayer/main.go
    log_success "Relayer built"

    # Build Migrator
    log_info "Building Migrator..."
    go build $BUILD_FLAGS -o bin/migrator cmd/migrator/main.go
    log_success "Migrator built"

    # Verify binaries
    log_info "Verifying binaries..."
    for binary in api listener relayer migrator; do
        if [ ! -f "bin/$binary" ]; then
            log_error "Binary not found: bin/$binary"
            exit 1
        fi
        # Check binary hash
        HASH=$(sha256sum "bin/$binary" | awk '{print $1}')
        log_info "$binary SHA256: $HASH"
    done
    log_success "All binaries verified"
}

backup_existing_data() {
    log_info "Creating backup of existing data..."

    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    BACKUP_PATH="$BACKUP_DIR/pre-deployment-$TIMESTAMP"

    mkdir -p "$BACKUP_PATH"

    # Backup database if exists
    if docker ps | grep -q metabridge-postgres; then
        log_info "Backing up database..."
        docker exec metabridge-postgres pg_dump -U metabridge metabridge_mainnet > "$BACKUP_PATH/database.sql"
        log_success "Database backed up to $BACKUP_PATH/database.sql"
    fi

    # Backup configuration
    if [ -f "$CONFIG_FILE" ]; then
        cp "$CONFIG_FILE" "$BACKUP_PATH/config.mainnet.yaml"
    fi

    log_success "Backup completed: $BACKUP_PATH"
}

start_infrastructure() {
    log_info "Starting production infrastructure services..."

    cd "$PROJECT_ROOT/deployments/docker"

    # Start infrastructure with production settings
    docker-compose -f docker-compose.infrastructure.yaml up -d

    log_info "Waiting for services to be ready (this may take a few minutes)..."
    sleep 15

    # Wait for PostgreSQL
    log_info "Waiting for PostgreSQL..."
    POSTGRES_READY=false
    for i in {1..30}; do
        if docker-compose -f docker-compose.infrastructure.yaml exec -T postgres pg_isready -U metabridge > /dev/null 2>&1; then
            POSTGRES_READY=true
            break
        fi
        echo -n "."
        sleep 2
    done

    if [ "$POSTGRES_READY" = false ]; then
        log_critical "PostgreSQL failed to start within timeout"
        exit 1
    fi
    log_success "PostgreSQL is ready"

    # Wait for NATS
    log_info "Waiting for NATS..."
    NATS_READY=false
    for i in {1..30}; do
        if docker-compose -f docker-compose.infrastructure.yaml logs nats 2>&1 | grep -q "Server is ready"; then
            NATS_READY=true
            break
        fi
        echo -n "."
        sleep 2
    done

    if [ "$NATS_READY" = false ]; then
        log_critical "NATS failed to start within timeout"
        exit 1
    fi
    log_success "NATS is ready"

    # Wait for Redis
    log_info "Waiting for Redis..."
    REDIS_READY=false
    for i in {1..30}; do
        if docker-compose -f docker-compose.infrastructure.yaml exec -T redis redis-cli ping > /dev/null 2>&1; then
            REDIS_READY=true
            break
        fi
        echo -n "."
        sleep 2
    done

    if [ "$REDIS_READY" = false ]; then
        log_critical "Redis failed to start within timeout"
        exit 1
    fi
    log_success "Redis is ready"

    log_success "All infrastructure services are running"
}

run_migrations() {
    log_info "Running database migrations..."

    cd "$PROJECT_ROOT"

    # Create mainnet database if not exists
    docker exec metabridge-postgres psql -U metabridge -tc "SELECT 1 FROM pg_database WHERE datname = 'metabridge_mainnet'" | grep -q 1 || \
    docker exec metabridge-postgres psql -U metabridge -c "CREATE DATABASE metabridge_mainnet;"

    # Run SQL schema
    if [ -f "$PROJECT_ROOT/internal/database/migrations/schema.sql" ]; then
        docker exec -i metabridge-postgres psql -U metabridge -d metabridge_mainnet < "$PROJECT_ROOT/internal/database/migrations/schema.sql"
        log_success "Database schema applied"
    else
        log_error "Schema file not found"
        exit 1
    fi

    # Verify migration
    TABLE_COUNT=$(docker exec metabridge-postgres psql -U metabridge -d metabridge_mainnet -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';")
    log_info "Database tables created: $TABLE_COUNT"

    log_success "Database migrations completed"
}

start_services() {
    log_info "Starting Metabridge services..."

    cd "$PROJECT_ROOT"

    # Load mainnet environment
    if [ -f "$PROJECT_ROOT/.env.mainnet" ]; then
        export $(cat "$PROJECT_ROOT/.env.mainnet" | grep -v '^#' | xargs)
    fi

    # Set environment
    export BRIDGE_ENVIRONMENT=mainnet
    export DATABASE_HOST=localhost
    export DATABASE_PORT=5432
    export DATABASE_USER=metabridge
    export DATABASE_NAME=metabridge_mainnet
    export NATS_URL=nats://localhost:4222
    export REDIS_URL=redis://localhost:6379

    # Start API in background
    log_info "Starting API service..."
    nohup ./bin/api --config "$CONFIG_FILE" > "$LOG_DIR/api.log" 2>&1 &
    echo $! > "$DATA_DIR/api.pid"
    log_success "API service started (PID: $(cat $DATA_DIR/api.pid))"

    # Wait and verify API started
    sleep 3
    if ! ps -p $(cat $DATA_DIR/api.pid) > /dev/null; then
        log_critical "API service failed to start. Check logs: $LOG_DIR/api.log"
        exit 1
    fi

    # Start Listener in background
    log_info "Starting Listener service..."
    nohup ./bin/listener --config "$CONFIG_FILE" > "$LOG_DIR/listener.log" 2>&1 &
    echo $! > "$DATA_DIR/listener.pid"
    log_success "Listener service started (PID: $(cat $DATA_DIR/listener.pid))"

    # Wait and verify Listener started
    sleep 3
    if ! ps -p $(cat $DATA_DIR/listener.pid) > /dev/null; then
        log_critical "Listener service failed to start. Check logs: $LOG_DIR/listener.log"
        exit 1
    fi

    # Start Relayer in background
    log_info "Starting Relayer service..."
    nohup ./bin/relayer --config "$CONFIG_FILE" > "$LOG_DIR/relayer.log" 2>&1 &
    echo $! > "$DATA_DIR/relayer.pid"
    log_success "Relayer service started (PID: $(cat $DATA_DIR/relayer.pid))"

    # Wait and verify Relayer started
    sleep 3
    if ! ps -p $(cat $DATA_DIR/relayer.pid) > /dev/null; then
        log_critical "Relayer service failed to start. Check logs: $LOG_DIR/relayer.log"
        exit 1
    fi

    # Wait for services to fully initialize
    log_info "Waiting for services to initialize..."
    sleep 10
}

verify_deployment() {
    log_info "Running comprehensive deployment verification..."

    # Check API health
    log_info "Checking API health..."
    for i in {1..10}; do
        if curl -f http://localhost:8080/health > /dev/null 2>&1; then
            log_success "API is healthy"
            break
        fi
        if [ $i -eq 10 ]; then
            log_critical "API health check failed after 10 attempts"
            log_error "Check logs: tail -f $LOG_DIR/api.log"
            exit 1
        fi
        echo -n "."
        sleep 3
    done

    # Check API status endpoint
    log_info "Checking system status..."
    STATUS_RESPONSE=$(curl -s http://localhost:8080/v1/status)
    if echo "$STATUS_RESPONSE" | grep -q "healthy"; then
        log_success "System status: healthy"
    else
        log_warning "System status check returned unexpected response"
    fi

    # Check database connection
    log_info "Checking database connection..."
    if docker exec metabridge-postgres psql -U metabridge -d metabridge_mainnet -c "SELECT 1;" > /dev/null 2>&1; then
        log_success "Database is accessible"
    else
        log_critical "Database connection failed"
        exit 1
    fi

    # Check NATS
    log_info "Checking NATS connection..."
    if docker exec metabridge-nats nats stream ls > /dev/null 2>&1; then
        log_success "NATS is running"
    else
        log_critical "NATS check failed"
        exit 1
    fi

    # Check Redis
    log_info "Checking Redis connection..."
    if docker exec metabridge-redis redis-cli ping | grep -q "PONG"; then
        log_success "Redis is running"
    else
        log_critical "Redis check failed"
        exit 1
    fi

    # Check all services are running
    log_info "Verifying all service processes..."
    for service in api listener relayer; do
        if [ -f "$DATA_DIR/${service}.pid" ]; then
            PID=$(cat "$DATA_DIR/${service}.pid")
            if ps -p $PID > /dev/null; then
                log_success "$service is running (PID: $PID)"
            else
                log_critical "$service process not found (expected PID: $PID)"
                exit 1
            fi
        else
            log_critical "$service PID file not found"
            exit 1
        fi
    done

    # Check supported chains
    log_info "Verifying blockchain configurations..."
    CHAINS_RESPONSE=$(curl -s http://localhost:8080/v1/chains)
    CHAIN_COUNT=$(echo "$CHAINS_RESPONSE" | grep -o '"name"' | wc -l)
    log_info "Configured chains: $CHAIN_COUNT"

    if [ "$CHAIN_COUNT" -lt 4 ]; then
        log_warning "Expected at least 4 chains, found $CHAIN_COUNT"
    else
        log_success "All blockchain configurations loaded"
    fi

    log_success "All deployment verification checks passed!"
}

setup_monitoring() {
    log_info "Configuring production monitoring..."

    # Check if Prometheus is accessible
    if curl -f http://localhost:9090/-/healthy > /dev/null 2>&1; then
        log_success "Prometheus is running"
    else
        log_warning "Prometheus health check failed"
    fi

    # Check if Grafana is accessible
    if curl -f http://localhost:3000/api/health > /dev/null 2>&1; then
        log_success "Grafana is running"
    else
        log_warning "Grafana health check failed"
    fi

    log_info "Configure alerting in Grafana for production monitoring"
}

show_status() {
    echo ""
    echo "======================================================================"
    echo -e "${GREEN}Metabridge Engine Mainnet - Deployment Complete!${NC}"
    echo "======================================================================"
    echo ""
    echo -e "${CYAN}📊 Service URLs:${NC}"
    echo "  API:        https://your-domain.com (or http://localhost:8080)"
    echo "  Prometheus: http://localhost:9090"
    echo "  Grafana:    http://localhost:3000 (configure production password)"
    echo ""
    echo -e "${CYAN}📝 Logs:${NC}"
    echo "  API:      tail -f $LOG_DIR/api.log"
    echo "  Listener: tail -f $LOG_DIR/listener.log"
    echo "  Relayer:  tail -f $LOG_DIR/relayer.log"
    echo ""
    echo -e "${CYAN}🔧 Quick Commands:${NC}"
    echo "  Health:   curl http://localhost:8080/health"
    echo "  Chains:   curl http://localhost:8080/v1/chains"
    echo "  Status:   curl http://localhost:8080/v1/status"
    echo ""
    echo -e "${CYAN}🔒 Security:${NC}"
    echo "  Validators: $MIN_VALIDATORS (3-of-5 multisig)"
    echo "  Key Management: AWS KMS ($AWS_KMS_KEY_ID)"
    echo "  Multi-sig Wallet: $MULTI_SIG_WALLET"
    echo ""
    echo -e "${CYAN}💾 Backups:${NC}"
    echo "  Location: $BACKUP_DIR"
    echo "  Latest: $(ls -t $BACKUP_DIR | head -n1)"
    echo ""
    echo -e "${CYAN}📊 Monitoring:${NC}"
    echo "  Setup alerts for:"
    echo "  - Failed transactions"
    echo "  - High gas prices"
    echo "  - Validator signature failures"
    echo "  - Database connection issues"
    echo "  - Unusual transaction volumes"
    echo ""
    echo -e "${YELLOW}⚠️  IMPORTANT:${NC}"
    echo "  1. Configure SSL/TLS for public access"
    echo "  2. Set up automated backups (every 6 hours recommended)"
    echo "  3. Configure alerting to team channels (Slack/PagerDuty)"
    echo "  4. Monitor gas prices and adjust limits as needed"
    echo "  5. Keep security contact information updated"
    echo "  6. Review logs daily for anomalies"
    echo ""
    echo "🛑 To stop all services:"
    echo "  ./stop-mainnet.sh"
    echo ""
    echo "📖 Documentation:"
    echo "  Deployment:  docs/runbooks/DEPLOYMENT.md"
    echo "  Monitoring:  docs/runbooks/MONITORING.md"
    echo "  Emergency:   docs/runbooks/EMERGENCY_PROCEDURES.md"
    echo ""
    echo "======================================================================"
    echo -e "${GREEN}Production deployment successful. Monitor closely for 24 hours.${NC}"
    echo "======================================================================"
    echo ""
}

create_rollback_script() {
    log_info "Creating rollback script..."

    cat > "$PROJECT_ROOT/rollback-mainnet.sh" << 'ROLLBACK_EOF'
#!/bin/bash
# Emergency rollback script for mainnet deployment

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${RED}======================================${NC}"
echo -e "${RED}  EMERGENCY MAINNET ROLLBACK${NC}"
echo -e "${RED}======================================${NC}"
echo ""

# Find latest backup
LATEST_BACKUP=$(ls -t backups/ | head -n1)

if [ -z "$LATEST_BACKUP" ]; then
    echo -e "${RED}No backup found!${NC}"
    exit 1
fi

echo -e "${YELLOW}Latest backup: $LATEST_BACKUP${NC}"
echo "Type 'ROLLBACK' to restore from this backup:"
read -r confirmation

if [ "$confirmation" != "ROLLBACK" ]; then
    echo "Rollback cancelled"
    exit 1
fi

# Stop services
./stop-mainnet.sh

# Restore database
if [ -f "backups/$LATEST_BACKUP/database.sql" ]; then
    echo "Restoring database..."
    docker exec -i metabridge-postgres psql -U metabridge metabridge_mainnet < "backups/$LATEST_BACKUP/database.sql"
    echo -e "${GREEN}Database restored${NC}"
fi

# Restore config
if [ -f "backups/$LATEST_BACKUP/config.mainnet.yaml" ]; then
    cp "backups/$LATEST_BACKUP/config.mainnet.yaml" config/config.mainnet.yaml
    echo -e "${GREEN}Configuration restored${NC}"
fi

echo ""
echo -e "${GREEN}Rollback completed. Review and restart services if needed.${NC}"
echo "To restart: ./deploy-mainnet.sh"
echo ""
ROLLBACK_EOF

    chmod +x "$PROJECT_ROOT/rollback-mainnet.sh"
    log_success "Rollback script created: rollback-mainnet.sh"
}

cleanup_on_error() {
    log_critical "Deployment failed! Initiating cleanup..."

    # Stop services
    if [ -f "$DATA_DIR/api.pid" ]; then
        kill $(cat "$DATA_DIR/api.pid") 2>/dev/null || true
        rm -f "$DATA_DIR/api.pid"
    fi
    if [ -f "$DATA_DIR/listener.pid" ]; then
        kill $(cat "$DATA_DIR/listener.pid") 2>/dev/null || true
        rm -f "$DATA_DIR/listener.pid"
    fi
    if [ -f "$DATA_DIR/relayer.pid" ]; then
        kill $(cat "$DATA_DIR/relayer.pid") 2>/dev/null || true
        rm -f "$DATA_DIR/relayer.pid"
    fi

    # Show recent logs
    echo ""
    log_error "Recent API logs:"
    if [ -f "$LOG_DIR/api.log" ]; then
        tail -n 20 "$LOG_DIR/api.log"
    fi

    echo ""
    log_error "Deployment failed. Services stopped. Review logs for details."
    echo "Backup available in: $BACKUP_DIR"

    exit 1
}

# Main execution
main() {
    # Print warning banner
    print_banner

    # Require explicit confirmation
    confirm_deployment

    # Set error handler
    trap cleanup_on_error ERR

    # Execute deployment steps
    verify_security_audit
    check_prerequisites
    verify_validator_configuration
    verify_multisig_wallet
    create_directories
    build_services
    backup_existing_data
    start_infrastructure
    run_migrations
    start_services
    verify_deployment
    setup_monitoring
    create_rollback_script
    show_status

    log_success "Mainnet deployment completed successfully!"
    echo ""
    log_warning "Monitor all services closely for the next 24-48 hours"
    log_warning "Ensure team is on standby for immediate response to any issues"
}

# Run main function
main "$@"
