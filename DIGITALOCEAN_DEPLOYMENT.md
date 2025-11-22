# Complete DigitalOcean Deployment Guide for Metabridge

**Your Droplet IP**: `159.65.73.133`

This guide will take you from SSH login to a fully running bridge in ~30 minutes.

---

## 📋 Quick Reference

### What You'll Build
- 5 Go binaries (API, Relayer, Listener, Batcher, Migrator)
- Total binary size: ~106 MB
- Expected build time: 2-5 minutes

### Key Expected Responses

**✅ Successful Compilation:**
- No error messages
- Commands complete silently (silence = success!)
- Binary files created in `bin/` directory

**✅ Successful Service Start:**
- `systemctl status` shows "active (running)"
- Health endpoint returns `{"status":"healthy"}`
- All 6 chains show `"healthy": true`

**✅ Successful Database Setup:**
- Tables created: messages, batches, users, api_keys, routes, webhooks
- Admin user exists
- Database size: ~50-100 MB fresh install

### Quick Health Check Commands

```bash
# Check all services at once
sudo systemctl status metabridge-api metabridge-relayer | grep "Active:"
# Expected: Active: active (running) for both

# Check API is responding
curl -s http://159.65.73.133:8080/health | grep status
# Expected: "status":"healthy"

# Check all chains
curl -s http://159.65.73.133:8080/v1/chains/status | jq 'to_entries[] | {chain: .key, healthy: .value.healthy}'
# Expected: All show "healthy": true
```

### Common Expected Outputs Reference

| Command | Expected Output | Meaning |
|---------|----------------|---------|
| `go build ...` | (silence) | ✅ Compilation successful |
| `systemctl status` | `Active: active (running)` | ✅ Service running |
| `curl /health` | `{"status":"healthy"}` | ✅ API responding |
| `docker ps` | `Up (healthy)` | ✅ Container running |
| `psql -c "\dt"` | List of tables | ✅ Database initialized |

### Detailed Documentation References

For comprehensive compilation information, troubleshooting, and expected responses, see:
- **Step 13**: Detailed compilation process and expected build outputs
- **Step 16**: Comprehensive testing with all expected responses
- `Documentations/COMPILATION_TEST_REPORT.md`: Full compilation report
- `Documentations/BUILD_VERIFICATION.md`: Build verification checklist

---

## Step 1: Connect to Your Droplet

```bash
# SSH into your DigitalOcean droplet
ssh root@159.65.73.133

# If you're using a non-root user:
# ssh your-username@159.65.73.133
```

If prompted about host authenticity, type `yes` and press Enter.

## Step 2: System Update & Upgrade

```bash
# Update package lists
sudo apt update

# Upgrade all packages (this may take 5-10 minutes)
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
  lsb-release \
  htop \
  vim

# Set timezone to UTC
sudo timedatectl set-timezone UTC

# Verify
date
```

## Step 3: Install Go 1.21+

```bash
# Download Go
cd ~
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz

# Remove old Go installation (if exists)
sudo rm -rf /usr/local/go

# Extract Go
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz

# Add Go to PATH
cat >> ~/.bashrc << 'EOF'
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
EOF

# Apply changes
source ~/.bashrc

# Verify Go installation
go version
# Should output: go version go1.21.6 linux/amd64

# Clean up
rm go1.21.6.linux-amd64.tar.gz
```

## Step 4: Install Node.js 18+

```bash
# Install Node.js via NodeSource
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt install -y nodejs

# Verify installation
node --version  # Should be v18.x.x
npm --version   # Should be 9.x.x or higher

# Install Yarn globally (optional)
sudo npm install -g yarn
```

## Step 5: Install Docker & Docker Compose

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

# Start Docker
sudo systemctl start docker
sudo systemctl enable docker

# Add current user to docker group (if not root)
sudo usermod -aG docker $USER

# Verify installation
docker --version
docker compose version

# Test Docker
sudo docker run hello-world
```

## Step 6: Clone Metabridge Repository

```bash
# Create project directory
mkdir -p ~/projects
cd ~/projects

# Clone repository
git clone https://github.com/EmekaIwuagwu/metabridge-engine-hub.git
cd metabridge-engine-hub

# Check repository
ls -la
git status
git branch

# Check what branch you're on and switch to main if needed
git checkout main
```

## Step 7: Install Project Dependencies

```bash
# Ensure you're in the project root
cd ~/projects/metabridge-engine-hub

# Install Go dependencies
go mod download
go mod verify

# Install smart contract dependencies (EVM)
cd contracts/evm
npm install
cd ../..

# Verify installation
echo "✅ Dependencies installed successfully"
```

## Step 8: Configure Environment Variables

```bash
cd ~/projects/metabridge-engine-hub

# Copy environment template
cp .env.example .env.production

# Edit environment file
nano .env.production
```

**Paste this configuration** (customize the marked fields):

```bash
# Environment
BRIDGE_ENVIRONMENT=production

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=bridge_user
DB_PASSWORD=YourStrongPassword123!  # ⚠️ CHANGE THIS
DB_NAME=metabridge_production
DB_SSLMODE=disable

# Server Configuration
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# JWT Authentication (generate with: openssl rand -hex 32)
JWT_SECRET=your_super_secret_jwt_key_at_least_32_characters_long_here  # ⚠️ CHANGE THIS
JWT_EXPIRATION_HOURS=24

# CORS (allow all for testing, restrict in production)
CORS_ALLOWED_ORIGINS=*

# Rate Limiting
RATE_LIMIT_PER_MINUTE=100
REQUIRE_AUTH=false  # Set to true after creating admin user
API_KEY_ENABLED=true

# RPC Endpoints - Get free API keys from these services
# Alchemy: https://www.alchemy.com/
# Infura: https://infura.io/
# Helius: https://helius.dev/
ALCHEMY_API_KEY=your_alchemy_api_key_here  # ⚠️ GET FREE KEY
INFURA_API_KEY=your_infura_api_key_here    # ⚠️ GET FREE KEY
HELIUS_API_KEY=your_helius_api_key_here    # ⚠️ GET FREE KEY (optional)

# Chain RPC URLs (Testnet)
POLYGON_RPC_URL=https://rpc-amoy.polygon.technology/
BNB_RPC_URL=https://data-seed-prebsc-1-s1.binance.org:8545/
AVALANCHE_RPC_URL=https://api.avax-test.network/ext/bc/C/rpc
ETHEREUM_RPC_URL=https://sepolia.infura.io/v3/${INFURA_API_KEY}
SOLANA_RPC_URL=https://api.devnet.solana.com
NEAR_RPC_URL=https://rpc.testnet.near.org

# Smart Contract Addresses (leave empty for now)
POLYGON_BRIDGE_CONTRACT=
BNB_BRIDGE_CONTRACT=
AVALANCHE_BRIDGE_CONTRACT=
ETHEREUM_BRIDGE_CONTRACT=
SOLANA_BRIDGE_PROGRAM=
NEAR_BRIDGE_CONTRACT=

# Validator Configuration (generate a new test wallet)
VALIDATOR_PRIVATE_KEY=your_private_key_here  # ⚠️ GENERATE NEW TEST WALLET

# NATS Configuration
NATS_URL=nats://localhost:4222

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
```

**Save and exit**: Press `Ctrl+X`, then `Y`, then `Enter`

### Generate JWT Secret

```bash
# Generate a secure JWT secret
openssl rand -hex 32

# Copy the output and paste it into your .env.production file as JWT_SECRET
```

## Step 9: Create Docker Compose File

```bash
cd ~/projects/metabridge-engine-hub

# Create docker-compose.production.yaml
cat > docker-compose.production.yaml << 'EOF'
version: '3.8'

services:
  postgres:
    image: postgres:15
    container_name: metabridge-postgres
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres_admin_password
      POSTGRES_DB: metabridge_production
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: always
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  nats:
    image: nats:2.10
    container_name: metabridge-nats
    ports:
      - "4222:4222"
      - "8222:8222"
    command: ["-js", "-m", "8222"]
    restart: always
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8222/healthz"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: metabridge-redis
    ports:
      - "6379:6379"
    restart: always
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
EOF

echo "✅ Docker Compose file created"
```

## Step 10: Start Infrastructure Services

```bash
cd ~/projects/metabridge-engine-hub

# Start PostgreSQL, NATS, and Redis
sudo docker compose -f docker-compose.production.yaml up -d

# Wait for services to start (30 seconds)
echo "⏳ Waiting for services to start..."
sleep 30

# Check service status
sudo docker compose -f docker-compose.production.yaml ps

# You should see all three services running (Up)

# Check logs if needed
sudo docker compose -f docker-compose.production.yaml logs
```

## Step 11: Initialize Database

```bash
cd ~/projects/metabridge-engine-hub

