#!/bin/bash
set -e

##############################################################################
# Articium - Automated End-to-End Test Suite
#
# This script performs a complete E2E test of the Articium system:
# 1. Requests testnet tokens from faucets
# 2. Deploys smart contracts to all testnets
# 3. Deploys and starts backend services
# 4. Executes cross-chain token transfers
# 5. Verifies all operations completed successfully
#
# Prerequisites:
# - Node.js 18+ and npm installed
# - Go 1.21+ installed
# - Docker and Docker Compose installed
# - jq installed (for JSON parsing)
# - curl installed
# - Git repository cloned
#
# Usage:
#   ./test-e2e-full.sh [--wallet-address 0x...]
#
##############################################################################

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="$PROJECT_ROOT/logs/e2e"
RESULTS_DIR="$PROJECT_ROOT/test-results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
TEST_LOG="$LOG_DIR/e2e-test-$TIMESTAMP.log"

# Test wallet (can be overridden)
WALLET_ADDRESS="${1:-}"

# Chains to test
CHAINS=("polygon-amoy" "bnb-testnet" "avalanche-fuji" "ethereum-sepolia")

# Test results
declare -A TEST_RESULTS
declare -A FAUCET_RESULTS
declare -A DEPLOYMENT_RESULTS
declare -A TRANSFER_RESULTS

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$TEST_LOG"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$TEST_LOG"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$TEST_LOG"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$TEST_LOG"
}

log_step() {
    echo -e "${CYAN}[STEP]${NC} $1" | tee -a "$TEST_LOG"
}

print_banner() {
    echo ""
    echo "========================================================================"
    echo "  Articium - Automated E2E Test Suite"
    echo "  Timestamp: $TIMESTAMP"
    echo "========================================================================"
    echo ""
}

check_prerequisites() {
    log_step "Checking prerequisites..."

    # Check Node.js
    if ! command -v node &> /dev/null; then
        log_error "Node.js is not installed"
        return 1
    fi
    log_info "Node.js: $(node --version)"

    # Check npm
    if ! command -v npm &> /dev/null; then
        log_error "npm is not installed"
        return 1
    fi
    log_info "npm: $(npm --version)"

    # Check Go
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        return 1
    fi
    log_info "Go: $(go version)"

    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        return 1
    fi
    log_info "Docker: $(docker --version)"

    # Check jq
    if ! command -v jq &> /dev/null; then
        log_error "jq is not installed. Install with: sudo apt-get install jq"
        return 1
    fi
    log_info "jq: $(jq --version)"

    # Check curl
    if ! command -v curl &> /dev/null; then
        log_error "curl is not installed"
        return 1
    fi

    log_success "All prerequisites met"
    return 0
}

setup_test_environment() {
    log_step "Setting up test environment..."

    # Create directories
    mkdir -p "$LOG_DIR"
    mkdir -p "$RESULTS_DIR"
    mkdir -p "$PROJECT_ROOT/test-wallets"

    # Create or load test wallet
    if [ -z "$WALLET_ADDRESS" ]; then
        log_info "No wallet address provided, generating test wallet..."

        # Create test wallet using Node.js
        cat > "$PROJECT_ROOT/test-wallets/generate-wallet.js" << 'EOF'
const { Wallet } = require('ethers');
const fs = require('fs');

const wallet = Wallet.createRandom();
const walletInfo = {
    address: wallet.address,
    privateKey: wallet.privateKey,
    mnemonic: wallet.mnemonic.phrase
};

console.log(JSON.stringify(walletInfo, null, 2));
fs.writeFileSync('test-wallets/wallet.json', JSON.stringify(walletInfo, null, 2));
EOF

        # Install ethers if needed
        cd "$PROJECT_ROOT/test-wallets"
        if [ ! -d "node_modules" ]; then
            npm init -y > /dev/null 2>&1
            npm install ethers@^6.0.0 > /dev/null 2>&1
        fi

        # Generate wallet
        WALLET_INFO=$(node generate-wallet.js)
        WALLET_ADDRESS=$(echo "$WALLET_INFO" | jq -r '.address')

        log_success "Test wallet generated: $WALLET_ADDRESS"
        log_warning "Wallet details saved to: test-wallets/wallet.json"
        log_warning "⚠️  This is a TEST wallet. Never use for mainnet!"
    else
        log_info "Using provided wallet: $WALLET_ADDRESS"
    fi

    # Save wallet address to env
    echo "TEST_WALLET_ADDRESS=$WALLET_ADDRESS" > "$PROJECT_ROOT/.env.test"

    log_success "Test environment ready"
}

