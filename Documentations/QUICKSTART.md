# Metabridge Engine - Quick Start Guide

Get the Metabridge Engine running locally in minutes!

## 🚀 One-Command Deployment

```bash
./deploy-testnet.sh
```

This script will:
- ✅ Check prerequisites (Docker, Go, RAM)
- ✅ Build all Go services
- ✅ Start infrastructure (PostgreSQL, NATS, Redis, Prometheus, Grafana)
- ✅ Run database migrations
- ✅ Start all backend services
- ✅ Verify deployment health

## 📋 Prerequisites

### Required Software
- **Docker** 20.10+ with Docker Compose
- **Go** 1.21+
- **8GB RAM minimum** (16GB recommended)
- **20GB disk space**

### Quick Install (macOS)
```bash
# Install Homebrew if not already installed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install required tools
brew install docker docker-compose go
```

### Quick Install (Ubuntu/Debian)
```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Install Docker Compose
sudo apt-get update
sudo apt-get install docker-compose-plugin

# Install Go
sudo apt-get install golang-1.21
```

## 🎯 System Requirements

### Minimum (Testing)
- **CPU**: 2 cores
- **RAM**: 8 GB
- **Disk**: 20 GB SSD
- **Network**: Stable internet for RPC connections

### Recommended (Development)
- **CPU**: 4 cores
- **RAM**: 16 GB
- **Disk**: 50 GB SSD
- **Network**: High-speed internet

### Production (Mainnet)
- **CPU**: 8+ cores
- **RAM**: 32 GB
- **Disk**: 500 GB SSD (NVMe preferred)
- **Network**: Dedicated high-bandwidth connection

## 📊 RAM Breakdown

Here's how the 8GB minimum is allocated:

| Component | Minimum | Recommended | Notes |
|-----------|---------|-------------|-------|
| PostgreSQL | 512 MB | 2 GB | Database |
| NATS JetStream | 256 MB | 1 GB | Message queue |
| Redis | 256 MB | 512 MB | Cache |
| API Service | 256 MB | 512 MB | REST API |
| Listener Service | 256 MB | 512 MB | Per chain (×4 = 2GB) |
| Relayer Service | 512 MB | 1 GB | Message processor |
| Prometheus | 512 MB | 1 GB | Metrics |
| Grafana | 256 MB | 512 MB | Dashboards |
| **System Overhead** | 1 GB | 2 GB | OS + Docker |
| **TOTAL** | **~4 GB** | **~12 GB** | |

**Note**: 4GB chains × 4 listeners = additional 2GB for full chain support

## 🔧 Step-by-Step Deployment

### 1. Clone and Setup

```bash
# Navigate to project directory
cd metabridge-hub

# Make scripts executable
chmod +x deploy-testnet.sh stop-testnet.sh
```

### 2. Deploy

```bash
# Deploy everything
./deploy-testnet.sh

# Expected output:
# ✓ Checking prerequisites...
# ✓ Building Go services...
# ✓ Starting infrastructure...
# ✓ Running migrations...
# ✓ Starting services...
# ✓ All health checks passed!
```

### 3. Verify

```bash
# Check API health
curl http://localhost:8080/health

# List supported chains
curl http://localhost:8080/v1/chains

# View system status
curl http://localhost:8080/v1/status
```

### 4. Access Dashboards

- **API**: http://localhost:8080
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Database UI**: http://localhost:8081

## 📝 View Logs

```bash
# API logs
tail -f logs/api.log

# Listener logs
tail -f logs/listener.log

# Relayer logs
tail -f logs/relayer.log

# All infrastructure logs
cd deployments/docker
docker-compose -f docker-compose.infrastructure.yaml logs -f
```

## 🛑 Stop Services

```bash
# Stop everything cleanly
./stop-testnet.sh
```

## 🧪 Testing the Bridge

### Test 1: Check Chain Connectivity

```bash
curl http://localhost:8080/v1/chains | jq '.'
```

Expected: List of 6 chains (Polygon, BNB, Avalanche, Ethereum, Solana, NEAR)

### Test 2: Monitor Metrics

```bash
# Total messages
curl http://localhost:9090/api/v1/query?query=bridge_messages_total

# Chain health
curl http://localhost:9090/api/v1/query?query=chain_health
```

