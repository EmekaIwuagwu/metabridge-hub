# End-to-End Testing Guide

Complete guide for running automated E2E tests on the Metabridge Engine.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Test Script Features](#test-script-features)
- [Prerequisites](#prerequisites)
- [Running the E2E Test](#running-the-e2e-test)
- [What the Test Does](#what-the-test-does)
- [Manual Steps Required](#manual-steps-required)
- [Test Report](#test-report)
- [Troubleshooting](#troubleshooting)
- [Advanced Usage](#advanced-usage)

---

## Overview

The automated E2E test suite (`test-e2e-full.sh`) validates the entire Metabridge system from end to end:

1. ✅ Generates test wallet (or uses provided one)
2. ✅ Guides you through getting testnet tokens from faucets
3. ✅ Deploys smart contracts to all 4 testnets
4. ✅ Deploys and starts all backend services
5. ✅ Executes cross-chain token transfers
6. ✅ Verifies all operations completed successfully
7. ✅ Generates comprehensive test report

**Estimated time:** 30-60 minutes (including manual faucet requests)

---

## Quick Start

```bash
# Navigate to project root
cd /path/to/metabridge-hub

# Run E2E test
./test-e2e-full.sh

# Or with your own wallet
./test-e2e-full.sh 0xYourWalletAddress
```

---

## Test Script Features

### Automatic Features

- ✅ Wallet generation
- ✅ Balance checking across all chains
- ✅ Smart contract deployment to all testnets
- ✅ Backend service deployment
- ✅ Health checks and verification
- ✅ Cross-chain transfer testing
- ✅ Transfer monitoring and status tracking
- ✅ Comprehensive test reporting
- ✅ Cleanup and service management

### Semi-Automatic Features

- 🔄 Faucet token requests (requires manual CAPTCHA completion)
- 🔄 Balance verification (manual confirmation required)

---

## Prerequisites

### Software Requirements

```bash
# Check all prerequisites
node --version    # Should be 18.0.0 or higher
npm --version     # Should be 9.0.0 or higher
go version        # Should be 1.21 or higher
docker --version  # Should be 24.0 or higher
jq --version      # For JSON parsing
```

### Install Missing Prerequisites

#### Ubuntu/Debian

```bash
# Install jq
sudo apt-get update
sudo apt-get install -y jq

# Install Node.js 18
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs

# Install Go 1.21
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
```

#### macOS

```bash
# Install jq
brew install jq

# Install Node.js
brew install node@18

# Install Go
brew install go@1.21

# Install Docker
brew install --cask docker
```

### System Resources

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| RAM | 8GB | 16GB |
| Disk Space | 20GB | 50GB |
| CPU | 4 cores | 8 cores |

---

## Running the E2E Test

### Option 1: Auto-Generated Wallet (Recommended for Testing)

```bash
./test-e2e-full.sh
```

The script will:
- Generate a new test wallet
- Display the address
- Save wallet details to `test-wallets/wallet.json`

### Option 2: Use Your Own Wallet

```bash
./test-e2e-full.sh 0xYourWalletAddress
```

**Note:** You'll need the private key in `test-wallets/wallet.json` for contract deployment.

---

## What the Test Does

### Phase 1: Environment Setup (2-3 minutes)

```
[STEP] Checking prerequisites...
[SUCCESS] Node.js: v18.17.0
[SUCCESS] npm: 9.6.7
[SUCCESS] Go: go1.21.0
[SUCCESS] Docker: 24.0.6
[SUCCESS] jq: jq-1.6
[SUCCESS] All prerequisites met

[STEP] Setting up test environment...
[SUCCESS] Test wallet generated: 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0
[SUCCESS] Test environment ready
```

### Phase 2: Faucet Token Requests (10-20 minutes)

The script will display instructions for each faucet:

```
═══════════════════════════════════════════════════════════════
  MANUAL FAUCET REQUESTS REQUIRED
═══════════════════════════════════════════════════════════════

1. Polygon Amoy Testnet (MATIC)
   URL: https://faucet.polygon.technology/
   Address: 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0
   Network: POL (Amoy)
   Amount: 0.5-1 MATIC

2. BNB Smart Chain Testnet (tBNB)
   URL: https://testnet.bnbchain.org/faucet-smart
   Address: 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0
   Amount: 0.1-0.5 tBNB

...
```

**What to do:**
1. Open each URL in your browser
2. Paste your wallet address
3. Complete the CAPTCHA
4. Click "Submit" or "Request"
5. Wait 1-2 minutes for tokens to arrive
6. Repeat for all 4 chains
7. Return to terminal and press ENTER

### Phase 3: Balance Verification

```
[INFO] Checking balances for wallet: 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0
polygon-amoy: 1.0000
bnb-testnet: 0.5000
avalanche-fuji: 2.0000
ethereum-sepolia: 0.8000

Do you have sufficient tokens on all chains? (yes/no)
```

Type `yes` to continue or `no` to exit and get more tokens.

### Phase 4: Smart Contract Deployment (5-10 minutes)

```
[STEP] Deploying smart contracts to all testnets...

[INFO] Deploying to polygon-amoy...
[SUCCESS] polygon-amoy: Contract deployed

[INFO] Deploying to bnb-testnet...
[SUCCESS] bnb-testnet: Contract deployed

[INFO] Deploying to avalanche-fuji...
[SUCCESS] avalanche-fuji: Contract deployed

[INFO] Deploying to ethereum-sepolia...
[SUCCESS] ethereum-sepolia: Contract deployed

Deployment Summary:
polygon-amoy: ✓
bnb-testnet: ✓
avalanche-fuji: ✓
ethereum-sepolia: ✓
```

### Phase 5: Backend Service Deployment (3-5 minutes)

```
[STEP] Deploying backend services...
[INFO] Starting Metabridge testnet deployment...

======================================================================
  Metabridge Engine - Testnet Deployment
======================================================================

[INFO] Building Go services...
[SUCCESS] API built
[SUCCESS] Listener built
[SUCCESS] Relayer built

[INFO] Starting infrastructure services...
[SUCCESS] PostgreSQL is ready
[SUCCESS] NATS is ready
[SUCCESS] Redis is ready

[SUCCESS] Backend services deployed
```

### Phase 6: System Health Verification (1-2 minutes)

```
[STEP] Waiting for services to be ready...
[SUCCESS] Services are ready

[STEP] Verifying system health...
[SUCCESS] API health check: PASS
[SUCCESS] Chain configuration: PASS (4 chains)
[SUCCESS] Database connection: PASS
[SUCCESS] NATS connection: PASS
[SUCCESS] Redis connection: PASS
[SUCCESS] System health verification complete
```

### Phase 7: Cross-Chain Transfer Testing (10-15 minutes)

```
[STEP] Testing cross-chain transfers...

[INFO] Testing: polygon-amoy → avalanche-fuji
[SUCCESS] Transfer initiated: 0x1234567890abcdef
[SUCCESS] polygon-amoy → avalanche-fuji: COMPLETED ✓

[INFO] Testing: bnb-testnet → ethereum-sepolia
[SUCCESS] Transfer initiated: 0xabcdef1234567890
[SUCCESS] bnb-testnet → ethereum-sepolia: COMPLETED ✓

[INFO] Testing: avalanche-fuji → polygon-amoy
[SUCCESS] Transfer initiated: 0x567890abcdef1234
[SUCCESS] avalanche-fuji → polygon-amoy: COMPLETED ✓

[INFO] Testing: ethereum-sepolia → bnb-testnet
[SUCCESS] Transfer initiated: 0x90abcdef12345678
[SUCCESS] ethereum-sepolia → bnb-testnet: COMPLETED ✓
```

### Phase 8: Test Report Generation

```
[STEP] Generating test report...

========================================================================
Metabridge Engine - E2E Test Report
========================================================================
Timestamp: 20240115_143045
Test Wallet: 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0

========================================================================
SMART CONTRACT DEPLOYMENTS
========================================================================
polygon-amoy: SUCCESS
bnb-testnet: SUCCESS
avalanche-fuji: SUCCESS
ethereum-sepolia: SUCCESS

========================================================================
CROSS-CHAIN TRANSFERS
========================================================================
polygon-amoy:avalanche-fuji: SUCCESS
bnb-testnet:ethereum-sepolia: SUCCESS
avalanche-fuji:polygon-amoy: SUCCESS
ethereum-sepolia:bnb-testnet: SUCCESS

========================================================================
OVERALL RESULTS
========================================================================
Deployments: 4/4 successful
Transfers: 4/4 successful

Overall Status: PASS ✓

Test Log: logs/e2e/e2e-test-20240115_143045.log
Report: test-results/e2e-report-20240115_143045.txt
========================================================================
```

---

## Manual Steps Required

### 1. Faucet Token Requests

**Why manual?** Most faucets require CAPTCHA completion to prevent abuse.

**How long?** 10-20 minutes total

**What to do:**
- Follow the on-screen instructions
- Visit each faucet URL
- Complete CAPTCHA
- Wait for tokens to arrive (usually 1-2 minutes)

### 2. Balance Confirmation

**Why manual?** Ensures you have sufficient tokens before proceeding.

**What to do:**
- Review the displayed balances
- Type `yes` if all balances are sufficient
- Type `no` if you need more tokens

### 3. Cleanup Decision

**Why manual?** Allows you to keep services running for inspection.

**What to do:**
- Type `yes` to stop all services
- Type `no` to leave services running

---

## Test Report

The test generates two outputs:

### 1. Detailed Log File

Location: `logs/e2e/e2e-test-TIMESTAMP.log`

Contains:
- All command outputs
- Deployment transaction hashes
- Transfer message IDs
- Error messages (if any)

### 2. Summary Report

Location: `test-results/e2e-report-TIMESTAMP.txt`

Contains:
- Test execution summary
- Deployment results for each chain
- Transfer test results
- Overall pass/fail status

**Example:**

```bash
# View latest test report
cat test-results/e2e-report-*.txt | tail -50

# View detailed log
tail -100 logs/e2e/e2e-test-*.log
```

---

## Troubleshooting

### Issue: "jq command not found"

**Solution:**
```bash
# Ubuntu/Debian
sudo apt-get install jq

# macOS
brew install jq
```

### Issue: "Insufficient funds on [chain]"

**Solution:**
1. Visit the faucet again
2. Try alternative faucets (see `docs/guides/TESTNET_FAUCETS.md`)
3. Wait 24 hours if rate-limited
4. Ask in community Discord channels

### Issue: "Contract deployment failed"

**Common causes:**
- Insufficient gas
- Network congestion
- RPC endpoint issues

**Solution:**
```bash
# Check your balance
node test-wallets/check-balance.js YOUR_ADDRESS

# Try deploying to one chain at a time
cd contracts/evm
npm run deploy:polygon-amoy
```

### Issue: "Services failed to start"

**Solution:**
```bash
# Check Docker is running
docker ps

# Check logs
tail -f logs/api.log

# Restart services
./stop-testnet.sh
./deploy-testnet.sh
```

### Issue: "Transfer stuck in pending"

**Solution:**
- Wait longer (some testnets are slow)
- Check validator status
- Check blockchain explorer for transaction
- Review relayer logs: `tail -f logs/relayer.log`

### Issue: "Docker permission denied"

**Solution:**
```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Logout and login again
# Or run:
newgrp docker
```

---

## Advanced Usage

### Running Specific Test Phases

```bash
# Just deploy contracts (skip faucets)
cd contracts/evm
npm run deploy:polygon-amoy
npm run deploy:bnb-testnet
npm run deploy:avalanche-fuji
npm run deploy:ethereum-sepolia

# Just deploy backend
./deploy-testnet.sh

# Just test transfers
node scripts/cross-chain-transfer.js
```

### Using Different Test Amounts

Edit the test script and modify:

```bash
# Line ~320 in test-e2e-full.sh
"amount": "10000000000000000",  # 0.01 token (default)

# Change to:
"amount": "100000000000000000",  # 0.1 token
```

### Running Continuous Tests

```bash
# Run test in loop
while true; do
    ./test-e2e-full.sh 0xYourWallet
    sleep 3600  # Wait 1 hour
done
```

### Parallel Testing

```bash
# Run multiple test instances with different wallets
./test-e2e-full.sh 0xWallet1 &
./test-e2e-full.sh 0xWallet2 &
./test-e2e-full.sh 0xWallet3 &
wait
```

---

## Test Metrics

### Expected Results

| Metric | Target | Acceptable |
|--------|--------|-----------|
| Contract Deployment Success | 100% | ≥75% |
| Transfer Success Rate | 100% | ≥90% |
| Transfer Completion Time | <5 min | <15 min |
| API Response Time | <200ms | <1s |
| System Uptime | 100% | ≥99% |

### Performance Benchmarks

Based on typical testnet conditions:

- **Polygon Amoy**: 2-5 second confirmations
- **BNB Testnet**: 3-6 second confirmations
- **Avalanche Fuji**: 1-2 second confirmations
- **Ethereum Sepolia**: 12-15 second confirmations

**Cross-chain transfer time:** 1-5 minutes (depending on confirmation requirements)

---

## Cleanup

### After Testing

```bash
# Stop all services
./stop-testnet.sh

# Remove test data (optional)
rm -rf test-wallets/
rm -rf logs/e2e/
rm -rf test-results/

# Remove Docker volumes (⚠️ deletes all data)
docker-compose -f deployments/docker/docker-compose.infrastructure.yaml down -v
```

### Preserve Test Results

```bash
# Archive test results
tar -czf e2e-results-$(date +%Y%m%d).tar.gz test-results/ logs/e2e/

# Upload to storage
aws s3 cp e2e-results-*.tar.gz s3://your-bucket/test-results/
```

---

## Continuous Integration

### GitHub Actions Example

```yaml
# .github/workflows/e2e-test.yml
name: E2E Tests

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  e2e-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run E2E Tests
        run: ./test-e2e-full.sh ${{ secrets.TEST_WALLET }}
      - name: Upload Test Results
        uses: actions/upload-artifact@v3
        with:
          name: test-results
          path: test-results/
```

---

## Support

### Getting Help

- **Documentation**: `docs/guides/`
- **Discord**: (your Discord link)
- **GitHub Issues**: (your GitHub link)
- **Email**: support@yourdomain.com

### Reporting Issues

When reporting test failures, include:

1. Test report file
2. Detailed log file
3. System information (`uname -a`, `docker version`, etc.)
4. Steps to reproduce
5. Expected vs actual behavior

---

**Happy Testing! 🚀**

For more information, see:
- [Testnet Faucets Guide](TESTNET_FAUCETS.md)
- [EVM Deployment Guide](EVM_DEPLOYMENT.md)
- [Azure Deployment Guide](AZURE_DEPLOYMENT.md)
