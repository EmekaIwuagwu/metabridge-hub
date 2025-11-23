# Systemd Service Troubleshooting Guide

This guide helps you troubleshoot and resolve issues when starting Articium services.

## Quick Diagnosis

### Run the Dependency Checker

```bash
bash scripts/check-dependencies.sh
```

This script automatically checks:
- PostgreSQL (port 5432)
- NATS (port 4222)
- Redis (port 6379)
- Binary files
- Configuration files
- Log directories

### Start All Dependencies

```bash
bash scripts/start-dependencies.sh
```

This script automatically starts all required services.

---

## Common Issues and Solutions

### ❌ Error: "Job failed because of unavailable resources"

**Cause**: Required dependencies (PostgreSQL, NATS, Redis) are not running.

**Solution**:

1. Check which services are missing:
   ```bash
   bash scripts/check-dependencies.sh
   ```

2. Start the missing services:
   ```bash
   # PostgreSQL
   sudo systemctl start postgresql

   # Redis
   sudo systemctl start redis

   # NATS (if not installed)
   bash scripts/start-dependencies.sh
   ```

3. Verify services are running:
   ```bash
   sudo systemctl status postgresql
   sudo systemctl status redis
   pgrep -a nats-server
   ```

---

### ❌ Error: "No such file or directory" for config file

**Cause**: Configuration file is missing or in the wrong location.

**Solution**:

1. Check if config file exists:
   ```bash
   ls -la /root/projects/articium/config/config.production.yaml
   ```

2. If missing, copy from template:
   ```bash
   cp config/config.testnet.yaml config/config.production.yaml
   ```

3. Update the config with your production settings:
   ```bash
   nano config/config.production.yaml
   ```

---

### ❌ Error: "Permission denied" when accessing binary

**Cause**: Binary files don't have execute permissions.

**Solution**:

```bash
chmod +x bin/*
ls -lh bin/
```

Expected output: All files should have `rwxr-xr-x` permissions.

---

### ❌ Error: "Failed to connect to database"

**Cause**: PostgreSQL is not running or credentials are incorrect.

**Solution**:

1. Start PostgreSQL:
   ```bash
   sudo systemctl start postgresql
   sudo systemctl enable postgresql
   ```

2. Create the database:
   ```bash
   sudo -u postgres psql -c "CREATE DATABASE articium_production;"
   sudo -u postgres psql -c "ALTER USER postgres WITH PASSWORD 'postgres_admin_password';"
   ```

3. Test connection:
   ```bash
   psql -h localhost -U postgres -d articium_production -c "SELECT version();"
   ```

4. Run migrations:
   ```bash
   ./bin/migrator -config config/config.production.yaml
   ```

---

### ❌ Error: "Failed to connect to NATS"

**Cause**: NATS server is not running.

**Solution**:

1. Check if NATS is installed:
   ```bash
   which nats-server
   ```

2. If not installed, run:
   ```bash
   bash scripts/start-dependencies.sh
   ```

3. Or manually install:
   ```bash
   wget https://github.com/nats-io/nats-server/releases/download/v2.10.7/nats-server-v2.10.7-linux-amd64.tar.gz
   tar -xzf nats-server-v2.10.7-linux-amd64.tar.gz
   sudo mv nats-server-v2.10.7-linux-amd64/nats-server /usr/local/bin/
   ```

4. Start NATS with JetStream:
   ```bash
   nohup nats-server -js > logs/nats.log 2>&1 &
   ```

5. Verify NATS is running:
   ```bash
   curl http://localhost:8222/varz
   ```

---

### ❌ Error: "ReadWritePaths directive failed"

**Cause**: The logs directory doesn't exist.

**Solution**:

```bash
mkdir -p /root/projects/articium/logs
chmod 755 /root/projects/articium/logs
```

---

## Viewing Logs

### Real-time Service Logs

```bash
# API logs
sudo journalctl -u articium-api -f

# Relayer logs
sudo journalctl -u articium-relayer -f

# Listener logs
sudo journalctl -u articium-listener -f

# Batcher logs
sudo journalctl -u articium-batcher -f

# All services
sudo journalctl -u articium-* -f
```