# Create database and user
sudo docker exec -i metabridge-postgres psql -U postgres << EOF
CREATE DATABASE metabridge_production;
CREATE USER bridge_user WITH ENCRYPTED PASSWORD 'YourStrongPassword123!';
GRANT ALL PRIVILEGES ON DATABASE metabridge_production TO bridge_user;
ALTER DATABASE metabridge_production OWNER TO bridge_user;
\c metabridge_production
GRANT ALL ON SCHEMA public TO bridge_user;
EOF

# Run main database schema
sudo docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production < internal/database/schema.sql

# Run authentication schema
sudo docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production < internal/database/auth.sql

# Verify tables were created
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c "\dt"

# You should see tables like: messages, batches, webhooks, routes, users, api_keys, etc.
```

## Step 12: Create Admin User

```bash
# Install bcrypt tool for password hashing
go install github.com/bitnami/bcrypt-cli@latest

# Hash your admin password (replace 'admin123' with your desired password)
~/go/bin/bcrypt-cli admin123

# Copy the hash output (starts with $2a$...)
# Example output: $2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# Insert admin user (replace <YOUR_BCRYPT_HASH> with the hash from above)
sudo docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production << 'EOF'
INSERT INTO users (id, email, name, password_hash, role, active, created_at, updated_at)
VALUES (
  'admin-001',
  'admin@metabridge.local',
  'System Administrator',
  '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
  'admin',
  true,
  NOW(),
  NOW()
);
EOF

# Verify user was created
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c "SELECT id, email, role FROM users;"
```

## Step 13: Build Bridge Services

This is a critical step where you compile all the Go binaries for your bridge system.

### Build Process Overview

The Metabridge Engine consists of 5 main binaries:
1. **metabridge-api** - Main API server (handles HTTP requests)
2. **metabridge-relayer** - Message relayer (processes cross-chain messages)
3. **metabridge-listener** - Blockchain listener (monitors chain events)
4. **metabridge-batcher** - Batch aggregator (optimizes gas costs)
5. **metabridge-migrator** - Database migrator (sets up database schema)

### Expected Build Time
- **Total Time**: 2-5 minutes (depending on your server specs)
- **Per Binary**: 30-60 seconds each
- **Download Time**: Additional 1-2 minutes for first build (dependencies)

### Build Commands

```bash
cd ~/projects/metabridge-engine-hub

# Create bin directory
mkdir -p bin

# Build 1: API Server
echo "🔨 Building API server..."
CGO_ENABLED=0 go build -o bin/metabridge-api cmd/api/main.go

# Expected Output:
# (Downloading dependencies on first build - you'll see progress bars)
# go: downloading github.com/ethereum/go-ethereum v1.13.8
# go: downloading github.com/gorilla/mux v1.8.1
# go: downloading github.com/rs/zerolog v1.31.0
# ... (20-30 more packages)
# (Then silence as it compiles - this is normal!)
# (After 30-60 seconds, command completes with no output = SUCCESS)

echo "✅ API server built"

# Build 2: Relayer Service
echo "🔨 Building relayer..."
CGO_ENABLED=0 go build -o bin/metabridge-relayer cmd/relayer/main.go

# Expected Output:
# (Faster this time since dependencies are cached)
# (15-30 seconds of silence)
# (Completes with no output = SUCCESS)

echo "✅ Relayer built"

# Build 3: Listener Service
echo "🔨 Building listener..."
CGO_ENABLED=0 go build -o bin/metabridge-listener cmd/listener/main.go

# Expected Output:
# (15-30 seconds of compilation)
# (No output = SUCCESS)

echo "✅ Listener built"

# Build 4: Batcher Service
echo "🔨 Building batcher..."
CGO_ENABLED=0 go build -o bin/metabridge-batcher cmd/batcher/main.go

# Expected Output:
# (10-20 seconds of compilation)
# (No output = SUCCESS)

echo "✅ Batcher built"

# Build 5: Database Migrator
echo "🔨 Building migrator..."
CGO_ENABLED=0 go build -o bin/metabridge-migrator cmd/migrator/main.go

# Expected Output:
# (10-20 seconds of compilation)
# (No output = SUCCESS)

echo "✅ Migrator built"

echo ""
echo "==================== BUILD VERIFICATION ===================="
echo ""

# Verify all binaries were created
ls -lh bin/

# Expected Output:
# total 106M
# -rwxr-xr-x 1 root root 27M Nov 22 14:23 metabridge-api
# -rwxr-xr-x 1 root root 13M Nov 22 14:24 metabridge-batcher
# -rwxr-xr-x 1 root root 27M Nov 22 14:24 metabridge-listener
# -rwxr-xr-x 1 root root 11M Nov 22 14:25 metabridge-migrator
# -rwxr-xr-x 1 root root 28M Nov 22 14:23 metabridge-relayer

