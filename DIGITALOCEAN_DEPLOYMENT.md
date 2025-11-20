# Complete DigitalOcean Deployment Guide for Metabridge

**Your Droplet IP**: `159.65.73.133`

This guide will take you from SSH login to a fully running bridge in ~30 minutes.

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

```bash
cd ~/projects/metabridge-engine-hub

# Create bin directory
mkdir -p bin

# Build API server
echo "🔨 Building API server..."
go build -o bin/metabridge-api cmd/api/main.go

# Build relayer
echo "🔨 Building relayer..."
go build -o bin/metabridge-relayer cmd/relayer/main.go

# Verify binaries
ls -lh bin/

# You should see:
# - metabridge-api
# - metabridge-relayer

echo "✅ Binaries built successfully"
```

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

### Test 1: Infrastructure Health Checks

```bash
# Check all Docker containers are running
sudo docker compose -f ~/projects/metabridge-engine-hub/docker-compose.production.yaml ps

# Expected output: All containers should show "Up" status
# - metabridge-postgres  Up (healthy)
# - metabridge-nats      Up (healthy)
# - metabridge-redis     Up (healthy)

# Test PostgreSQL connection
sudo docker exec -it metabridge-postgres psql -U bridge_user -d metabridge_production -c "SELECT version();"

# Expected: PostgreSQL version info

# Test NATS connection
curl http://localhost:8222/varz

# Expected: JSON with NATS server stats

# Test Redis connection
sudo docker exec -it metabridge-redis redis-cli ping

# Expected: PONG
```

### Test 2: API Health Checks

```bash
# Test 1: Basic health endpoint
curl http://159.65.73.133:8080/health

# Expected output:
# {"status":"ok","version":"1.0.0","timestamp":"2025-11-20T..."}

# Test 2: Detailed API status
curl http://159.65.73.133:8080/v1/status

# Expected: Detailed status of all services

# Test 3: Chain connectivity
curl http://159.65.73.133:8080/v1/chains/status

# Expected: JSON with all 6 chains and their RPC status
# Should show which chains are connected/disconnected

# Test 4: Bridge statistics
curl http://159.65.73.133:8080/v1/stats

# Expected:
# {
#   "total_messages": 0,
#   "pending_messages": 0,
#   "completed_messages": 0,
#   "failed_messages": 0,
#   "total_volume": "0",
#   "total_fees": "0"
# }
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

### Test 7: End-to-End Bridge Flow Test (Optional)

```bash
# This test requires deployed smart contracts and test tokens

# Test 1: Create a bridge request
curl -X POST http://159.65.73.133:8080/v1/bridge/request \
  -H "Content-Type: application/json" \
  -d '{
    "source_chain": "polygon",
    "destination_chain": "bnb",
    "token_address": "0x...",
    "amount": "1000000000000000000",
    "recipient": "0x...",
    "sender": "0x..."
  }'

# Expected: Bridge request ID

# Test 2: Check request status
curl http://159.65.73.133:8080/v1/bridge/request/<request_id>

# Expected: Request details with status

# Test 3: List all bridge requests
curl http://159.65.73.133:8080/v1/messages?limit=10

# Expected: List of bridge messages
```

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
