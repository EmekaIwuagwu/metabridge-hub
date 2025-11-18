# Metabridge Engine - Production-Grade Multi-Chain Bridge Protocol

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![Solidity](https://img.shields.io/badge/Solidity-0.8.20-orange.svg)](https://soliditylang.org)

**Metabridge** is a production-ready, enterprise-grade cross-chain messaging and asset bridge protocol written in Golang that supports **heterogeneous blockchain architectures** across both **testnet and mainnet** environments.

## 🌟 Key Features

### Multi-Chain Support
- **6 Blockchain Networks** with full testnet and mainnet configurations
- **EVM Chains**: Polygon, BNB Smart Chain, Avalanche, Ethereum
- **Non-EVM Chains**: Solana, NEAR Protocol

### Cross-Platform Capabilities
- ✅ Different signature schemes (ECDSA for EVM, Ed25519 for Solana/NEAR)
- ✅ Varied finality models (probabilistic vs deterministic)
- ✅ Transaction model abstraction (account-based and UTXO-like)
- ✅ Cross-platform token standards (ERC-20/721, SPL, NEP-141/171)
- ✅ Environment-aware security (2-of-3 testnet, 3-of-5 mainnet)

### Production Features
- 🔐 Multi-signature validation
- 🚨 Emergency pause mechanism
- 📊 Comprehensive monitoring and metrics
- 🔄 Automatic failover and retry logic
- ⚡ High-availability architecture
- 🛡️ Rate limiting and fraud detection
- 📈 Real-time statistics and analytics

---

## 📋 Table of Contents

- [Architecture](#architecture)
- [Self-Hosted Relayer System](#self-hosted-relayer-system)
- [Supported Networks](#supported-networks)
- [Prerequisites](#prerequisites)
- [🚀 Quick Deploy on Render (Recommended for Testing)](#-quick-deploy-on-render-recommended-for-testing)
- [Azure Production Deployment](#azure-production-deployment)
- [Quick Start](#quick-start)
- [Testnet Deployment](#testnet-deployment)
- [Mainnet Deployment](#mainnet-deployment)
- [Configuration](#configuration)
- [API Documentation](#api-documentation)
- [Monitoring](#monitoring)
- [Security](#security)
- [Testing](#testing)

---

## 🏗️ Architecture

```
┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│   Polygon   │         │   Solana    │         │    NEAR     │
│  (EVM)      │         │ (Non-EVM)   │         │  (Non-EVM)  │
└──────┬──────┘         └──────┬──────┘         └──────┬──────┘
       │                       │                        │
       │                       │                        │
       └───────────────────────┼────────────────────────┘
                               │
                    ┌──────────▼──────────┐
                    │   Event Listeners   │
                    │  (Multi-Chain)      │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │   Message Queue     │
                    │   (NATS JetStream)  │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │   Relayer Service   │
                    │  (Multi-Sig)        │
                    └──────────┬──────────┘
                               │
       ┌───────────────────────┼────────────────────────┐
       │                       │                        │
┌──────▼──────┐         ┌──────▼──────┐         ┌──────▼──────┐
│    BNB      │         │  Avalanche  │         │  Ethereum   │
│   (EVM)     │         │   (EVM)     │         │   (EVM)     │
└─────────────┘         └─────────────┘         └─────────────┘
```

### Components

1. **Blockchain Clients**: Universal interface supporting EVM, Solana, and NEAR
2. **Event Listeners**: Monitor and decode events from all supported chains
3. **Message Queue**: NATS JetStream for reliable message delivery
4. **Relayer**: Processes cross-chain messages with multi-sig validation
5. **API Server**: RESTful API for bridge operations and status queries
6. **Database**: PostgreSQL for persistent state and audit logs
7. **Cache**: Redis for performance optimization
8. **Monitoring**: Prometheus + Grafana for observability

---

## 🔄 Self-Hosted Relayer System

Metabridge includes a **production-ready, self-hosted relayer** that eliminates dependency on third-party relayer networks. You control the entire message relay infrastructure.

### Relayer Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Blockchain Networks                       │
│  EVM (Polygon, BNB, Avalanche, ETH) | Solana | NEAR         │
└────────────────────┬────────────────────────────────────────┘
                     │ Lock/Burn Events
          ┌──────────▼──────────────┐
          │    Event Listeners      │
          │  - EVM Listener         │
          │  - Solana Listener      │
          │  - NEAR Listener        │
          └──────────┬──────────────┘
                     │ Parse Events
          ┌──────────▼──────────────┐
          │    NATS Queue           │
          │  (Message Persistence)  │
          └──────────┬──────────────┘
                     │ Dequeue
          ┌──────────▼──────────────┐
          │   Relayer Workers       │
          │  (Configurable Pool)    │
          └──────────┬──────────────┘
                     │
          ┌──────────▼──────────────┐
          │    Message Processor    │
          │  - Validate signatures  │
          │  - Check security rules │
          │  - Build transactions   │
          └──────────┬──────────────┘
                     │ Sign & Broadcast
          ┌──────────▼──────────────┐
          │  Destination Chains     │
          │  Unlock/Mint Assets     │
          └─────────────────────────┘
```

### Key Features

✅ **Multi-Chain Support**: EVM, Solana, and NEAR processing
✅ **Worker Pool**: Configurable concurrent message processing
✅ **Fault Tolerance**: Automatic retry with exponential backoff
✅ **Health Monitoring**: Per-chain health checks every 30 seconds
✅ **Queue Persistence**: Messages survive relayer restarts
✅ **Multi-Signature Verification**: Validates validator signatures before relay
✅ **Transaction Confirmation**: Waits for blockchain confirmations
✅ **Metrics & Monitoring**: Prometheus metrics for all operations

### Relayer Components

#### 1. Event Listeners

**EVM Listener** (`internal/listener/evm/listener.go`):
- Polls EVM chains for bridge contract events
- Handles block confirmations (128-256 blocks)
- Decodes `TokenLocked` and `NFTLocked` events
- Batch processing (100 blocks at a time)

**Solana Listener** (`internal/listener/solana/listener.go`):
- Monitors Solana program accounts for lock events
- Handles slot confirmations (32 slots)
- Parses account data for bridge events
- Supports SPL token and Metaplex NFT standards

**NEAR Listener** (`internal/listener/near/listener.go`):
- Queries NEAR contract events via RPC
- Handles block confirmations (3 blocks)
- Parses NEP-141 (token) and NEP-171 (NFT) events
- Compatible with NEAR Indexer integration

#### 2. Message Processor

**Security Validation**:
```go
// Validates before processing
- Multi-signature verification (2-of-3 testnet, 3-of-5 mainnet)
- Transaction limit checks
- Daily volume limits
- Rate limiting per sender
- Duplicate message detection
```

**EVM Transaction Building**:
```solidity
// Calls bridge contract unlock function
unlockToken(
    bytes32 messageId,
    address recipient,
    address token,
    uint256 amount,
    bytes[] signatures
)
```

**Solana Transaction Building**:
```rust
// Builds Solana instruction
- Program ID: Bridge program
- Accounts: [relayer, vault_pda, recipient_ata, token_mint, token_program]
- Data: [discriminator, message_id, amount, signatures]
- Uses Associated Token Accounts (ATA) for recipients
```

**NEAR Transaction Building**:
```rust
// Builds NEAR function call
{
  "method_name": "unlock_token",
  "args": {
    "message_id": "...",
    "recipient": "user.near",
    "token": "token.near",
    "amount": "1000000",
    "signatures": ["sig1", "sig2", "sig3"]
  },
  "gas": 100000000000000,
  "deposit": "0"
}
```

### Running the Relayer

#### Development Mode

```bash
# Start relayer with testnet config
./relayer --config config/config.testnet.yaml
```

#### Production Mode (Systemd)

```bash
# Create systemd service
sudo cat > /etc/systemd/system/metabridge-relayer.service <<EOF
[Unit]
Description=Metabridge Relayer Service
After=network.target postgresql.service nats.service

[Service]
Type=simple
User=bridge
WorkingDirectory=/opt/metabridge
ExecStart=/opt/metabridge/relayer --config /opt/metabridge/config/config.mainnet.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
Environment="BRIDGE_ENVIRONMENT=mainnet"

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl enable metabridge-relayer
sudo systemctl start metabridge-relayer

# Check status
sudo systemctl status metabridge-relayer
sudo journalctl -u metabridge-relayer -f
```

#### Docker Deployment

```bash
# Build relayer image
docker build -t metabridge-relayer:latest -f Dockerfile.relayer .

# Run relayer container
docker run -d \
  --name metabridge-relayer \
  --network metabridge-network \
  -v /opt/metabridge/config:/config:ro \
  -v /opt/metabridge/keys:/keys:ro \
  -e BRIDGE_ENVIRONMENT=mainnet \
  metabridge-relayer:latest \
  --config /config/config.mainnet.yaml
```

### Relayer Configuration

```yaml
# config/config.mainnet.yaml
relayer:
  workers: 10                    # Number of concurrent workers
  message_batch_size: 50         # Messages to process per batch
  retry_attempts: 3              # Retry failed messages
  retry_delay: "30s"             # Delay between retries
  confirmation_timeout: "5m"     # Wait time for confirmations
  health_check_interval: "30s"   # Health check frequency

security:
  required_signatures: 3         # Multi-sig threshold
  max_transaction_value: 1000000 # Max value in USD
  daily_volume_limit: 10000000   # Daily limit in USD
  rate_limit_per_minute: 20      # Rate limit per sender
```

### Monitoring Relayer Health

**Prometheus Metrics**:
```prometheus
# Messages processed
bridge_relayer_messages_processed_total{source, destination}

# Processing duration
bridge_relayer_message_processing_duration_seconds{source, destination}

# Failed messages
bridge_relayer_messages_failed_total{source, destination, reason}

# Queue depth
bridge_queue_size
bridge_queue_consumers

# Chain health
bridge_chain_health{chain} # 1 = healthy, 0 = unhealthy
bridge_chain_block_number{chain}
```

**Grafana Alerts**:
- Alert when chain health = 0 for > 5 minutes
- Alert when failed messages > 10 in 10 minutes
- Alert when processing duration > 60 seconds
- Alert when queue depth > 1000 messages

### Scaling the Relayer

**Horizontal Scaling**:
```bash
# Run multiple relayer instances
# Each worker pool processes from shared NATS queue
# Messages are load-balanced automatically

# Instance 1
./relayer --config config.yaml --workers 10

# Instance 2
./relayer --config config.yaml --workers 10

# Instance 3
./relayer --config config.yaml --workers 10
```

**Vertical Scaling**:
```yaml
# Increase workers per instance
relayer:
  workers: 50  # Adjust based on CPU cores
```

### Transaction Signing

**Development (Private Keys)**:
```go
// Load from environment/config
signer, _ := evmCrypto.NewECDSASigner(privateKeyHex)
```

**Production (AWS KMS)**:
```go
// Use AWS KMS for secure key management
signer, _ := kms.NewKMSSigner(kmsKeyID)
```

**Production (Hardware Security Module)**:
```go
// Use HSM for maximum security
signer, _ := hsm.NewHSMSigner(hsmConfig)
```

### Troubleshooting

**Relayer not processing messages**:
1. Check NATS connection: `nats stream info BRIDGE_MESSAGES`
2. Check database connection: `psql -U bridge_user -d metabridge`
3. Check RPC endpoints: View health metrics
4. Check logs: `journalctl -u metabridge-relayer -f`

**Transactions failing on destination**:
1. Verify signer has sufficient gas funds
2. Check destination chain RPC is responsive
3. Verify bridge contract addresses are correct
4. Check transaction nonce management

**High processing latency**:
1. Increase worker count
2. Optimize RPC endpoint selection
3. Check database query performance
4. Review gas price settings

---

## 🌐 Supported Networks

### Testnet Configurations

| Chain | Network | Chain ID | RPC Endpoint | Confirmations |
|-------|---------|----------|--------------|---------------|
| **Polygon** | Amoy | 80002 | https://rpc-amoy.polygon.technology/ | 128 |
| **BNB** | Testnet | 97 | https://data-seed-prebsc-1-s1.binance.org:8545/ | 15 |
| **Avalanche** | Fuji | 43113 | https://api.avax-test.network/ext/bc/C/rpc | 10 |
| **Ethereum** | Sepolia | 11155111 | https://sepolia.infura.io/v3/YOUR-KEY | 32 |
| **Solana** | Devnet | - | https://api.devnet.solana.com | 32 slots |
| **NEAR** | Testnet | - | https://rpc.testnet.near.org | 3 blocks |

### Mainnet Configurations

| Chain | Network | Chain ID | RPC Endpoint | Confirmations |
|-------|---------|----------|--------------|---------------|
| **Polygon** | Mainnet | 137 | https://polygon-rpc.com/ | 256 |
| **BNB** | Mainnet | 56 | https://bsc-dataseed.binance.org/ | 30 |
| **Avalanche** | C-Chain | 43114 | https://api.avax.network/ext/bc/C/rpc | 20 |
| **Ethereum** | Mainnet | 1 | https://mainnet.infura.io/v3/YOUR-KEY | 64 |
| **Solana** | Mainnet-Beta | - | https://api.mainnet-beta.solana.com | 32 slots |
| **NEAR** | Mainnet | - | https://rpc.mainnet.near.org | 3 blocks |

---

## 📦 Prerequisites

### Software Requirements

- **Go**: 1.21 or higher
- **Node.js**: 18.x or higher (for smart contract deployment)
- **Docker**: 20.10 or higher
- **Docker Compose**: 2.0 or higher
- **PostgreSQL**: 15.x
- **Redis**: 7.x
- **NATS**: 2.10 or higher

### For Smart Contract Deployment

- **Hardhat**: For EVM contracts
- **Anchor**: For Solana programs
- **Rust**: For NEAR contracts

### API Keys Required

- Alchemy API Key (for EVM chains)
- Infura API Key (for Ethereum)
- Helius API Key (for Solana)

---

## 🚀 Quick Deploy on Render (Recommended for Testing)

**Want to test the bridge in 10 minutes without managing servers?** Use Render!

### What is Render Deployment?

- ✅ **No server management**: Deploy with clicks, no SSH or Docker
- ✅ **Free tier**: Test for free (with 15-min sleep on inactivity)
- ✅ **Automatic HTTPS**: Free SSL certificates
- ✅ **Git-based**: Push to GitHub = auto-deploy
- ✅ **Managed database**: PostgreSQL included
- ✅ **$0-$21/month**: Much cheaper than Azure ($140/month)

### Quick Start (5 Steps)

1. **Push to GitHub** (if not already)
   ```bash
   git remote add origin https://github.com/YOUR_USERNAME/metabridge-engine-hub.git
   git push origin main
   ```

2. **Sign up for Render** → https://render.com (free account)

3. **Create PostgreSQL Database**
   - Click "New +" → "PostgreSQL"
   - Name: `metabridge-db`
   - Plan: Free (for testing)
   - Copy the "Internal Database URL"

4. **Deploy API Server**
   - Click "New +" → "Web Service"
   - Connect your GitHub repo
   - Build Command: `go build -o bin/metabridge-api cmd/api/main.go`
   - Start Command: `./bin/metabridge-api`
   - Add environment variables (see full guide)

5. **Test Your Deployment**
   ```bash
   curl https://your-app.onrender.com/health
   # Should return: {"status":"ok","version":"1.0.0"}
   ```

### Full Render Deployment Guide

👉 **See [RENDER_DEPLOYMENT.md](./RENDER_DEPLOYMENT.md)** for complete step-by-step instructions

This guide includes:
- Complete environment variable setup
- Database migration steps
- API server deployment
- Relayer (background worker) deployment
- Testing and monitoring
- Cost breakdown (free vs paid)
- Troubleshooting

### When to Use Render vs Azure

| Use Case | Recommended Platform |
|----------|---------------------|
| **"I want to test how this works"** | ✅ **Render** (10 min setup, $0) |
| **"I'm developing/prototyping"** | ✅ **Render** (easy iteration) |
| **"I need low-cost testnet"** | ✅ **Render** ($0-$21/month) |
| **"I need production for <1000 users"** | ✅ **Render** ($21-$50/month) |
| **"I need enterprise production"** | ✅ **Azure** (full control, $140+/month) |
| **"I need custom infrastructure"** | ✅ **Azure** (VMs, networking, etc.) |

### Cost Comparison

| Tier | Render | Azure |
|------|--------|-------|
| **Free Testing** | ✅ $0 (with sleep) | ❌ N/A |
| **Basic Production** | ✅ $21/month | ❌ ~$140/month |
| **Professional** | $110/month | $280/month |

### What's Deployed on Render

When you deploy on Render, you get:

1. **API Server** (`https://your-app.onrender.com`)
   - REST API for bridge operations
   - Authentication & authorization
   - Chain status endpoints
   - Transaction tracking

2. **Relayer** (Background Worker)
   - Processes cross-chain messages
   - Validates signatures
   - Broadcasts transactions

3. **PostgreSQL Database** (Managed)
   - Message storage
   - User authentication
   - Audit logs

4. **Automatic Features**
   - HTTPS/SSL certificate
   - Auto-deploy on git push
   - Built-in monitoring
   - Log aggregation

### Limitations of Render Free Tier

- **Sleep after 15 min**: First request wakes it up (30-50s delay)
- **512 MB RAM**: Enough for testing
- **Shared CPU**: Slower than dedicated
- **100 GB bandwidth/month**: Plenty for testing

**Solution**: Upgrade to Starter ($7/month) to remove sleep and get more resources

---

## 🔷 Azure Production Deployment

Complete step-by-step guide to deploy Metabridge on Azure from scratch.

### Prerequisites

- Azure account with active subscription
- SSH key pair generated on your local machine
- Domain name (optional, for HTTPS)

### Step 1: Create Azure VM

```bash
# From Azure Portal or CLI
az vm create \
  --resource-group metabridge-rg \
  --name metabridge-vm \
  --image Ubuntu2204 \
  --size Standard_D4s_v3 \
  --admin-username bridge \
  --ssh-key-values ~/.ssh/id_rsa.pub \
  --public-ip-sku Standard

# Get the public IP
az vm show -d -g metabridge-rg -n metabridge-vm --query publicIps -o tsv
```

**Recommended VM Sizes**:
- **Testnet**: Standard_D2s_v3 (2 vCPU, 8 GB RAM) - $70/month
- **Production**: Standard_D4s_v3 (4 vCPU, 16 GB RAM) - $140/month
- **High Volume**: Standard_D8s_v3 (8 vCPU, 32 GB RAM) - $280/month

### Step 2: Connect via SSH

```bash
# SSH into your Azure VM
ssh bridge@<YOUR_VM_PUBLIC_IP>

# Or if you saved the IP
export BRIDGE_VM_IP=<YOUR_VM_PUBLIC_IP>
ssh bridge@$BRIDGE_VM_IP
```

### Step 3: System Preparation

```bash
# Update package lists
sudo apt update

# Upgrade all packages
sudo apt upgrade -y

# Install essential tools
sudo apt install -y \
  curl \
  wget \
  git \
  build-essential \
  jq \
  unzip \
  software-properties-common \
  apt-transport-https \
  ca-certificates \
  gnupg \
  lsb-release

# Set timezone (recommended)
sudo timedatectl set-timezone UTC
```

### Step 4: Install Go 1.21+

```bash
# Download Go
cd ~
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz

# Extract and install
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
source ~/.bashrc

# Verify installation
go version
# Should output: go version go1.21.6 linux/amd64
```

### Step 5: Install Node.js 18+

```bash
# Install Node.js via NodeSource
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt install -y nodejs

# Verify installation
node --version  # Should be v18.x.x
npm --version   # Should be 9.x.x or higher

# Install Yarn (optional, for faster package management)
npm install -g yarn
```

### Step 6: Install Docker & Docker Compose

```bash
# Add Docker's official GPG key
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

# Add Docker repository
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install Docker
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Add user to docker group
sudo usermod -aG docker $USER

# Apply group changes (or logout/login)
newgrp docker

# Verify installation
docker --version
docker compose version
```

### Step 7: Clone Repository

```bash
# Create project directory
mkdir -p ~/metabridge
cd ~/metabridge

# Clone the repository
git clone https://github.com/EmekaIwuagwu/metabridge-engine-hub.git
cd metabridge-engine-hub

# Check current branch
git branch
git status
```

### Step 8: Install Project Dependencies

```bash
# Install Go dependencies
go mod download
go mod verify

# Install smart contract dependencies (EVM)
cd contracts/evm
npm install

# Return to project root
cd ~/metabridge/metabridge-engine-hub
```

### Step 9: Configure Environment

```bash
# Copy environment template
cp .env.example .env.production

# Edit environment file
nano .env.production
```

**Required environment variables**:

```bash
# Environment
BRIDGE_ENVIRONMENT=production

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=bridge_user
DB_PASSWORD=<GENERATE_STRONG_PASSWORD>
DB_NAME=metabridge_production
DB_SSLMODE=disable

# Server Configuration
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# JWT Authentication (generate with: openssl rand -hex 32)
JWT_SECRET=<YOUR_64_CHAR_SECRET_HERE>
JWT_EXPIRATION_HOURS=24

# CORS (your frontend domain)
CORS_ALLOWED_ORIGINS=https://yourdomain.com,https://app.yourdomain.com

# Rate Limiting
RATE_LIMIT_PER_MINUTE=100
REQUIRE_AUTH=true
API_KEY_ENABLED=true

# RPC Endpoints (use your own API keys)
ALCHEMY_API_KEY=<YOUR_ALCHEMY_KEY>
INFURA_API_KEY=<YOUR_INFURA_KEY>
HELIUS_API_KEY=<YOUR_HELIUS_KEY>

# Chain RPC URLs
POLYGON_RPC_URL=https://polygon-mainnet.g.alchemy.com/v2/${ALCHEMY_API_KEY}
BNB_RPC_URL=https://bsc-dataseed.binance.org/
AVALANCHE_RPC_URL=https://api.avax.network/ext/bc/C/rpc
ETHEREUM_RPC_URL=https://mainnet.infura.io/v3/${INFURA_API_KEY}
SOLANA_RPC_URL=https://api.mainnet-beta.solana.com
NEAR_RPC_URL=https://rpc.mainnet.near.org

# Smart Contract Addresses (fill after deployment)
POLYGON_BRIDGE_CONTRACT=
BNB_BRIDGE_CONTRACT=
AVALANCHE_BRIDGE_CONTRACT=
ETHEREUM_BRIDGE_CONTRACT=
SOLANA_BRIDGE_PROGRAM=
NEAR_BRIDGE_CONTRACT=

# Validator Configuration (use secure key management in production)
VALIDATOR_PRIVATE_KEY=<YOUR_VALIDATOR_KEY>

# NATS Configuration
NATS_URL=nats://localhost:4222

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
```

### Step 10: Start Infrastructure Services

```bash
# Start PostgreSQL, NATS, and Redis
docker compose -f docker-compose.production.yaml up -d postgres nats redis

# Wait for services to be ready (30 seconds)
sleep 30

# Check service status
docker compose -f docker-compose.production.yaml ps

# Should show postgres, nats, and redis as "running"
```

### Step 11: Initialize Database

```bash
# Create database and user
docker exec -i metabridge-postgres psql -U postgres <<EOF
CREATE DATABASE metabridge_production;
CREATE USER bridge_user WITH ENCRYPTED PASSWORD '<YOUR_DB_PASSWORD>';
GRANT ALL PRIVILEGES ON DATABASE metabridge_production TO bridge_user;
ALTER DATABASE metabridge_production OWNER TO bridge_user;
\q
EOF

# Run database migrations
docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production < internal/database/schema.sql

# Run authentication schema
docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production < internal/database/auth.sql

# Verify tables created
docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c "\dt"
```

### Step 12: Create Admin User

```bash
# Generate password hash (install bcrypt tool)
go install github.com/bitnami/bcrypt-cli@latest

# Hash your admin password
bcrypt-cli <YOUR_ADMIN_PASSWORD>
# Copy the hash output

# Insert admin user
docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production <<EOF
INSERT INTO users (id, email, name, password_hash, role, active, created_at, updated_at)
VALUES (
  'admin-001',
  'admin@metabridge.local',
  'System Administrator',
  '<YOUR_BCRYPT_HASH>',
  'admin',
  true,
  NOW(),
  NOW()
);
EOF

# Verify user created
docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c "SELECT id, email, role FROM users;"
```

### Step 13: Deploy Smart Contracts

#### EVM Contracts (Polygon, BNB, Avalanche, Ethereum)

```bash
cd ~/metabridge/metabridge-engine-hub/contracts/evm

# Create deployment configuration
cat > hardhat.config.js <<'EOF'
require("@nomicfoundation/hardhat-toolbox");
require("hardhat-deploy");

module.exports = {
  solidity: "0.8.20",
  networks: {
    polygon: {
      url: process.env.POLYGON_RPC_URL,
      accounts: [process.env.DEPLOYER_PRIVATE_KEY],
    },
    bsc: {
      url: process.env.BNB_RPC_URL,
      accounts: [process.env.DEPLOYER_PRIVATE_KEY],
    },
    avalanche: {
      url: process.env.AVALANCHE_RPC_URL,
      accounts: [process.env.DEPLOYER_PRIVATE_KEY],
    },
    ethereum: {
      url: process.env.ETHEREUM_RPC_URL,
      accounts: [process.env.DEPLOYER_PRIVATE_KEY],
    },
  },
};
EOF

# Load environment
export $(cat ../../.env.production | grep -v '^#' | xargs)

# Deploy to Polygon
npx hardhat deploy --network polygon --tags Bridge
export POLYGON_BRIDGE=$(cat deployments/polygon/BridgeBase.json | jq -r '.address')
echo "Polygon Bridge: $POLYGON_BRIDGE"

# Deploy to BNB
npx hardhat deploy --network bsc --tags Bridge
export BNB_BRIDGE=$(cat deployments/bsc/BridgeBase.json | jq -r '.address')
echo "BNB Bridge: $BNB_BRIDGE"

# Deploy to Avalanche
npx hardhat deploy --network avalanche --tags Bridge
export AVALANCHE_BRIDGE=$(cat deployments/avalanche/BridgeBase.json | jq -r '.address')
echo "Avalanche Bridge: $AVALANCHE_BRIDGE"

# Deploy to Ethereum
npx hardhat deploy --network ethereum --tags Bridge
export ETHEREUM_BRIDGE=$(cat deployments/ethereum/BridgeBase.json | jq -r '.address')
echo "Ethereum Bridge: $ETHEREUM_BRIDGE"

# Update .env.production with contract addresses
echo "POLYGON_BRIDGE_CONTRACT=$POLYGON_BRIDGE" >> ../../.env.production
echo "BNB_BRIDGE_CONTRACT=$BNB_BRIDGE" >> ../../.env.production
echo "AVALANCHE_BRIDGE_CONTRACT=$AVALANCHE_BRIDGE" >> ../../.env.production
echo "ETHEREUM_BRIDGE_CONTRACT=$ETHEREUM_BRIDGE" >> ../../.env.production
```

#### Solana Program (Optional)

```bash
# Install Solana CLI
sh -c "$(curl -sSfL https://release.solana.com/stable/install)"
export PATH="$HOME/.local/share/solana/install/active_release/bin:$PATH"

# Install Anchor
cargo install --git https://github.com/coral-xyz/anchor --tag v0.29.0 anchor-cli

cd ~/metabridge/metabridge-engine-hub/contracts/solana

# Build program
anchor build

# Set Solana to mainnet
solana config set --url https://api.mainnet-beta.solana.com

# Deploy (requires SOL for deployment)
anchor deploy

# Get program ID
solana address -k target/deploy/bridge-keypair.json
```

#### NEAR Contract (Optional)

```bash
# Install NEAR CLI
npm install -g near-cli

cd ~/metabridge/metabridge-engine-hub/contracts/near

# Build contract
./build.sh

# Deploy to NEAR mainnet
near deploy --accountId bridge.near --wasmFile res/bridge.wasm

# Initialize contract
near call bridge.near new '{"owner":"validator.near","required_signatures":3}' --accountId bridge.near
```

### Step 14: Build Bridge Services

```bash
cd ~/metabridge/metabridge-engine-hub

# Build API server
go build -o bin/metabridge-api cmd/api/main.go

# Build relayer
go build -o bin/metabridge-relayer cmd/relayer/main.go

# Build batcher (if exists)
go build -o bin/metabridge-batcher cmd/batcher/main.go

# Verify binaries
ls -lh bin/
```

### Step 15: Create Systemd Services

**API Server Service**:

```bash
sudo tee /etc/systemd/system/metabridge-api.service > /dev/null <<EOF
[Unit]
Description=Metabridge API Server
After=network.target postgresql.service nats.service redis.service

[Service]
Type=simple
User=bridge
WorkingDirectory=/home/bridge/metabridge/metabridge-engine-hub
ExecStart=/home/bridge/metabridge/metabridge-engine-hub/bin/metabridge-api
EnvironmentFile=/home/bridge/metabridge/metabridge-engine-hub/.env.production
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```

**Relayer Service**:

```bash
sudo tee /etc/systemd/system/metabridge-relayer.service > /dev/null <<EOF
[Unit]
Description=Metabridge Relayer Service
After=network.target postgresql.service nats.service metabridge-api.service

[Service]
Type=simple
User=bridge
WorkingDirectory=/home/bridge/metabridge/metabridge-engine-hub
ExecStart=/home/bridge/metabridge/metabridge-engine-hub/bin/metabridge-relayer --config /home/bridge/metabridge/metabridge-engine-hub/config/config.mainnet.yaml
EnvironmentFile=/home/bridge/metabridge/metabridge-engine-hub/.env.production
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```

**Enable and start services**:

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable services to start on boot
sudo systemctl enable metabridge-api
sudo systemctl enable metabridge-relayer

# Start services
sudo systemctl start metabridge-api
sudo systemctl start metabridge-relayer

# Check status
sudo systemctl status metabridge-api
sudo systemctl status metabridge-relayer

# View logs
sudo journalctl -u metabridge-api -f --lines=50
sudo journalctl -u metabridge-relayer -f --lines=50
```

### Step 16: Configure Firewall

```bash
# Install UFW if not present
sudo apt install -y ufw

# Allow SSH (IMPORTANT: Do this first!)
sudo ufw allow 22/tcp

# Allow API server
sudo ufw allow 8080/tcp

# Allow Prometheus (if using external monitoring)
sudo ufw allow 9090/tcp

# Allow Grafana (if using external access)
sudo ufw allow 3000/tcp

# Enable firewall
sudo ufw --force enable

# Check status
sudo ufw status verbose
```

### Step 17: Set Up HTTPS (Optional but Recommended)

```bash
# Install Nginx
sudo apt install -y nginx

# Install Certbot for Let's Encrypt
sudo apt install -y certbot python3-certbot-nginx

# Create Nginx configuration
sudo tee /etc/nginx/sites-available/metabridge > /dev/null <<EOF
server {
    listen 80;
    server_name api.yourdomain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_cache_bypass \$http_upgrade;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

# Enable site
sudo ln -s /etc/nginx/sites-available/metabridge /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx

# Get SSL certificate
sudo certbot --nginx -d api.yourdomain.com

# Auto-renewal is set up automatically
sudo certbot renew --dry-run
```

### Step 18: Run Integration Tests

```bash
cd ~/metabridge/metabridge-engine-hub

# Set test environment
export TEST_ENV=production
export $(cat .env.production | grep -v '^#' | xargs)

# Run unit tests
go test ./internal/... -v

# Run integration tests
go test ./tests/integration/... -v

# Run API tests
go test ./tests/api/... -v

# Test health endpoint
curl http://localhost:8080/health
# Expected: {"status":"ok","version":"1.0.0"}

# Test chain status
curl http://localhost:8080/v1/chains/status
# Expected: JSON with all chain statuses

# Test authentication
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@metabridge.local","password":"<YOUR_ADMIN_PASSWORD>"}'
# Expected: JWT token in response
```

### Step 19: Set Up Monitoring

```bash
# Install Prometheus
cd /tmp
wget https://github.com/prometheus/prometheus/releases/download/v2.45.0/prometheus-2.45.0.linux-amd64.tar.gz
tar xvf prometheus-2.45.0.linux-amd64.tar.gz
sudo mv prometheus-2.45.0.linux-amd64 /opt/prometheus
sudo useradd --no-create-home --shell /bin/false prometheus
sudo chown -R prometheus:prometheus /opt/prometheus

# Create Prometheus config
sudo tee /opt/prometheus/prometheus.yml > /dev/null <<EOF
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'metabridge'
    static_configs:
      - targets: ['localhost:8080']
EOF

# Create Prometheus service
sudo tee /etc/systemd/system/prometheus.service > /dev/null <<EOF
[Unit]
Description=Prometheus
After=network.target

[Service]
User=prometheus
Group=prometheus
Type=simple
ExecStart=/opt/prometheus/prometheus --config.file=/opt/prometheus/prometheus.yml --storage.tsdb.path=/opt/prometheus/data

[Install]
WantedBy=multi-user.target
EOF

# Start Prometheus
sudo systemctl daemon-reload
sudo systemctl enable prometheus
sudo systemctl start prometheus

# Install Grafana
sudo apt-get install -y software-properties-common
wget -q -O - https://packages.grafana.com/gpg.key | sudo apt-key add -
echo "deb https://packages.grafana.com/oss/deb stable main" | sudo tee /etc/apt/sources.list.d/grafana.list
sudo apt-get update
sudo apt-get install -y grafana

# Start Grafana
sudo systemctl enable grafana-server
sudo systemctl start grafana-server

# Access Grafana at http://<YOUR_VM_IP>:3000
# Default credentials: admin/admin
```

### Step 20: Verify Full Deployment

```bash
# Check all services are running
sudo systemctl status metabridge-api
sudo systemctl status metabridge-relayer
sudo systemctl status prometheus
sudo systemctl status grafana-server

# Check Docker services
docker compose -f docker-compose.production.yaml ps

# Test API endpoints
curl http://localhost:8080/health
curl http://localhost:8080/v1/chains/status
curl http://localhost:8080/v1/stats

# Check database
docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c "SELECT COUNT(*) FROM messages;"

# Check logs for errors
sudo journalctl -u metabridge-api --since "10 minutes ago" | grep -i error
sudo journalctl -u metabridge-relayer --since "10 minutes ago" | grep -i error

# Monitor resource usage
htop  # Or: sudo apt install htop && htop
```

### Deployment Checklist

After deployment, verify:

- [ ] All systemd services are active and running
- [ ] Docker containers (postgres, nats, redis) are healthy
- [ ] API health endpoint returns 200 OK
- [ ] Chain status endpoint returns all chains
- [ ] Database contains admin user and schema
- [ ] Smart contracts are deployed on all chains
- [ ] Firewall is configured (UFW enabled)
- [ ] HTTPS/SSL is configured (if using domain)
- [ ] Prometheus is scraping metrics
- [ ] Grafana dashboards are accessible
- [ ] Log rotation is configured
- [ ] Automated backups are scheduled
- [ ] Monitoring alerts are configured

### Common Commands

```bash
# Restart all services
sudo systemctl restart metabridge-api metabridge-relayer

# View live logs
sudo journalctl -u metabridge-api -f
sudo journalctl -u metabridge-relayer -f

# Check resource usage
docker stats
htop

# Database backup
docker exec metabridge-postgres pg_dump -U bridge_user metabridge_production > backup_$(date +%Y%m%d).sql

# Update code
cd ~/metabridge/metabridge-engine-hub
git pull origin main
go build -o bin/metabridge-api cmd/api/main.go
sudo systemctl restart metabridge-api

# View all bridge transactions
docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c \
  "SELECT id, source_chain, destination_chain, status, created_at FROM messages ORDER BY created_at DESC LIMIT 10;"
```

### Troubleshooting

**Service won't start**:
```bash
sudo journalctl -u metabridge-api -n 100 --no-pager
# Check for configuration errors or missing environment variables
```

**Database connection failed**:
```bash
# Verify PostgreSQL is running
docker compose ps postgres
# Check connection
docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c "SELECT 1;"
```

**Out of memory**:
```bash
# Check memory usage
free -h
# Increase VM size or add swap
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

**High CPU usage**:
```bash
# Check which process
top
# Reduce relayer workers in config
nano config/config.mainnet.yaml
# Set workers: 5 (instead of 10)
```

---

## 🚀 Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/EmekaIwuagwu/metabridge-hub.git
cd metabridge-hub
```

### 2. Install Dependencies

```bash
# Install Go dependencies
go mod download

# Install smart contract dependencies (EVM)
cd contracts/evm
npm install
cd ../..
```

### 3. Set Environment Variables

```bash
# Create .env file
cat > .env.testnet <<EOF
# RPC API Keys
ALCHEMY_API_KEY=your_alchemy_key
INFURA_API_KEY=your_infura_key
HELIUS_API_KEY=your_helius_key

# Database
DB_PASSWORD=bridge_password

# Keystore
TESTNET_KEYSTORE_PASSWORD=your_keystore_password

# Contract Addresses (will be filled after deployment)
POLYGON_AMOY_BRIDGE_CONTRACT=
BNB_TESTNET_BRIDGE_CONTRACT=
AVALANCHE_FUJI_BRIDGE_CONTRACT=
ETHEREUM_SEPOLIA_BRIDGE_CONTRACT=
SOLANA_DEVNET_BRIDGE_PROGRAM=
NEAR_TESTNET_BRIDGE_CONTRACT=
EOF
```

### 4. Start Infrastructure (Testnet)

```bash
# Start PostgreSQL, NATS, and Redis
docker-compose -f docker-compose.testnet.yaml up -d postgres nats redis

# Wait for services to be healthy
docker-compose -f docker-compose.testnet.yaml ps
```

### 5. Run Database Migrations

```bash
# Apply database schema
psql -h localhost -U bridge_user -d metabridge_testnet -f internal/database/schema.sql

# Or use Docker
docker exec -i metabridge-postgres-testnet psql -U bridge_user -d metabridge_testnet < internal/database/schema.sql
```

---

## 🧪 Testnet Deployment

### Step 1: Deploy Smart Contracts

#### EVM Contracts (Polygon, BNB, Avalanche, Ethereum)

```bash
cd contracts/evm

# Deploy to Polygon Amoy Testnet
npx hardhat deploy --network polygon-amoy --tags Bridge
export POLYGON_AMOY_BRIDGE_CONTRACT=$(cat deployments/polygon-amoy/Bridge.json | jq -r '.address')

# Deploy to BNB Testnet
npx hardhat deploy --network bnb-testnet --tags Bridge
export BNB_TESTNET_BRIDGE_CONTRACT=$(cat deployments/bnb-testnet/Bridge.json | jq -r '.address')

# Deploy to Avalanche Fuji
npx hardhat deploy --network avalanche-fuji --tags Bridge
export AVALANCHE_FUJI_BRIDGE_CONTRACT=$(cat deployments/avalanche-fuji/Bridge.json | jq -r '.address')

# Deploy to Ethereum Sepolia
npx hardhat deploy --network ethereum-sepolia --tags Bridge
export ETHEREUM_SEPOLIA_BRIDGE_CONTRACT=$(cat deployments/ethereum-sepolia/Bridge.json | jq -r '.address')

# Verify contracts
npx hardhat verify --network polygon-amoy $POLYGON_AMOY_BRIDGE_CONTRACT
```

#### Solana Program (Devnet)

```bash
cd contracts/solana

# Build program
anchor build

# Set Solana to devnet
solana config set --url devnet

# Deploy
anchor deploy --provider.cluster devnet
export SOLANA_DEVNET_BRIDGE_PROGRAM=$(solana address -k target/deploy/bridge-keypair.json)

# Initialize
anchor run initialize --provider.cluster devnet
```

#### NEAR Contract (Testnet)

```bash
cd contracts/near

# Build contract
./build.sh

# Create testnet account
near create-account bridge.testnet --masterAccount your-account.testnet

# Deploy
near deploy --accountId bridge.testnet --wasmFile res/bridge.wasm
export NEAR_TESTNET_BRIDGE_CONTRACT="bridge.testnet"

# Initialize
near call bridge.testnet new '{"owner":"validator.testnet","required_signatures":2}' --accountId bridge.testnet
```

### Step 2: Update Configuration

Update `.env.testnet` with deployed contract addresses:

```bash
# Update environment variables
echo "POLYGON_AMOY_BRIDGE_CONTRACT=$POLYGON_AMOY_BRIDGE_CONTRACT" >> .env.testnet
echo "BNB_TESTNET_BRIDGE_CONTRACT=$BNB_TESTNET_BRIDGE_CONTRACT" >> .env.testnet
# ... etc
```

### Step 3: Start Backend Services

```bash
# Set environment
export BRIDGE_ENVIRONMENT=testnet

# Load environment variables
source .env.testnet

# Start all services
docker-compose -f docker-compose.testnet.yaml up -d

# Check logs
docker-compose -f docker-compose.testnet.yaml logs -f
```

### Step 4: Verify Deployment

```bash
# Check API health
curl http://localhost:8080/health

# Check chain status
curl http://localhost:8080/v1/chains/status

# Check bridge stats
curl http://localhost:8080/v1/stats
```

---

## 🏭 Mainnet Deployment

### ⚠️ Pre-Deployment Checklist

Before deploying to mainnet, ensure:

- [ ] All smart contracts audited by reputable security firm
- [ ] Bug bounty program established
- [ ] Multi-signature wallets configured (3-of-5 minimum)
- [ ] Emergency pause mechanism tested
- [ ] Rate limiting configured
- [ ] Monitoring and alerting configured
- [ ] Incident response plan documented
- [ ] Insurance coverage secured
- [ ] Testnet stress testing completed

### Step 1: Deploy Smart Contracts to Mainnet

```bash
# ⚠️ CAUTION: Deploying to mainnet with real funds

# EVM Contracts
cd contracts/evm

npx hardhat deploy --network polygon-mainnet --tags Bridge
npx hardhat deploy --network bnb-mainnet --tags Bridge
npx hardhat deploy --network avalanche-mainnet --tags Bridge
npx hardhat deploy --network ethereum-mainnet --tags Bridge

# Verify all contracts
npx hardhat verify --network polygon-mainnet $POLYGON_MAINNET_BRIDGE_CONTRACT

# Transfer ownership to multi-sig
npx hardhat run scripts/transfer-ownership.js --network polygon-mainnet
```

### Step 2: Deploy Solana and NEAR Contracts

```bash
# Solana Mainnet
cd contracts/solana
solana config set --url mainnet-beta
anchor deploy --provider.cluster mainnet-beta

# NEAR Mainnet
cd contracts/near
near deploy --accountId bridge.near --wasmFile res/bridge_release.wasm
```

### Step 3: Production Infrastructure

```bash
# Use Kubernetes for production
kubectl create namespace metabridge-mainnet

# Create secrets
kubectl create secret generic bridge-secrets \
  --from-env-file=.env.mainnet \
  -n metabridge-mainnet

# Deploy services
kubectl apply -f deployments/kubernetes/mainnet/
```

### Step 4: Gradual Rollout

Start with conservative limits and gradually increase:

**Week 1**: $1,000 max per transaction
**Week 2**: $10,000 max per transaction
**Week 3**: $50,000 max per transaction
**Week 4+**: $100,000+ with monitoring

---

## ⚙️ Configuration

### Environment Variables

```bash
# Environment Selection
BRIDGE_ENVIRONMENT=testnet  # or mainnet

# Database
DB_HOST=localhost
DB_PASSWORD=secure_password

# RPC Keys
ALCHEMY_API_KEY=your_key
INFURA_API_KEY=your_key
HELIUS_API_KEY=your_key

# Security
TESTNET_KEYSTORE_PASSWORD=password
MAINNET_KEYSTORE_PASSWORD=secure_password

# AWS KMS (mainnet only)
AWS_KMS_EVM_KEY_ID=your_kms_key
```

### Chain Configuration

See `config/config.testnet.yaml` and `config/config.mainnet.yaml` for complete chain configurations.

---

## 📊 Monitoring

### Prometheus Metrics

Access Prometheus at: `http://localhost:9090`

Key metrics:
- `bridge_messages_total` - Total messages processed
- `bridge_messages_by_status` - Messages by status
- `bridge_transaction_value_usd` - Transaction volumes
- `bridge_gas_price_gwei` - Gas prices per chain
- `bridge_processing_time_seconds` - Processing latency

### Grafana Dashboards

Access Grafana at: `http://localhost:3000`

Default credentials: `admin/admin`

Pre-built dashboards:
1. **Bridge Overview**: High-level metrics
2. **Chain Status**: Per-chain health
3. **Transaction Monitoring**: Real-time transactions
4. **Security Dashboard**: Anomaly detection

---

## 🔐 Security

### Testnet Security (2-of-3 Multi-Sig)

- Transaction limit: $10,000
- Daily volume: $100,000
- Rate limit: 100 tx/hour

### Mainnet Security (3-of-5 Multi-Sig)

- Transaction limit: $1,000,000
- Daily volume: $10,000,000
- Rate limit: 20 tx/hour
- Mandatory emergency pause
- Fraud detection enabled
- 24/7 monitoring

### Emergency Procedures

```bash
# Emergency pause (requires multi-sig)
npx hardhat run scripts/emergency-pause.js --network polygon-mainnet

# Stop relayer services
kubectl scale deployment relayer --replicas=0 -n metabridge-mainnet
```

---

## 🧪 Testing

```bash
# Unit tests
go test ./... -v

# Integration tests
go test ./tests/integration/... -v

# E2E tests (requires deployed contracts)
go test ./tests/e2e/... -v -run TestPolygonToBNB
```

---

## 📚 API Documentation

### Health Check

```bash
GET /health
```

### Get Chain Status

```bash
GET /v1/chains/status
```

### Bridge Token

```bash
POST /v1/bridge/token
{
  "source_chain": "polygon-amoy",
  "dest_chain": "bnb-testnet",
  "token_address": "0x...",
  "amount": "1000000000000000000",
  "recipient": "0x..."
}
```

### Get Message Status

```bash
GET /v1/messages/{messageId}
```

---

## 📝 License

MIT License

---

## ⚖️ Disclaimer

This software is provided "as is" without warranty. Use at your own risk. Always conduct thorough security audits before handling real user funds.

---

**Built with ❤️ for the decentralized future**
