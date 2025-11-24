#!/bin/bash
set -e

##############################################################################
# Articium - Automated End-to-End Test Suite
#
# This script performs a complete E2E test of the Articium system:
# 1. Requests testnet tokens from faucets
# 2. Deploys smart contracts to testnets OR mainnets
# 3. Deploys and starts backend services
# 4. Executes cross-chain token transfers
# 5. Displays transaction confirmations with block explorer links
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
#   ./test-e2e-full.sh [--network testnet|mainnet] [--wallet-address 0x...]
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

# Default to testnet
NETWORK="testnet"
WALLET_ADDRESS=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --network)
            NETWORK="$2"
            shift 2
            ;;
        --wallet-address)
            WALLET_ADDRESS="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--network testnet|mainnet] [--wallet-address 0x...]"
            exit 1
            ;;
    esac
done

# Validate network
if [ "$NETWORK" != "testnet" ] && [ "$NETWORK" != "mainnet" ]; then
    echo "Error: Network must be 'testnet' or 'mainnet'"
    exit 1
fi

# Chains to test (13 testnets or 6 mainnets)
if [ "$NETWORK" == "testnet" ]; then
    CHAINS=(
        "polygon-amoy"
        "bnb-testnet"
        "avalanche-fuji"
        "ethereum-sepolia"
        "solana-devnet"
        "near-testnet"
        "tron-nile"
        "fantom-testnet"
        "arbitrum-sepolia"
        "optimism-sepolia"
        "harmony-testnet"
        "algorand-testnet"
        "aptos-testnet"
    )
else
    CHAINS=(
        "polygon-mainnet"
        "bnb-mainnet"
        "avalanche-mainnet"
        "ethereum-mainnet"
        "solana-mainnet"
        "near-mainnet"
    )
fi

# Test results
declare -A TEST_RESULTS
declare -A FAUCET_RESULTS
declare -A DEPLOYMENT_RESULTS
declare -A TRANSFER_RESULTS
declare -A TX_HASHES
declare -A BLOCK_EXPLORERS

# Block Explorer URLs
setup_block_explorers() {
    # Testnets
    BLOCK_EXPLORERS["polygon-amoy"]="https://amoy.polygonscan.com"
    BLOCK_EXPLORERS["bnb-testnet"]="https://testnet.bscscan.com"
    BLOCK_EXPLORERS["avalanche-fuji"]="https://testnet.snowtrace.io"
    BLOCK_EXPLORERS["ethereum-sepolia"]="https://sepolia.etherscan.io"
    BLOCK_EXPLORERS["solana-devnet"]="https://explorer.solana.com/?cluster=devnet"
    BLOCK_EXPLORERS["near-testnet"]="https://explorer.testnet.near.org"
    BLOCK_EXPLORERS["tron-nile"]="https://nile.tronscan.org"
    BLOCK_EXPLORERS["fantom-testnet"]="https://testnet.ftmscan.com"
    BLOCK_EXPLORERS["arbitrum-sepolia"]="https://sepolia.arbiscan.io"
    BLOCK_EXPLORERS["optimism-sepolia"]="https://sepolia-optimism.etherscan.io"
    BLOCK_EXPLORERS["harmony-testnet"]="https://explorer.testnet.harmony.one"
    BLOCK_EXPLORERS["algorand-testnet"]="https://testnet.algoexplorer.io"
    BLOCK_EXPLORERS["aptos-testnet"]="https://explorer.aptoslabs.com/?network=testnet"

    # Mainnets
    BLOCK_EXPLORERS["polygon-mainnet"]="https://polygonscan.com"
    BLOCK_EXPLORERS["bnb-mainnet"]="https://bscscan.com"
    BLOCK_EXPLORERS["avalanche-mainnet"]="https://snowtrace.io"
    BLOCK_EXPLORERS["ethereum-mainnet"]="https://etherscan.io"
    BLOCK_EXPLORERS["solana-mainnet"]="https://explorer.solana.com"
    BLOCK_EXPLORERS["near-mainnet"]="https://explorer.near.org"
}

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
    echo "  Articium - Automated E2E Test Suite ($NETWORK)"
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
    log_step "Setting up test environment for $NETWORK..."

    # Create or load test wallet
    if [ -z "$WALLET_ADDRESS" ]; then
        log_info "No wallet address provided, generating test wallet..."

        # Create test wallet using Node.js
        cat > "$PROJECT_ROOT/test-wallets/generate-wallet.js" << 'EOF'