### Last 100 Lines

```bash
sudo journalctl -u articium-api -n 100 --no-pager
```

### Logs Since Boot

```bash
sudo journalctl -u articium-api -b --no-pager
```

### Error Logs Only

```bash
sudo journalctl -u articium-api -p err --no-pager
```

---

## Service Management Commands

### Start Services

```bash
sudo systemctl start articium-api
sudo systemctl start articium-relayer
sudo systemctl start articium-listener
sudo systemctl start articium-batcher
```

### Stop Services

```bash
sudo systemctl stop articium-api
sudo systemctl stop articium-relayer
sudo systemctl stop articium-listener
sudo systemctl stop articium-batcher
```

### Restart Services

```bash
sudo systemctl restart articium-api
```

### Check Status

```bash
sudo systemctl status articium-api
sudo systemctl status articium-relayer
sudo systemctl status articium-listener
sudo systemctl status articium-batcher
```

### Enable Auto-start on Boot

```bash
sudo systemctl enable articium-api
sudo systemctl enable articium-relayer
sudo systemctl enable articium-listener
sudo systemctl enable articium-batcher
```

### Disable Auto-start

```bash
sudo systemctl disable articium-api
```

---

## Reloading Service Files

After modifying service files, reload systemd:

```bash
sudo systemctl daemon-reload
```

Then restart the affected service:

```bash
sudo systemctl restart articium-api
```

---

## Dependency Order

Services start in this order:

1. **PostgreSQL** - Database (required by all services)
2. **NATS** - Message queue (required by listener)
3. **Redis** - Cache (optional for API)
4. **articium-api** - API server (required by other services)
5. **articium-listener** - Blockchain listeners
6. **articium-batcher** - Transaction batcher
7. **articium-relayer** - Transaction relayer

---

## Complete Setup Checklist

- [ ] PostgreSQL installed and running
- [ ] NATS installed and running
- [ ] Redis installed and running
- [ ] Database created (`articium_production`)
- [ ] Database user configured
- [ ] Configuration file exists (`config/config.production.yaml`)
- [ ] Configuration file updated with correct values
- [ ] Binaries built and executable (`bin/*`)
- [ ] Logs directory created (`logs/`)
- [ ] Migrations run (`bin/migrator`)
- [ ] Service files copied to `/etc/systemd/system/`
- [ ] Systemd daemon reloaded
- [ ] Services enabled for auto-start
- [ ] Services started successfully

---

## Testing the Setup

### 1. Check All Dependencies

```bash
bash scripts/check-dependencies.sh
```

### 2. Test Database Connection

```bash
psql -h localhost -U postgres -d articium_production -c "SELECT COUNT(*) FROM messages;"
```

### 3. Test API Endpoint

```bash
curl http://localhost:8080/api/v1/health
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2024-11-22T15:00:00Z",
  "components": {
    "database": "healthy",
    "queue": "healthy",
    "cache": "healthy"
  }
}
```

### 4. Check Prometheus Metrics

```bash
curl http://localhost:9090/metrics
```

---

## Emergency Procedures

### Stop All Services

```bash
sudo systemctl stop articium-api articium-relayer articium-listener articium-batcher
```

### Reset Database (WARNING: Deletes all data)

```bash
sudo -u postgres psql -c "DROP DATABASE IF EXISTS articium_production;"
sudo -u postgres psql -c "CREATE DATABASE articium_production;"
./bin/migrator -config config/config.production.yaml
```

### View All Service Status

```bash
systemctl list-units "articium-*" --all
```

---

## Getting Help

If you're still experiencing issues:

1. Run the dependency checker and save output:
   ```bash
   bash scripts/check-dependencies.sh > diagnosis.txt 2>&1
   ```

2. Collect service logs:
   ```bash
   sudo journalctl -u articium-* --no-pager > service-logs.txt
   ```

3. Check file permissions:
   ```bash
   ls -laR /root/projects/articium/ > file-permissions.txt
   ```

4. Share the output files for support.
