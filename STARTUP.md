# 🚀 Articium Hub - Quick Start Guide

## One-Command Startup

Start everything with a single command:

```bash
./start-all.sh
```

That's it! This will:
- ✅ Start PostgreSQL
- ✅ Configure database
- ✅ Start backend API (port 8080)
- ✅ Start frontend (port 3000)

## Commands

```bash
# Start all services
./start-all.sh

# Stop all services
./start-all.sh stop

# Restart all services
./start-all.sh restart

# Check service status
./start-all.sh status
```

## Access URLs

Once started, access:

- **Frontend**: http://localhost:3000 (or http://YOUR_SERVER_IP:3000)
- **API**: http://localhost:8080
- **Health Check**: http://localhost:8080/health

## View Logs

```bash
# API logs
tail -f /tmp/articium-api.log

# Frontend logs
tail -f /tmp/articium-frontend.log

# PostgreSQL logs
tail -f /var/log/postgresql/postgresql-16-main.log
```

## Troubleshooting

### Port Already in Use
```bash
# Kill old processes
./start-all.sh stop
./start-all.sh start
```

### Database Connection Issues
```bash
# Check PostgreSQL
service postgresql status

# Restart PostgreSQL
service postgresql restart
```

### Frontend Not Building
```bash
cd frontend
rm -rf node_modules
npm install
cd ..
./start-all.sh restart
```

## Manual Startup (if needed)

If you need to start services manually:

### Backend:
```bash
export DB_PASSWORD=bridge_password
export REQUIRE_AUTH=false
export BRIDGE_ENVIRONMENT=testnet
./bin/api -config config/config.testnet.yaml
```

### Frontend:
```bash
cd frontend
npm run dev
```

## Testing the Bridge

1. **Install Wallets**:
   - MetaMask: https://metamask.io/
   - Phantom: https://phantom.app/

2. **Get Testnet Tokens**:
   - BNB: https://testnet.bnbchain.org/faucet-smart
   - Solana: https://faucet.solana.com/

3. **Open Frontend**: http://localhost:3000

4. **Connect Wallet** and start bridging!

## Need Help?

- Check logs: `tail -f /tmp/articium-*.log`
- Check status: `./start-all.sh status`
- Restart everything: `./start-all.sh restart`