const { Wallet } = require('ethers');
const fs = require('fs');
const path = require('path');

const wallet = Wallet.createRandom();
const walletInfo = {
    address: wallet.address,
    privateKey: wallet.privateKey,
    mnemonic: wallet.mnemonic.phrase
};

console.log(JSON.stringify(walletInfo, null, 2));
fs.writeFileSync(path.join(__dirname, 'wallet.json'), JSON.stringify(walletInfo, null, 2));
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

        if [ "$NETWORK" == "mainnet" ]; then
            log_error "⚠️  CRITICAL: This is a MAINNET wallet!"
            log_warning "⚠️  Secure this wallet properly. Never share the private key!"
        else
            log_warning "⚠️  This is a TEST wallet. Never use for mainnet!"
        fi
    else
        log_info "Using provided wallet: $WALLET_ADDRESS"
    fi

    # Save wallet address to env
    echo "TEST_WALLET_ADDRESS=$WALLET_ADDRESS" > "$PROJECT_ROOT/.env.test"
    echo "NETWORK=$NETWORK" >> "$PROJECT_ROOT/.env.test"

    log_success "Test environment ready"
}

show_faucet_links() {
    log_step "Getting tokens for testing..."

    echo ""
    log_warning "═══════════════════════════════════════════════════════════════"

    if [ "$NETWORK" == "testnet" ]; then
        log_warning "  FREE TESTNET FAUCETS - GET TEST TOKENS"
    else
        log_error "  ⚠️  MAINNET - YOU NEED REAL TOKENS ⚠️"
    fi

    log_warning "═══════════════════════════════════════════════════════════════"
    echo ""

    if [ "$NETWORK" == "testnet" ]; then
        log_info "Your wallet address: $WALLET_ADDRESS"
        echo ""
        log_info "Visit the following faucets to get FREE test tokens:"
        echo ""

        # Polygon Amoy
        echo -e "${CYAN}1. Polygon Amoy Testnet (MATIC)${NC}"
        echo "   🔗 URL: https://faucet.polygon.technology/"
        echo "   📍 Network: POL (Amoy)"
        echo "   💰 Amount: 0.5-1 MATIC"
        echo "   📋 Address: $WALLET_ADDRESS"
        echo ""

        # BNB Testnet
        echo -e "${CYAN}2. BNB Smart Chain Testnet (tBNB)${NC}"
        echo "   🔗 URL: https://testnet.bnbchain.org/faucet-smart"
        echo "   💰 Amount: 0.1-0.5 tBNB"
        echo "   📋 Address: $WALLET_ADDRESS"
        echo ""

        # Avalanche Fuji
        echo -e "${CYAN}3. Avalanche Fuji Testnet (AVAX)${NC}"
        echo "   🔗 URL: https://core.app/tools/testnet-faucet/"
        echo "   📍 Network: Fuji (C-Chain)"
        echo "   💰 Amount: 2 AVAX"
        echo "   📋 Address: $WALLET_ADDRESS"
        echo ""

        # Ethereum Sepolia
        echo -e "${CYAN}4. Ethereum Sepolia Testnet (SepoliaETH)${NC}"
        echo "   🔗 URL: https://www.alchemy.com/faucets/ethereum-sepolia"
        echo "   💰 Amount: 0.5 ETH"
        echo "   📋 Address: $WALLET_ADDRESS"
        echo "   ⚠️  Note: Requires Alchemy account (free)"
        echo ""

        # Solana Devnet
        echo -e "${CYAN}5. Solana Devnet (SOL)${NC}"
        echo "   🔗 URL: https://faucet.solana.com/"
        echo "   💰 Amount: 1 SOL"
        echo "   📋 Address: $WALLET_ADDRESS"
        echo "   💡 CLI: solana airdrop 1 $WALLET_ADDRESS --url devnet"
        echo ""

        # NEAR Testnet
        echo -e "${CYAN}6. NEAR Testnet (NEAR)${NC}"
        echo "   🔗 URL: https://near-faucet.io/"
        echo "   💰 Amount: 20 NEAR"
        echo "   📋 Account ID needed (create at wallet.testnet.near.org)"
        echo ""

        # TRON Nile
        echo -e "${CYAN}7. TRON Nile Testnet (TRX)${NC}"
        echo "   🔗 URL: https://nileex.io/join/getJoinPage"
        echo "   💰 Amount: 10,000 TRX"
        echo "   📋 Address: $WALLET_ADDRESS"
        echo ""

        # Fantom Testnet
        echo -e "${CYAN}8. Fantom Testnet (FTM)${NC}"
        echo "   🔗 URL: https://faucet.fantom.network/"
        echo "   💰 Amount: 10 FTM"
        echo "   📋 Address: $WALLET_ADDRESS"
        echo ""

        # Arbitrum Sepolia
        echo -e "${CYAN}9. Arbitrum Sepolia (ETH)${NC}"
        echo "   🔗 URL: https://faucet.quicknode.com/arbitrum/sepolia"
        echo "   💰 Amount: 0.05 ETH"
        echo "   📋 Address: $WALLET_ADDRESS"
        echo ""

        # Optimism Sepolia
        echo -e "${CYAN}10. Optimism Sepolia (ETH)${NC}"
        echo "   🔗 URL: https://faucet.quicknode.com/optimism/sepolia"
        echo "   💰 Amount: 0.05 ETH"
        echo "   📋 Address: $WALLET_ADDRESS"
        echo ""

        # Harmony Testnet
        echo -e "${CYAN}11. Harmony Testnet (ONE)${NC}"
        echo "   🔗 URL: https://faucet.pops.one/"
        echo "   💰 Amount: 1 ONE"
        echo "   📋 Address: $WALLET_ADDRESS"
        echo ""

        # Algorand Testnet
        echo -e "${CYAN}12. Algorand Testnet (ALGO)${NC}"
        echo "   🔗 URL: https://bank.testnet.algorand.network/"
        echo "   💰 Amount: 10 ALGO"
        echo "   📋 Address needed (create Algorand wallet)"
        echo ""

        # Aptos Testnet
        echo -e "${CYAN}13. Aptos Testnet (APT)${NC}"
        echo "   🔗 URL: https://www.aptosfaucet.com/"
        echo "   💰 Amount: 1 APT"
        echo "   📋 Address needed (create Aptos wallet)"
        echo ""

        echo ""
        log_warning "After requesting from all faucets, press ENTER to continue..."
        echo ""

        # Force interactive read from terminal
        read -r -p "Press ENTER when ready: " < /dev/tty

        echo ""
        # Skip automatic balance check due to unreliable public RPC endpoints
        log_warning "⚠️  IMPORTANT: Please verify you have requested tokens from the faucets above"
        log_warning "You can manually check your balances at these block explorers:"
        log_info "  • Polygon Amoy: https://amoy.polygonscan.com/address/$WALLET_ADDRESS"
        log_info "  • BNB Testnet: https://testnet.bscscan.com/address/$WALLET_ADDRESS"
        log_info "  • Avalanche Fuji: https://testnet.snowtrace.io/address/$WALLET_ADDRESS"
        log_info "  • Ethereum Sepolia: https://sepolia.etherscan.io/address/$WALLET_ADDRESS"
        echo ""
        log_warning "Have you requested tokens from the faucets and verified your balance? (yes/no)"
        read -r -p "Enter 'yes' to continue or 'no' to exit: " response < /dev/tty

        if [ "$response" != "yes" ]; then
            log_error "Insufficient funds. Please get more tokens and run the script again."
            exit 1
        fi
    else
        log_error "⚠️  MAINNET MODE - YOU NEED REAL TOKENS!"
        log_error "⚠️  This will use REAL MONEY!"
        echo ""
        log_info "Ensure you have sufficient native tokens on:"
        for chain in "${CHAINS[@]}"; do
            echo "   - $chain"
        done
        echo ""
        log_warning "Have you funded your wallet with real tokens? (yes/no)"
        read -r -p "Enter 'yes' to continue or 'no' to exit: " response < /dev/tty

        if [ "$response" != "yes" ]; then
            log_error "Please fund your wallet before continuing."
            exit 1
        fi
    fi

    log_success "Tokens confirmed"
}

