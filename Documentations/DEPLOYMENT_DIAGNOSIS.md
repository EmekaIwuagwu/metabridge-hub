# Metabridge Deployment Diagnosis & Fix Guide

## Root Cause Analysis

Your systemd service is failing with a "resources" error. After analyzing the codebase, I've identified **5 critical issues**:

### Issue 1: Database Name Mismatch
- **Expected**: `metabridge_testnet` (from config/config.testnet.yaml:14)
- **Actual**: `metabridge_production` (your Docker setup)
- **Impact**: API server can't connect to the database

### Issue 2: Missing Production Config File
- **Available**: `config.testnet.yaml`, `config.mainnet.yaml`
- **Missing**: Production-specific config file
- **Impact**: Service uses testnet config with wrong database name

### Issue 3: Binary Names Don't Match
- **Makefile builds**: `bin/api`, `bin/relayer`
- **Systemd expects**: `bin/api-server`, `bin/relayer-server` (possibly)
- **Impact**: systemd can't find the correct binary path

### Issue 4: Missing Database Schema
- **Status**: Database created, but tables not initialized
- **Required**: Run schema.sql, auth.sql, and other SQL files
- **Impact**: API server crashes when trying to query non-existent tables

### Issue 5: Environment Variables Not Set
- Config uses placeholders like `${DB_PASSWORD}`, `${ALCHEMY_API_KEY}`
- **Impact**: These need to be set or replaced with actual values

---

## Step-by-Step Fix (Run on your DigitalOcean server)

### Step 1: Verify Binary Names
```bash
cd /root/projects/metabridge-engine-hub
ls -lh bin/
```

**Expected output**: You should see `api` and `relayer` (NOT `api-server`)

---

### Step 2: Create Production Config File

Create `config/config.production.yaml`:

```bash
cat > config/config.production.yaml << 'EOF'
environment: "testnet"

server:
  host: "0.0.0.0"
  port: 8080
  tls_enabled: false
  read_timeout: "30s"
  write_timeout: "30s"
  max_header_bytes: 1048576

database:
  host: "localhost"
  port: 5432
  database: "metabridge_production"
  username: "bridge_user"
  password: "secure_bridge_pass_2024"
  ssl_mode: "disable"
  max_open_conns: 25
  max_idle_conns: 5
  max_lifetime: "5m"

queue:
  type: "nats"
  urls:
    - "nats://localhost:4222"
  subject: "bridge.messages"
  stream_name: "BRIDGE_MESSAGES"
  max_retries: 3

cache:
  type: "redis"
  addresses:
    - "localhost:6379"
  password: ""
  db: 0
  ttl: "1h"

relayer:
  workers: 5
  max_retries: 3
  retry_backoff: "5s"
  processing_timeout: "2m"
  enable_circuit_breaker: false
  circuit_breaker_threshold: 5
  batch_size: 10

security:
  required_signatures: 2
  validator_addresses:
    - "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0"
    - "0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199"
    - "0xdD2FD4581271e230360230F9337D5c0430Bf44C0"
  max_transaction_amount: "10000"
  daily_volume_limit: "100000"
  enable_rate_limiting: true
  rate_limit_per_hour: 100
  rate_limit_per_address: 20
  enable_emergency_pause: false
  enable_fraud_detection: false
  large_transaction_threshold: "5000"

crypto:
  evm_keystore_path: "./keystore/testnet/evm_validator.json"
  solana_keystore_path: "./keystore/testnet/solana_validator.json"
  near_keystore_path: "./keystore/testnet/near_validator.json"
  password_env_var: "TESTNET_KEYSTORE_PASSWORD"
  use_aws_kms: false

monitoring:
  prometheus_port: 9090
  enable_tracing: false
  jaeger_endpoint: ""
  log_level: "debug"
  enable_metrics_export: false
  metrics_export_interval: "30s"

alerting:
  enabled: false
  slack_webhook: ""
  pagerduty_key: ""
  alert_on_failure_threshold: 5
  alert_on_high_gas: false
  alert_on_large_transaction: false

chains:
  - name: "polygon-amoy"
    chain_type: "EVM"
    environment: "testnet"
    chain_id: "80002"
    rpc_endpoints:
      - "https://rpc-amoy.polygon.technology/"
    bridge_contract: "0x0000000000000000000000000000000000000000"
    start_block: 0
    confirmation_blocks: 128
    block_time: "2s"
    max_gas_price: "500"
    gas_limit_multiplier: 1.2
    max_reorg_depth: 256
    enabled: false
EOF
```

---

### Step 3: Initialize Database Schema

Run all schema files in order:

```bash
cd /root/projects/metabridge-engine-hub

# Main schema
docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production < internal/database/schema.sql

# Auth schema
docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production < internal/database/auth.sql

# Additional schemas
docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production < internal/database/batches.sql
docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production < internal/database/webhooks.sql
docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production < internal/database/routes.sql
```

**Verify tables were created:**
```bash
docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production -c "\dt"
```

You should see tables like: chains, messages, validators, transactions, etc.

---

### Step 4: Update Systemd Service Files

**For API Service** (`/etc/systemd/system/metabridge-api.service`):

```bash
cat > /etc/systemd/system/metabridge-api.service << 'EOF'
[Unit]
Description=Metabridge API Server
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=root
WorkingDirectory=/root/projects/metabridge-engine-hub
Environment="PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin"
ExecStart=/root/projects/metabridge-engine-hub/bin/api -config config/config.production.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```

**For Relayer Service** (`/etc/systemd/system/metabridge-relayer.service`):

```bash
cat > /etc/systemd/system/metabridge-relayer.service << 'EOF'
[Unit]
Description=Metabridge Relayer Service
After=network.target docker.service metabridge-api.service
Requires=docker.service
Wants=metabridge-api.service

[Service]
Type=simple
User=root
WorkingDirectory=/root/projects/metabridge-engine-hub
Environment="PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin"
ExecStart=/root/projects/metabridge-engine-hub/bin/relayer -config config/config.production.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```

---

### Step 5: Reload and Start Services

```bash
# Reload systemd configuration
systemctl daemon-reload

# Start API service
systemctl start metabridge-api

# Check status
systemctl status metabridge-api

# View detailed logs if it fails
journalctl -xeu metabridge-api -n 100
```

**If successful**, start the relayer:
```bash
systemctl start metabridge-relayer
systemctl status metabridge-relayer
```

---

## Security Warning

Your PostgreSQL is receiving brute-force attacks! Configure firewall NOW:

```bash
# Allow only essential ports
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw allow 8080/tcp  # API
ufw enable
ufw status
```

---

## Verification Commands

After services start successfully:

```bash
# Check all services
systemctl status metabridge-api metabridge-relayer

# View live logs
journalctl -u metabridge-api -f

# Test API endpoint
curl http://localhost:8080/health

# Check Docker services
docker ps

# Check database connection
docker exec -i metabridge-postgres psql -U bridge_user -d metabridge_production -c "SELECT COUNT(*) FROM chains;"
```

---

## Expected Result

After following these steps, you should see:

✅ API server running on port 8080
✅ Relayer service running
✅ Both services showing "active (running)" in systemctl status
✅ No errors in journalctl logs
✅ Database tables populated with seed data

---

## Next Actions After Success

1. Test API: `curl http://159.65.73.133:8080/health`
2. Enable on boot: `systemctl enable metabridge-api metabridge-relayer`
3. Configure proper SSL/TLS for production
4. Set up monitoring and alerting
5. Deploy smart contracts to testnets
