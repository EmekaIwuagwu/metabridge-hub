# Azure Deployment Guide - Metabridge Engine

Complete step-by-step guide to deploy Metabridge Engine on Azure from scratch.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Create Azure VM](#create-azure-vm)
3. [Initial Server Setup](#initial-server-setup)
4. [Install Dependencies](#install-dependencies)
5. [Clone and Configure Project](#clone-and-configure-project)
6. [Deploy Smart Contracts](#deploy-smart-contracts)
7. [Configure and Start Services](#configure-and-start-services)
8. [Setup SSL/HTTPS](#setup-sslhttps)
9. [Configure Monitoring](#configure-monitoring)
10. [Production Hardening](#production-hardening)
11. [Backup and Recovery](#backup-and-recovery)

---

## Prerequisites

### What You Need

- **Azure Account** with active subscription
- **Domain name** (optional but recommended)
- **SSH key pair** (we'll create if needed)
- **API Keys**:
  - Alchemy API key (https://www.alchemy.com)
  - Infura API key (https://www.infura.io)
  - Block explorer API keys (Polygonscan, BscScan, Snowtrace, Etherscan)

### Cost Estimate

| Component | Specs | Monthly Cost (USD) |
|-----------|-------|-------------------|
| **VM (Testnet)** | Standard_D4s_v3 (4 vCPU, 16GB RAM) | ~$140 |
| **VM (Mainnet)** | Standard_D8s_v3 (8 vCPU, 32GB RAM) | ~$280 |
| **Storage** | 500GB Premium SSD | ~$75 |
| **Bandwidth** | ~500GB/month | ~$40 |
| **Total Testnet** | | **~$255/month** |
| **Total Mainnet** | | **~$395/month** |

---

## Create Azure VM

### Step 1: Login to Azure Portal

1. Go to https://portal.azure.com
2. Sign in with your Azure account
3. Click "Create a resource"

### Step 2: Create Virtual Machine

**2.1. Basics Tab:**

1. Click **"Virtual machines"** → **"Create"** → **"Azure virtual machine"**

2. **Project details:**
   - Subscription: Select your subscription
   - Resource group: **Create new** → Name: `metabridge-testnet-rg`

3. **Instance details:**
   - Virtual machine name: `metabridge-testnet-vm`
   - Region: **East US** (or closest to your location)
   - Availability options: **No infrastructure redundancy required** (for testnet)
   - Security type: **Standard**
   - Image: **Ubuntu Server 22.04 LTS - x64 Gen2**
   - VM architecture: **x64**
   - Size: Click **"See all sizes"**
     - Filter: `D4s_v3`
     - Select: **Standard_D4s_v3** (4 vCPUs, 16 GB RAM)
     - Click **"Select"**

4. **Administrator account:**
   - Authentication type: **SSH public key**
   - Username: `azureuser`
   - SSH public key source:
     - **Option A**: **Generate new key pair** → Key pair name: `metabridge-key`
     - **Option B**: **Use existing public key** → Paste your public key

5. **Inbound port rules:**
   - Public inbound ports: **Allow selected ports**
   - Select inbound ports:
     - [x] SSH (22)
     - [x] HTTP (80)
     - [x] HTTPS (443)

**2.2. Disks Tab:**

1. **OS disk:**
   - OS disk type: **Premium SSD (locally-redundant storage)**
   - Size: **Default** (30GB)
   - Delete with VM: **Yes**

2. **Data disks:**
   - Click **"Create and attach a new disk"**
   - Name: `metabridge-data-disk`
   - Size: **512 GiB**
   - Disk SKU: **Premium SSD LRS**
   - Click **"OK"**

**2.3. Networking Tab:**

1. **Network interface:**
   - Virtual network: **Create new** → Name: `metabridge-vnet`
   - Subnet: **default** (10.0.0.0/24)
   - Public IP: **Create new** → Name: `metabridge-public-ip`
     - SKU: **Standard**
     - Assignment: **Static**
   - NIC network security group: **Basic**
   - Public inbound ports: **Allow selected ports**
   - Select inbound ports: SSH (22), HTTP (80), HTTPS (443)

2. **Advanced:**
   - Click **"Create new"** under Network security group
   - Add inbound rules:

   | Priority | Name | Port | Protocol | Source | Action |
   |----------|------|------|----------|--------|--------|
   | 100 | SSH | 22 | TCP | My IP | Allow |
   | 200 | HTTP | 80 | TCP | Internet | Allow |
   | 300 | HTTPS | 443 | TCP | Internet | Allow |
   | 400 | API | 8080 | TCP | Internet | Allow |
   | 500 | Prometheus | 9090 | TCP | My IP | Allow |
   | 600 | Grafana | 3000 | TCP | My IP | Allow |

   - Click **"OK"**

**2.4. Management Tab:**

1. **Monitoring:**
   - Boot diagnostics: **Enable with managed storage account**
   - Enable OS guest diagnostics: **Off** (to save costs)

2. **Auto-shutdown:**
   - Enable auto-shutdown: **On** (optional, for cost savings)
   - Shutdown time: **7:00 PM** (adjust as needed)
   - Time zone: Your timezone
   - Email notification: **On** → Enter your email

**2.5. Review + Create:**

1. Review all settings
2. Click **"Create"**
3. If you chose "Generate new key pair":
   - **Download private key** (metabridge-key.pem)
   - **SAVE THIS FILE SECURELY** - you cannot download it again!
4. Wait 3-5 minutes for deployment

### Step 3: Get VM Details

1. Go to **Virtual machines** → **metabridge-testnet-vm**
2. Note down:
   - **Public IP address**: `XX.XX.XX.XX` (you'll use this for SSH)
   - **Private IP address**: `10.0.0.4` (for internal use)

---

## Initial Server Setup

### Step 1: Connect via SSH

**On macOS/Linux:**

```bash
# Set permissions on private key
chmod 400 metabridge-key.pem

# Connect to VM
ssh -i metabridge-key.pem azureuser@XX.XX.XX.XX
```

**On Windows (PowerShell):**

```powershell
# Connect to VM
ssh -i metabridge-key.pem azureuser@XX.XX.XX.XX
```

**Expected output:**
```
Welcome to Ubuntu 22.04.3 LTS (GNU/Linux 5.15.0-1045-azure x86_64)
...
azureuser@metabridge-testnet-vm:~$
```

✅ **You're now connected to your Azure VM!**

### Step 2: Update System

```bash
# Update package list
sudo apt update

# Upgrade all packages
sudo apt upgrade -y

# Install essential tools
sudo apt install -y curl wget git vim htop net-tools
```

**Time:** ~5 minutes

### Step 3: Configure Firewall (UFW)

```bash
# Install UFW
sudo apt install -y ufw

# Allow SSH (IMPORTANT: do this first!)
sudo ufw allow 22/tcp

# Allow HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Allow API
sudo ufw allow 8080/tcp

# Allow Prometheus (your IP only - replace XX.XX.XX.XX)
sudo ufw allow from YOUR_IP to any port 9090

# Allow Grafana (your IP only)
sudo ufw allow from YOUR_IP to any port 3000

# Enable firewall
sudo ufw enable

# Check status
sudo ufw status
```

### Step 4: Setup Data Disk

```bash
# List disks
lsblk

# Expected output:
# NAME    MAJ:MIN RM   SIZE RO TYPE MOUNTPOINT
# sda       8:0    0    30G  0 disk
# ├─sda1    8:1    0  29.9G  0 part /
# sdb       8:16   0   512G  0 disk    <-- This is your data disk
# sr0      11:0    1   628K  0 rom

# Format data disk (assuming it's sdb)
sudo mkfs.ext4 /dev/sdb

# Create mount point
sudo mkdir -p /mnt/metabridge-data

# Mount disk
sudo mount /dev/sdb /mnt/metabridge-data

# Get UUID
sudo blkid /dev/sdb
# Output: /dev/sdb: UUID="xxxx-xxxx-xxxx-xxxx" TYPE="ext4"

# Add to fstab for auto-mount
echo "UUID=xxxx-xxxx-xxxx-xxxx /mnt/metabridge-data ext4 defaults 0 2" | sudo tee -a /etc/fstab

# Verify mount
df -h /mnt/metabridge-data
```

### Step 5: Create Application Directory

```bash
# Create app directory
sudo mkdir -p /mnt/metabridge-data/metabridge

# Set ownership
sudo chown -R azureuser:azureuser /mnt/metabridge-data/metabridge

# Create subdirectories
mkdir -p /mnt/metabridge-data/metabridge/{data,logs,config,backups}

# Create symlink for easy access
ln -s /mnt/metabridge-data/metabridge ~/metabridge
```

---

## Install Dependencies

### Step 1: Install Docker

```bash
# Remove old versions
sudo apt remove docker docker-engine docker.io containerd runc

# Install prerequisites
sudo apt install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

# Add Docker's official GPG key
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

# Set up repository
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install Docker
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Add user to docker group
sudo usermod -aG docker azureuser

# Apply group changes (or logout/login)
newgrp docker

# Verify installation
docker --version
docker compose version
```

**Expected output:**
```
Docker version 24.0.7, build afdd53b
Docker Compose version v2.23.0
```

### Step 2: Install Go

```bash
# Download Go 1.21
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz

# Remove old installation
sudo rm -rf /usr/local/go

# Extract
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc

# Reload bashrc
source ~/.bashrc

# Verify
go version
```

**Expected output:**
```
go version go1.21.5 linux/amd64
```

### Step 3: Install Node.js and npm

```bash
# Install NVM
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.5/install.sh | bash

# Reload shell
source ~/.bashrc

# Install Node.js 18
nvm install 18
nvm use 18

# Verify
node --version
npm --version
```

**Expected output:**
```
v18.18.2
10.2.0
```

### Step 4: Install Additional Tools

```bash
# Install jq (JSON processor)
sudo apt install -y jq

# Install PostgreSQL client
sudo apt install -y postgresql-client

# Install Redis CLI
sudo apt install -y redis-tools

# Verify
jq --version
psql --version
redis-cli --version
```

---

## Clone and Configure Project

### Step 1: Clone Repository

```bash
# Navigate to app directory
cd ~/metabridge

# Clone repository
git clone https://github.com/EmekaIwuagwu/metabridge-engine-hub.git
cd metabridge-engine-hub

# Checkout your branch (or main)
git checkout claude/multi-chain-bridge-protocol-014mAq2r9WZ9CyBp9wSuuMGe

# Verify
ls -la
```

### Step 2: Configure Environment Variables

```bash
# Create .env file
cat > .env << 'EOF'
# Environment
BRIDGE_ENVIRONMENT=testnet

# Database
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=metabridge
DATABASE_PASSWORD=CHANGE_THIS_PASSWORD
DATABASE_NAME=metabridge_testnet
DB_PASSWORD=CHANGE_THIS_PASSWORD

# NATS
NATS_URL=nats://localhost:4222

# Redis
REDIS_URL=redis://localhost:6379

# RPC API Keys
ALCHEMY_API_KEY=your_alchemy_key_here
INFURA_API_KEY=your_infura_key_here
NODEREAL_API_KEY=your_nodereal_key_here
HELIUS_API_KEY=your_helius_key_here

# Block Explorer API Keys
POLYGONSCAN_API_KEY=your_polygonscan_key_here
BSCSCAN_API_KEY=your_bscscan_key_here
SNOWTRACE_API_KEY=your_snowtrace_key_here
ETHERSCAN_API_KEY=your_etherscan_key_here

# Deployer (for contract deployment)
DEPLOYER_PRIVATE_KEY=your_private_key_without_0x

# Contract Addresses (will be set after deployment)
POLYGON_AMOY_BRIDGE_CONTRACT=
BNB_TESTNET_BRIDGE_CONTRACT=
AVALANCHE_FUJI_BRIDGE_CONTRACT=
ETHEREUM_SEPOLIA_BRIDGE_CONTRACT=
SOLANA_DEVNET_BRIDGE_PROGRAM=
NEAR_TESTNET_BRIDGE_CONTRACT=
EOF

# Secure the file
chmod 600 .env

# Edit with your actual values
nano .env
```

**Press Ctrl+X, then Y, then Enter to save**

### Step 3: Set Data Directories

```bash
# Update paths to use data disk
export DATA_DIR=/mnt/metabridge-data/metabridge/data
export LOG_DIR=/mnt/metabridge-data/metabridge/logs

# Create directories
mkdir -p $DATA_DIR/{postgres,redis,nats,prometheus,grafana}
mkdir -p $LOG_DIR

# Set permissions
chmod -R 755 $DATA_DIR
chmod -R 755 $LOG_DIR
```

---

## Deploy Smart Contracts

### Step 1: Deploy EVM Contracts

```bash
cd contracts/evm

# Install dependencies
npm install

# Copy environment
cp .env.example .env

# Edit with your keys
nano .env

# Deploy to Polygon Amoy
npm run deploy:polygon-amoy

# Deploy to BNB Testnet
npm run deploy:bnb-testnet

# Deploy to Avalanche Fuji
npm run deploy:avalanche-fuji

# Deploy to Ethereum Sepolia
npm run deploy:ethereum-sepolia

# Save contract addresses
ls -la deployments/
```

**Save all contract addresses - you'll need them!**

### Step 2: Deploy Solana Contract

```bash
cd ../solana

# Install Anchor (if not already installed)
cargo install --git https://github.com/coral-xyz/anchor --tag v0.29.0 anchor-cli

# Build
anchor build

# Deploy to Devnet
anchor deploy --provider.cluster devnet

# Save program ID
```

### Step 3: Deploy NEAR Contract

```bash
cd ../near

# Install NEAR CLI
npm install -g near-cli

# Build contract
./build.sh

# Login to NEAR
near login

# Create sub-account for contract
near create-account bridge.YOUR_ACCOUNT.testnet --masterAccount YOUR_ACCOUNT.testnet --initialBalance 10

# Deploy
near deploy --accountId bridge.YOUR_ACCOUNT.testnet --wasmFile ./res/near_bridge.wasm

# Initialize
near call bridge.YOUR_ACCOUNT.testnet new \
  '{"owner":"YOUR_ACCOUNT.testnet","validators":["ed25519:..."],"required_signatures":2}' \
  --accountId YOUR_ACCOUNT.testnet
```

### Step 4: Update Configuration

```bash
cd ~/metabridge/metabridge-engine-hub

# Update config with contract addresses
nano config/config.testnet.yaml

# Update these sections:
# - bridge_contract for each EVM chain
# - bridge_program for Solana
# - bridge_contract for NEAR
```

---

## Configure and Start Services

### Step 1: Update Docker Compose Paths

```bash
# Edit docker-compose file
nano deployments/docker/docker-compose.infrastructure.yaml

# Update volume paths to use data disk:
# Change:
#   - ../../data/postgres:/var/lib/postgresql/data
# To:
#   - /mnt/metabridge-data/metabridge/data/postgres:/var/lib/postgresql/data
```

### Step 2: Start Infrastructure

```bash
cd deployments/docker

# Start infrastructure services
docker compose -f docker-compose.infrastructure.yaml up -d

# Check status
docker compose -f docker-compose.infrastructure.yaml ps

# Check logs
docker compose -f docker-compose.infrastructure.yaml logs -f
```

**Wait for all services to be healthy (1-2 minutes)**

### Step 3: Build and Start Backend Services

```bash
cd ~/metabridge/metabridge-engine-hub

# Run deployment script
./deploy-testnet.sh
```

**Expected output:**
```
✅ Bridge is healthy
✅ All services started
```

### Step 4: Verify Deployment

```bash
# Check API
curl http://localhost:8080/health

# Check chains
curl http://localhost:8080/v1/chains | jq '.'

# Check Docker containers
docker ps

# Check processes
ps aux | grep metabridge
```

---

## Setup SSL/HTTPS

### Step 1: Configure Domain (Optional)

If you have a domain:

```bash
# Point your domain A record to:
# bridge.yourdomain.com → XX.XX.XX.XX (your VM IP)
```

### Step 2: Install Nginx

```bash
# Install Nginx
sudo apt install -y nginx

# Stop Apache if running
sudo systemctl stop apache2
sudo systemctl disable apache2

# Start Nginx
sudo systemctl start nginx
sudo systemctl enable nginx
```

### Step 3: Install Certbot (Let's Encrypt)

```bash
# Install Certbot
sudo apt install -y certbot python3-certbot-nginx

# Get SSL certificate
sudo certbot --nginx -d bridge.yourdomain.com

# Follow prompts:
# - Enter email
# - Agree to terms
# - Choose redirect (2)

# Test auto-renewal
sudo certbot renew --dry-run
```

### Step 4: Configure Nginx Reverse Proxy

```bash
# Edit Nginx config
sudo nano /etc/nginx/sites-available/metabridge

# Add this configuration:
```

```nginx
server {
    listen 80;
    server_name bridge.yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name bridge.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/bridge.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/bridge.yourdomain.com/privkey.pem;

    # API
    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Prometheus
    location /prometheus/ {
        proxy_pass http://localhost:9090/;
        auth_basic "Restricted";
        auth_basic_user_file /etc/nginx/.htpasswd;
    }

    # Grafana
    location /grafana/ {
        proxy_pass http://localhost:3000/;
        proxy_set_header Host $http_host;
    }
}
```

```bash
# Save and exit (Ctrl+X, Y, Enter)

# Create password for Prometheus
sudo apt install -y apache2-utils
sudo htpasswd -c /etc/nginx/.htpasswd admin

# Enable site
sudo ln -s /etc/nginx/sites-available/metabridge /etc/nginx/sites-enabled/

# Test configuration
sudo nginx -t

# Reload Nginx
sudo systemctl reload nginx
```

---

## Configure Monitoring

### Step 1: Access Grafana

```bash
# Get Grafana admin password
docker exec metabridge-grafana grafana-cli admin reset-admin-password NewPassword123
```

**Access:** https://bridge.yourdomain.com/grafana
- Username: `admin`
- Password: `NewPassword123`

### Step 2: Import Dashboards

1. Login to Grafana
2. Click **"+"** → **"Import"**
3. Upload dashboard JSON or use ID
4. Select Prometheus datasource
5. Click **"Import"**

### Step 3: Setup Alerts

Configure alerts for:
- API down
- High error rate
- Chain disconnection
- Low disk space
- High memory usage

---

## Production Hardening

### Step 1: Setup Systemd Services

```bash
# Create service files
sudo nano /etc/systemd/system/metabridge-api.service
```

```ini
[Unit]
Description=Metabridge API Service
After=network.target docker.service

[Service]
Type=simple
User=azureuser
WorkingDirectory=/home/azureuser/metabridge/metabridge-engine-hub
EnvironmentFile=/home/azureuser/metabridge/metabridge-engine-hub/.env
ExecStart=/home/azureuser/metabridge/metabridge-engine-hub/bin/api --config /home/azureuser/metabridge/metabridge-engine-hub/config/config.testnet.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable services
sudo systemctl enable metabridge-api
sudo systemctl enable metabridge-listener
sudo systemctl enable metabridge-relayer

# Start services
sudo systemctl start metabridge-api
sudo systemctl start metabridge-listener
sudo systemctl start metabridge-relayer

# Check status
sudo systemctl status metabridge-api
```

### Step 2: Setup Log Rotation

```bash
sudo nano /etc/logrotate.d/metabridge
```

```
/mnt/metabridge-data/metabridge/logs/*.log {
    daily
    rotate 30
    compress
    delaycompress
    notifempty
    create 0640 azureuser azureuser
    sharedscripts
    postrotate
        systemctl reload metabridge-api
    endscript
}
```

### Step 3: Setup Monitoring Alerts

Configure Azure Monitor or external monitoring:
- Uptime monitoring
- Resource utilization alerts
- Log aggregation
- Error tracking

---

## Backup and Recovery

### Step 1: Database Backup

```bash
# Create backup script
nano ~/backup-database.sh
```

```bash
#!/bin/bash
BACKUP_DIR=/mnt/metabridge-data/metabridge/backups
DATE=$(date +%Y%m%d_%H%M%S)

docker exec metabridge-postgres pg_dump -U metabridge metabridge_testnet > \
  $BACKUP_DIR/db_backup_$DATE.sql

# Keep only last 7 days
find $BACKUP_DIR -name "db_backup_*.sql" -mtime +7 -delete

echo "Backup completed: db_backup_$DATE.sql"
```

```bash
chmod +x ~/backup-database.sh

# Add to crontab (daily at 2 AM)
crontab -e

# Add line:
0 2 * * * /home/azureuser/backup-database.sh
```

### Step 2: Backup to Azure Blob Storage

```bash
# Install Azure CLI
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash

# Login
az login

# Create storage account
az storage account create \
  --name metabridgebackups \
  --resource-group metabridge-testnet-rg \
  --location eastus \
  --sku Standard_LRS

# Upload backups
az storage blob upload-batch \
  --destination backups \
  --source /mnt/metabridge-data/metabridge/backups \
  --account-name metabridgebackups
```

---

## Verification Checklist

After deployment, verify:

- [ ] VM is accessible via SSH
- [ ] All Docker containers running
- [ ] API responds at http://YOUR_IP:8080/health
- [ ] All chains listed at /v1/chains
- [ ] Prometheus accessible
- [ ] Grafana accessible
- [ ] SSL certificate valid
- [ ] Firewall rules correct
- [ ] Systemd services enabled
- [ ] Backups configured
- [ ] Monitoring alerts set

---

## Maintenance Commands

```bash
# Restart all services
./stop-testnet.sh && ./deploy-testnet.sh

# View logs
tail -f /mnt/metabridge-data/metabridge/logs/*.log

# Check Docker logs
docker compose -f deployments/docker/docker-compose.infrastructure.yaml logs -f

# Check system resources
htop

# Check disk usage
df -h

# Check database size
docker exec -it metabridge-postgres psql -U metabridge -d metabridge_testnet -c "SELECT pg_size_pretty(pg_database_size('metabridge_testnet'));"

# Manual backup
~/backup-database.sh

# Update code
cd ~/metabridge/metabridge-engine-hub
git pull
./stop-testnet.sh
go build -o bin/api cmd/api/main.go
./deploy-testnet.sh
```

---

## Troubleshooting

### Issue: Can't connect via SSH

**Solution:**
```bash
# Check NSG rules in Azure Portal
# Ensure your IP is whitelisted
# Check VM is running
```

### Issue: Out of disk space

**Solution:**
```bash
# Check usage
df -h

# Clean Docker
docker system prune -a

# Clean logs
sudo journalctl --vacuum-time=3d
```

### Issue: Service won't start

**Solution:**
```bash
# Check logs
sudo journalctl -u metabridge-api -n 50

# Check config
cat config/config.testnet.yaml

# Restart
sudo systemctl restart metabridge-api
```

---

## Cost Optimization

1. **Use Auto-shutdown**: Enable for non-production
2. **Use Spot Instances**: Save up to 90% for testnet
3. **Reserved Instances**: Save 40-60% for mainnet
4. **Monitor Bandwidth**: Use Azure CDN if needed
5. **Optimize Storage**: Delete old logs and backups

---

## Next Steps

1. ✅ Deploy to testnet
2. ✅ Test cross-chain transfers
3. ✅ Monitor for 1 week
4. ✅ Security audit
5. ✅ Deploy to mainnet
6. ✅ Set up production monitoring
7. ✅ Enable auto-scaling

---

**Congratulations!** Your Metabridge Engine is now running on Azure! 🎉