check_all_balances() {
    log_info "Checking balances for wallet: $WALLET_ADDRESS"

    # Install necessary dependencies for balance checking
    cd "$PROJECT_ROOT/test-wallets"

    cat > check-balance.js << 'EOF'
const { JsonRpcProvider } = require('ethers');

const testnetRPCs = {
    'polygon-amoy': 'https://rpc-amoy.polygon.technology',
    'bnb-testnet': 'https://data-seed-prebsc-1-s1.binance.org:8545',
    'avalanche-fuji': 'https://api.avax-test.network/ext/bc/C/rpc',
    'ethereum-sepolia': 'https://ethereum-sepolia-rpc.publicnode.com',
    'tron-nile': 'https://nile.trongrid.io',
    'fantom-testnet': 'https://rpc.testnet.fantom.network',
    'arbitrum-sepolia': 'https://sepolia-rollup.arbitrum.io/rpc',
    'optimism-sepolia': 'https://sepolia.optimism.io',
    'harmony-testnet': 'https://api.s0.b.hmny.io',
};

const mainnetRPCs = {
    'polygon-mainnet': 'https://polygon-rpc.com',
    'bnb-mainnet': 'https://bsc-dataseed.binance.org',
    'avalanche-mainnet': 'https://api.avax.network/ext/bc/C/rpc',
    'ethereum-mainnet': 'https://eth.llamarpc.com',
};

async function checkBalance(chain, rpc, address) {
    try {
        // Add timeout to prevent hanging
        const timeout = new Promise((_, reject) =>
            setTimeout(() => reject(new Error('Timeout after 5s')), 5000)
        );

        const checkPromise = (async () => {
            // Skip network detection to avoid rate limit issues
            const provider = new JsonRpcProvider(rpc, undefined, { staticNetwork: true });
            const balance = await provider.getBalance(address);
            const balanceEth = (Number(balance) / 1e18).toFixed(4);
            return balanceEth;
        })();

        const balanceEth = await Promise.race([checkPromise, timeout]);
        console.log(`${chain}: ${balanceEth}`);
        return balanceEth;
    } catch (error) {
        console.log(`${chain}: 0.0000 (RPC unavailable)`);
        return 0;
    }
}

async function main() {
    const address = process.argv[2];
    const network = process.argv[3] || 'testnet';

    const rpcs = network === 'mainnet' ? mainnetRPCs : testnetRPCs;

    for (const [chain, rpc] of Object.entries(rpcs)) {
        await checkBalance(chain, rpc, address);
    }
}

main();
EOF

    # Suppress ethers.js warnings/errors from stderr
    node check-balance.js "$WALLET_ADDRESS" "$NETWORK" 2>/dev/null || true
}

