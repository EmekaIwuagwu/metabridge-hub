# Mainnet Deployment Guide

**⚠️ CRITICAL: Production Mainnet Deployment with Real Funds**

This guide covers the complete process for deploying Metabridge Engine to production mainnet networks. This deployment will handle real cryptocurrency transactions and requires extensive preparation.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Pre-Deployment Checklist](#pre-deployment-checklist)
- [Security Requirements](#security-requirements)
- [Infrastructure Setup](#infrastructure-setup)
- [Environment Configuration](#environment-configuration)
- [Smart Contract Deployment](#smart-contract-deployment)
- [Service Deployment](#service-deployment)
- [Post-Deployment Verification](#post-deployment-verification)
- [Monitoring & Alerts](#monitoring--alerts)
- [Emergency Procedures](#emergency-procedures)
- [Rollback Procedures](#rollback-procedures)

---

## Prerequisites

### Minimum Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| **RAM** | 32GB | 64GB |
| **CPU** | 8 cores | 16 cores |
| **Disk** | 1TB SSD | 2TB NVMe SSD |
| **Network** | 1 Gbps | 10 Gbps |
| **Uptime** | 99.9% SLA | 99.99% SLA |

### Software Requirements

- Ubuntu Server 22.04 LTS (or similar)
- Docker 24.0+ with Docker Compose
- Go 1.21+
- Node.js 18+ (for contract deployment)
- PostgreSQL 15+ (production-grade)
- SSL/TLS certificates (Let's Encrypt or commercial)
- AWS KMS or HSM for key management

### Team Requirements

- **Security Lead**: Responsible for security audit and compliance
- **DevOps Lead**: Responsible for infrastructure and deployment
- **Blockchain Engineers**: 2+ engineers familiar with all supported chains
- **24/7 On-Call**: Team available for immediate response
- **Legal/Compliance**: For regulatory requirements

---

## Pre-Deployment Checklist

### 1. Security Audit (MANDATORY)

```bash
# Before deployment, you MUST have:
✓ Professional security audit completed
✓ All critical issues resolved
✓ All high-severity issues resolved
✓ Audit report reviewed and approved
✓ security-audit.verified file created
```

**Recommended Audit Firms:**
- Trail of Bits: https://www.trailofbits.com/
- ConsenSys Diligence: https://consensys.net/diligence/
- OpenZeppelin: https://www.openzeppelin.com/security-audits
- CertiK: https://www.certik.com/
- Quantstamp: https://quantstamp.com/

**Cost:** $50,000 - $200,000 depending on scope

### 2. Testing Completion

```bash
✓ All unit tests passing
✓ All integration tests passing
✓ End-to-end tests on all testnets
✓ Load testing completed (expected volume + 10x)
✓ Stress testing completed
✓ Failure scenario testing
✓ Cross-chain transfer verification on testnets
```

### 3. Infrastructure Preparation

```bash
✓ Production servers provisioned
✓ SSL certificates installed
✓ DNS configured and tested
✓ Firewall rules configured
✓ VPN access for team
✓ Backup systems tested
✓ Monitoring systems configured
✓ Alerting configured (Slack, PagerDuty, email)
```

### 4. Operational Readiness

```bash
✓ Runbooks prepared and reviewed
✓ Incident response plan documented
✓ Emergency contact list updated
✓ Communication channels established
✓ Post-deployment monitoring plan
✓ Rollback procedures tested
```

### 5. Legal & Compliance

```bash
✓ Legal review completed
✓ Terms of Service finalized
✓ Privacy Policy published
✓ Regulatory requirements met
✓ Insurance coverage obtained (recommended)
✓ Bug bounty program planned
```

---

## Security Requirements

### Multi-Signature Configuration

Mainnet requires **3-of-5 multisig** for all validator operations:

```yaml
# config/config.mainnet.yaml
security:
  signature_scheme: "ecdsa_secp256k1"
  required_signatures: 3
  total_validators: 5

validators:
  - address: "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0"
    name: "Validator-1"
    kms_key_id: "arn:aws:kms:us-east-1:xxx:key/xxx"

  - address: "0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199"
    name: "Validator-2"
    kms_key_id: "arn:aws:kms:us-east-1:xxx:key/xxx"

  - address: "0xdD2FD4581271e230360230F9337D5c0430Bf44C0"
    name: "Validator-3"
    kms_key_id: "arn:aws:kms:us-east-1:xxx:key/xxx"

  - address: "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed"
    name: "Validator-4"
    kms_key_id: "arn:aws:kms:us-east-1:xxx:key/xxx"

  - address: "0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359"
    name: "Validator-5"
    kms_key_id: "arn:aws:kms:us-east-1:xxx:key/xxx"
```

### AWS KMS Setup

**Never store private keys in plaintext on mainnet servers!**

#### Create KMS Keys

```bash
# Create KMS key for each validator
for i in {1..5}; do
  aws kms create-key \
    --description "Metabridge Mainnet Validator $i" \
    --key-usage SIGN_VERIFY \
    --customer-master-key-spec ECC_SECG_P256K1 \
    --tags TagKey=Project,TagValue=Metabridge \
           TagKey=Environment,TagValue=Mainnet \
           TagKey=Validator,TagValue=$i
done
```

#### Configure IAM Permissions

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "kms:Sign",
        "kms:Verify",
        "kms:DescribeKey",
        "kms:GetPublicKey"
      ],
      "Resource": "arn:aws:kms:us-east-1:*:key/*",
      "Condition": {
        "StringEquals": {
          "kms:RequestAlias": "alias/metabridge-mainnet-*"
        }
      }
    }
  ]
}
```

### HSM Alternative

For maximum security, consider using a Hardware Security Module:

**Recommended HSM Providers:**
- AWS CloudHSM
- Thales Luna HSM
- Ledger Enterprise

**Cost:** $1,000 - $5,000/month

---

## Infrastructure Setup

### Production Server Configuration

#### Azure VM Setup (Recommended)

```bash
# Create Resource Group
az group create \
  --name metabridge-mainnet-rg \
  --location eastus

# Create Virtual Network
az network vnet create \
  --resource-group metabridge-mainnet-rg \
  --name metabridge-vnet \
  --address-prefix 10.0.0.0/16 \
  --subnet-name metabridge-subnet \
  --subnet-prefix 10.0.1.0/24

# Create VM
az vm create \
  --resource-group metabridge-mainnet-rg \
  --name metabridge-mainnet-vm \
  --image UbuntuLTS \
  --size Standard_E8s_v5 \
  --admin-username metabridge-admin \
  --ssh-key-values @~/.ssh/id_rsa.pub \
  --public-ip-address-allocation static \
  --public-ip-sku Standard

# Attach Premium SSD Data Disk (2TB)
az vm disk attach \
  --resource-group metabridge-mainnet-rg \
  --vm-name metabridge-mainnet-vm \
  --name metabridge-data-disk \
  --size-gb 2048 \
  --sku Premium_LRS \
  --new

# Configure NSG Rules
az network nsg rule create \
  --resource-group metabridge-mainnet-rg \
  --nsg-name metabridge-mainnet-vmNSG \
  --name AllowHTTPS \
  --priority 1000 \
  --destination-port-ranges 443 \
  --protocol Tcp \
  --access Allow
```

#### Firewall Configuration

```bash
# Allow only required ports
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp    # SSH (from specific IPs only)
sudo ufw allow 80/tcp    # HTTP (for Let's Encrypt)
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable

# Restrict SSH to specific IPs
sudo ufw delete allow 22/tcp
sudo ufw allow from YOUR_IP_ADDRESS to any port 22
```

---

## Environment Configuration

### 1. Copy and Configure Environment File

```bash
cd /home/metabridge-admin/metabridge-hub
cp .env.mainnet.example .env.mainnet
```

### 2. Configure Critical Settings

Edit `.env.mainnet`:

```bash
# Database (use strong passwords!)
DATABASE_PASSWORD=$(openssl rand -base64 32)

# Redis
REDIS_PASSWORD=$(openssl rand -base64 32)

# Validators (3-of-5)
VALIDATOR_ADDRESSES=0x742d...,0x8626...,0xdD2F...,0x5aAe...,0xfB69...
REQUIRED_SIGNATURES=3

# AWS KMS
AWS_KMS_KEY_ID=arn:aws:kms:us-east-1:123456789012:key/12345678-...
AWS_REGION=us-east-1

# Multi-sig Wallet
MULTI_SIG_WALLET=0x1234567890123456789012345678901234567890

# RPC Endpoints (use dedicated endpoints!)
POLYGON_RPC_URL=https://polygon-mainnet.g.alchemy.com/v2/YOUR_KEY
BNB_RPC_URL=https://bsc-dataseed1.binance.org/
AVALANCHE_RPC_URL=https://api.avax.network/ext/bc/C/rpc
ETHEREUM_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY

# SSL/TLS
DOMAIN=bridge.yourdomain.com
SSL_CERT_PATH=/etc/letsencrypt/live/bridge.yourdomain.com/fullchain.pem
SSL_KEY_PATH=/etc/letsencrypt/live/bridge.yourdomain.com/privkey.pem

# Monitoring
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/WEBHOOK/URL
PAGERDUTY_INTEGRATION_KEY=YOUR_PAGERDUTY_KEY
EMAIL_ALERTS=ops@yourdomain.com,security@yourdomain.com
```

### 3. Secure the Environment File

```bash
chmod 600 .env.mainnet
chown metabridge-admin:metabridge-admin .env.mainnet
```

---

## Smart Contract Deployment

### 1. Prepare Deployment Wallet

```bash
# Create or import deployment wallet with sufficient funds
cd contracts/evm

# Estimated costs per chain (as of 2024):
# Polygon: ~$5-10
# BNB: ~$2-5
# Avalanche: ~$3-8
# Ethereum: ~$100-500 (varies with gas prices)
```

### 2. Configure Hardhat for Mainnet

Edit `contracts/evm/.env`:

```bash
# Deployment wallet private key (use hardware wallet if possible)
PRIVATE_KEY=0x...

# API Keys
ALCHEMY_API_KEY=...
INFURA_API_KEY=...

# Block explorer keys (for verification)
POLYGONSCAN_API_KEY=...
BSCSCAN_API_KEY=...
SNOWTRACE_API_KEY=...
ETHERSCAN_API_KEY=...
```

### 3. Deploy Contracts

```bash
cd contracts/evm
npm install

# Deploy to each mainnet (one at a time, verify each)

# Polygon Mainnet
npm run deploy:polygon -- --network polygon
# Wait for confirmation, verify on PolygonScan

# BNB Smart Chain Mainnet
npm run deploy:bnb -- --network bnb
# Wait for confirmation, verify on BscScan

# Avalanche C-Chain Mainnet
npm run deploy:avalanche -- --network avalanche
# Wait for confirmation, verify on Snowtrace

# Ethereum Mainnet (most expensive - do last)
npm run deploy:ethereum -- --network ethereum
# Wait for confirmation, verify on Etherscan
```

### 4. Verify Contracts

```bash
# Verify each contract on block explorers
npx hardhat verify --network polygon 0xYOUR_CONTRACT_ADDRESS "arg1" "arg2"
npx hardhat verify --network bnb 0xYOUR_CONTRACT_ADDRESS "arg1" "arg2"
npx hardhat verify --network avalanche 0xYOUR_CONTRACT_ADDRESS "arg1" "arg2"
npx hardhat verify --network ethereum 0xYOUR_CONTRACT_ADDRESS "arg1" "arg2"
```

### 5. Transfer Ownership to Multi-sig

**CRITICAL:** Transfer contract ownership to multi-signature wallet:

```bash
# For each chain, transfer ownership
node scripts/transfer-ownership.js \
  --network polygon \
  --contract 0xCONTRACT_ADDRESS \
  --multisig 0xMULTISIG_WALLET_ADDRESS
```

### 6. Update Configuration

Update `config/config.mainnet.yaml` with deployed contract addresses:

```yaml
chains:
  - name: "polygon"
    chain_type: "EVM"
    chain_id: "137"
    bridge_contract: "0xYOUR_DEPLOYED_CONTRACT_ADDRESS"

  - name: "bnb"
    chain_type: "EVM"
    chain_id: "56"
    bridge_contract: "0xYOUR_DEPLOYED_CONTRACT_ADDRESS"

  # ... etc for all chains
```

---

## Service Deployment

### 1. Complete Security Audit Verification

```bash
# Create security audit verification file
cat > security-audit.verified << 'EOF'
AUDIT_COMPANY: Trail of Bits
AUDIT_DATE: 2024-01-15
AUDIT_REPORT: /docs/security/audit-report.pdf
CRITICAL_ISSUES: 0
HIGH_ISSUES: 0
VERIFIED_BY: John Doe, CTO
VERIFIED_DATE: 2024-01-20
EOF
```

### 2. Run Deployment Script

```bash
cd /home/metabridge-admin/metabridge-hub

# Source environment
export $(cat .env.mainnet | grep -v '^#' | xargs)

# Run deployment (will ask for confirmation)
./deploy-mainnet.sh
```

The script will:
1. ✅ Verify security audit
2. ✅ Check system prerequisites (32GB RAM, 1TB disk, etc.)
3. ✅ Verify validator configuration (5 validators)
4. ✅ Verify AWS KMS access
5. ✅ Verify multi-sig wallet configuration
6. ✅ Create secure directories
7. ✅ Build optimized binaries
8. ✅ Create automatic backup
9. ✅ Start infrastructure (PostgreSQL, NATS, Redis)
10. ✅ Run database migrations
11. ✅ Start services (API, Listener, Relayer)
12. ✅ Verify deployment
13. ✅ Configure monitoring
14. ✅ Create rollback script

### 3. Expected Output

```
======================================================================
  ⚠️  MAINNET DEPLOYMENT - REAL FUNDS AT RISK ⚠️
======================================================================

This script will deploy Metabridge Engine to PRODUCTION MAINNET.
All transactions will use REAL cryptocurrency.

Prerequisites Checklist:
  ✓ Security audit completed and signed off
  ✓ Multi-signature wallets configured (3-of-5)
  ✓ AWS KMS or HSM for validator key management
  ...

Type 'DEPLOY TO MAINNET' to proceed:
DEPLOY TO MAINNET

[INFO] Verifying security audit...
[SUCCESS] Security audit verified
[SUCCESS] All deployment verification checks passed!

======================================================================
  Metabridge Engine Mainnet - Deployment Complete!
======================================================================
```

---

## Post-Deployment Verification

### 1. Health Checks

```bash
# API health
curl https://bridge.yourdomain.com/health
# Expected: {"status":"healthy","timestamp":"..."}

# System status
curl https://bridge.yourdomain.com/v1/status
# Expected: Full system status with all chains

# Supported chains
curl https://bridge.yourdomain.com/v1/chains
# Expected: List of all 6 chains with mainnet configs
```

### 2. Test Small Transfer

```bash
# Test with MINIMAL amount first
node scripts/cross-chain-transfer.js \
  --source polygon \
  --destination avalanche \
  --amount "10000000000000000" \  # 0.01 token
  --token "0xTOKEN_ADDRESS"

# Monitor closely
# Wait for completion
# Verify on block explorers
```

### 3. Verify Monitoring

- ✅ Prometheus: http://your-server:9090
- ✅ Grafana: http://your-server:3000
- ✅ Alerts configured in Slack/PagerDuty
- ✅ Email alerts working

### 4. Database Verification

```bash
# Check database tables
docker exec metabridge-postgres psql -U metabridge -d metabridge_mainnet -c "\dt"

# Check validators
docker exec metabridge-postgres psql -U metabridge -d metabridge_mainnet -c "SELECT * FROM validators;"

# Check initial state
docker exec metabridge-postgres psql -U metabridge -d metabridge_mainnet -c "SELECT COUNT(*) FROM messages;"
```

---

## Monitoring & Alerts

### Critical Metrics to Monitor

1. **Transaction Success Rate**
   - Target: >99.9%
   - Alert: <99%

2. **Validator Availability**
   - Target: All 5 validators online
   - Alert: Any validator offline >5 minutes

3. **Signature Collection Time**
   - Target: <30 seconds
   - Alert: >60 seconds

4. **Gas Prices**
   - Target: Within configured limits
   - Alert: Approaching 90% of max

5. **Database Performance**
   - Target: Query time <100ms
   - Alert: >500ms

6. **API Response Time**
   - Target: <200ms
   - Alert: >1000ms

### Alert Configuration

```yaml
# grafana/alerts.yaml
alerts:
  - name: "High Failed Transaction Rate"
    condition: "failed_transactions > 1%"
    severity: "critical"
    notify: ["slack", "pagerduty", "email"]

  - name: "Validator Offline"
    condition: "validator_status == offline"
    severity: "critical"
    notify: ["slack", "pagerduty", "sms"]

  - name: "High Gas Prices"
    condition: "gas_price > max_gas_price * 0.9"
    severity: "warning"
    notify: ["slack"]
```

---

## Emergency Procedures

### Pause Bridge Operations

```bash
# Emergency pause (requires multi-sig)
curl -X POST https://bridge.yourdomain.com/v1/admin/pause \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"reason":"Emergency maintenance","signatures":["..."]}'
```

### Stop Services

```bash
# Graceful shutdown
./stop-mainnet.sh

# This will:
# 1. Create automatic backup
# 2. Stop all services gracefully
# 3. Preserve all data
```

### Emergency Rollback

```bash
# If deployment has critical issues
./rollback-mainnet.sh

# This will:
# 1. Stop all services
# 2. Restore from latest backup
# 3. Restore configuration
```

---

## Rollback Procedures

### Automatic Rollback

```bash
# The rollback script is automatically created during deployment
ls -l rollback-mainnet.sh

# To execute rollback
./rollback-mainnet.sh

# Type 'ROLLBACK' to confirm
```

### Manual Rollback

```bash
# 1. Stop services
./stop-mainnet.sh

# 2. Find backup
ls -lt backups/

# 3. Restore database
docker exec -i metabridge-postgres psql -U metabridge metabridge_mainnet < \
  backups/pre-deployment-20240115_103045/database.sql

# 4. Restore config
cp backups/pre-deployment-20240115_103045/config.mainnet.yaml \
   config/config.mainnet.yaml

# 5. Restart with previous version
git checkout PREVIOUS_COMMIT_HASH
./deploy-mainnet.sh
```

---

## Cost Estimates

### Infrastructure (Monthly)

| Component | Cost |
|-----------|------|
| Azure VM (Standard_E8s_v5) | $400-500 |
| Premium SSD (2TB) | $200-300 |
| Static IP | $5 |
| Bandwidth | $50-200 |
| Backups (S3) | $50-100 |
| **Total Infrastructure** | **~$705-1,105/month** |

### Security & Monitoring

| Component | Cost |
|-----------|------|
| Security Audit (one-time) | $50,000-200,000 |
| AWS KMS (5 keys) | $5-10/month |
| Monitoring (Grafana Cloud) | $50-200/month |
| PagerDuty | $25-100/month |
| Bug Bounty Program | $1,000-10,000/month |
| Insurance (optional) | $500-2,000/month |

### RPC Endpoints

| Provider | Cost |
|----------|------|
| Alchemy Pro | $199-499/month |
| Infura Pro | $225-500/month |
| QuickNode Pro | $299-999/month |

**Total Estimated Monthly Cost:** $1,500 - $5,000

---

## Best Practices

### 1. Gradual Rollout

```bash
# Week 1: Deploy but keep low limits
MAX_TRANSACTION_AMOUNT_USD=1000
DAILY_VOLUME_LIMIT_USD=10000

# Week 2-4: Monitor and gradually increase
MAX_TRANSACTION_AMOUNT_USD=10000
DAILY_VOLUME_LIMIT_USD=100000

# Month 2+: Full production limits
MAX_TRANSACTION_AMOUNT_USD=100000
DAILY_VOLUME_LIMIT_USD=1000000
```

### 2. Regular Backups

```bash
# Configure automated backups every 6 hours
0 */6 * * * /home/metabridge-admin/metabridge-hub/scripts/backup.sh

# Keep backups for 30 days
find /backups/* -mtime +30 -delete
```

### 3. Security Updates

```bash
# Weekly security reviews
# Monthly dependency updates
# Quarterly penetration testing
# Yearly full security audit
```

### 4. Incident Response

1. **Detect**: Automated monitoring alerts team
2. **Assess**: On-call engineer evaluates severity
3. **Respond**: Execute appropriate runbook
4. **Communicate**: Update status page and notify users
5. **Resolve**: Fix issue and verify
6. **Review**: Post-mortem within 48 hours

---

## Support & Resources

- **Emergency Contact:** security@yourdomain.com
- **Status Page:** https://status.yourdomain.com
- **Documentation:** https://docs.yourdomain.com
- **Discord:** https://discord.gg/your-server

---

## Legal & Compliance

### Terms of Service

Ensure users accept:
- Risk disclosure
- Transaction limits
- Supported chains
- Fee structure
- Liability limitations

### Regulatory Compliance

- AML/KYC requirements (varies by jurisdiction)
- Geographic restrictions
- Transaction reporting
- Data protection (GDPR, etc.)

---

**REMEMBER: Mainnet deployment involves REAL FUNDS. Take your time, follow all steps, and have your team ready for immediate support.**

Good luck! 🚀