echo ""
echo "Checking binary types..."
file bin/*

# Expected Output:
# bin/metabridge-api:      ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, Go BuildID=..., not stripped
# bin/metabridge-batcher:  ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, Go BuildID=..., not stripped
# bin/metabridge-listener: ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, Go BuildID=..., not stripped
# bin/metabridge-migrator: ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, Go BuildID=..., not stripped
# bin/metabridge-relayer:  ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, Go BuildID=..., not stripped

echo ""
echo "Testing binaries respond to --help..."

# Test each binary
bin/metabridge-api --help 2>&1 | head -5
# Expected: Shows help text or "Usage:" message

bin/metabridge-relayer --help 2>&1 | head -5
# Expected: Shows help text or "Usage:" message

echo ""
echo "✅ All binaries built successfully!"
echo ""
echo "Binary Sizes:"
du -h bin/* | column -t
echo ""
echo "Total Size: $(du -sh bin/ | awk '{print $1}')"
echo ""
```

### Alternative: Build All at Once Using Makefile

```bash
# Use the Makefile to build everything
make build

# Expected Output:
# Building Go binaries...
# CGO_ENABLED=0 go build -o bin/api ./cmd/api
# CGO_ENABLED=0 go build -o bin/relayer ./cmd/relayer
# CGO_ENABLED=0 go build -o bin/listener ./cmd/listener
# CGO_ENABLED=0 go build -o bin/batcher ./cmd/batcher
# CGO_ENABLED=0 go build -o bin/migrator ./cmd/migrator
# Build complete! Binaries in ./bin/
# total 106M
# -rwxr-xr-x 1 root root 27M Nov 22 14:23 api
# -rwxr-xr-x 1 root root 13M Nov 22 14:24 batcher
# -rwxr-xr-x 1 root root 27M Nov 22 14:24 listener
# -rwxr-xr-x 1 root root 11M Nov 22 14:25 migrator
# -rwxr-xr-x 1 root root 28M Nov 22 14:23 relayer
```

### What Does "SUCCESS" Look Like?

✅ **Successful Build Indicators:**
- No error messages displayed
- Command completes and returns to shell prompt
- Binary file created in `bin/` directory
- Binary is executable (shown as green in `ls` with colors)
- Binary responds to `--help` flag
- Binary shows "ELF 64-bit LSB executable" in `file` command

❌ **Build Failure Indicators:**
- Error messages containing "undefined:", "not found", "cannot find package"
- No binary file created
- Build process exits early
- Red error text displayed

### Common Build Issues & Solutions

#### Issue 1: Cannot Download Packages

**Error Message:**
```
go: github.com/ethereum/go-ethereum@v1.13.8: Get "https://proxy.golang.org/...": dial tcp: lookup proxy.golang.org: no such host
```

**Solution:**
```bash
# Check DNS settings
cat /etc/resolv.conf

# Fix DNS if needed
echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
echo "nameserver 8.8.4.4" | sudo tee -a /etc/resolv.conf

# Test connectivity
ping -c 3 proxy.golang.org

# Retry build
go clean -modcache
go build -o bin/metabridge-api cmd/api/main.go
```

#### Issue 2: Out of Memory

**Error Message:**
```
signal: killed
```

**Solution:**
```bash
# Check available memory
free -h

# Add swap space if needed (2GB)
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# Make permanent
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab

# Retry build
go build -o bin/metabridge-api cmd/api/main.go
```

#### Issue 3: Compilation Errors

**Error Message:**
```
# github.com/EmekaIwuagwu/metabridge-hub/internal/api
internal/api/handlers.go:123:45: undefined: SomeFunction
```

**Solution:**
```bash
# Make sure you're on the correct branch
git status
git branch

# Pull latest code
git pull origin main

# Clean and rebuild
go clean -cache
go mod tidy
go build -o bin/metabridge-api cmd/api/main.go
```

#### Issue 4: Permission Denied

**Error Message:**
```
permission denied
```

**Solution:**
```bash
# Make bin directory writable
chmod 755 bin/
chmod 644 bin/*

# Or run with sudo
sudo go build -o bin/metabridge-api cmd/api/main.go
```

### Build Verification Checklist

After building, verify everything is correct:

```bash
# ✅ 1. Check all 5 binaries exist
ls bin/ | wc -l
# Expected: 5

# ✅ 2. Check total size is reasonable
du -sh bin/
# Expected: 100M-120M (statically linked binaries)

# ✅ 3. Check binaries are executable
ls -la bin/ | grep rwx
# Expected: All 5 files show -rwxr-xr-x

# ✅ 4. Check architecture matches your server
file bin/metabridge-api
# Expected: x86-64 (for most servers)
# If you see "ARM aarch64", that's also fine (for ARM servers)

# ✅ 5. Check binaries are statically linked
ldd bin/metabridge-api
# Expected: "not a dynamic executable" or "statically linked"

# ✅ 6. Test binary execution
bin/metabridge-api --version 2>&1
# Expected: Version info or error message (but binary runs)

# ✅ 7. Check Go build cache
go clean -cache -n
# Shows what would be cleaned (means cache exists)
```

### Performance Metrics

**Expected Build Performance:**

| Binary | Size | Build Time | Dependencies |
|--------|------|------------|--------------|
| metabridge-api | ~27 MB | 30-60s | High (HTTP, DB, chains) |
| metabridge-relayer | ~28 MB | 30-60s | High (all chains, NATS) |
| metabridge-listener | ~27 MB | 30-60s | High (all chains, events) |
| metabridge-batcher | ~13 MB | 15-30s | Medium (batch logic) |
| metabridge-migrator | ~11 MB | 10-20s | Low (DB only) |

**Total:** ~106 MB, 2-5 minutes

### Advanced: Optimized Build

For production deployment with optimizations:

```bash
# Build with optimizations and version info
VERSION=$(git describe --tags --always --dirty)
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse HEAD)

go build \
  -ldflags="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}" \
  -trimpath \
  -o bin/metabridge-api \
  cmd/api/main.go

# Explanation:
# -ldflags="-s -w"  : Strip debug symbols (smaller binary)
# -X main.Version   : Inject version information
# -trimpath         : Remove file system paths from binary
# Result: Smaller binaries (~20-25% reduction)
```

### Next Steps

Once all binaries are built successfully, proceed to Step 14 to create systemd services that will run these binaries automatically.

## Step 14: Create Systemd Services

### API Server Service

```bash
sudo tee /etc/systemd/system/metabridge-api.service > /dev/null << EOF
[Unit]
Description=Metabridge API Server
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=$USER
WorkingDirectory=/home/$USER/projects/metabridge-engine-hub
ExecStart=/home/$USER/projects/metabridge-engine-hub/bin/metabridge-api
EnvironmentFile=/home/$USER/projects/metabridge-engine-hub/.env.production
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```

### Relayer Service

```bash
sudo tee /etc/systemd/system/metabridge-relayer.service > /dev/null << EOF
[Unit]
Description=Metabridge Relayer Service
After=network.target docker.service metabridge-api.service
Requires=docker.service

[Service]
Type=simple
User=$USER
WorkingDirectory=/home/$USER/projects/metabridge-engine-hub
ExecStart=/home/$USER/projects/metabridge-engine-hub/bin/metabridge-relayer --config /home/$USER/projects/metabridge-engine-hub/config/config.testnet.yaml
EnvironmentFile=/home/$USER/projects/metabridge-engine-hub/.env.production
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```

### Enable and Start Services

```bash
# Reload systemd daemon
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
```

## Step 15: Configure Firewall

```bash
# Install UFW if not already installed
sudo apt install -y ufw

# Allow SSH (CRITICAL - do this first!)
sudo ufw allow 22/tcp

# Allow HTTP
sudo ufw allow 80/tcp

# Allow HTTPS
sudo ufw allow 443/tcp

# Allow API server
sudo ufw allow 8080/tcp

# Enable firewall
sudo ufw --force enable

# Check firewall status
sudo ufw status verbose
```

## Step 16: Run Comprehensive Tests

This section provides detailed tests to verify your deployment is working correctly. Each test includes the exact command, expected output, and what to do if the test fails.

### Test 1: Infrastructure Health Checks

These tests verify that all your infrastructure services (PostgreSQL, NATS, Redis) are running properly.

```bash
echo "========================================="
echo "Test 1: Infrastructure Health Checks"
echo "========================================="
echo ""

# Check all Docker containers are running
echo "Checking Docker containers..."
sudo docker compose -f ~/projects/metabridge-engine-hub/docker-compose.production.yaml ps

# Expected output:
# NAME                    IMAGE              COMMAND                  SERVICE    CREATED          STATUS                    PORTS
# metabridge-nats         nats:2.10          "/nats-server -js -m…"   nats       10 minutes ago   Up 10 minutes (healthy)   0.0.0.0:4222->4222/tcp, 0.0.0.0:8222->8222/tcp
# metabridge-postgres     postgres:15        "docker-entrypoint.s…"   postgres   10 minutes ago   Up 10 minutes (healthy)   0.0.0.0:5432->5432/tcp
# metabridge-redis        redis:7-alpine     "docker-entrypoint.s…"   redis      10 minutes ago   Up 10 minutes (healthy)   0.0.0.0:6379->6379/tcp

# What to look for:
# ✅ STATUS column shows "Up" for all containers
# ✅ "(healthy)" appears next to each container
# ❌ If "Exit" or "Restarting" appears, container has issues

echo ""
echo "Testing PostgreSQL connection..."
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c "SELECT version();"

# Expected output:
#                                                           version
# -----------------------------------------------------------------------------------------------------------------------------
#  PostgreSQL 15.5 (Debian 15.5-1.pgdg120+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 12.2.0-14) 12.2.0, 64-bit
# (1 row)

# ✅ Shows PostgreSQL 15.x version
# ❌ If error "could not connect": Check container is running

echo ""
echo "Testing NATS connection..."
curl -s http://localhost:8222/varz | jq '.' | head -20

# Expected output (JSON with NATS stats):
# {
#   "server_id": "NDJZ...",
#   "server_name": "NDJZ...",
#   "version": "2.10.0",
#   "proto": 1,
#   "git_commit": "...",
#   "go": "go1.21.3",
#   "host": "0.0.0.0",
#   "port": 4222,
#   "max_connections": 65536,
#   "ping_interval": 120000000000,
#   "ping_max": 2,
#   "http_host": "0.0.0.0",
#   "http_port": 8222,
#   "https_port": 0,
#   "auth_timeout": 1,
#   "max_control_line": 4096,
#   ...
# }

# ✅ Shows JSON with server stats and version 2.10.x
# ❌ If "Connection refused": NATS container not running
# Note: Install jq if not available: sudo apt install -y jq

echo ""
echo "Testing Redis connection..."
sudo docker exec -it metabridge-redis redis-cli ping

# Expected output:
# PONG

# ✅ Responds with "PONG"
# ❌ If "Could not connect": Redis container not running

echo ""
echo "Testing Redis data operations..."
sudo docker exec -it metabridge-redis redis-cli SET test_key "test_value"
sudo docker exec -it metabridge-redis redis-cli GET test_key
sudo docker exec -it metabridge-redis redis-cli DEL test_key

# Expected output:
# OK
# "test_value"
# (integer) 1

# ✅ Redis can store and retrieve data
# ❌ If errors: Check Redis logs

echo ""
echo "✅ All infrastructure services are healthy!"
echo ""
```

### Test 2: API Health Checks

These tests verify that your API server is running and responding to requests correctly.

```bash
echo "========================================="
echo "Test 2: API Health Checks"
echo "========================================="
echo ""

# Test 1: Basic health endpoint
echo "Testing basic health endpoint..."
curl -s http://159.65.73.133:8080/health | jq '.'

# Expected output:
# {
#   "status": "healthy",
#   "timestamp": "2025-11-22T14:30:45Z",
#   "version": "1.0.0",
#   "uptime": 3600
# }

# ✅ Status is "healthy"
# ✅ Returns valid JSON
# ✅ Timestamp is current
# ❌ If "Connection refused": API server not running
# ❌ If HTML error page: Wrong port or nginx issue
# ❌ If timeout: Firewall blocking port 8080

echo ""
echo "Testing with verbose output for debugging..."
curl -v http://159.65.73.133:8080/health 2>&1 | grep -E '(HTTP|status)'

# Expected output:
# > GET /health HTTP/1.1
# < HTTP/1.1 200 OK
# < Content-Type: application/json
# {"status":"healthy",...}

# ✅ HTTP/1.1 200 OK response
# ❌ HTTP/1.1 404 Not Found: Route not configured
# ❌ HTTP/1.1 500 Internal Server Error: Server crash

echo ""
echo "Testing detailed API status..."
curl -s http://159.65.73.133:8080/v1/status | jq '.'

# Expected output:
# {
#   "api": {
#     "status": "healthy",
#     "version": "1.0.0",
#     "uptime_seconds": 3600
#   },
#   "database": {
#     "status": "connected",
#     "type": "postgresql",
#     "ping_ms": 2
#   },
#   "nats": {
#     "status": "connected",
#     "url": "nats://localhost:4222",
#     "servers": 1
#   },
#   "redis": {
#     "status": "connected",
#     "ping_ms": 1
#   }
# }

# ✅ All services show "connected" or "healthy"
# ❌ If any service shows "disconnected": Check that service
# ❌ If database ping_ms > 100: Database performance issue

echo ""
echo "Testing chain connectivity..."
curl -s http://159.65.73.133:8080/v1/chains/status | jq '.'

# Expected output:
# {
#   "ethereum": {
#     "chain_id": 11155111,
#     "name": "Ethereum Sepolia",
#     "type": "evm",
#     "healthy": true,
#     "rpc_url": "https://sepolia.infura.io/v3/...",
#     "block_number": 5234567,
#     "last_check": "2025-11-22T14:30:45Z",
#     "latency_ms": 245
#   },
#   "polygon": {
#     "chain_id": 80002,
#     "name": "Polygon Amoy",
#     "type": "evm",
#     "healthy": true,
#     "rpc_url": "https://rpc-amoy.polygon.technology/",
#     "block_number": 12345678,
#     "last_check": "2025-11-22T14:30:45Z",
#     "latency_ms": 189
#   },
#   "bnb": {
#     "chain_id": 97,
#     "name": "BNB Testnet",
#     "type": "evm",
#     "healthy": true,
#     "block_number": 34567890,
#     "latency_ms": 156
#   },
#   "avalanche": {
#     "chain_id": 43113,
#     "name": "Avalanche Fuji",
#     "type": "evm",
#     "healthy": true,
#     "block_number": 23456789,
#     "latency_ms": 203
#   },
#   "solana": {
#     "name": "Solana Devnet",
#     "type": "solana",
#     "healthy": true,
#     "rpc_url": "https://api.devnet.solana.com",
#     "slot": 287654321,
#     "latency_ms": 178
#   },
#   "near": {
#     "name": "NEAR Testnet",
#     "type": "near",
#     "healthy": true,
#     "rpc_url": "https://rpc.testnet.near.org",
#     "block_height": 123456789,
#     "latency_ms": 312
#   }
# }

# ✅ All chains show "healthy": true
# ✅ Block numbers/slots are recent
# ✅ Latency < 500ms (acceptable < 1000ms)
# ⚠️ If healthy: false - RPC endpoint down or API key issue
# ⚠️ If high latency (>1000ms) - Network congestion or slow RPC

echo ""
echo "Testing bridge statistics..."
curl -s http://159.65.73.133:8080/v1/stats | jq '.'

# Expected output (fresh deployment):
# {
#   "total_messages": 0,
#   "pending_messages": 0,
#   "processing_messages": 0,
#   "completed_messages": 0,
#   "failed_messages": 0,
#   "total_volume_usd": "0",
#   "total_fees_usd": "0",
#   "success_rate": 0,
#   "average_processing_time_seconds": 0,
#   "chains": {
#     "ethereum": {"sent": 0, "received": 0},
#     "polygon": {"sent": 0, "received": 0},
#     "bnb": {"sent": 0, "received": 0},
#     "avalanche": {"sent": 0, "received": 0},
#     "solana": {"sent": 0, "received": 0},
#     "near": {"sent": 0, "received": 0}
#   }
# }

# ✅ Returns valid stats structure
# ✅ All values are 0 for fresh deployment
# ❌ If error: Database connection issue

echo ""
echo "Testing API response times..."
time curl -s http://159.65.73.133:8080/health > /dev/null

# Expected output:
# real    0m0.052s
# user    0m0.012s
# sys     0m0.008s

# ✅ Response time < 100ms is excellent
# ✅ Response time < 500ms is acceptable
# ⚠️ Response time > 1s indicates performance issues

echo ""
echo "Testing CORS headers..."
curl -s -I http://159.65.73.133:8080/health | grep -i 'access-control'

# Expected output:
# Access-Control-Allow-Origin: *
# Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
# Access-Control-Allow-Headers: Content-Type, Authorization

# ✅ CORS headers present (needed for web frontends)
# ⚠️ If missing: Check .env.production CORS settings

echo ""
echo "✅ All API health checks passed!"
echo ""
```

### Test 3: Authentication Tests

```bash
# Login test (if auth enabled)
curl -X POST http://159.65.73.133:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@metabridge.local",
    "password": "admin123"
  }'

# Expected: JWT token in response
# {
#   "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
#   "user": {
#     "id": "admin-001",
#     "email": "admin@metabridge.local",
#     "role": "admin"
#   }
# }

# Save the token for later use
export JWT_TOKEN="<your_token_here>"

# Test authenticated endpoint
curl http://159.65.73.133:8080/v1/admin/users \
  -H "Authorization: Bearer $JWT_TOKEN"

# Expected: List of users
```

### Test 4: Database Tests

```bash
# Test 1: Check all tables exist
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production << 'EOF'
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'public'
ORDER BY table_name;
EOF

# Expected tables:
# - messages
# - batches
# - batch_messages
# - routes
# - webhooks
# - users
# - api_keys
# - sessions

# Test 2: Verify admin user exists
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production << 'EOF'
SELECT id, email, role, active FROM users;
EOF

# Expected: admin-001 | admin@metabridge.local | admin | true

# Test 3: Check database size
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c \
  "SELECT pg_size_pretty(pg_database_size('metabridge_production')) as size;"

# Expected: Database size (should be ~50MB for fresh install)
```

### Test 5: Service Monitoring Tests

```bash
# Check systemd service status
sudo systemctl status metabridge-api --no-pager
sudo systemctl status metabridge-relayer --no-pager

# Both should show:
# Active: active (running)

# Check service logs for errors
sudo journalctl -u metabridge-api --since "5 minutes ago" --no-pager | grep -i error

# Expected: No critical errors (some warnings are normal)

# Check resource usage
ps aux | grep metabridge

# Expected: See api and relayer processes running

# Check memory usage
free -h

# Expected: At least 1GB free memory

# Check disk usage
df -h

# Expected: At least 10GB free disk space
```

### Test 6: Network Connectivity Tests

```bash
# Test 1: Check open ports
sudo netstat -tlnp | grep -E '(8080|5432|4222|6379)'

# Expected ports:
# - 8080  (API)
# - 5432  (PostgreSQL)
# - 4222  (NATS)
# - 6379  (Redis)

# Test 2: Test external API access
curl -I http://159.65.73.133:8080/health

# Expected:
# HTTP/1.1 200 OK
# Content-Type: application/json

# Test 3: Test firewall rules
sudo ufw status numbered

# Expected rules:
# [1] 22/tcp                     ALLOW IN    Anywhere
# [2] 80/tcp                     ALLOW IN    Anywhere
# [3] 443/tcp                    ALLOW IN    Anywhere
# [4] 8080/tcp                   ALLOW IN    Anywhere
```

### Test 7: Production-Ready End-to-End Bridge Flow Test

This comprehensive test walks you through the complete bridge lifecycle from smart contract deployment to successful cross-chain token transfer.

**Test Overview:**
- Deploy smart contracts to Polygon Amoy and BNB Testnet
- Deploy test ERC20 token on both chains
- Bridge 10 tokens from Polygon → BNB
- Verify at every step with expected responses

**Prerequisites:**
- Metamask wallet with testnet funds (Polygon Amoy, BNB Testnet)
- Hardhat installed: `cd contracts/evm && npm install`
- Test wallet private key in `.env.production`

---

#### Part 1: Smart Contract Deployment

##### Step 1.1: Deploy Bridge Contract to Polygon Amoy

```bash
cd ~/projects/metabridge-engine-hub/contracts/evm

# Set environment variables
export PRIVATE_KEY="your_test_wallet_private_key_here"
export INFURA_API_KEY="your_infura_api_key"

# Deploy to Polygon Amoy (Testnet)
echo "Deploying bridge contract to Polygon Amoy..."
npx hardhat run scripts/deploy.js --network polygon-amoy

# Expected Output:
# Deploying MetabridgeV1...
# Waiting for confirmations...
# MetabridgeV1 deployed to: 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb5
# Transaction hash: 0x1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f
# Block number: 12345678
# Gas used: 2,847,391
# Deployer address: 0xYourWalletAddress
#
# Initializing contract...
# Setting validators: [0xValidator1, 0xValidator2, 0xValidator3]
# Threshold set to: 2 (2-of-3 multisig)
# ✅ Deployment complete!

# Save the contract address
export POLYGON_BRIDGE_ADDRESS="0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb5"
echo "POLYGON_BRIDGE_CONTRACT=$POLYGON_BRIDGE_ADDRESS" >> ~/.env.production

# Verify on PolygonScan
echo "Verify at: https://amoy.polygonscan.com/address/$POLYGON_BRIDGE_ADDRESS"
```

**✅ What to verify on PolygonScan:**
- Contract shows as "Contract" (green checkmark)
- Contract Creation transaction is successful
- Contract Code tab shows verified source code (after running verify script)
- Read Contract shows correct validator addresses
- Threshold shows "2"

##### Step 1.2: Deploy Bridge Contract to BNB Testnet

```bash
echo "Deploying bridge contract to BNB Testnet..."
npx hardhat run scripts/deploy.js --network bnb-testnet

# Expected Output:
# Deploying MetabridgeV1...
# Waiting for confirmations...
# MetabridgeV1 deployed to: 0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063
# Transaction hash: 0x2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g
# Block number: 34567890
# Gas used: 2,847,391
# Deployer address: 0xYourWalletAddress
#
# Initializing contract...
# Setting validators: [0xValidator1, 0xValidator2, 0xValidator3]
# Threshold set to: 2 (2-of-3 multisig)
# ✅ Deployment complete!

# Save the contract address
export BNB_BRIDGE_ADDRESS="0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063"
echo "BNB_BRIDGE_CONTRACT=$BNB_BRIDGE_ADDRESS" >> ~/.env.production

# Verify on BscScan
echo "Verify at: https://testnet.bscscan.com/address/$BNB_BRIDGE_ADDRESS"
```

##### Step 1.3: Verify Contract Deployment

```bash
echo "Verifying contracts on block explorers..."

# Verify Polygon Amoy
npx hardhat verify --network polygon-amoy $POLYGON_BRIDGE_ADDRESS

# Expected Output:
# Verifying contract on Etherscan...
# Successfully submitted source code for contract
# contracts/MetabridgeV1.sol:MetabridgeV1 at 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb5
# for verification on the block explorer. Waiting for verification result...
#
# Successfully verified contract MetabridgeV1 on PolygonScan.
# https://amoy.polygonscan.com/address/0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb5#code

# Verify BNB Testnet
npx hardhat verify --network bnb-testnet $BNB_BRIDGE_ADDRESS

# Expected Output:
# Successfully verified contract MetabridgeV1 on BscScan.
# https://testnet.bscscan.com/address/0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063#code
```

##### Step 1.4: Deploy Test ERC20 Token

```bash
echo "Deploying test token to Polygon Amoy..."
npx hardhat run scripts/deploy-token.js --network polygon-amoy

# Expected Output:
# Deploying TestToken (USDC)...
# Token deployed to: 0x9c3C9283D3e44854697Cd22D3Faa240Cfb032889
# Name: USD Coin Test
# Symbol: USDC
# Decimals: 6
# Total Supply: 1,000,000 USDC (1000000000000)
# ✅ Token deployment complete!

export POLYGON_TOKEN_ADDRESS="0x9c3C9283D3e44854697Cd22D3Faa240Cfb032889"

# Mint test tokens to your wallet
npx hardhat run scripts/mint-tokens.js --network polygon-amoy

# Expected Output:
# Minting 1000 USDC to 0xYourWalletAddress...
# Transaction hash: 0x3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h
# ✅ Minted 1000 USDC
# Your balance: 1000000000 (1000 USDC)

echo "Deploying wrapped token to BNB Testnet..."
npx hardhat run scripts/deploy-token.js --network bnb-testnet

# Expected Output:
# Deploying TestToken (USDC)...
# Token deployed to: 0x4B0897b0513fdC7C541B6d9D7E929C4e5364D2dB
# ✅ Token deployment complete!

export BNB_TOKEN_ADDRESS="0x4B0897b0513fdC7C541B6d9D7E929C4e5364D2dB"
```

##### Step 1.5: Update Backend Configuration

```bash
# Update .env.production with contract addresses
cat >> ~/projects/metabridge-engine-hub/.env.production << EOF

# Smart Contract Addresses (Updated)
POLYGON_BRIDGE_CONTRACT=$POLYGON_BRIDGE_ADDRESS
BNB_BRIDGE_CONTRACT=$BNB_BRIDGE_ADDRESS
POLYGON_USDC_TOKEN=$POLYGON_TOKEN_ADDRESS
BNB_USDC_TOKEN=$BNB_TOKEN_ADDRESS
EOF

# Restart services to load new configuration
sudo systemctl restart metabridge-api
sudo systemctl restart metabridge-relayer
sudo systemctl restart metabridge-listener

# Wait for services to restart
sleep 10

# Verify services picked up new contracts
curl -s http://159.65.73.133:8080/v1/contracts | jq '.'

# Expected Output:
# {
#   "polygon": {
#     "bridge": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb5",
#     "tokens": {
#       "USDC": "0x9c3C9283D3e44854697Cd22D3Faa240Cfb032889"
#     }
#   },
#   "bnb": {
#     "bridge": "0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063",
#     "tokens": {
#       "USDC": "0x4B0897b0513fdC7C541B6d9D7E929C4e5364D2dB"
#     }
#   }
# }
```

---

#### Part 2: End-to-End Token Bridge Test (Polygon → BNB)

##### Step 2.1: Check Initial Balances

```bash
# Check your token balance on Polygon
npx hardhat run scripts/check-balance.js --network polygon-amoy

# Expected Output:
# Checking balance for: 0xYourWalletAddress
# USDC Balance on Polygon: 1000.000000 USDC
# Native MATIC Balance: 2.456789 MATIC

# Check your token balance on BNB
npx hardhat run scripts/check-balance.js --network bnb-testnet

# Expected Output:
# Checking balance for: 0xYourWalletAddress
# USDC Balance on BNB: 0.000000 USDC
# Native BNB Balance: 0.234567 BNB

# Save initial state
export INITIAL_POLYGON_BALANCE="1000000000"  # 1000 USDC (6 decimals)
export INITIAL_BNB_BALANCE="0"
export BRIDGE_AMOUNT="10000000"  # 10 USDC (6 decimals)
```

##### Step 2.2: Approve Bridge Contract

```bash
echo "Step 1: Approving bridge contract to spend tokens..."

# Approve the Polygon bridge contract to spend your USDC
npx hardhat run scripts/approve-bridge.js --network polygon-amoy

# Expected Output:
# Approving Polygon Bridge (0x742d35Cc...) to spend 10 USDC...
# Token: 0x9c3C9283D3e44854697Cd22D3Faa240Cfb032889
# Spender: 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb5
# Amount: 10000000 (10 USDC)
#
# Sending approval transaction...
# Transaction hash: 0x4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i
# Waiting for confirmation...
#
# ✅ Approval confirmed!
# Block number: 12345690
# Gas used: 46,523
#
# Checking allowance...
# Current allowance: 10000000 (10 USDC)
# ✅ Approval successful!

# Verify on PolygonScan
# https://amoy.polygonscan.com/tx/0x4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i
```

**✅ What to verify on PolygonScan:**
- Transaction Status: Success ✓
- Method: "Approve"
- Logs show "Approval" event
- Spender: 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb5 (Bridge Contract)
- Value: 10000000 (10 USDC)

##### Step 2.3: Lock Tokens on Source Chain (Polygon)

```bash
echo "Step 2: Locking tokens on Polygon..."

# Lock tokens on Polygon bridge
npx hardhat run scripts/bridge-lock.js --network polygon-amoy

# Expected Output:
# ================================================
# BRIDGE LOCK TRANSACTION
# ================================================
# Source Chain: Polygon Amoy (Chain ID: 80002)
# Destination Chain: BNB Testnet (Chain ID: 97)
# Token: 0x9c3C9283D3e44854697Cd22D3Faa240Cfb032889 (USDC)
# Amount: 10000000 (10 USDC)
# Sender: 0xYourWalletAddress
# Recipient: 0xYourWalletAddress
# Bridge Contract: 0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb5
#
# Sending lock transaction...
# Transaction hash: 0x5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j
# Waiting for confirmation...
#
# ✅ Transaction confirmed!
# Block number: 12345695
# Gas used: 127,845
#
# Event: TokensLocked
#   - messageId: 0x7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j7k
#   - sourceChain: 80002
#   - destinationChain: 97
#   - token: 0x9c3C9283D3e44854697Cd22D3Faa240Cfb032889
#   - sender: 0xYourWalletAddress
#   - recipient: 0xYourWalletAddress
#   - amount: 10000000
#   - nonce: 1
#   - timestamp: 1700654321
#
# ✅ Tokens successfully locked!
#
# Next steps:
# 1. Wait for listener to detect event (~30 seconds)
# 2. Wait for validators to sign message (~2 minutes)
# 3. Wait for relayer to submit on BNB (~3 minutes)
#
# Track your bridge request:
# Message ID: 0x7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j7k
# View on PolygonScan: https://amoy.polygonscan.com/tx/0x5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j

export MESSAGE_ID="0x7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j7k"
export LOCK_TX_HASH="0x5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j"
```

**✅ What to verify on PolygonScan:**
- Transaction Status: Success ✓
- Method: "lock" or "lockTokens"
- Logs show "TokensLocked" event with correct parameters
- Your USDC balance decreased by 10 USDC
- Bridge contract balance increased by 10 USDC

##### Step 2.4: Monitor Listener Detection

```bash
echo "Step 3: Monitoring listener for event detection..."

# Check listener logs
sudo journalctl -u metabridge-listener -f --since "1 minute ago" | grep -E "(TokensLocked|$MESSAGE_ID)"

# Expected Log Output:
# Nov 22 14:35:12 metabridge-listener[12345]: {"level":"info","time":"2025-11-22T14:35:12Z","message":"New block detected","chain":"polygon","block":12345695}
# Nov 22 14:35:13 metabridge-listener[12345]: {"level":"info","time":"2025-11-22T14:35:13Z","message":"Event detected","event":"TokensLocked","txHash":"0x5e6f7g8h..."}
# Nov 22 14:35:13 metabridge-listener[12345]: {"level":"info","time":"2025-11-22T14:35:13Z","message":"Processing lock event","messageId":"0x7g8h9i0j...","amount":"10000000"}
# Nov 22 14:35:14 metabridge-listener[12345]: {"level":"info","time":"2025-11-22T14:35:14Z","message":"Message stored in database","messageId":"0x7g8h9i0j...","status":"pending"}
# Nov 22 14:35:14 metabridge-listener[12345]: {"level":"info","time":"2025-11-22T14:35:14Z","message":"Message published to NATS","subject":"bridge.message.new","messageId":"0x7g8h9i0j..."}

# Check via API (wait 30 seconds after lock transaction)
sleep 30
curl -s "http://159.65.73.133:8080/v1/messages/$MESSAGE_ID" | jq '.'

# Expected API Response:
# {
#   "id": "0x7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j7k",
#   "source_chain": "polygon",
#   "destination_chain": "bnb",
#   "source_tx_hash": "0x5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j",
#   "destination_tx_hash": null,
#   "token_address": "0x9c3C9283D3e44854697Cd22D3Faa240Cfb032889",
#   "sender": "0xYourWalletAddress",
#   "recipient": "0xYourWalletAddress",
#   "amount": "10000000",
#   "status": "pending",
#   "validator_signatures": [],
#   "created_at": "2025-11-22T14:35:13Z",
#   "updated_at": "2025-11-22T14:35:14Z"
# }

# ✅ Message is detected and in "pending" status
```

##### Step 2.5: Monitor Validator Signing

```bash
echo "Step 4: Waiting for validator signatures..."

# Monitor relayer logs for validator signing
sudo journalctl -u metabridge-relayer -f --since "1 minute ago" | grep -E "(Signing|Signature|$MESSAGE_ID)"

# Expected Log Output:
# Nov 22 14:35:30 metabridge-relayer[12346]: {"level":"info","time":"2025-11-22T14:35:30Z","message":"Processing new message","messageId":"0x7g8h9i0j..."}
# Nov 22 14:35:30 metabridge-relayer[12346]: {"level":"info","time":"2025-11-22T14:35:30Z","message":"Validating message","messageId":"0x7g8h9i0j..."}
# Nov 22 14:35:31 metabridge-relayer[12346]: {"level":"info","time":"2025-11-22T14:35:31Z","message":"Verifying source transaction","txHash":"0x5e6f7g8h...","chain":"polygon"}
# Nov 22 14:35:32 metabridge-relayer[12346]: {"level":"info","time":"2025-11-22T14:35:32Z","message":"Source transaction verified","confirmations":12}
# Nov 22 14:35:33 metabridge-relayer[12346]: {"level":"info","time":"2025-11-22T14:35:33Z","message":"Generating signature","messageId":"0x7g8h9i0j...","validator":"0xValidator1"}
# Nov 22 14:35:33 metabridge-relayer[12346]: {"level":"info","time":"2025-11-22T14:35:33Z","message":"Signature created","signature":"0x8h9i0j...","validator":"0xValidator1"}
# Nov 22 14:35:34 metabridge-relayer[12346]: {"level":"info","time":"2025-11-22T14:35:34Z","message":"Signature stored","messageId":"0x7g8h9i0j...","signatureCount":1,"required":2}
# Nov 22 14:36:45 metabridge-relayer[12347]: {"level":"info","time":"2025-11-22T14:36:45Z","message":"Signature created","signature":"0x9i0j1k...","validator":"0xValidator2"}
# Nov 22 14:36:45 metabridge-relayer[12347]: {"level":"info","time":"2025-11-22T14:36:45Z","message":"Threshold reached","messageId":"0x7g8h9i0j...","signatureCount":2,"required":2}
# Nov 22 14:36:46 metabridge-relayer[12347]: {"level":"info","time":"2025-11-22T14:36:46Z","message":"Message ready for relay","messageId":"0x7g8h9i0j..."}

# Check signatures via API (wait 2 minutes)
sleep 90
curl -s "http://159.65.73.133:8080/v1/messages/$MESSAGE_ID" | jq '.'

# Expected API Response:
# {
#   "id": "0x7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j7k",
#   "status": "signed",
#   "validator_signatures": [
#     {
#       "validator": "0xValidator1Address",
#       "signature": "0x8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j7k8l",
#       "signed_at": "2025-11-22T14:35:33Z"
#     },
#     {
#       "validator": "0xValidator2Address",
#       "signature": "0x9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j7k8l9m",
#       "signed_at": "2025-11-22T14:36:45Z"
#     }
#   ],
#   ...
# }

# ✅ Message has 2/2 required signatures
```

##### Step 2.6: Monitor Unlock on Destination Chain (BNB)

```bash
echo "Step 5: Monitoring unlock transaction on BNB..."

# Monitor relayer logs for unlock submission
sudo journalctl -u metabridge-relayer -f --since "1 minute ago" | grep -E "(Unlock|Submitting|$MESSAGE_ID)"

# Expected Log Output:
# Nov 22 14:37:00 metabridge-relayer[12347]: {"level":"info","time":"2025-11-22T14:37:00Z","message":"Preparing unlock transaction","messageId":"0x7g8h9i0j...","chain":"bnb"}
# Nov 22 14:37:01 metabridge-relayer[12347]: {"level":"info","time":"2025-11-22T14:37:01Z","message":"Building unlock payload","recipient":"0xYourWalletAddress","amount":"10000000"}
# Nov 22 14:37:02 metabridge-relayer[12347]: {"level":"info","time":"2025-11-22T14:37:02Z","message":"Estimating gas","chain":"bnb","estimatedGas":145000}
# Nov 22 14:37:03 metabridge-relayer[12347]: {"level":"info","time":"2025-11-22T14:37:03Z","message":"Submitting unlock transaction","messageId":"0x7g8h9i0j..."}
# Nov 22 14:37:05 metabridge-relayer[12347]: {"level":"info","time":"2025-11-22T14:37:05Z","message":"Unlock transaction sent","txHash":"0x6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j7k","chain":"bnb"}
# Nov 22 14:37:06 metabridge-relayer[12347]: {"level":"info","time":"2025-11-22T14:37:06Z","message":"Waiting for transaction confirmation","txHash":"0x6f7g8h9i..."}
# Nov 22 14:37:25 metabridge-relayer[12347]: {"level":"info","time":"2025-11-22T14:37:25Z","message":"Transaction confirmed","txHash":"0x6f7g8h9i...","blockNumber":34567920,"confirmations":12}
# Nov 22 14:37:26 metabridge-relayer[12347]: {"level":"info","time":"2025-11-22T14:37:26Z","message":"Message completed","messageId":"0x7g8h9i0j...","status":"completed"}

export UNLOCK_TX_HASH="0x6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j7k"

# Check final message status via API
curl -s "http://159.65.73.133:8080/v1/messages/$MESSAGE_ID" | jq '.'

# Expected API Response:
# {
#   "id": "0x7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j7k",
#   "source_chain": "polygon",
#   "destination_chain": "bnb",
#   "source_tx_hash": "0x5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j",
#   "destination_tx_hash": "0x6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j7k",
#   "token_address": "0x9c3C9283D3e44854697Cd22D3Faa240Cfb032889",
#   "sender": "0xYourWalletAddress",
#   "recipient": "0xYourWalletAddress",
#   "amount": "10000000",
#   "status": "completed",
#   "validator_signatures": [ /* 2 signatures */ ],
#   "processing_time_seconds": 193,
#   "source_block": 12345695,
#   "destination_block": 34567920,
#   "created_at": "2025-11-22T14:35:13Z",
#   "completed_at": "2025-11-22T14:37:26Z",
#   "updated_at": "2025-11-22T14:37:26Z"
# }