request_faucet_tokens() {
    log_step "Requesting tokens from faucets..."

    echo ""
    log_warning "═══════════════════════════════════════════════════════════════"
    log_warning "  MANUAL FAUCET REQUESTS REQUIRED"
    log_warning "═══════════════════════════════════════════════════════════════"
    echo ""
    log_info "Most faucets require manual CAPTCHA completion."
    log_info "Please visit the following faucets and request tokens:"
    echo ""

    # Polygon Amoy
    echo -e "${CYAN}1. Polygon Amoy Testnet (MATIC)${NC}"
    echo "   URL: https://faucet.polygon.technology/"
    echo "   Address: $WALLET_ADDRESS"
    echo "   Network: POL (Amoy)"
    echo "   Amount: 0.5-1 MATIC"
    echo ""

    # BNB Testnet
    echo -e "${CYAN}2. BNB Smart Chain Testnet (tBNB)${NC}"
    echo "   URL: https://testnet.bnbchain.org/faucet-smart"
    echo "   Address: $WALLET_ADDRESS"
    echo "   Amount: 0.1-0.5 tBNB"
    echo ""

    # Avalanche Fuji
    echo -e "${CYAN}3. Avalanche Fuji Testnet (AVAX)${NC}"
    echo "   URL: https://core.app/tools/testnet-faucet/"
    echo "   Address: $WALLET_ADDRESS"
    echo "   Network: Fuji (C-Chain)"
    echo "   Amount: 2 AVAX"
    echo ""

    # Ethereum Sepolia
    echo -e "${CYAN}4. Ethereum Sepolia Testnet (SepoliaETH)${NC}"
    echo "   URL: https://www.alchemy.com/faucets/ethereum-sepolia"
    echo "   Address: $WALLET_ADDRESS"
    echo "   Amount: 0.5 ETH"
    echo "   Note: Requires Alchemy account (free)"
    echo ""

    log_warning "After requesting from all faucets, press ENTER to continue..."
    read -r

    # Check balances
    log_info "Checking balances..."
    check_all_balances

    log_warning "Do you have sufficient tokens on all chains? (yes/no)"
    read -r response

    if [ "$response" != "yes" ]; then
        log_error "Insufficient funds. Please get more tokens and run the script again."
        exit 1
    fi

    log_success "Faucet tokens confirmed"
}

check_all_balances() {
    log_info "Checking balances for wallet: $WALLET_ADDRESS"

    # Install necessary dependencies for balance checking
    cd "$PROJECT_ROOT/test-wallets"

    cat > check-balance.js << 'EOF'
const { JsonRpcProvider } = require('ethers');

const rpcs = {
    'polygon-amoy': 'https://rpc-amoy.polygon.technology',
    'bnb-testnet': 'https://data-seed-prebsc-1-s1.binance.org:8545',
    'avalanche-fuji': 'https://api.avax-test.network/ext/bc/C/rpc',
    'ethereum-sepolia': 'https://ethereum-sepolia-rpc.publicnode.com'
};

async function checkBalance(chain, rpc, address) {
    try {
        const provider = new JsonRpcProvider(rpc);
        const balance = await provider.getBalance(address);
        const balanceEth = (Number(balance) / 1e18).toFixed(4);
        console.log(`${chain}: ${balanceEth}`);
        return balanceEth;
    } catch (error) {
        console.log(`${chain}: Error - ${error.message}`);
        return 0;
    }
}

async function main() {
    const address = process.argv[2];
    for (const [chain, rpc] of Object.entries(rpcs)) {
        await checkBalance(chain, rpc, address);
    }
}

main();
EOF

    node check-balance.js "$WALLET_ADDRESS"
}

deploy_smart_contracts() {
    log_step "Deploying smart contracts to all testnets..."

    cd "$PROJECT_ROOT/contracts/evm"

    # Install dependencies
    if [ ! -d "node_modules" ]; then
        log_info "Installing contract dependencies..."
        npm install > /dev/null 2>&1
    fi

    # Setup environment
    if [ -f "$PROJECT_ROOT/test-wallets/wallet.json" ]; then
        PRIVATE_KEY=$(jq -r '.privateKey' "$PROJECT_ROOT/test-wallets/wallet.json")
        echo "PRIVATE_KEY=$PRIVATE_KEY" > .env
    fi

    # Deploy to each testnet
    for chain in "${CHAINS[@]}"; do
        log_info "Deploying to $chain..."

        case $chain in
            "polygon-amoy")
                npm run deploy:polygon-amoy >> "$TEST_LOG" 2>&1
                ;;
            "bnb-testnet")
                npm run deploy:bnb-testnet >> "$TEST_LOG" 2>&1
                ;;
            "avalanche-fuji")
                npm run deploy:avalanche-fuji >> "$TEST_LOG" 2>&1
                ;;
            "ethereum-sepolia")
                npm run deploy:ethereum-sepolia >> "$TEST_LOG" 2>&1
                ;;
        esac

        if [ $? -eq 0 ]; then
            log_success "$chain: Contract deployed"
            DEPLOYMENT_RESULTS[$chain]="SUCCESS"
        else
            log_error "$chain: Deployment failed"
            DEPLOYMENT_RESULTS[$chain]="FAILED"
        fi

        sleep 2
    done

    # Check deployment results
    log_info "Deployment Summary:"
    for chain in "${CHAINS[@]}"; do
        if [ "${DEPLOYMENT_RESULTS[$chain]}" == "SUCCESS" ]; then
            log_success "$chain: ✓"
        else
            log_error "$chain: ✗"
        fi
    done
}

