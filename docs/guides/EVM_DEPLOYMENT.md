# Articium EVM Contracts - Deployment Guide

Complete guide for deploying Articium bridge contracts to all supported EVM chains.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Setup](#setup)
3. [Get Testnet Tokens](#get-testnet-tokens)
4. [Deploy to Individual Chains](#deploy-to-individual-chains)
5. [Deploy to All Testnets](#deploy-to-all-testnets)
6. [Verify Contracts](#verify-contracts)
7. [Update Configuration](#update-configuration)
8. [Troubleshooting](#troubleshooting)

## Prerequisites

### Required Software
- Node.js 18+
- npm or yarn
- Git

### Required Accounts
- Wallet with private key (for deployment)
- RPC provider accounts (Alchemy, Infura, etc.)
- Block explorer API keys (for verification)

## Setup

### 1. Install Dependencies

```bash
cd contracts/evm
npm install
```

### 2. Configure Environment

```bash
# Copy example environment file
cp .env.example .env

# Edit .env with your values
nano .env
```

Fill in:
- `DEPLOYER_PRIVATE_KEY`: Your wallet private key (without 0x)
- `ALCHEMY_API_KEY`: Get from https://www.alchemy.com
- `INFURA_API_KEY`: Get from https://www.infura.io
- `POLYGONSCAN_API_KEY`: Get from https://polygonscan.com/apis
- `BSCSCAN_API_KEY`: Get from https://bscscan.com/apis
- `SNOWTRACE_API_KEY`: Get from https://snowtrace.io/apis

### 3. Generate Deployer Wallet (if needed)

```bash
# Using cast (Foundry)
cast wallet new

# Or using Node.js
node -e "console.log(require('crypto').randomBytes(32).toString('hex'))"
```

⚠️ **IMPORTANT**: Save your private key securely! Never commit it to git!

## Get Testnet Tokens

Before deploying, you need native tokens on each testnet:

### Polygon Amoy Testnet (MATIC)
- **Faucet**: https://faucet.polygon.technology/
- **Amount**: 1-2 MATIC
- **How to**:
  1. Visit faucet
  2. Connect wallet or paste address
  3. Select "Amoy Testnet"
  4. Click "Submit"
  5. Wait 1-2 minutes

### BNB Smart Chain Testnet (tBNB)
- **Faucet**: https://testnet.bnbchain.org/faucet-smart
- **Amount**: 0.5-1 tBNB
- **How to**:
  1. Visit faucet
  2. Paste your address
  3. Complete CAPTCHA
  4. Click "Give me BNB"
  5. Wait 1-2 minutes

### Avalanche Fuji Testnet (AVAX)
- **Faucet**: https://core.app/tools/testnet-faucet/
- **Amount**: 1-2 AVAX
- **How to**:
  1. Visit Core faucet
  2. Select "Fuji (C-Chain)"
  3. Paste your address
  4. Complete CAPTCHA
  5. Click "Request"
  6. Wait 1-2 minutes

**Alternative Fuji Faucet**:
- https://faucet.avax.network/ (requires Twitter/GitHub)

### Ethereum Sepolia Testnet (SepoliaETH)
- **Faucets**:
  - https://sepoliafaucet.com/
  - https://www.alchemy.com/faucets/ethereum-sepolia
  - https://faucet.quicknode.com/ethereum/sepolia
- **Amount**: 0.5-1 ETH
- **How to**:
  1. Choose a faucet
  2. Paste your address
  3. Complete verification (CAPTCHA/Twitter/GitHub)
  4. Wait 1-5 minutes

### Verify You Have Funds

```bash
# Check balances
npx hardhat run scripts/check-balance.js --network polygon-amoy
npx hardhat run scripts/check-balance.js --network bnb-testnet
npx hardhat run scripts/check-balance.js --network avalanche-fuji
npx hardhat run scripts/check-balance.js --network ethereum-sepolia
```

## Deploy to Individual Chains

### Polygon Amoy

```bash
npm run deploy:polygon-amoy
```

**Expected output:**
```
Network: polygon-amoy
Chain ID: 80002
Deployer address: 0x...
Deployer balance: 1.5 MATIC
✅ BridgeBase deployed to: 0x...
```

**Time**: ~2-3 minutes
**Gas cost**: ~0.1-0.2 MATIC

### BNB Smart Chain Testnet

```bash
npm run deploy:bnb-testnet
```

**Expected output:**
```
Network: bnb-testnet
Chain ID: 97
✅ BridgeBase deployed to: 0x...
```

**Time**: ~1-2 minutes
**Gas cost**: ~0.01-0.02 tBNB

### Avalanche Fuji

```bash
npm run deploy:avalanche-fuji
```

**Expected output:**
```
Network: avalanche-fuji
Chain ID: 43113
✅ BridgeBase deployed to: 0x...
```

**Time**: ~1-2 minutes
**Gas cost**: ~0.1-0.2 AVAX

### Ethereum Sepolia

```bash
npm run deploy:ethereum-sepolia
```

**Expected output:**
```
Network: ethereum-sepolia
Chain ID: 11155111
✅ BridgeBase deployed to: 0x...
```

**Time**: ~2-3 minutes
**Gas cost**: ~0.05-0.1 ETH

## Deploy to All Testnets

Deploy to all 4 testnets sequentially:

```bash
npm run deploy-all-testnet
```

This will:
1. Deploy to Polygon Amoy
2. Wait 5 seconds
3. Deploy to BNB Testnet
4. Wait 5 seconds
5. Deploy to Avalanche Fuji
6. Wait 5 seconds
7. Deploy to Ethereum Sepolia
8. Show summary

**Total time**: ~10-15 minutes
**Total gas cost**: ~0.3-0.5 USD equivalent

## Verify Contracts

After deployment, verify on block explorers:

### Automatic Verification (Recommended)

The deployment script attempts automatic verification. If it fails:

### Manual Verification

```bash
# Polygon Amoy
npx hardhat verify --network polygon-amoy <CONTRACT_ADDRESS> \
  '["0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0","0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199","0xdD2FD4581271e230360230F9337D5c0430Bf44C0"]' \
  2

# BNB Testnet
npx hardhat verify --network bnb-testnet <CONTRACT_ADDRESS> \
  '["0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0","0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199","0xdD2FD4581271e230360230F9337D5c0430Bf44C0"]' \
  2

# Avalanche Fuji
npx hardhat verify --network avalanche-fuji <CONTRACT_ADDRESS> \
  '["0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0","0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199","0xdD2FD4581271e230360230F9337D5c0430Bf44C0"]' \
  2

# Ethereum Sepolia
npx hardhat verify --network ethereum-sepolia <CONTRACT_ADDRESS> \
  '["0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0","0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199","0xdD2FD4581271e230360230F9337D5c0430Bf44C0"]' \
  2
```

### Check Verification Status

- **Polygon Amoy**: https://amoy.polygonscan.com/address/<CONTRACT_ADDRESS>
- **BNB Testnet**: https://testnet.bscscan.com/address/<CONTRACT_ADDRESS>
- **Avalanche Fuji**: https://testnet.snowtrace.io/address/<CONTRACT_ADDRESS>
- **Ethereum Sepolia**: https://sepolia.etherscan.io/address/<CONTRACT_ADDRESS>

## Update Configuration

After deployment, update the backend configuration:

### 1. Find Deployment Info

Contract addresses are saved in `deployments/` directory:

```bash
cat deployments/polygon-amoy_80002.json
cat deployments/bnb-testnet_97.json
cat deployments/avalanche-fuji_43113.json
cat deployments/ethereum-sepolia_11155111.json
```

### 2. Update config.testnet.yaml

```bash
cd ../../  # Back to project root
nano config/config.testnet.yaml
```

Update the `bridge_contract` fields:

```yaml
chains:
  - name: "polygon-amoy"
    bridge_contract: "0xYOUR_POLYGON_CONTRACT_ADDRESS"

  - name: "bnb-testnet"
    bridge_contract: "0xYOUR_BNB_CONTRACT_ADDRESS"

  - name: "avalanche-fuji"
    bridge_contract: "0xYOUR_AVALANCHE_CONTRACT_ADDRESS"

  - name: "ethereum-sepolia"
    bridge_contract: "0xYOUR_ETHEREUM_CONTRACT_ADDRESS"
```

### 3. Set Environment Variables

```bash
export POLYGON_AMOY_BRIDGE_CONTRACT="0x..."
export BNB_TESTNET_BRIDGE_CONTRACT="0x..."
export AVALANCHE_FUJI_BRIDGE_CONTRACT="0x..."
export ETHEREUM_SEPOLIA_BRIDGE_CONTRACT="0x..."
```

### 4. Restart Services

```bash
./stop-testnet.sh
./deploy-testnet.sh
```

## Troubleshooting

### "Insufficient funds" Error

**Problem**: Deployer doesn't have enough tokens

**Solution**:
```bash
# Check balance
npx hardhat run scripts/check-balance.js --network <NETWORK>

# Get more tokens from faucets (see above)
```

### "Nonce too high" Error

**Problem**: Transaction nonce mismatch

**Solution**:
```bash
# Reset account nonce in MetaMask/wallet
# Or wait a few minutes and try again
```

### "Gas price too low" Error

**Problem**: Network congestion

**Solution**:
Edit `hardhat.config.js` and increase gas price:
```javascript
gasPrice: 50000000000, // 50 Gwei (increase this)
```

### Verification Failed

**Problem**: Etherscan can't verify automatically

**Solution**:
1. Flatten the contract:
   ```bash
   npm run flatten
   ```

2. Manually verify on block explorer:
   - Go to contract page
   - Click "Contract" tab
   - Click "Verify and Publish"
   - Select "Solidity (Single file)"
   - Upload `flattened/BridgeBase_flat.sol`
   - Set compiler version: `0.8.20`
   - Set optimization: `Yes, 200 runs`

### RPC Rate Limiting

**Problem**: "Too many requests" error

**Solution**:
- Use paid RPC tier
- Add delays between deployments
- Use different RPC providers

### Transaction Taking Too Long

**Problem**: Stuck pending transaction

**Solution**:
```bash
# Check transaction on block explorer
# If stuck, try:
# 1. Increase gas price
# 2. Wait for network congestion to clear
# 3. Cancel and resubmit (in wallet)
```

## Gas Cost Estimates

| Network | Deployment Cost | Current Gas Price |
|---------|----------------|-------------------|
| Polygon Amoy | ~0.15 MATIC | ~30-50 Gwei |
| BNB Testnet | ~0.015 tBNB | ~10 Gwei |
| Avalanche Fuji | ~0.15 AVAX | ~25 Gwei |
| Ethereum Sepolia | ~0.08 ETH | ~20-50 Gwei |

**Total testnet cost**: ~$0.50 USD equivalent (with free faucet tokens)

## Mainnet Deployment

⚠️ **CRITICAL**: Mainnet deployment requires:

1. **Security audit** completed
2. **Multi-sig wallet** for deployment
3. **Sufficient funds** (0.5-1 ETH equivalent per chain)
4. **Validator keys** secured in HSM/KMS
5. **Emergency procedures** tested

**Mainnet commands**:
```bash
# ⚠️  ONLY run after security audit!
npm run deploy:polygon-mainnet
npm run deploy:bnb-mainnet
npm run deploy:avalanche-mainnet
npm run deploy:ethereum-mainnet
```

## Next Steps

After successful deployment:

1. ✅ Verify all contracts on block explorers
2. ✅ Update backend configuration
3. ✅ Test lock/unlock operations
4. ✅ Monitor events on each chain
5. ✅ Document contract addresses
6. ✅ Set up monitoring alerts

---

**Need help?** Check the main documentation or open an issue.