# ✅ Message status is "completed"
# ✅ destination_tx_hash is set
# ✅ Processing time: ~3 minutes

echo "Verify unlock transaction on BscScan:"
echo "https://testnet.bscscan.com/tx/$UNLOCK_TX_HASH"
```

**✅ What to verify on BscScan:**
- Transaction Status: Success ✓
- Method: "unlock" or "unlockTokens"
- Logs show "TokensUnlocked" event
- To: 0xYourWalletAddress
- Token Transfer: 10 USDC to your address
- Message ID matches

##### Step 2.7: Verify Final Balances

```bash
echo "Step 6: Verifying final token balances..."

# Check final balance on Polygon
npx hardhat run scripts/check-balance.js --network polygon-amoy

# Expected Output:
# Checking balance for: 0xYourWalletAddress
# USDC Balance on Polygon: 990.000000 USDC (was 1000 USDC)
# Change: -10 USDC ✓

# Check final balance on BNB
npx hardhat run scripts/check-balance.js --network bnb-testnet

# Expected Output:
# Checking balance for: 0xYourWalletAddress
# USDC Balance on BNB: 10.000000 USDC (was 0 USDC)
# Change: +10 USDC ✓

# Verify bridge statistics
curl -s http://159.65.73.133:8080/v1/stats | jq '.'