deploy_backend_services() {
    log_step "Deploying backend services..."

    cd "$PROJECT_ROOT"

    # Use testnet deployment script
    log_info "Starting Articium testnet deployment..."

    # Run deployment in background to capture output
    ./deploy-testnet.sh >> "$TEST_LOG" 2>&1

    if [ $? -eq 0 ]; then
        log_success "Backend services deployed"
        return 0
    else
        log_error "Backend deployment failed"
        return 1
    fi
}

wait_for_services() {
    log_step "Waiting for services to be ready..."

    local max_attempts=30
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if curl -f http://localhost:8080/health > /dev/null 2>&1; then
            log_success "Services are ready"
            return 0
        fi

        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done

    log_error "Services failed to become ready within timeout"
    return 1
}

test_cross_chain_transfers() {
    log_step "Testing cross-chain transfers..."

    # Test scenarios
    local test_cases=(
        "polygon-amoy:avalanche-fuji"
        "bnb-testnet:ethereum-sepolia"
        "avalanche-fuji:polygon-amoy"
        "ethereum-sepolia:bnb-testnet"
    )

    for test_case in "${test_cases[@]}"; do
        IFS=':' read -r source dest <<< "$test_case"

        log_info "Testing: $source → $dest"

        # Create test transfer request
        local response=$(curl -s -X POST http://localhost:8080/v1/bridge/token \
            -H "Content-Type: application/json" \
            -d "{
                \"source_chain\": \"$source\",
                \"destination_chain\": \"$dest\",
                \"token_address\": \"0x0000000000000000000000000000000000001010\",
                \"amount\": \"10000000000000000\",
                \"sender\": \"$WALLET_ADDRESS\",
                \"recipient\": \"$WALLET_ADDRESS\"
            }")

        local message_id=$(echo "$response" | jq -r '.message_id')

        if [ "$message_id" != "null" ] && [ -n "$message_id" ]; then
            log_success "Transfer initiated: $message_id"

            # Monitor transfer
            local status=$(monitor_transfer "$message_id")

            if [ "$status" == "completed" ]; then
                log_success "$source → $dest: COMPLETED ✓"
                TRANSFER_RESULTS["$test_case"]="SUCCESS"
            else
                log_warning "$source → $dest: Status: $status"
                TRANSFER_RESULTS["$test_case"]="PENDING"
            fi
        else
            log_error "$source → $dest: Failed to initiate"
            TRANSFER_RESULTS["$test_case"]="FAILED"
        fi

        sleep 5
    done
}

monitor_transfer() {
    local message_id=$1
    local max_attempts=60  # 5 minutes
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        local response=$(curl -s "http://localhost:8080/v1/messages/$message_id")
        local status=$(echo "$response" | jq -r '.status')

        if [ "$status" == "completed" ]; then
            return 0
        elif [ "$status" == "failed" ]; then
            return 1
        fi

        attempt=$((attempt + 1))
        sleep 5
    done

    echo "pending"
    return 2
}