### Test 3: Database Check

```bash
# Connect to database
docker exec -it metabridge-postgres psql -U metabridge -d metabridge_testnet

# Inside psql:
SELECT * FROM chains;
SELECT COUNT(*) FROM messages;
```

## 🔍 Troubleshooting

### Port Already in Use

```bash
# Check what's using the port
lsof -i :8080  # API port
lsof -i :5432  # PostgreSQL port

# Kill process if needed
kill -9 <PID>
```

### Docker Issues

```bash
# Reset Docker
docker system prune -a

# Restart Docker daemon
sudo systemctl restart docker  # Linux
# or
# Restart Docker Desktop  # macOS/Windows
```

### Out of Memory

```bash
# Check current usage
free -h  # Linux
vm_stat  # macOS

# Increase Docker memory limit
# Docker Desktop -> Settings -> Resources -> Memory -> 8GB+
```

### Services Won't Start

```bash
# Check logs
tail -f logs/*.log

# Check Docker containers
docker ps -a
docker logs metabridge-postgres
docker logs metabridge-nats
docker logs metabridge-redis

# Restart infrastructure
cd deployments/docker
docker-compose -f docker-compose.infrastructure.yaml restart
```

## 🎓 Next Steps

### 1. Smart Contract Deployment

Before the bridge can process real transactions, deploy smart contracts:

```bash
# See docs/runbooks/DEPLOYMENT.md for detailed instructions

# EVM contracts (Hardhat)
cd contracts/evm
npm install
npx hardhat deploy --network polygon-amoy

# Solana program (Anchor)
cd contracts/solana
anchor build
anchor deploy --provider.cluster devnet

# NEAR contract
cd contracts/near
./build.sh
near deploy --accountId metabridge.testnet --wasmFile ./res/near_bridge.wasm
```

### 2. Configure RPC Endpoints

Update `config/config.testnet.yaml` with your RPC endpoints:

```yaml
chains:
  - name: polygon
    rpc_urls:
      - "https://polygon-amoy.g.alchemy.com/v2/YOUR_KEY"
```

Get free RPC endpoints from:
- **Alchemy**: https://www.alchemy.com
- **Infura**: https://www.infura.io
- **QuickNode**: https://www.quicknode.com

### 3. Set Up Validators

Generate validator keys for production:

```bash
# For testnet, the system uses placeholder keys
# For production, use HSM/KMS:

# Generate EVM validator key
openssl ecparam -name secp256k1 -genkey -noout -out validator.pem

# Generate Ed25519 validator key (Solana/NEAR)
solana-keygen new --outfile validator-solana.json
near generate-key validator.near
```

### 4. Run Tests

```bash
# Unit tests
go test ./...

# Integration tests
go test ./tests/integration/... -v

# E2E tests (requires deployed contracts)
go test ./tests/e2e/... -v
```

### 5. Monitor in Production

- Set up alerts in Prometheus
- Configure Grafana dashboards
- Enable log aggregation
- Set up PagerDuty/Opsgenie for on-call

## 📚 Documentation

- **Deployment Guide**: [docs/runbooks/DEPLOYMENT.md](docs/runbooks/DEPLOYMENT.md)
- **Emergency Procedures**: [docs/runbooks/EMERGENCY_PROCEDURES.md](docs/runbooks/EMERGENCY_PROCEDURES.md)
- **Monitoring Guide**: [docs/runbooks/MONITORING.md](docs/runbooks/MONITORING.md)
- **Main README**: [README.md](README.md)

## 🆘 Getting Help

- **Issues**: Check logs in `logs/` directory
- **Health**: Run `curl http://localhost:8080/health`
- **Status**: Run `docker-compose ps`
- **Docs**: See `docs/runbooks/`

## 🎉 Success Indicators

Your deployment is successful when:

✅ All health checks pass
✅ API responds at http://localhost:8080/health
✅ Grafana shows metrics at http://localhost:3000
✅ Database has tables: `SELECT COUNT(*) FROM chains;`
✅ NATS is connected: `docker exec metabridge-nats nats-server --signal ping`
✅ All services show in logs without errors

---

**Ready to deploy?**

```bash
./deploy-testnet.sh
```

Let's build the future of cross-chain bridges! 🌉