# Expected Output:
# {
#   "total_messages": 1,
#   "pending_messages": 0,
#   "processing_messages": 0,
#   "completed_messages": 1,
#   "failed_messages": 0,
#   "total_volume_usd": "10.00",
#   "total_fees_usd": "0.15",
#   "success_rate": 100,
#   "average_processing_time_seconds": 193,
#   "chains": {
#     "polygon": {"sent": 1, "received": 0, "volume_usd": "10.00"},
#     "bnb": {"sent": 0, "received": 1, "volume_usd": "10.00"}
#   }
# }
```

##### Step 2.8: Database Verification

```bash
echo "Step 7: Verifying database records..."

# Check message in database
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production << EOF
SELECT
  id,
  source_chain,
  destination_chain,
  status,
  amount,
  created_at,
  completed_at,
  EXTRACT(EPOCH FROM (completed_at - created_at)) as processing_seconds
FROM messages
WHERE id = '$MESSAGE_ID';
EOF

# Expected Output:
#                                        id                                        | source_chain | destination_chain | status    | amount   |         created_at         |        completed_at        | processing_seconds
# ---------------------------------------------------------------------------------+--------------+-------------------+-----------+----------+----------------------------+----------------------------+--------------------
#  0x7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j7k | polygon      | bnb               | completed | 10000000 | 2025-11-22 14:35:13.123456 | 2025-11-22 14:37:26.789012 |             193.67
# (1 row)