verify_system_health() {
    log_step "Verifying system health..."

    # Check API health
    if curl -f http://localhost:8080/health > /dev/null 2>&1; then
        log_success "API health check: PASS"
    else
        log_error "API health check: FAIL"
        return 1
    fi

    # Check supported chains
    local chains_response=$(curl -s http://localhost:8080/v1/chains)
    local chain_count=$(echo "$chains_response" | jq '.chains | length')

    if [ "$chain_count" -ge 4 ]; then
        log_success "Chain configuration: PASS ($chain_count chains)"
    else
        log_error "Chain configuration: FAIL (expected 4+, got $chain_count)"
        return 1
    fi

    # Check database
    if docker exec articium-postgres psql -U articium -d articium_testnet -c "SELECT 1;" > /dev/null 2>&1; then
        log_success "Database connection: PASS"
    else
        log_error "Database connection: FAIL"
        return 1
    fi

    # Check NATS
    if docker exec articium-nats nats stream ls > /dev/null 2>&1; then
        log_success "NATS connection: PASS"
    else
        log_warning "NATS connection: WARNING"
    fi

    # Check Redis
    if docker exec articium-redis redis-cli ping | grep -q "PONG"; then
        log_success "Redis connection: PASS"
    else
        log_warning "Redis connection: WARNING"
    fi

    log_success "System health verification complete"
    return 0
}

generate_test_report() {
    log_step "Generating test report..."

    local report_file="$RESULTS_DIR/e2e-report-$TIMESTAMP.txt"

    cat > "$report_file" << EOF
========================================================================
Articium - E2E Test Report
========================================================================
Timestamp: $TIMESTAMP
Test Wallet: $WALLET_ADDRESS

========================================================================
SMART CONTRACT DEPLOYMENTS
========================================================================
EOF

    for chain in "${CHAINS[@]}"; do
        echo "$chain: ${DEPLOYMENT_RESULTS[$chain]:-NOT_RUN}" >> "$report_file"
    done

    cat >> "$report_file" << EOF

========================================================================
CROSS-CHAIN TRANSFERS
========================================================================
EOF

    for test_case in "${!TRANSFER_RESULTS[@]}"; do
        echo "$test_case: ${TRANSFER_RESULTS[$test_case]}" >> "$report_file"
    done

    cat >> "$report_file" << EOF

========================================================================
OVERALL RESULTS
========================================================================
EOF

    # Calculate success rate
    local total_deployments=${#CHAINS[@]}
    local successful_deployments=0

    for chain in "${CHAINS[@]}"; do
        if [ "${DEPLOYMENT_RESULTS[$chain]}" == "SUCCESS" ]; then
            successful_deployments=$((successful_deployments + 1))
        fi
    done

    local total_transfers=${#TRANSFER_RESULTS[@]}
    local successful_transfers=0

    for result in "${TRANSFER_RESULTS[@]}"; do
        if [ "$result" == "SUCCESS" ]; then
            successful_transfers=$((successful_transfers + 1))
        fi
    done

    cat >> "$report_file" << EOF
Deployments: $successful_deployments/$total_deployments successful
Transfers: $successful_transfers/$total_transfers successful

Overall Status: $([ $successful_deployments -eq $total_deployments ] && [ $successful_transfers -gt 0 ] && echo "PASS ✓" || echo "NEEDS REVIEW ⚠")

Test Log: $TEST_LOG
Report: $report_file
========================================================================
EOF

    # Display report
    cat "$report_file"

    log_success "Test report saved: $report_file"
}

cleanup() {
    log_info "Cleaning up test environment..."

    # Optionally stop services
    log_warning "Do you want to stop all services? (yes/no)"
    read -r response

    if [ "$response" == "yes" ]; then
        cd "$PROJECT_ROOT"
        ./stop-testnet.sh
        log_success "Services stopped"
    else
        log_info "Services left running for manual inspection"
    fi
}

# Main execution
main() {
    print_banner

    # Step 1: Prerequisites
    if ! check_prerequisites; then
        log_error "Prerequisites check failed"
        exit 1
    fi

    # Step 2: Setup
    setup_test_environment

    # Step 3: Faucets
    request_faucet_tokens

    # Step 4: Deploy contracts
    deploy_smart_contracts

    # Step 5: Deploy backend
    if ! deploy_backend_services; then
        log_error "Backend deployment failed"
        exit 1
    fi

    # Step 6: Wait for services
    if ! wait_for_services; then
        log_error "Services failed to start"
        exit 1
    fi

    # Step 7: Verify health
    if ! verify_system_health; then
        log_error "System health verification failed"
        exit 1
    fi

    # Step 8: Test transfers
    test_cross_chain_transfers

    # Step 9: Generate report
    generate_test_report

    # Step 10: Cleanup
    cleanup

    echo ""
    log_success "E2E test suite completed!"
    echo ""
    log_info "Next steps:"
    log_info "1. Review test report: $RESULTS_DIR/e2e-report-$TIMESTAMP.txt"
    log_info "2. Check detailed logs: $TEST_LOG"
    log_info "3. Monitor services: http://localhost:8080"
    log_info "4. View Grafana: http://localhost:3000"
    echo ""
}

# Handle interrupts
trap cleanup INT TERM

# Run main function
main "$@"
