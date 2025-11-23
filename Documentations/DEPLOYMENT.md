# Articium Hub - Production Deployment Guide

This guide covers deploying the Articium Hub to production with all security features enabled.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Database Setup](#database-setup)
3. [Environment Configuration](#environment-configuration)
4. [Smart Contract Deployment](#smart-contract-deployment)
5. [Server Deployment](#server-deployment)
6. [Security Hardening](#security-hardening)
7. [Monitoring & Maintenance](#monitoring--maintenance)
8. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Software

- **Go 1.21+** - Backend runtime
- **PostgreSQL 14+** - Primary database
- **Node.js 18+** - Smart contract deployment
- **NATS Server** - Message queue (optional but recommended)
- **Nginx** - Reverse proxy with SSL/TLS
- **Prometheus** - Metrics collection
- **Grafana** - Monitoring dashboards

### Infrastructure Requirements

**Minimum (Small Scale)**:
- 2 CPU cores
- 4GB RAM
- 50GB SSD storage
- 1Gbps network

**Recommended (Production)**:
- 4-8 CPU cores
- 16GB RAM
- 200GB SSD storage
- 10Gbps network
- Load balancer
- Database replica

---

## Database Setup

### 1. Install PostgreSQL

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install postgresql-14 postgresql-contrib

# Start service
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

### 2. Create Database and User

```bash
sudo -u postgres psql
```

```sql
-- Create database
CREATE DATABASE articium;

-- Create user with strong password
CREATE USER articium_user WITH ENCRYPTED PASSWORD 'YOUR_VERY_STRONG_PASSWORD_HERE';

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE articium TO articium_user;

-- Exit
\q
```

### 3. Run Database Migrations

```bash
# Run migrations in order
psql -U articium_user -d articium -f internal/database/schema.sql
psql -U articium_user -d articium -f internal/database/auth.sql
psql -U articium_user -d articium -f internal/database/batches.sql
psql -U articium_user -d articium -f internal/database/webhooks.sql
psql -U articium_user -d articium -f internal/database/routes.sql
```

### 4. Create Admin User

**CRITICAL: Change default admin password!**

```bash
# Generate bcrypt hash for your password
# Install bcrypt tool
go install github.com/bitnami/bcrypt-cli/cmd/bcrypt-cli@latest

# Generate hash (replace 'your_secure_password' with actual password)
bcrypt-cli hash your_secure_password

# Update admin user in database
psql -U articium_user -d articium
```

```sql
UPDATE users
SET password_hash = '$2a$10$YOUR_BCRYPT_HASH_HERE'
WHERE email = 'admin@articium.local';

-- Change email to your actual email
UPDATE users
SET email = 'admin@yourdomain.com'
WHERE email = 'admin@articium.local';
```

### 5. Configure PostgreSQL for Production

Edit `/etc/postgresql/14/main/postgresql.conf`:

```conf
# Performance tuning
shared_buffers = 4GB                    # 25% of RAM
effective_cache_size = 12GB              # 75% of RAM
maintenance_work_mem = 1GB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1                   # For SSD
effective_io_concurrency = 200           # For SSD
work_mem = 10MB
min_wal_size = 1GB
max_wal_size = 4GB

# Connection settings
max_connections = 200

# Logging
log_destination = 'stderr'
logging_collector = on
log_directory = '/var/log/postgresql'
log_filename = 'postgresql-%Y-%m-%d.log'
log_rotation_age = 1d
log_rotation_size = 100MB
log_line_prefix = '%t [%p]: [%l-1] user=%u,db=%d,app=%a,client=%h '
log_checkpoints = on
log_connections = on
log_disconnections = on
log_duration = on
log_lock_waits = on
log_statement = 'ddl'
```

Edit `/etc/postgresql/14/main/pg_hba.conf`:

```conf
# Allow local connections with password
local   all             all                                     scram-sha-256
host    all             all             127.0.0.1/32            scram-sha-256
host    all             all             ::1/128                 scram-sha-256

# Allow from application servers (adjust IP range)
host    articium      articium_user 10.0.0.0/8              scram-sha-256
```

Restart PostgreSQL:

```bash
sudo systemctl restart postgresql
```

---

## Environment Configuration

### 1. Create .env File

```bash
cd /opt/articium
cp .env.example .env
chmod 600 .env  # Protect secrets
```

### 2. Configure Critical Settings

Edit `.env`:

```bash
# Generate JWT secret
openssl rand -hex 32

# Update .env with generated secret
JWT_SECRET=<your_generated_secret_here>

# Set production database
DB_HOST=localhost
DB_PORT=5432
DB_USER=articium_user
DB_PASSWORD=YOUR_VERY_STRONG_PASSWORD_HERE
DB_NAME=articium
DB_SSLMODE=require  # IMPORTANT: Enable SSL

# Production CORS
CORS_ALLOWED_ORIGINS=https://app.yourdomain.com,https://dashboard.yourdomain.com

# Enable authentication
REQUIRE_AUTH=true

# Production logging
LOG_LEVEL=info
LOG_FORMAT=json
SERVER_ENV=production
```

### 3. Set File Permissions

```bash
sudo chown articium:articium .env
sudo chmod 400 .env  # Read-only for owner
```

---

## Smart Contract Deployment

### 1. Configure Hardhat

Edit `contracts/evm/hardhat.config.js`:

```javascript
networks: {
  polygon: {
    url: process.env.POLYGON_RPC_URL,
    accounts: [process.env.DEPLOYER_PRIVATE_KEY],
    gasPrice: 50000000000 // 50 gwei
  },
  ethereum: {
    url: process.env.ETHEREUM_RPC_URL,
    accounts: [process.env.DEPLOYER_PRIVATE_KEY],
    gasPrice: 30000000000 // 30 gwei
  }
}
```

### 2. Deploy Contracts

```bash
cd contracts/evm

# Install dependencies
npm install

# Deploy to Polygon
npx hardhat run scripts/deploy.js --network polygon

# Deploy BatchSettler
npx hardhat run scripts/deploy-batch-settler.js --network polygon

# Save contract addresses to .env
```

### 3. Verify Contracts

```bash
npx hardhat verify --network polygon <CONTRACT_ADDRESS> <CONSTRUCTOR_ARGS>
```

---

## Server Deployment

### 1. Build Application

```bash
# Clone repository
git clone https://github.com/yourusername/articium.git
cd articium

# Build server
go build -o articium-server cmd/server/main.go

# Build batcher service
go build -o articium-batcher cmd/batcher/main.go
```

### 2. Create Systemd Services

**Main Server**: `/etc/systemd/system/articium-server.service`

```ini
[Unit]
Description=Articium Hub Server
After=network.target postgresql.service

[Service]
Type=simple
User=articium
Group=articium
WorkingDirectory=/opt/articium
Environment="PATH=/usr/local/go/bin:/usr/bin"
EnvironmentFile=/opt/articium/.env
ExecStart=/opt/articium/articium-server
Restart=always
RestartSec=10

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/articium/logs

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=articium-server

[Install]
WantedBy=multi-user.target
```

**Batcher Service**: `/etc/systemd/system/articium-batcher.service`

```ini
[Unit]
Description=Articium Transaction Batcher
After=network.target articium-server.service

[Service]
Type=simple
User=articium
Group=articium
WorkingDirectory=/opt/articium
Environment="PATH=/usr/local/go/bin:/usr/bin"
EnvironmentFile=/opt/articium/.env
ExecStart=/opt/articium/articium-batcher
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### 3. Start Services

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable services
sudo systemctl enable articium-server
sudo systemctl enable articium-batcher

# Start services
sudo systemctl start articium-server
sudo systemctl start articium-batcher

# Check status
sudo systemctl status articium-server
sudo systemctl status articium-batcher

# View logs
sudo journalctl -u articium-server -f
sudo journalctl -u articium-batcher -f
```

---

## Security Hardening

### 1. Enable HTTPS with Nginx

Install Nginx and Certbot:

```bash
sudo apt install nginx certbot python3-certbot-nginx
```

Configure Nginx (`/etc/nginx/sites-available/articium`):

```nginx
# HTTP -> HTTPS redirect
server {
    listen 80;
    server_name api.yourdomain.com;
    return 301 https://$server_name$request_uri;
}

# HTTPS server
server {
    listen 443 ssl http2;
    server_name api.yourdomain.com;

    # SSL configuration
    ssl_certificate /etc/letsencrypt/live/api.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.yourdomain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options DENY always;
    add_header X-Content-Type-Options nosniff always;
    add_header X-XSS-Protection "1; mode=block" always;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=100r/m;
    limit_req zone=api_limit burst=20 nodelay;

    # Proxy to application
    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Health check endpoint (no auth required)
    location /health {
        proxy_pass http://localhost:8080/health;
        access_log off;
    }

    # Metrics endpoint (restrict access)
    location /metrics {
        allow 10.0.0.0/8;  # Internal network only
        deny all;
        proxy_pass http://localhost:9090/metrics;
    }
}
```

Get SSL certificate:

```bash
sudo certbot --nginx -d api.yourdomain.com
```

Enable and start Nginx:

```bash
sudo ln -s /etc/nginx/sites-available/articium /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl enable nginx
sudo systemctl restart nginx
```

### 2. Firewall Configuration

```bash
# Install UFW
sudo apt install ufw

# Default policies
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Allow SSH
sudo ufw allow 22/tcp

# Allow HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Allow Prometheus (from monitoring server only)
sudo ufw allow from 10.0.0.10 to any port 9090

# Enable firewall
sudo ufw enable
```

### 3. Database Backup

Create backup script (`/opt/articium/scripts/backup-db.sh`):

```bash
#!/bin/bash
BACKUP_DIR="/opt/articium/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DB_NAME="articium"
DB_USER="articium_user"

# Create backup
pg_dump -U $DB_USER -Fc $DB_NAME > $BACKUP_DIR/articium_$TIMESTAMP.dump

# Keep only last 30 days
find $BACKUP_DIR -name "articium_*.dump" -mtime +30 -delete

# Upload to S3 (optional)
# aws s3 cp $BACKUP_DIR/articium_$TIMESTAMP.dump s3://your-bucket/backups/
```

Add to crontab:

```bash
sudo crontab -e

# Daily backup at 2 AM
0 2 * * * /opt/articium/scripts/backup-db.sh
```

---

## Monitoring & Maintenance

### 1. Prometheus Configuration

Create `/etc/prometheus/prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'articium-server'
    static_configs:
      - targets: ['localhost:9090']
        labels:
          instance: 'production'
          service: 'articium-server'

  - job_name: 'articium-batcher'
    static_configs:
      - targets: ['localhost:9091']
        labels:
          instance: 'production'
          service: 'articium-batcher'

  - job_name: 'node'
    static_configs:
      - targets: ['localhost:9100']
```

### 2. Key Metrics to Monitor

- **API Request Rate**: `rate(http_requests_total[5m])`
- **Error Rate**: `rate(http_requests_total{status=~"5.."}[5m])`
- **Response Time**: `histogram_quantile(0.95, http_request_duration_seconds_bucket)`
- **Batch Savings**: `bridge_route_total_cost_wei`
- **Webhook Success Rate**: `bridge_webhooks_delivered_total / bridge_webhooks_dispatched_total`
- **Route Discovery Time**: `bridge_route_discovery_latency_seconds`

### 3. Alerts

Create `/etc/prometheus/alerts.yml`:

```yaml
groups:
  - name: articium_alerts
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"

      - alert: SlowAPIResponse
        expr: histogram_quantile(0.95, http_request_duration_seconds_bucket) > 2
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "API response time degraded"

      - alert: DatabaseDown
        expr: up{job="postgresql"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Database is down"
```

---

## Troubleshooting

### Common Issues

**1. Authentication Failing**

```bash
# Check JWT secret is set
grep JWT_SECRET .env

# Verify user exists
psql -U articium_user -d articium -c "SELECT * FROM users;"

# Check auth logs
sudo journalctl -u articium-server | grep auth
```

**2. Database Connection Errors**

```bash
# Test connection
psql -U articium_user -h localhost -d articium

# Check PostgreSQL is running
sudo systemctl status postgresql

# Check logs
sudo tail -f /var/log/postgresql/postgresql-*.log
```

**3. Smart Contract Errors**

```bash
# Check RPC endpoints
curl -X POST $POLYGON_RPC_URL \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# Verify contract addresses
grep BRIDGE_CONTRACT .env
```

**4. High Memory Usage**

```bash
# Check process memory
ps aux | grep articium

# Adjust PostgreSQL shared_buffers if needed
sudo vim /etc/postgresql/14/main/postgresql.conf

# Restart services
sudo systemctl restart articium-server
```

---

## Post-Deployment Checklist

- [ ] Database migrations completed
- [ ] Admin password changed
- [ ] JWT secret configured
- [ ] CORS origins restricted
- [ ] SSL/TLS enabled
- [ ] Firewall configured
- [ ] Backups automated
- [ ] Monitoring configured
- [ ] Alerts set up
- [ ] Smart contracts deployed and verified
- [ ] Load testing completed
- [ ] Documentation updated

---

## Support

For issues and questions:
- GitHub Issues: https://github.com/yourusername/articium/issues
- Documentation: https://docs.articium.io
- Email: support@articium.io