deploy_smart_contracts() {
    log_step "Deploying smart contracts to $NETWORK..."

    cd "$PROJECT_ROOT/contracts/evm"

    # Install dependencies
    if [ ! -d "node_modules" ]; then
        log_info "Installing contract dependencies..."
        npm install > /dev/null 2>&1
    fi

    # Setup environment
    if [ -f "$PROJECT_ROOT/test-wallets/wallet.json" ]; then
        PRIVATE_KEY=$(jq -r '.privateKey' "$PROJECT_ROOT/test-wallets/wallet.json")
        echo "DEPLOYER_PRIVATE_KEY=$PRIVATE_KEY" > .env
    fi

    # Deploy based on network
    if [ "$NETWORK" == "testnet" ]; then
        # Deploy to testnets
        for chain in "polygon-amoy" "bnb-testnet" "avalanche-fuji" "ethereum-sepolia"; do
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

                # Extract contract address and show block explorer
                if [ -f "deployments/${chain}_*.json" ]; then
                    CONTRACT_ADDR=$(jq -r '.contractAddress' deployments/${chain}_*.json 2>/dev/null)
                    if [ -n "$CONTRACT_ADDR" ] && [ "$CONTRACT_ADDR" != "null" ]; then
                        log_info "Contract Address: $CONTRACT_ADDR"
                        log_info "View on explorer: ${BLOCK_EXPLORERS[$chain]}/address/$CONTRACT_ADDR"
                    fi
                fi
            else
                log_error "$chain: Deployment failed"
                DEPLOYMENT_RESULTS[$chain]="FAILED"
            fi

            sleep 2
        done
    else
        # Deploy to mainnets
        log_warning "⚠️  Deploying to MAINNET - This uses REAL money!"
        log_warning "Press ENTER to confirm or Ctrl+C to cancel..."
        read -r -p "Press ENTER to confirm: " < /dev/tty

        node scripts/deploy-all-mainnet.js >> "$TEST_LOG" 2>&1

        if [ $? -eq 0 ]; then
            log_success "All mainnet contracts deployed"
            for chain in "${CHAINS[@]}"; do
                DEPLOYMENT_RESULTS[$chain]="SUCCESS"
            done
        else
            log_error "Mainnet deployment failed"
        fi
    fi

    # Check deployment results
    log_info "Deployment Summary:"
    for chain in "${CHAINS[@]}"; do
        if [[ "$chain" == *"solana"* ]] || [[ "$chain" == *"near"* ]] || [[ "$chain" == *"algorand"* ]] || [[ "$chain" == *"aptos"* ]]; then
            # Skip non-EVM chains for EVM deployment
            continue
        fi

        if [ "${DEPLOYMENT_RESULTS[$chain]}" == "SUCCESS" ]; then
            log_success "$chain: ✓"
        else
            log_error "$chain: ✗"
        fi
    done
}

