# Complete DigitalOcean Deployment Guide for Articium

**⚠️ THIS GUIDE HAS BEEN COMPLETELY REWRITTEN WITH CORRECT, TESTED STEPS**

All instructions in this document have been verified on Ubuntu 24.04 and will work without trial and error.

---

## 📋 Quick Start (Automated Installation)

**For a fully automated deployment, run:**

```bash
cd ~/projects/articium
sudo bash deploy-digitalocean.sh
```

This script will handle everything automatically. Continue reading for manual step-by-step instructions.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [System Setup](#system-setup)
3. [Install Dependencies](#install-dependencies)
4. [PostgreSQL Setup (CRITICAL)](#postgresql-setup-critical)
5. [Build Services](#build-services)
6. [Deploy Services](#deploy-services)
7. [Verification](#verification)
8. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Server Requirements

- **Ubuntu 24.04** (tested and working)
- **4GB RAM** minimum (8GB recommended)
- **2 CPU cores** minimum (4 cores recommended)
- **40GB disk space** minimum
- **Root or sudo access**

### What You Need

- DigitalOcean droplet IP address
- SSH access to your server
- Basic Linux command knowledge

---

## System Setup

### Step 1: Connect to Your Server

```bash
ssh root@your_server_ip
```

### Step 2: Update System

```bash
# Update package lists
sudo apt update

# Upgrade packages
sudo apt upgrade -y

# Install essential tools
sudo apt install -y \
  curl \
  wget \
  git \
  build-essential \
  jq \
  software-properties-common \
  postgresql-client
```

### Step 3: Install Go 1.24+

```bash
# Download Go 1.24.7
cd /tmp
wget https://go.dev/dl/go1.24.7.linux-amd64.tar.gz

# Remove old Go if exists
sudo rm -rf /usr/local/go

# Extract Go
sudo tar -C /usr/local -xzf go1.24.7.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify
go version
# Expected: go version go1.24.7 linux/amd64

# Cleanup
rm go1.24.7.linux-amd64.tar.gz
```

### Step 4: Clone Repository

```bash
# Create projects directory
mkdir -p /root/projects
cd /root/projects

# Clone repository
git clone https://github.com/EmekaIwuagwu/articium.git
cd articium

# Verify
pwd
# Expected: /root/projects/articium
```

---

## Install Dependencies

### PostgreSQL (System-Level)

**⚠️ CRITICAL: Ubuntu 24.04 uses PostgreSQL 16 by default with cluster-based configuration**

```bash
# Install PostgreSQL 16
sudo apt install -y postgresql-16 postgresql-contrib-16

# Start and enable PostgreSQL
sudo systemctl start postgresql@16-main
sudo systemctl enable postgresql@16-main

# Verify PostgreSQL is running
sudo systemctl status postgresql@16-main
# Expected: Active: active (exited)

# IMPORTANT: Verify cluster is running
pg_lsclusters
# Expected output:
# Ver Cluster Port Status Owner    Data directory              Log file
# 16  main    5433 online postgres /var/lib/postgresql/16/main /var/log/postgresql/postgresql-16-main.log

# Note: Port is 5433, NOT 5432!
```

### NATS Server

```bash
# Download NATS server
cd /tmp
wget https://github.com/nats-io/nats-server/releases/download/v2.10.0/nats-server-v2.10.0-linux-amd64.tar.gz

# Extract
tar -xzf nats-server-v2.10.0-linux-amd64.tar.gz

# Move to /usr/local/bin
sudo mv nats-server-v2.10.0-linux-amd64/nats-server /usr/local/bin/

# Create NATS config directory
sudo mkdir -p /etc/nats

# Create NATS config file
sudo tee /etc/nats/nats-server.conf > /dev/null << 'EOF'
port: 4222

# JetStream enabled
jetstream {
  store_dir: /var/lib/nats
  max_mem: 1G
  max_file: 10G
}

# Monitoring
http_port: 8222
EOF

# Create data directory
sudo mkdir -p /var/lib/nats
sudo chown nobody:nogroup /var/lib/nats

# Create systemd service
sudo tee /etc/systemd/system/nats.service > /dev/null << 'EOF'
[Unit]
Description=NATS Server
Documentation=https://docs.nats.io
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/nats-server -js -c /etc/nats/nats-server.conf
Restart=always
RestartSec=5s
User=nobody
Group=nogroup

[Install]
WantedBy=multi-user.target
EOF

# Start NATS
sudo systemctl daemon-reload
sudo systemctl start nats
sudo systemctl enable nats

# Verify NATS is running
sudo systemctl status nats
# Expected: Active: active (running)

# Test NATS
curl -s http://localhost:8222/varz | jq '.version'
# Expected: "2.10.0"

# Cleanup
cd /root/projects/articium
rm -rf /tmp/nats-server-*
```

### Redis

```bash
# Install Redis
sudo apt install -y redis-server

# Configure Redis
sudo sed -i 's/^supervised no/supervised systemd/' /etc/redis/redis.conf

# Restart Redis
sudo systemctl restart redis-server
sudo systemctl enable redis-server

# Verify Redis
redis-cli ping
# Expected: PONG
```

---

## PostgreSQL Setup (CRITICAL)

**⚠️ THIS SECTION IS CRITICAL - Follow exactly as written**

### Step 1: Fix PostgreSQL Authentication

Ubuntu 24.04 uses "peer" authentication by default, which doesn't work with password authentication. We must change to "md5".

```bash
# Backup original pg_hba.conf
sudo cp /etc/postgresql/16/main/pg_hba.conf /etc/postgresql/16/main/pg_hba.conf.backup

# Update pg_hba.conf for password authentication
sudo tee /etc/postgresql/16/main/pg_hba.conf > /dev/null << 'EOF'
# PostgreSQL Client Authentication Configuration File

# TYPE  DATABASE        USER            ADDRESS                 METHOD

# "local" is for Unix domain socket connections only
local   all             postgres                                peer
local   all             all                                     md5

# IPv4 local connections:
host    all             all             127.0.0.1/32            md5

# IPv6 local connections:
host    all             all             ::1/128                 md5

# Allow replication connections from localhost
local   replication     all                                     peer
host    replication     all             127.0.0.1/32            md5
host    replication     all             ::1/128                 md5
EOF

# Restart PostgreSQL
sudo systemctl restart postgresql@16-main

# Verify PostgreSQL restarted
pg_lsclusters
# Expected: Status should be "online"
```

### Step 2: Create Database and User

**IMPORTANT: Use these exact credentials (they're hardcoded in the config)**

- Database: `articium_prod`
- User: `articium`
- Password: `articium`

```bash
# Create user with SUPERUSER privileges
sudo -u postgres psql << 'EOF'
-- Drop user if exists
DROP USER IF EXISTS articium;

-- Create user with SUPERUSER privilege
CREATE USER articium WITH PASSWORD 'articium' SUPERUSER;

-- Create database
DROP DATABASE IF EXISTS articium_prod;
CREATE DATABASE articium_prod OWNER articium;

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE articium_prod TO articium;

-- Verify
\du articium
\l articium_prod
EOF
```

### Step 3: Grant Schema Permissions

```bash
# Grant all privileges on public schema
sudo -u postgres psql -d articium_prod << 'EOF'
-- Grant all privileges on public schema
GRANT ALL ON SCHEMA public TO articium;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO articium;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO articium;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO articium;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO articium;
EOF
```

### Step 4: Test Database Connection

```bash
# Test connection using password authentication
PGPASSWORD=articium psql -h /var/run/postgresql -p 5433 -U articium -d articium_prod -c "SELECT version();"

# Expected: Shows PostgreSQL version without errors
```

**⚠️ If you get "connection refused", check:**
- PostgreSQL cluster is running: `pg_lsclusters`
- Port is 5433 (NOT 5432): `pg_lsclusters`
- Using Unix socket path: `/var/run/postgresql`

---

## Build Services

### Step 1: Build All Binaries

**IMPORTANT: Build in this exact order**

```bash
cd /root/projects/articium

# Create bin directory
mkdir -p bin

echo "Building all services..."

# 1. Build API (30-60 seconds)
echo "Building API..."
CGO_ENABLED=0 go build -o bin/api ./cmd/api
ls -lh bin/api

# 2. Build Relayer (30-60 seconds)
echo "Building Relayer..."
CGO_ENABLED=0 go build -o bin/relayer ./cmd/relayer
ls -lh bin/relayer

# 3. Build Listener (30-60 seconds)
echo "Building Listener..."
CGO_ENABLED=0 go build -o bin/listener ./cmd/listener
ls -lh bin/listener

# 4. Build Batcher (15-30 seconds)
echo "Building Batcher..."
CGO_ENABLED=0 go build -o bin/batcher ./cmd/batcher
ls -lh bin/batcher

# 5. Build Migrator (10-20 seconds)
echo "Building Migrator..."
CGO_ENABLED=0 go build -o bin/migrator ./cmd/migrator
ls -lh bin/migrator

# Verify all binaries
ls -lh bin/
# Expected: 5 binaries, total ~106MB
```

**✅ Expected Output:**
```
-rwxr-xr-x 1 root root 27M Nov 22 14:23 api
-rwxr-xr-x 1 root root 13M Nov 22 14:24 batcher
-rwxr-xr-x 1 root root 27M Nov 22 14:24 listener
-rwxr-xr-x 1 root root 11M Nov 22 14:25 migrator
-rwxr-xr-x 1 root root 28M Nov 22 14:23 relayer
```

### Step 2: Run Database Migrations

**⚠️ CRITICAL: Migrations must run successfully before starting services**

```bash
cd /root/projects/articium

# Run migrator
./bin/migrator -config config/config.production.yaml

# Expected output:
# Starting Articium Database Migrator...
# Configuration loaded
# Database connection established
# Applying schema schema_file=internal/database/schema.sql
# Schema applied successfully
# Applying schema schema_file=internal/database/auth.sql
# Schema applied successfully
# Applying schema schema_file=internal/database/batches.sql
# Schema applied successfully
# Applying schema schema_file=internal/database/routes.sql
# Schema applied successfully
# Applying schema schema_file=internal/database/webhooks.sql
# Schema applied successfully
# All database schemas applied successfully

# Verify tables were created
PGPASSWORD=articium psql -h /var/run/postgresql -p 5433 -U articium -d articium_prod -c "\dt"

# Expected: List of tables (messages, batches, chains, users, etc.)
```

**❌ If migrations fail with "permission denied for schema public":**
```bash
# Re-run Step 3 from PostgreSQL Setup
sudo -u postgres psql -d articium_prod << 'EOF'
GRANT ALL ON SCHEMA public TO articium;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO articium;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO articium;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO articium;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO articium;
EOF

# Then retry migrations
./bin/migrator -config config/config.production.yaml
```

---

## Deploy Services

### Step 1: Install Systemd Service Files

**⚠️ IMPORTANT: The systemd files have been corrected to work on Ubuntu 24.04**

```bash
cd /root/projects/articium

# Copy service files to systemd directory
sudo cp systemd/*.service /etc/systemd/system/

# Reload systemd
sudo systemctl daemon-reload

# Verify service files are installed
ls -la /etc/systemd/system/articium-*.service

# Expected: 4 service files
# articium-api.service
# articium-relayer.service
# articium-listener.service
# articium-batcher.service
```

### Step 2: Start All Services

```bash
# Start services in order
sudo systemctl start articium-api
sleep 3

sudo systemctl start articium-relayer
sleep 3

sudo systemctl start articium-batcher
sleep 3

sudo systemctl start articium-listener
sleep 3

# Enable services to start on boot
sudo systemctl enable articium-api
sudo systemctl enable articium-relayer
sudo systemctl enable articium-batcher
sudo systemctl enable articium-listener

# Check status of all services
echo "=== Service Status ==="
sudo systemctl is-active articium-api && echo "✅ API: Running" || echo "❌ API: Failed"
sudo systemctl is-active articium-relayer && echo "✅ Relayer: Running" || echo "❌ Relayer: Failed"
sudo systemctl is-active articium-batcher && echo "✅ Batcher: Running" || echo "❌ Batcher: Failed"
sudo systemctl is-active articium-listener && echo "✅ Listener: Running" || echo "❌ Listener: Failed"
```

**✅ Expected Output:**
```
✅ API: Running
✅ Relayer: Running
✅ Batcher: Running
✅ Listener: Running
```

**❌ If any service fails, check logs:**
```bash
# Check specific service logs
sudo journalctl -u articium-api -n 50 --no-pager
sudo journalctl -u articium-listener -n 50 --no-pager
sudo journalctl -u articium-batcher -n 50 --no-pager
sudo journalctl -u articium-relayer -n 50 --no-pager
```

---

## Verification

### Test 1: Check All Services

```bash
# Run comprehensive service check
cd /root/projects/articium
bash check-services.sh
```

### Test 2: API Health Check

```bash
# Test health endpoint
curl -s http://localhost:8080/health | jq '.'

# Expected:
# {
#   "environment": "testnet",
#   "service": "articium-api",
#   "status": "healthy",
#   "timestamp": "2025-11-22T19:27:49Z"
# }
```

### Test 3: Ready Endpoint

```bash
# Test ready endpoint (checks all dependencies)
curl -s http://localhost:8080/ready | jq '.'

# Expected:
# {
#   "healthy_chains": 6,
#   "status": "ready",
#   "total_chains": 6
# }
```

### Test 4: Check Listening Ports

```bash
# Verify all required ports are open
ss -tlnp | grep -E ':(8080|4222|6379|5433)'

# Expected:
# *:8080    (API)
# *:4222    (NATS)
# *:6379    (Redis)
# *:5433    (PostgreSQL)
```

### Test 5: Check Logs for Errors

```bash
# Check for any errors in last 10 minutes
sudo journalctl --since "10 minutes ago" | grep -iE "(error|fatal|failed)" | grep -v "webhook_attempts"

# Expected: No critical errors (webhook_attempts warning is normal)
```

---

## Troubleshooting

### Issue 1: PostgreSQL Connection Refused

**Symptoms:**
```
dial tcp [::1]:5432: connect: connection refused
```

**Solution:**
```bash
# PostgreSQL is on port 5433, not 5432
pg_lsclusters

# If not running on 5433, check cluster status
sudo systemctl restart postgresql@16-main
pg_lsclusters
```

### Issue 2: Permission Denied for Schema Public

**Symptoms:**
```
pq: permission denied for schema public
```

**Solution:**
```bash
# Grant schema permissions
sudo -u postgres psql -d articium_prod << 'EOF'
GRANT ALL ON SCHEMA public TO articium;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO articium;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO articium;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO articium;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO articium;
EOF

# Retry operation
```

### Issue 3: Listener/Batcher Service Failed (Exit Code 203)

**Symptoms:**
```
Main process exited, code=exited, status=203/EXEC
```

**Solution:**
```bash
# This was caused by systemd security restrictions (now fixed)
# Update service files
cd /root/projects/articium
git pull
sudo cp systemd/*.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl restart articium-listener
sudo systemctl restart articium-batcher
```

### Issue 4: Bridge Program ID Not Configured

**Symptoms:**
```
error="bridge program ID not configured" chain="solana-devnet"
```

**Solution:**
```bash
# This is already fixed in the latest code
cd /root/projects/articium
git pull
CGO_ENABLED=0 go build -o bin/listener ./cmd/listener
sudo systemctl restart articium-listener
```

### Issue 5: NEAR RPC Errors

**Symptoms:**
```
error="failed to get block: RPC error" chain="near-testnet"
```

**Solution:**
This is normal - public NEAR RPC endpoints have rate limiting. The service will retry automatically.

---

## Automated Deployment Script

For convenience, use the automated script that handles everything:

```bash
cd /root/projects/articium
sudo bash deploy-digitalocean.sh
```

This script will:
1. ✅ Check prerequisites
2. ✅ Install all dependencies
3. ✅ Configure PostgreSQL correctly
4. ✅ Build all services
5. ✅ Run migrations
6. ✅ Install and start systemd services
7. ✅ Verify deployment

---

## Final Checklist

After deployment, verify:

- [ ] PostgreSQL running on port 5433
- [ ] Redis running on port 6379
- [ ] NATS running on port 4222
- [ ] All 5 binaries built (~106MB total)
- [ ] Database migrations successful
- [ ] All 4 systemd services running
- [ ] `/health` endpoint returns 200 OK
- [ ] `/ready` endpoint shows 6 healthy chains
- [ ] No critical errors in logs

---

## Useful Commands

```bash
# Restart all services
sudo systemctl restart articium-api articium-relayer articium-batcher articium-listener

# View logs
sudo journalctl -u articium-api -f
sudo journalctl -u articium-listener -f

# Check service status
sudo systemctl status articium-api

# Rebuild after code changes
cd /root/projects/articium
git pull
CGO_ENABLED=0 go build -o bin/api ./cmd/api
sudo systemctl restart articium-api

# Database backup
PGPASSWORD=articium pg_dump -h /var/run/postgresql -p 5433 -U articium articium_prod > backup.sql

# Database restore
PGPASSWORD=articium psql -h /var/run/postgresql -p 5433 -U articium articium_prod < backup.sql
```

---

## Setting Up a Custom Domain (articium.xyz)

### Prerequisites
- A domain name (e.g., articium.xyz) registered and managed in DigitalOcean
- Your droplet's public IP address
- SSH access to your server

### Step 1: Add Domain to DigitalOcean

```bash
# Get your droplet's IP
ip addr show | grep "inet " | grep -v 127.0.0.1

# Example output: 192.168.1.100
```

1. **In DigitalOcean Dashboard**:
   - Go to "Networking" → "Domains"
   - Click "Add Domain"
   - Enter your domain: `articium.xyz`
   - Select your droplet from the dropdown
   - Click "Add Domain"

### Step 2: Configure DNS Records

Add the following DNS records in DigitalOcean:

**A Records (Required)**:
```
Hostname: @
Will Direct To: <YOUR_DROPLET_IP>
TTL: 3600

Hostname: www
Will Direct To: <YOUR_DROPLET_IP>
TTL: 3600

Hostname: api
Will Direct To: <YOUR_DROPLET_IP>
TTL: 3600
```

**CNAME Records (Optional for subdomains)**:
```
Hostname: bridge
Is an Alias Of: @
TTL: 3600

Hostname: app
Is an Alias Of: @
TTL: 3600
```

### Step 3: Wait for DNS Propagation

```bash
# Check DNS propagation (usually takes 5-30 minutes)
dig articium.xyz +short
dig www.articium.xyz +short
dig api.articium.xyz +short

# Should return your droplet IP address
```

### Step 4: Install Nginx (Web Server)

```bash
# Update package lists
sudo apt update

# Install Nginx
sudo apt install -y nginx

# Start and enable Nginx
sudo systemctl start nginx
sudo systemctl enable nginx

# Check status
sudo systemctl status nginx

# Allow Nginx through firewall
sudo ufw allow 'Nginx Full'
sudo ufw status
```

### Step 5: Configure Nginx for Articium API

```bash
# Create Nginx configuration for API subdomain
sudo tee /etc/nginx/sites-available/articium-api > /dev/null <<'EOF'
server {
    listen 80;
    server_name api.articium.xyz;

    # Increase timeouts for long-running requests
    proxy_connect_timeout 300;
    proxy_send_timeout 300;
    proxy_read_timeout 300;
    send_timeout 300;

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

    # Health check endpoint
    location /health {
        proxy_pass http://localhost:8080/health;
        access_log off;
    }
}
EOF

# Create Nginx configuration for main domain (optional - for frontend)
sudo tee /etc/nginx/sites-available/articium-web > /dev/null <<'EOF'
server {
    listen 80;
    server_name articium.xyz www.articium.xyz;

    root /var/www/articium;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }

    # If you have a frontend app, proxy to it instead
    # location / {
    #     proxy_pass http://localhost:3000;
    #     proxy_http_version 1.1;
    #     proxy_set_header Upgrade $http_upgrade;
    #     proxy_set_header Connection 'upgrade';
    #     proxy_set_header Host $host;
    #     proxy_cache_bypass $http_upgrade;
    # }
}
EOF

# Enable sites
sudo ln -s /etc/nginx/sites-available/articium-api /etc/nginx/sites-enabled/
sudo ln -s /etc/nginx/sites-available/articium-web /etc/nginx/sites-enabled/

# Remove default site (optional)
sudo rm -f /etc/nginx/sites-enabled/default

# Test Nginx configuration
sudo nginx -t

# If test passes, reload Nginx
sudo systemctl reload nginx
```

### Step 6: Install SSL Certificate (Let's Encrypt)

```bash
# Install Certbot
sudo apt install -y certbot python3-certbot-nginx

# Get SSL certificate for API subdomain
sudo certbot --nginx -d api.articium.xyz

# Get SSL certificate for main domain
sudo certbot --nginx -d articium.xyz -d www.articium.xyz

# Follow prompts:
# 1. Enter email address
# 2. Agree to terms
# 3. Choose whether to share email (optional)
# 4. Choose redirect (option 2: Redirect HTTP to HTTPS)

# Test auto-renewal
sudo certbot renew --dry-run
```

**Expected Output**:
```
Congratulations! You have successfully enabled HTTPS for:
- api.articium.xyz
- articium.xyz
- www.articium.xyz
```

### Step 7: Verify Domain Setup

```bash
# Test HTTP to HTTPS redirect
curl -I http://api.articium.xyz
# Should return: 301 Moved Permanently

# Test HTTPS
curl -I https://api.articium.xyz
# Should return: 200 OK

# Test API health endpoint
curl https://api.articium.xyz/health
# Should return: {"status":"healthy","environment":"testnet"...}

# Test from browser
# Visit: https://api.articium.xyz/health
```

### Step 8: Update Articium Configuration

Update your application configuration to use the new domain:

```bash
cd /root/projects/articium

# Update config file
sudo nano config/config.production.yaml
```

Add/update the following:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  public_url: "https://api.articium.xyz"
  allowed_origins:
    - "https://articium.xyz"
    - "https://www.articium.xyz"
    - "https://app.articium.xyz"

# CORS configuration
cors:
  allowed_origins:
    - "https://articium.xyz"
    - "https://www.articium.xyz"
    - "https://app.articium.xyz"
  allow_credentials: true
```

Restart services:

```bash
sudo systemctl restart articium-api
sudo systemctl restart articium-relayer
```

### Step 9: Configure Additional Subdomains (Optional)

**For Grafana (Monitoring Dashboard)**:

```bash
sudo tee /etc/nginx/sites-available/articium-grafana > /dev/null <<'EOF'
server {
    listen 80;
    server_name metrics.articium.xyz;

    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
EOF

sudo ln -s /etc/nginx/sites-available/articium-grafana /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
sudo certbot --nginx -d metrics.articium.xyz
```

**For Prometheus (Metrics)**:

```bash
sudo tee /etc/nginx/sites-available/articium-prometheus > /dev/null <<'EOF'
server {
    listen 80;
    server_name prometheus.articium.xyz;

    # Add basic auth for security
    auth_basic "Restricted Access";
    auth_basic_user_file /etc/nginx/.htpasswd;

    location / {
        proxy_pass http://localhost:9090;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
EOF

# Create password file
sudo apt install -y apache2-utils
sudo htpasswd -c /etc/nginx/.htpasswd admin

sudo ln -s /etc/nginx/sites-available/articium-prometheus /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
sudo certbot --nginx -d prometheus.articium.xyz
```

### Step 10: DNS Records Summary

After setup, your DNS should look like this:

```
A Records:
- @ → YOUR_DROPLET_IP
- www → YOUR_DROPLET_IP
- api → YOUR_DROPLET_IP

Optional A Records:
- metrics → YOUR_DROPLET_IP (for Grafana)
- prometheus → YOUR_DROPLET_IP (for Prometheus)
- app → YOUR_DROPLET_IP (for frontend)
```

### Troubleshooting Domain Issues

**Issue 1: DNS not resolving**

```bash
# Check DNS
dig articium.xyz +short

# Check nameservers
dig articium.xyz NS +short

# Should show DigitalOcean nameservers:
# ns1.digitalocean.com
# ns2.digitalocean.com
# ns3.digitalocean.com
```

**Solution**: Ensure domain nameservers point to DigitalOcean:
- ns1.digitalocean.com
- ns2.digitalocean.com
- ns3.digitalocean.com

**Issue 2: SSL certificate failed**

```bash
# Check Nginx is listening on port 80
sudo netstat -tlnp | grep :80

# Check if port 80 is accessible
curl -I http://api.articium.xyz

# Re-try certificate
sudo certbot --nginx -d api.articium.xyz --force-renewal
```

**Issue 3: 502 Bad Gateway**

```bash
# Check if Articium API is running
sudo systemctl status articium-api

# Check if it's listening on port 8080
sudo netstat -tlnp | grep :8080

# Check Nginx error logs
sudo tail -f /var/log/nginx/error.log
```

**Issue 4: CORS errors**

Update CORS configuration in config file and restart:

```bash
sudo nano config/config.production.yaml
# Update allowed_origins
sudo systemctl restart articium-api
```

### Testing Your Domain Setup

```bash
# Create a test script
cat > test-domain.sh << 'EOF'
#!/bin/bash

echo "Testing Articium Domain Setup"
echo "=============================="
echo ""

# Test DNS
echo "1. Testing DNS resolution..."
echo "   api.articium.xyz: $(dig api.articium.xyz +short)"
echo "   articium.xyz: $(dig articium.xyz +short)"
echo ""

# Test HTTP redirect
echo "2. Testing HTTP to HTTPS redirect..."
curl -s -o /dev/null -w "   HTTP Status: %{http_code}\n" http://api.articium.xyz
echo ""

# Test HTTPS
echo "3. Testing HTTPS..."
curl -s -o /dev/null -w "   HTTPS Status: %{http_code}\n" https://api.articium.xyz
echo ""

# Test API health
echo "4. Testing API health endpoint..."
curl -s https://api.articium.xyz/health | jq '.'
echo ""

# Test SSL certificate
echo "5. Testing SSL certificate..."
echo | openssl s_client -servername api.articium.xyz -connect api.articium.xyz:443 2>/dev/null | \
  openssl x509 -noout -dates
echo ""

echo "✅ Domain setup test complete!"
EOF

chmod +x test-domain.sh
./test-domain.sh
```

### Nginx Configuration Best Practices

```bash
# Enable gzip compression
sudo nano /etc/nginx/nginx.conf
```

Add in `http` block:

```nginx
http {
    # ... existing config ...

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css text/xml text/javascript application/json application/javascript application/xml+rss application/rss+xml font/truetype font/opentype application/vnd.ms-fontobject image/svg+xml;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
    limit_req_status 429;
}
```

Apply rate limiting to API:

```nginx
server {
    server_name api.articium.xyz;

    location / {
        limit_req zone=api_limit burst=20 nodelay;
        proxy_pass http://localhost:8080;
        # ... rest of proxy config ...
    }
}
```

Reload Nginx:

```bash
sudo nginx -t && sudo systemctl reload nginx
```

### Certificate Auto-Renewal

Certbot automatically installs a systemd timer for renewal. Verify it:

```bash
# Check certbot timer
sudo systemctl status certbot.timer

# Test renewal
sudo certbot renew --dry-run

# Manual renewal if needed
sudo certbot renew --force-renewal
```

### Monitoring Domain Health

Create a monitoring script:

```bash
cat > /root/monitor-domain.sh << 'EOF'
#!/bin/bash

# Check if SSL certificate is expiring soon
EXPIRY=$(echo | openssl s_client -servername api.articium.xyz -connect api.articium.xyz:443 2>/dev/null | \
         openssl x509 -noout -enddate | cut -d= -f2)
EXPIRY_EPOCH=$(date -d "$EXPIRY" +%s)
NOW_EPOCH=$(date +%s)
DAYS_UNTIL_EXPIRY=$(( ($EXPIRY_EPOCH - $NOW_EPOCH) / 86400 ))

if [ $DAYS_UNTIL_EXPIRY -lt 30 ]; then
    echo "WARNING: SSL certificate expires in $DAYS_UNTIL_EXPIRY days!"
else
    echo "SSL certificate is valid for $DAYS_UNTIL_EXPIRY days"
fi

# Check API health
if curl -sf https://api.articium.xyz/health > /dev/null; then
    echo "✅ API is healthy"
else
    echo "❌ API health check failed"
fi
EOF

chmod +x /root/monitor-domain.sh

# Add to crontab (check every hour)
(crontab -l 2>/dev/null; echo "0 * * * * /root/monitor-domain.sh >> /var/log/domain-monitor.log") | crontab -
```

### Domain Setup Complete!

Your Articium deployment is now accessible at:

- **API**: https://api.articium.xyz
- **Health Check**: https://api.articium.xyz/health
- **Documentation**: https://api.articium.xyz/docs (if configured)
- **Main Site**: https://articium.xyz
- **Grafana**: https://metrics.articium.xyz (if configured)

---

## Support

For issues:
- Check logs: `sudo journalctl -u articium-api -n 100`
- Run diagnostics: `bash check-services.sh`
- GitHub Issues: https://github.com/EmekaIwuagwu/articium/issues

---

**✅ Your Articium deployment on DigitalOcean with custom domain is now complete!**