# Check validator signatures
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production << EOF
SELECT
  message_id,
  validator_address,
  LEFT(signature, 20) as signature_prefix,
  created_at
FROM validator_signatures
WHERE message_id = '$MESSAGE_ID'
ORDER BY created_at;
EOF

# Expected Output:
#                                     message_id                                     |            validator_address              | signature_prefix |         created_at
# -----------------------------------------------------------------------------------+--------------------------------------------+------------------+----------------------------
#  0x7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j7k | 0xValidator1Address                        | 0x8h9i0j1k2l3m4n | 2025-11-22 14:35:33.456789
#  0x7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5i6j7k | 0xValidator2Address                        | 0x9i0j1k2l3m4n5o | 2025-11-22 14:36:45.123456
# (2 rows)

# ✅ Database shows complete record with both signatures
```

---

#### Part 3: Test Summary and Verification Checklist

##### ✅ Complete End-to-End Verification Checklist

```bash
echo "========================================="
echo "END-TO-END TEST VERIFICATION SUMMARY"
echo "========================================="
echo ""

# 1. Smart Contracts Deployed
echo "✅ 1. Smart Contracts:"
echo "   Polygon Bridge: $POLYGON_BRIDGE_ADDRESS"
echo "   BNB Bridge: $BNB_BRIDGE_ADDRESS"
echo "   Polygon USDC: $POLYGON_TOKEN_ADDRESS"
echo "   BNB USDC: $BNB_TOKEN_ADDRESS"
echo ""

