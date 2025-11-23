# Monitoring and Observability Runbook

This guide covers monitoring, alerting, and observability for the Articium bridge protocol.

## Table of Contents

1. [Monitoring Stack](#monitoring-stack)
2. [Key Metrics](#key-metrics)
3. [Alert Thresholds](#alert-thresholds)
4. [Dashboard Guide](#dashboard-guide)
5. [Log Analysis](#log-analysis)
6. [Performance Tuning](#performance-tuning)
7. [Capacity Planning](#capacity-planning)

## Monitoring Stack

### Components

- **Prometheus**: Metrics collection and storage
- **Grafana**: Visualization and dashboards
- **Alertmanager**: Alert routing and management
- **Loki**: Log aggregation
- **Jaeger**: Distributed tracing (optional)

### Access

- **Grafana**: https://grafana.articium.io
- **Prometheus**: https://prometheus.articium.io
- **Alertmanager**: https://alertmanager.articium.io

## Key Metrics

### Bridge Health Metrics

#### Message Processing
```promql
# Total messages processed
sum(bridge_messages_total)

# Messages by status
sum by (status) (bridge_messages_total)

# Processing rate (messages/second)
rate(bridge_messages_total[5m])

# Failed message rate
rate(bridge_messages_total{status="failed"}[5m])

# Message processing duration
histogram_quantile(0.95, rate(bridge_message_processing_duration_bucket[5m]))
```

#### Chain Health
```promql
# Chain connectivity (1 = healthy, 0 = unhealthy)
chain_health{chain="polygon"}

# Block lag
chain_current_block - chain_latest_block

# RPC response time
histogram_quantile(0.95, rate(chain_rpc_duration_bucket[5m]))

# Transaction confirmation time
histogram_quantile(0.95, rate(chain_tx_confirmation_duration_bucket[5m]))
```

#### Validator Metrics
```promql
# Signature success rate
rate(validator_signatures_total{status="success"}[5m]) /
rate(validator_signatures_total[5m])

# Signature verification time
histogram_quantile(0.95, rate(validator_signature_verification_duration_bucket[5m]))

# Active validators
validator_active_count
```

### System Metrics

#### API Service
```promql
# Request rate
rate(api_requests_total[5m])

# Error rate
rate(api_requests_total{status=~"5.."}[5m])

# Response time (p95)
histogram_quantile(0.95, rate(api_request_duration_bucket[5m]))

# Concurrent connections
api_current_connections
```

#### Relayer Service
```promql
# Worker utilization
relayer_workers_busy / relayer_workers_total

# Queue depth
queue_messages_pending

# Processing lag
queue_messages_age_seconds

# Throughput
rate(relayer_messages_processed[5m])
```

#### Database
```promql
# Connection pool utilization
database_connections_active / database_connections_max

# Query duration
histogram_quantile(0.95, rate(database_query_duration_bucket[5m]))

# Slow queries (> 1s)
rate(database_query_duration_bucket{le="1"}[5m])

# Database size
database_size_bytes
```

#### Message Queue (NATS)
```promql
# Queue depth
nats_stream_messages

# Consumer lag
nats_consumer_lag

# Delivery rate
rate(nats_deliveries_total[5m])

# Redelivery rate
rate(nats_redeliveries_total[5m])
```

## Alert Thresholds

### Critical Alerts (Page Immediately)

```yaml
# Bridge completely down
- alert: BridgeDown
  expr: up{job="bridge-api"} == 0
  for: 1m
  severity: critical
  annotations:
    summary: "Bridge API is down"
    description: "Bridge API has been down for more than 1 minute"

# High failure rate
- alert: HighFailureRate
  expr: |
    rate(bridge_messages_total{status="failed"}[5m]) /
    rate(bridge_messages_total[5m]) > 0.1
  for: 5m
  severity: critical
  annotations:
    summary: "High message failure rate: {{ $value }}%"

# Chain offline
- alert: ChainOffline
  expr: chain_health == 0
  for: 2m
  severity: critical
  annotations:
    summary: "Chain {{ $labels.chain }} is offline"

# Database down
- alert: DatabaseDown
  expr: up{job="postgres"} == 0
  for: 1m
  severity: critical
  annotations:
    summary: "Database is down"

# Queue backlog critical
- alert: QueueBacklogCritical
  expr: nats_stream_messages > 100000
  for: 10m
  severity: critical
  annotations:
    summary: "Message queue backlog: {{ $value }} messages"
```

### Warning Alerts (Notify Team)

```yaml
# Slow processing
- alert: SlowMessageProcessing
  expr: |
    histogram_quantile(0.95,
      rate(bridge_message_processing_duration_bucket[5m])
    ) > 60
  for: 10m
  severity: warning
  annotations:
    summary: "Slow message processing: {{ $value }}s"

# High memory usage
- alert: HighMemoryUsage
  expr: |
    (node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) /
    node_memory_MemTotal_bytes > 0.9
  for: 5m
  severity: warning
  annotations:
    summary: "High memory usage: {{ $value }}%"

# Disk space low
- alert: DiskSpaceLow
  expr: |
    (node_filesystem_avail_bytes /
    node_filesystem_size_bytes) < 0.1
  for: 5m
  severity: warning
  annotations:
    summary: "Disk space low: {{ $value }}% available"

# Connection pool exhausted
- alert: ConnectionPoolExhausted
  expr: |
    database_connections_active /
    database_connections_max > 0.9
  for: 5m
  severity: warning
  annotations:
    summary: "Database connection pool nearly exhausted"
```

### Info Alerts (Monitor)

```yaml
# Elevated error rate
- alert: ElevatedErrorRate
  expr: |
    rate(api_requests_total{status=~"5.."}[5m]) /
    rate(api_requests_total[5m]) > 0.01
  for: 10m
  severity: info
  annotations:
    summary: "Elevated API error rate: {{ $value }}%"

# Validator offline
- alert: ValidatorOffline
  expr: validator_active{status="online"} == 0
  for: 15m
  severity: info
  annotations:
    summary: "Validator {{ $labels.address }} offline"

# Queue growing
- alert: QueueGrowing
  expr: deriv(nats_stream_messages[5m]) > 10
  for: 15m
  severity: info
  annotations:
    summary: "Message queue growing: {{ $value }} msg/s"
```

## Dashboard Guide

### Main Bridge Dashboard

**URL**: https://grafana.articium.io/d/bridge-overview

**Panels**:
1. **Bridge Health**:
   - Overall status indicator
   - Messages processed (24h)
   - Success rate
   - Active chains

2. **Message Flow**:
   - Messages by source chain
   - Messages by destination chain
   - Message status distribution
   - Processing time histogram

3. **Chain Status**:
   - Health indicator per chain
   - Block lag
   - RPC latency
   - Transaction success rate

4. **System Resources**:
   - CPU usage
   - Memory usage
   - Disk I/O
   - Network throughput

### Service-Specific Dashboards

#### API Dashboard
- Request rate by endpoint
- Response time percentiles
- Error rate by status code
- Active connections
- Rate limit hits

#### Relayer Dashboard
- Worker utilization
- Processing throughput
- Failed transactions
- Gas costs
- Retry statistics

#### Database Dashboard
- Connections (active/idle/max)
- Query performance
- Table sizes
- Index usage
- Replication lag

### Business Metrics Dashboard

- Total volume locked (TVL)
- Volume by chain pair
- Unique users
- Average transaction size
- Fee revenue
- Top routes

## Log Analysis

### Accessing Logs

#### Kubernetes
```bash
# API logs
kubectl logs -f deployment/api -n articium-mainnet

# Relayer logs
kubectl logs -f deployment/relayer -n articium-mainnet

# All services
kubectl logs -f -l app=articium -n articium-mainnet --all-containers
```

#### Docker Compose
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f relayer
```

### Log Queries (Loki)

```logql
# All errors in last hour
{app="articium"} |= "error" | json

# Failed message processing
{app="articium",service="relayer"} |= "failed to process message"

# Slow queries
{app="articium"} |= "slow query" | duration > 1s

# Signature verification failures
{app="articium"} |= "signature verification failed"

# Chain connection issues
{app="articium"} |= "failed to connect" | chain =~ ".*"
```

### Common Log Patterns

```bash
# Find stuck messages
grep "message_status=pending" /var/log/articium/relayer.log | \
  awk '{print $1}' | sort | uniq -c | sort -n

# Identify error spikes
grep "level=error" /var/log/articium/*.log | \
  cut -d' ' -f1 | uniq -c

# Track transaction confirmations
grep "transaction_confirmed" /var/log/articium/relayer.log | \
  grep -oP 'duration=\K[0-9]+' | \
  awk '{sum+=$1; count++} END {print "Average:", sum/count, "seconds"}'
```

## Performance Tuning

### Database Optimization

```sql
-- Identify slow queries
SELECT
  query,
  calls,
  total_time,
  mean_time,
  stddev_time
FROM pg_stat_statements
WHERE mean_time > 1000
ORDER BY mean_time DESC
LIMIT 20;

-- Check missing indexes
SELECT
  schemaname,
  tablename,
  seq_scan,
  seq_tup_read,
  idx_scan,
  idx_tup_fetch
FROM pg_stat_user_tables
WHERE seq_scan > 0
ORDER BY seq_tup_read DESC
LIMIT 20;

-- Vacuum statistics
SELECT
  schemaname,
  tablename,
  last_vacuum,
  last_autovacuum,
  n_dead_tup
FROM pg_stat_user_tables
WHERE n_dead_tup > 1000
ORDER BY n_dead_tup DESC;
```

### Connection Pool Tuning

```yaml
# Adjust based on load
database:
  max_open_conns: 50  # Increase if pool exhausted
  max_idle_conns: 10  # Reduce if too many idle
  max_lifetime: 1h    # Rotate connections
```

### Relayer Worker Scaling

```bash
# Check current worker utilization
curl http://localhost:9090/api/v1/query?query='relayer_workers_busy/relayer_workers_total'

# If utilization > 0.8, scale up
kubectl scale deployment/relayer --replicas=5 -n articium-mainnet

# Monitor queue depth after scaling
watch -n 5 'nats-cli stream info articium | grep Messages'
```

### Message Queue Tuning

```bash
# Increase consumer ack wait time for slow processing
nats consumer edit articium relayer --ack-wait 60s

# Adjust max deliveries to reduce retries
nats consumer edit articium relayer --max-deliver 5

# Enable flow control for backpressure
nats consumer edit articium relayer --flow-control
```

## Capacity Planning

### Growth Projections

Monitor weekly to project capacity needs:

```promql
# Message volume growth rate
(
  rate(bridge_messages_total[7d]) -
  rate(bridge_messages_total[7d] offset 7d)
) / rate(bridge_messages_total[7d] offset 7d)

# Database growth rate
(
  database_size_bytes -
  database_size_bytes offset 7d
) / database_size_bytes offset 7d
```

### Scaling Triggers

| Metric | Threshold | Action |
|--------|-----------|--------|
| API CPU > 70% | 5m | Add API replicas |
| Relayer CPU > 80% | 5m | Add relayer workers |
| DB CPU > 75% | 10m | Vertical scale database |
| DB Connections > 80% | 5m | Increase pool size |
| Queue depth > 50k | 30m | Scale relayers |
| Disk usage > 80% | - | Add storage |

### Resource Recommendations

#### Minimum (Testnet)
- API: 1 vCPU, 2 GB RAM
- Listener: 1 vCPU, 2 GB RAM
- Relayer: 2 vCPU, 4 GB RAM
- Database: 2 vCPU, 8 GB RAM, 100 GB SSD
- NATS: 1 vCPU, 2 GB RAM

#### Production (Mainnet)
- API: 4 vCPU, 8 GB RAM (3 replicas)
- Listener: 2 vCPU, 4 GB RAM (per chain)
- Relayer: 4 vCPU, 8 GB RAM (5 replicas)
- Database: 8 vCPU, 32 GB RAM, 500 GB SSD
- NATS: 4 vCPU, 8 GB RAM (3 node cluster)

#### High Volume (> 10k msg/day)
- API: 8 vCPU, 16 GB RAM (5 replicas)
- Listener: 4 vCPU, 8 GB RAM (per chain)
- Relayer: 8 vCPU, 16 GB RAM (10 replicas)
- Database: 16 vCPU, 64 GB RAM, 1 TB SSD
- NATS: 8 vCPU, 16 GB RAM (5 node cluster)

## Health Check Endpoints

### API
```bash
# Basic health
curl http://localhost:8080/health

# Readiness
curl http://localhost:8080/ready

# Detailed status
curl http://localhost:8080/v1/status
```

### Services
```bash
# Check all services healthy
curl -f http://localhost:8080/health && \
curl -f http://localhost:8081/health && \
curl -f http://localhost:8082/health && \
echo "All services healthy"
```

### External Dependencies
```bash
# Database
pg_isready -h db.articium.io -U articium

# NATS
nats-cli server ping

# Redis
redis-cli ping
```

## Troubleshooting Guide

### High Latency

1. Check database query performance
2. Verify RPC endpoint response times
3. Review relayer worker utilization
4. Check network latency between services
5. Analyze slow log entries

### Memory Leaks

1. Monitor heap usage over time
2. Check for goroutine leaks
3. Review connection pool sizes
4. Analyze garbage collection metrics
5. Enable profiling if needed

### CPU Spikes

1. Identify which service
2. Check for signature verification load
3. Review database connection thrashing
4. Analyze log volume
5. Check for infinite loops

### Disk Space Issues

1. Check log rotation
2. Clean old database backups
3. Review temp file usage
4. Archive old transactions
5. Expand volume if needed