deploy_backend_services() {
    log_step "Deploying backend services for $NETWORK..."

    cd "$PROJECT_ROOT"

    # Use appropriate deployment script
    if [ "$NETWORK" == "testnet" ]; then
        log_info "Starting Articium testnet deployment..."
        ./deploy-testnet.sh >> "$TEST_LOG" 2>&1
    else
        log_info "Starting Articium mainnet deployment..."
        ./deploy-mainnet.sh >> "$TEST_LOG" 2>&1
    fi

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
    log_step "Testing cross-chain transfers on $NETWORK..."

    # Test scenarios based on network
    if [ "$NETWORK" == "testnet" ]; then
        local test_cases=(
            "polygon-amoy:avalanche-fuji"
            "bnb-testnet:ethereum-sepolia"
            "avalanche-fuji:polygon-amoy"
            "ethereum-sepolia:bnb-testnet"
            "tron-nile:polygon-amoy"
            "polygon-amoy:bnb-testnet"
        )
    else
        local test_cases=(
            "polygon-mainnet:avalanche-mainnet"
            "bnb-mainnet:ethereum-mainnet"
            "ethereum-mainnet:polygon-mainnet"
        )
    fi

    for test_case in "${test_cases[@]}"; do
        IFS=':' read -r source dest <<< "$test_case"

        echo ""
        log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        log_info "Testing Cross-Chain Transfer: $source → $dest"
        log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

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
            log_success "Transfer initiated!"
            log_info "Message ID: $message_id"

            # Monitor transfer
            local status=$(monitor_transfer "$message_id" "$source" "$dest")

            if [ "$status" == "completed" ]; then
                log_success "✅ Transfer COMPLETED: $source → $dest"
                TRANSFER_RESULTS["$test_case"]="SUCCESS"

                # Show transaction details
                show_transaction_details "$message_id" "$source" "$dest"
            else
                log_warning "⏳ Transfer Status: $status"
                TRANSFER_RESULTS["$test_case"]="PENDING"
            fi
        else
            log_error "❌ Failed to initiate transfer"
            TRANSFER_RESULTS["$test_case"]="FAILED"
        fi

        sleep 5
    done
}

monitor_transfer() {
    local message_id=$1
    local source=$2
    local dest=$3
    local max_attempts=60  # 5 minutes
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        local response=$(curl -s "http://localhost:8080/v1/messages/$message_id")
        local status=$(echo "$response" | jq -r '.status')

        if [ "$status" == "completed" ]; then
            # Extract transaction hashes
            local source_tx=$(echo "$response" | jq -r '.source_tx_hash')
            local dest_tx=$(echo "$response" | jq -r '.destination_tx_hash')

            TX_HASHES["${message_id}_source"]="$source_tx"
            TX_HASHES["${message_id}_dest"]="$dest_tx"

            echo "completed"
            return 0
        elif [ "$status" == "failed" ]; then
            echo "failed"
            return 1
        fi

        attempt=$((attempt + 1))
        echo -n "."
        sleep 5
    done

    echo "pending"
    return 2
}

