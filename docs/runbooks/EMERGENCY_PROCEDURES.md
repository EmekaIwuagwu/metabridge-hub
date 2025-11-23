# Emergency Procedures

This document outlines procedures for handling critical incidents and emergency situations with the Articium.

## Table of Contents

1. [Emergency Contacts](#emergency-contacts)
2. [Incident Severity Levels](#incident-severity-levels)
3. [Emergency Bridge Pause](#emergency-bridge-pause)
4. [Security Incident Response](#security-incident-response)
5. [Chain Outage Handling](#chain-outage-handling)
6. [Database Failure Recovery](#database-failure-recovery)
7. [Message Queue Failure](#message-queue-failure)
8. [Stuck Transactions](#stuck-transactions)
9. [Validator Compromise](#validator-compromise)
10. [Post-Incident Review](#post-incident-review)

## Emergency Contacts

### On-Call Rotation
- **Primary**: +1-XXX-XXX-XXXX (PagerDuty)
- **Secondary**: +1-XXX-XXX-XXXX
- **Security Team**: security@articium.io
- **Infrastructure**: devops@articium.io

### Escalation Path
1. On-call engineer (immediate)
2. Engineering manager (within 15 minutes)
3. CTO (within 30 minutes for SEV-1)
4. CEO (within 1 hour for SEV-0)

### External Contacts
- **Cloud Provider Support**: AWS/GCP/Azure
- **Security Auditor**: auditor@example.com
- **Legal Counsel**: legal@articium.io

## Incident Severity Levels

### SEV-0: Critical
**Impact**: Complete bridge outage, potential loss of funds
**Response Time**: Immediate
**Examples**:
- Smart contract exploit detected
- Multiple validator keys compromised
- Database completely inaccessible
- Funds locked indefinitely

**Actions**:
1. Trigger emergency pause (all chains)
2. Notify all stakeholders immediately
3. Assemble war room within 15 minutes
4. Engage security auditor
5. Prepare public communication

### SEV-1: High
**Impact**: Major functionality degraded, some chains offline
**Response Time**: < 15 minutes
**Examples**:
- Single chain completely offline
- Relayer stopped processing messages
- Database write failures
- Security vulnerability reported

**Actions**:
1. Page on-call team
2. Assess impact and scope
3. Implement temporary workaround
4. Begin investigation
5. Update status page

### SEV-2: Medium
**Impact**: Minor degradation, reduced capacity
**Response Time**: < 1 hour
**Examples**:
- Increased transaction failures (< 10%)
- Slow message processing
- Single validator offline
- API rate limiting triggered

**Actions**:
1. Notify on-call team
2. Monitor and document
3. Schedule fix during business hours
4. Update internal stakeholders

### SEV-3: Low
**Impact**: No user impact, cosmetic issues
**Response Time**: Next business day
**Examples**:
- Monitoring alert false positive
- Documentation issue
- Non-critical warning logs

## Emergency Bridge Pause

### When to Pause

Pause the bridge immediately if:
- Smart contract vulnerability discovered
- Unusual transaction patterns detected
- Multiple failed signature verifications
- Validator keys potentially compromised
- Major chain reorganization
- Regulatory/legal requirement

### How to Pause (EVM Chains)

```bash
# Connect to owner wallet (hardware wallet recommended)
# For each EVM chain:

# Polygon
npx hardhat run scripts/emergency-pause.ts --network polygon-mainnet

# BNB Chain
npx hardhat run scripts/emergency-pause.ts --network bnb-mainnet

# Avalanche
npx hardhat run scripts/emergency-pause.ts --network avalanche-mainnet

# Ethereum
npx hardhat run scripts/emergency-pause.ts --network ethereum-mainnet

# Verify pause
npx hardhat run scripts/verify-pause.ts --network polygon-mainnet
```

### How to Pause (Solana)

```bash
# Connect with upgrade authority
solana config set --url https://api.mainnet-beta.solana.com

# Call pause instruction
anchor run pause-mainnet

# Verify
solana program show <PROGRAM_ID>
```

### How to Pause (NEAR)

```bash
# Call pause from owner account
near call bridge.articium.near pause '{}' \
  --accountId owner.articium.near \
  --networkId mainnet \
  --gas 50000000000000

# Verify
near view bridge.articium.near get_config
```

### How to Pause (Backend Services)

```bash
# Emergency stop all services
# Kubernetes
kubectl scale deployment api --replicas=0 -n articium-mainnet
kubectl scale deployment listener --replicas=0 -n articium-mainnet
kubectl scale deployment relayer --replicas=0 -n articium-mainnet

# Docker Compose
docker-compose down

# Verify
kubectl get pods -n articium-mainnet
```

### Communication During Pause

1. **Immediate** (< 5 minutes):
   ```
   Update status page: "Bridge temporarily paused for maintenance"
   ```

2. **Within 15 minutes**:
   ```
   Twitter: "The Articium is temporarily paused while we investigate
   an issue. All funds are secure. We will provide updates every 30 minutes."
   ```

3. **Within 30 minutes**:
   - Update Discord/Telegram
   - Email major integrators
   - Prepare detailed technical explanation

## Security Incident Response

### Suspected Exploit

**Immediate Actions** (< 5 minutes):
1. Pause all bridge contracts
2. Stop all backend services
3. Take database snapshot
4. Preserve all logs
5. Notify security team

**Investigation** (< 1 hour):
```bash
# Collect evidence
cd /var/log/articium
tar -czf incident-$(date +%Y%m%d-%H%M%S).tar.gz *.log

# Export database for forensics
pg_dump -h localhost -U articium articium_mainnet \
  > incident_db_$(date +%Y%m%d-%H%M%S).sql

# Collect blockchain transaction data
# For each affected transaction, gather:
# - Transaction hash
# - Block number
# - Sender/receiver
# - Amount
# - Signatures

# Save to incident report
```

**Analysis**:
1. Determine attack vector
2. Estimate funds at risk
3. Identify affected accounts
4. Check if exploit is ongoing
5. Assess blast radius

**Mitigation**:
1. Deploy fix if available
2. Upgrade contracts if possible
3. Contact affected chains
4. Coordinate with law enforcement if needed
5. Work with security auditor on patch

### Vulnerability Disclosure

If vulnerability reported via bug bounty:

1. **Acknowledge** (< 24 hours):
   ```
   Thank researcher
   Confirm receipt
   Provide triage timeline
   ```

2. **Assess** (< 48 hours):
   ```
   Reproduce vulnerability
   Determine severity
   Estimate impact
   Calculate bounty payout
   ```

3. **Remediate** (varies by severity):
   ```
   SEV-0: Emergency patch within hours
   SEV-1: Patch within days
   SEV-2: Patch within weeks
   SEV-3: Include in next release
   ```

4. **Disclose** (after fix deployed):
   ```
   Coordinate with researcher
   Publish post-mortem
   Credit researcher (if permitted)
   Pay bounty
   ```

## Chain Outage Handling

### Detection

Monitor for:
- RPC endpoint failures
- Block number not advancing
- Increased transaction failures
- Network upgrade announcements

### Response

```bash
# Check chain status
curl https://api.polygonscan.com/api?module=proxy&action=eth_blockNumber

# If chain is down:
# 1. Switch to backup RPC endpoint
export POLYGON_RPC="https://polygon-backup-rpc.example.com"

# 2. Update configuration
kubectl edit configmap bridge-config -n articium-mainnet

# 3. Restart affected services
kubectl rollout restart deployment/listener -n articium-mainnet

# 4. Monitor recovery
kubectl logs -f deployment/listener -n articium-mainnet
```

### During Network Upgrade

1. **Pre-upgrade** (24 hours before):
   - Review upgrade notes
   - Test on testnet
   - Prepare rollback plan
   - Schedule maintenance window

2. **During upgrade**:
   - Monitor upgrade progress
   - Watch for consensus issues
   - Check RPC compatibility
   - Test basic operations

3. **Post-upgrade**:
   - Verify all chains operational
   - Run smoke tests
   - Resume normal operations
   - Document any issues

## Database Failure Recovery

### Database Unavailable

```bash
# Check database status
pg_isready -h db.articium.io -U articium

# Attempt connection
psql -h db.articium.io -U articium -d articium_mainnet

# If primary database down:
# 1. Failover to replica
kubectl patch service postgres \
  -p '{"spec":{"selector":{"role":"replica"}}}'

# 2. Promote replica to primary
kubectl exec -it postgres-replica-0 -- \
  pg_ctl promote -D /var/lib/postgresql/data

# 3. Update connection strings
kubectl set env deployment/api \
  DATABASE_HOST=postgres-replica-0.postgres
```

### Data Corruption Detected

```bash
# Stop all services immediately
kubectl scale deployment --all --replicas=0 -n articium-mainnet

# Restore from last known good backup
# Find latest backup
aws s3 ls s3://articium-backups/mainnet/ | tail -n 5

# Restore
pg_restore -h localhost -U articium -d articium_mainnet_restored \
  --clean --if-exists \
  s3://articium-backups/mainnet/backup_TIMESTAMP.dump

# Verify data integrity
psql -h localhost -U articium -d articium_mainnet_restored \
  -c "SELECT COUNT(*), MAX(created_at) FROM messages;"

# Switch to restored database
kubectl set env deployment --all \
  DATABASE_NAME=articium_mainnet_restored

# Restart services
kubectl scale deployment --all --replicas=1 -n articium-mainnet
```

## Message Queue Failure

### NATS Server Down

```bash
# Check NATS status
nats-cli server ping

# Restart NATS
kubectl rollout restart statefulset/nats -n nats-system

# Verify recovery
nats-cli stream ls

# Check message backlog
nats-cli stream info articium

# Resume processing
kubectl scale deployment/relayer --replicas=3 -n articium-mainnet
```

### Message Backlog

```bash
# Check queue depth
nats-cli stream info articium | grep Messages

# If backlog > 10,000:
# 1. Scale up relayers
kubectl scale deployment/relayer --replicas=10 -n articium-mainnet

# 2. Monitor processing rate
watch -n 5 'nats-cli stream info articium | grep Messages'

# 3. If still growing, investigate:
# - Are messages failing?
# - Is relayer stuck?
# - Are chains accepting transactions?

# Check failed messages
psql -h localhost -U articium -d articium_mainnet \
  -c "SELECT COUNT(*) FROM messages WHERE status = 'failed' AND created_at > NOW() - INTERVAL '1 hour';"
```

## Stuck Transactions

### Identifying Stuck Messages

```bash
# Find messages pending > 1 hour
psql -h localhost -U articium -d articium_mainnet -c "
SELECT id, source_chain_name, destination_chain_name, status, created_at
FROM messages
WHERE status = 'pending'
AND created_at < NOW() - INTERVAL '1 hour'
ORDER BY created_at DESC
LIMIT 50;
"
```

### Manual Intervention

```bash
# For a stuck message:
MESSAGE_ID="abc123..."

# Check validator signatures
curl http://localhost:8080/v1/messages/$MESSAGE_ID/signatures

# If insufficient signatures:
# - Check validator service logs
# - Verify validator connectivity
# - Manually trigger signature collection

# If signatures valid but not relayed:
# - Check relayer logs
# - Verify destination chain RPC
# - Check transaction gas price
# - Manually retry transaction

# Retry message processing
curl -X POST http://localhost:8080/v1/admin/retry-message/$MESSAGE_ID \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Gas Price Issues

```bash
# Check current gas prices
curl https://api.etherscan.io/api?module=gastracker&action=gasoracle

# Update gas price multiplier
kubectl set env deployment/relayer \
  GAS_PRICE_MULTIPLIER=1.5 \
  -n articium-mainnet

# Restart relayer
kubectl rollout restart deployment/relayer -n articium-mainnet
```

## Validator Compromise

### Immediate Actions

1. **Revoke compromised validator**:
   ```bash
   # EVM
   npx hardhat run scripts/remove-validator.ts \
     --network polygon-mainnet \
     --validator 0xCOMPROMISED_ADDRESS

   # Solana
   anchor run remove-validator-mainnet \
     --validator COMPROMISED_PUBKEY

   # NEAR
   near call bridge.articium.near remove_validator \
     '{"validator": "ed25519:COMPROMISED_KEY"}' \
     --accountId owner.articium.near \
     --networkId mainnet
   ```

2. **Add replacement validator**:
   ```bash
   # Generate new keypair securely
   # Store in HSM/KMS
   # Add to all chains
   ```

3. **Investigate**:
   - Review all transactions signed by compromised validator
   - Check for unauthorized messages
   - Audit access logs
   - Determine compromise vector

4. **Notify**:
   - Other validators
   - Security team
   - Legal counsel (if needed)
   - Users (if funds at risk)

## Post-Incident Review

### Incident Report Template

```markdown
# Incident Report: [TITLE]

**Date**: YYYY-MM-DD
**Severity**: SEV-X
**Duration**: X hours
**Impact**: [Description]

## Timeline

- HH:MM - Incident detected
- HH:MM - Team paged
- HH:MM - Investigation began
- HH:MM - Root cause identified
- HH:MM - Mitigation deployed
- HH:MM - Service restored
- HH:MM - Incident closed

## Root Cause

[Detailed explanation]

## Impact Assessment

- Users affected: X
- Transactions affected: X
- Funds at risk: $X
- Downtime: X hours

## Resolution

[What was done to fix it]

## Prevention

- [ ] Action item 1
- [ ] Action item 2
- [ ] Action item 3

## Lessons Learned

1. What went well
2. What could be improved
3. Lucky breaks
```

### Post-Mortem Meeting

Within 48 hours of incident resolution:

1. **Attendees**:
   - Engineering team
   - DevOps
   - Security
   - Management

2. **Agenda**:
   - Timeline review
   - Root cause analysis
   - Impact assessment
   - Prevention measures
   - Process improvements

3. **Outcomes**:
   - Action items assigned
   - Timeline for implementation
   - Follow-up meeting scheduled

4. **Communication**:
   - Internal summary
   - External post-mortem (if appropriate)
   - Blog post/documentation update