# 2. Lock Transaction Successful
echo "✅ 2. Lock Transaction (Polygon):"
echo "   TX Hash: $LOCK_TX_HASH"
echo "   Block: View on https://amoy.polygonscan.com/tx/$LOCK_TX_HASH"
echo "   Status: ✓ Confirmed"
echo "   Amount Locked: 10 USDC"
echo ""

# 3. Message Detected and Stored
echo "✅ 3. Message Detection:"
echo "   Message ID: $MESSAGE_ID"
echo "   Status: Detected by listener"
echo "   Stored in database: ✓"
echo ""

# 4. Validator Signatures Collected
echo "✅ 4. Validator Signatures:"
echo "   Required: 2-of-3"
echo "   Collected: 2 signatures"
echo "   Validators: Validator1, Validator2"
echo ""

# 5. Unlock Transaction Successful
echo "✅ 5. Unlock Transaction (BNB):"
echo "   TX Hash: $UNLOCK_TX_HASH"
echo "   Block: View on https://testnet.bscscan.com/tx/$UNLOCK_TX_HASH"
echo "   Status: ✓ Confirmed"
echo "   Amount Unlocked: 10 USDC"
echo ""

# 6. Balance Changes Verified
echo "✅ 6. Token Balances:"
echo "   Polygon: 1000 → 990 USDC (-10 USDC) ✓"
echo "   BNB: 0 → 10 USDC (+10 USDC) ✓"
echo ""

# 7. Processing Time
echo "✅ 7. Performance:"
echo "   Total Processing Time: ~3 minutes"
echo "   Listener Detection: ~30 seconds"
echo "   Validator Signing: ~2 minutes"
echo "   Relay to BNB: ~30 seconds"
echo ""

echo "========================================="
echo "🎉 END-TO-END TEST PASSED!"
echo "========================================="
echo ""
echo "Your bridge is fully operational and ready for production!"
echo ""
```

---

#### Troubleshooting End-to-End Test

##### Issue 1: Lock Transaction Fails

**Symptoms:**
- Transaction reverts on Polygon
- Error: "ERC20: insufficient allowance"

**Solution:**
```bash
# Check allowance
npx hardhat run scripts/check-allowance.js --network polygon-amoy

# If allowance is 0, re-approve
npx hardhat run scripts/approve-bridge.js --network polygon-amoy

# Retry lock
npx hardhat run scripts/bridge-lock.js --network polygon-amoy
```

##### Issue 2: Listener Not Detecting Event

**Symptoms:**
- Message not appearing in API after 2 minutes
- No logs in listener service

**Solution:**
```bash
# Check listener is running
sudo systemctl status metabridge-listener

# Check listener logs for errors
sudo journalctl -u metabridge-listener --since "5 minutes ago" | grep -i error

# Check RPC connection
curl -s http://159.65.73.133:8080/v1/chains/status | jq '.polygon.healthy'

# Restart listener if needed
sudo systemctl restart metabridge-listener

# Manual event fetch (if needed)
npx hardhat run scripts/fetch-events.js --network polygon-amoy
```

##### Issue 3: Not Enough Validator Signatures

**Symptoms:**
- Message stuck in "pending" status
- Less than 2 signatures after 5 minutes

**Solution:**
```bash
# Check validator configuration
curl -s http://159.65.73.133:8080/v1/config/validators | jq '.'

# Check relayer logs
sudo journalctl -u metabridge-relayer --since "5 minutes ago" | grep -i signature

# Verify validator private keys are configured
grep VALIDATOR ~/.env.production

# Restart relayer
sudo systemctl restart metabridge-relayer
```

##### Issue 4: Unlock Transaction Fails on BNB

**Symptoms:**
- Message has signatures but no unlock TX
- Relayer logs show transaction errors

**Solution:**
```bash
# Check relayer has BNB for gas
npx hardhat run scripts/check-balance.js --network bnb-testnet

# Check bridge contract has tokens to unlock
npx hardhat run scripts/check-bridge-balance.js --network bnb-testnet

# If bridge is empty, mint tokens to bridge contract
npx hardhat run scripts/mint-to-bridge.js --network bnb-testnet

# Check relayer logs for specific error
sudo journalctl -u metabridge-relayer -n 100 | grep -A 5 "unlock"

# Manual retry unlock (if needed)
curl -X POST http://159.65.73.133:8080/v1/admin/retry-message \
  -H "Content-Type: application/json" \
  -d "{\"message_id\": \"$MESSAGE_ID\"}"
```

##### Issue 5: Balance Not Updated on BNB

**Symptoms:**
- Unlock transaction successful
- But balance shows 0 USDC

**Solution:**
```bash
# Check you're looking at correct token contract
echo "Token contract: $BNB_TOKEN_ADDRESS"