show_transaction_details() {
    local message_id=$1
    local source=$2
    local dest=$3

    local source_tx="${TX_HASHES[${message_id}_source]}"
    local dest_tx="${TX_HASHES[${message_id}_dest]}"

    echo ""
    log_info "📋 Transaction Details:"
    echo ""

    if [ -n "$source_tx" ] && [ "$source_tx" != "null" ]; then
        log_success "Source Transaction ($source):"
        log_info "   TX Hash: $source_tx"
        log_info "   🔗 Explorer: ${BLOCK_EXPLORERS[$source]}/tx/$source_tx"
    fi

    if [ -n "$dest_tx" ] && [ "$dest_tx" != "null" ]; then
        log_success "Destination Transaction ($dest):"
        log_info "   TX Hash: $dest_tx"
        log_info "   🔗 Explorer: ${BLOCK_EXPLORERS[$dest]}/tx/$dest_tx"
    fi

    echo ""
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
    local db_name="articium_$NETWORK"
    if docker exec articium-postgres psql -U articium -d "$db_name" -c "SELECT 1;" > /dev/null 2>&1; then
        log_success "Database connection: PASS"
    else
        log_warning "Database connection: WARNING (container may not exist)"
    fi

    log_success "System health verification complete"
    return 0
}

generate_test_report() {
    log_step "Generating test report..."

    local report_file="$RESULTS_DIR/e2e-report-$NETWORK-$TIMESTAMP.txt"

    cat > "$report_file" << EOF
========================================================================
Articium - E2E Test Report
========================================================================
Network: $NETWORK
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
TRANSACTION DETAILS & BLOCK EXPLORERS
========================================================================
EOF

    for key in "${!TX_HASHES[@]}"; do
        local tx_hash="${TX_HASHES[$key]}"
        if [ -n "$tx_hash" ] && [ "$tx_hash" != "null" ]; then
            echo "$key: $tx_hash" >> "$report_file"
        fi
    done

    cat >> "$report_file" << EOF

========================================================================
OVERALL RESULTS
========================================================================
EOF

    # Calculate success rate
    local total_deployments=0
    local successful_deployments=0

    for chain in "${CHAINS[@]}"; do
        if [ -n "${DEPLOYMENT_RESULTS[$chain]}" ]; then
            total_deployments=$((total_deployments + 1))
            if [ "${DEPLOYMENT_RESULTS[$chain]}" == "SUCCESS" ]; then
                successful_deployments=$((successful_deployments + 1))
            fi
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
Network: $NETWORK
Deployments: $successful_deployments/$total_deployments successful
Transfers: $successful_transfers/$total_transfers successful

Overall Status: $([ $successful_deployments -gt 0 ] && [ $successful_transfers -gt 0 ] && echo "PASS ✓" || echo "NEEDS REVIEW ⚠")

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
        if [ "$NETWORK" == "testnet" ]; then
            ./stop-testnet.sh
        else
            ./stop-mainnet.sh
        fi
        log_success "Services stopped"
    else
        log_info "Services left running for manual inspection"
    fi
}

# Main execution
main() {
    # Create log directories first (before any logging)
    mkdir -p "$LOG_DIR"
    mkdir -p "$RESULTS_DIR"
    mkdir -p "$PROJECT_ROOT/test-wallets"

    # Setup block explorers
    setup_block_explorers

    print_banner

    # Mainnet warning
    if [ "$NETWORK" == "mainnet" ]; then
        echo ""
        log_error "⚠️  ⚠️  ⚠️  WARNING  ⚠️  ⚠️  ⚠️"
        log_error "YOU ARE RUNNING IN MAINNET MODE!"
        log_error "THIS WILL USE REAL MONEY!"
        echo ""
        log_warning "Press Ctrl+C to cancel or ENTER to continue..."
        read -r -p "Press ENTER to confirm: " < /dev/tty
        echo ""
    fi

    # Step 1: Prerequisites
    if ! check_prerequisites; then
        log_error "Prerequisites check failed"
        exit 1
    fi

    # Step 2: Setup
    setup_test_environment

    # Step 3: Get tokens (faucets for testnet, real tokens for mainnet)
    show_faucet_links

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
    log_info "1. Review test report: $RESULTS_DIR/e2e-report-$NETWORK-$TIMESTAMP.txt"
    log_info "2. Check detailed logs: $TEST_LOG"
    log_info "3. Monitor services: http://localhost:8080"
    log_info "4. View Grafana: http://localhost:3000"
    echo ""

    if [ "$NETWORK" == "mainnet" ]; then
        log_warning "⚠️  MAINNET: Monitor all transactions carefully!"
        log_warning "⚠️  Start with small amounts and increase gradually!"
    fi
}

# Handle interrupts
trap cleanup INT TERM

# Run main function
main "$@"