# Force refresh balance
npx hardhat run scripts/check-balance.js --network bnb-testnet

# Check block explorer
echo "https://testnet.bscscan.com/token/$BNB_TOKEN_ADDRESS?a=0xYourWalletAddress"

# Add token to Metamask if not visible:
# - Token Address: $BNB_TOKEN_ADDRESS
# - Symbol: USDC
# - Decimals: 6
```

---

This completes the comprehensive end-to-end testing guide with detailed expected responses at every step!

### Test 8: Performance Tests

```bash
# Test 1: API response time
time curl http://159.65.73.133:8080/health

# Expected: < 100ms

# Test 2: Database query performance
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production << 'EOF'
\timing on
SELECT COUNT(*) FROM messages;
EOF

# Expected: Query time < 10ms

# Test 3: Concurrent requests test (requires Apache Bench)
sudo apt install -y apache2-utils

ab -n 100 -c 10 http://159.65.73.133:8080/health

# Expected:
# - 100% successful requests
# - Average response time < 100ms
# - No failed requests
```

### Test 9: Log Analysis

```bash
# Check API logs for startup messages
sudo journalctl -u metabridge-api --since "10 minutes ago" --no-pager | head -50

# Expected to see:
# - "Starting Metabridge API..."
# - "Database connected"
# - "NATS connected"
# - "Redis connected"
# - "Server listening on :8080"

# Check for any error patterns
sudo journalctl -u metabridge-api --since "1 hour ago" --no-pager | grep -iE '(error|fatal|panic|failed)' | wc -l

# Expected: 0 or very low number

# Check relayer logs
sudo journalctl -u metabridge-relayer --since "10 minutes ago" --no-pager | head -50

# Expected to see:
# - "Relayer starting..."
# - "Connected to NATS"
# - "Listening for messages..."
```

### Test 10: Backup & Recovery Test

```bash
# Test 1: Create a manual backup
sudo docker exec metabridge-postgres pg_dump -U bridge_user metabridge_production > ~/test_backup_$(date +%Y%m%d).sql

# Verify backup file was created
ls -lh ~/test_backup_*.sql

# Expected: Backup file with size > 0 bytes

# Test 2: Insert test data
sudo docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production << 'EOF'
INSERT INTO users (id, email, name, password_hash, role, active, created_at, updated_at)
VALUES (
  'test-user-001',
  'test@metabridge.local',
  'Test User',
  '$2a$10$test',
  'user',
  true,
  NOW(),
  NOW()
);
EOF

# Verify insertion
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c \
  "SELECT COUNT(*) FROM users WHERE id='test-user-001';"

# Expected: 1

# Test 3: Test restore (to verify backup integrity)
# First, delete the test user
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c \
  "DELETE FROM users WHERE id='test-user-001';"

# Restore from backup
sudo docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production < ~/test_backup_*.sql

# Verify restore
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c \
  "SELECT COUNT(*) FROM users;"

# Expected: Original count + test user

# Clean up test data
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c \
  "DELETE FROM users WHERE id='test-user-001';"
rm ~/test_backup_*.sql
```

## Step 17: View Logs

```bash
# View API logs (live)
sudo journalctl -u metabridge-api -f

# View relayer logs (live)
sudo journalctl -u metabridge-relayer -f

# View last 100 lines of API logs
sudo journalctl -u metabridge-api -n 100 --no-pager

# View logs with errors only
sudo journalctl -u metabridge-api --since "10 minutes ago" | grep -i error

# View Docker container logs
sudo docker compose -f docker-compose.production.yaml logs -f
```

## Step 18: Install Nginx (Optional - for production)

```bash
# Install Nginx
sudo apt install -y nginx

# Create Nginx configuration
sudo tee /etc/nginx/sites-available/metabridge > /dev/null << 'EOF'
server {
    listen 80;
    server_name 159.65.73.133;

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
}
EOF

# Enable site
sudo ln -s /etc/nginx/sites-available/metabridge /etc/nginx/sites-enabled/

# Remove default site
sudo rm /etc/nginx/sites-enabled/default

# Test Nginx configuration
sudo nginx -t

# Restart Nginx
sudo systemctl restart nginx

# Now you can access via: http://159.65.73.133
curl http://159.65.73.133/health
```

## Step 19: Set Up SSL (Optional - if you have a domain)

```bash
# Install Certbot
sudo apt install -y certbot python3-certbot-nginx

# Get SSL certificate (replace yourdomain.com with your actual domain)
sudo certbot --nginx -d api.yourdomain.com

# Auto-renewal is configured automatically
# Test auto-renewal
sudo certbot renew --dry-run
```

## Verification Checklist

Run these checks to ensure everything is working:

```bash
# ✅ Check Docker containers
sudo docker compose -f docker-compose.production.yaml ps
# All should be "Up"

# ✅ Check systemd services
sudo systemctl status metabridge-api
sudo systemctl status metabridge-relayer
# Both should be "active (running)"

# ✅ Check API health
curl http://159.65.73.133:8080/health
# Should return: {"status":"ok","version":"1.0.0"}

# ✅ Check database
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c "SELECT COUNT(*) FROM users;"
# Should return: 1

# ✅ Check disk space
df -h
# Should have >10GB free

# ✅ Check memory
free -h
# Should have >1GB free

# ✅ Check firewall
sudo ufw status
# Should show ports 22, 80, 443, 8080 allowed
```

## Common Commands

```bash
# Restart API server
sudo systemctl restart metabridge-api

# Restart relayer
sudo systemctl restart metabridge-relayer

# Restart all Docker services
cd ~/projects/metabridge-engine-hub
sudo docker compose -f docker-compose.production.yaml restart

# View live logs
sudo journalctl -u metabridge-api -f

# Update code
cd ~/projects/metabridge-engine-hub
git pull origin main
go build -o bin/metabridge-api cmd/api/main.go
sudo systemctl restart metabridge-api

# Database backup
sudo docker exec metabridge-postgres pg_dump -U bridge_user metabridge_production > ~/backup_$(date +%Y%m%d).sql

# Restore database
sudo docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production < ~/backup_20240101.sql
```

## Troubleshooting

### Service won't start

```bash
# Check logs
sudo journalctl -u metabridge-api -n 100 --no-pager

# Check if port is already in use
sudo lsof -i :8080

# Check environment file
cat ~/projects/metabridge-engine-hub/.env.production
```

### Database connection failed

```bash
# Check if PostgreSQL is running
sudo docker compose -f docker-compose.production.yaml ps postgres

# Test connection
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c "SELECT 1;"

# Check password in .env.production matches the one used in Step 11
```

### Port already in use

```bash
# Find what's using port 8080
sudo lsof -i :8080

# Kill the process
sudo kill -9 <PID>

# Or change the port in .env.production
nano ~/projects/metabridge-engine-hub/.env.production
# Change SERVER_PORT=8080 to SERVER_PORT=8081
```

### Out of disk space

```bash
# Check disk usage
df -h

# Clean up Docker
sudo docker system prune -a --volumes

# Clean up old logs
sudo journalctl --vacuum-time=7d
```

## Next Steps

1. **Get API Keys** (if you haven't already):
   - Alchemy: https://www.alchemy.com/
   - Infura: https://infura.io/
   - Update `.env.production` with your keys

2. **Deploy Smart Contracts** (from your local machine or the droplet):
   - Follow the contract deployment guide in README.md
   - Update contract addresses in `.env.production`

3. **Enable Authentication** (optional):
   - Change `REQUIRE_AUTH=true` in `.env.production`
   - Restart API: `sudo systemctl restart metabridge-api`

4. **Set Up Domain** (optional):
   - Point your domain to `159.65.73.133`
   - Follow Step 19 to set up SSL

5. **Monitor Your Bridge**:
   - Access: http://159.65.73.133:8080/v1/stats
   - Check logs: `sudo journalctl -u metabridge-api -f`

## Your Deployment Summary

**Access Points**:
- API: `http://159.65.73.133:8080`
- Health: `http://159.65.73.133:8080/health`
- Chain Status: `http://159.65.73.133:8080/v1/chains/status`
- Stats: `http://159.65.73.133:8080/v1/stats`

**Admin Credentials**:
- Email: `admin@metabridge.local`
- Password: `admin123` (or whatever you set in Step 12)

**Services**:
- API Server: `sudo systemctl status metabridge-api`
- Relayer: `sudo systemctl status metabridge-relayer`
- PostgreSQL: `sudo docker ps | grep postgres`
- NATS: `sudo docker ps | grep nats`
- Redis: `sudo docker ps | grep redis`

**Important Files**:
- Environment: `~/projects/metabridge-engine-hub/.env.production`
- Binaries: `~/projects/metabridge-engine-hub/bin/`
- Logs: `sudo journalctl -u metabridge-api`

🎉 **Congratulations! Your Metabridge is now running on DigitalOcean!**
