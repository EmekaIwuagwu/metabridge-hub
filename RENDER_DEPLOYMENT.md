# Deploying Metabridge on Render.com

Complete guide to deploy Metabridge bridge protocol on Render for testing and production.

## What You'll Deploy

- **API Server**: REST API for bridge operations (accessible via HTTPS)
- **Relayer**: Background worker that processes cross-chain messages
- **PostgreSQL**: Managed database (Render provides this)
- **Redis**: Managed cache (optional, can add later)

## Prerequisites

1. Render account (free tier available): https://render.com
2. GitHub account (to connect your repo)
3. RPC API keys (Alchemy, Infura) - free tiers available

## Step 1: Fork/Push Repository to GitHub

```bash
# If you haven't already, push your code to GitHub
cd ~/metabridge/metabridge-engine-hub
git remote set-url origin https://github.com/YOUR_USERNAME/metabridge-engine-hub.git
git push origin main
```

## Step 2: Create Render Account

1. Go to https://render.com
2. Sign up with GitHub
3. Authorize Render to access your repositories

## Step 3: Create PostgreSQL Database

1. In Render Dashboard, click **"New +"** → **"PostgreSQL"**
2. Configure database:
   - **Name**: `metabridge-db`
   - **Database**: `metabridge`
   - **User**: `metabridge_user` (auto-generated)
   - **Region**: Choose closest to you
   - **Plan**: Free (for testing) or Starter ($7/month for production)
3. Click **"Create Database"**
4. Wait 2-3 minutes for database to provision
5. Copy the **Internal Database URL** (starts with `postgres://`)

## Step 4: Create Environment Variables File

Create a `.env` file in your repo root (or use Render's dashboard):

```bash
# Environment
BRIDGE_ENVIRONMENT=testnet

# Database (will be set by Render)
DATABASE_URL=${DATABASE_URL}

# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# JWT Authentication
JWT_SECRET=your-super-secret-jwt-key-here-min-32-chars
JWT_EXPIRATION_HOURS=24

# CORS
CORS_ALLOWED_ORIGINS=*

# Rate Limiting
RATE_LIMIT_PER_MINUTE=100
REQUIRE_AUTH=false
API_KEY_ENABLED=true

# RPC Endpoints (GET FREE KEYS FROM THESE PROVIDERS)
ALCHEMY_API_KEY=your_alchemy_key_here
INFURA_API_KEY=your_infura_key_here
HELIUS_API_KEY=your_helius_key_here

# Chain RPC URLs
POLYGON_RPC_URL=https://rpc-amoy.polygon.technology/
BNB_RPC_URL=https://data-seed-prebsc-1-s1.binance.org:8545/
AVALANCHE_RPC_URL=https://api.avax-test.network/ext/bc/C/rpc
ETHEREUM_RPC_URL=https://sepolia.infura.io/v3/${INFURA_API_KEY}
SOLANA_RPC_URL=https://api.devnet.solana.com
NEAR_RPC_URL=https://rpc.testnet.near.org

# Smart Contract Addresses (leave empty for now, will fill after deployment)
POLYGON_BRIDGE_CONTRACT=
BNB_BRIDGE_CONTRACT=
AVALANCHE_BRIDGE_CONTRACT=
ETHEREUM_BRIDGE_CONTRACT=
SOLANA_BRIDGE_PROGRAM=
NEAR_BRIDGE_CONTRACT=

# Validator (for testing, use a new test wallet)
VALIDATOR_PRIVATE_KEY=your_test_private_key_here

# NATS (we'll use in-memory for Render free tier)
NATS_URL=nats://localhost:4222
```

## Step 5: Create Build Script

Create `build.sh` in your repo root:

```bash
#!/bin/bash
set -e

echo "Building Metabridge API Server..."
go build -o bin/metabridge-api cmd/api/main.go

echo "Build complete!"
```

Make it executable:
```bash
chmod +x build.sh
git add build.sh
git commit -m "Add Render build script"
git push
```

## Step 6: Create Start Script

Create `start.sh` in your repo root:

```bash
#!/bin/bash
set -e

echo "Starting Metabridge API Server..."

# Parse DATABASE_URL to individual components for compatibility
if [ -n "$DATABASE_URL" ]; then
  export DB_HOST=$(echo $DATABASE_URL | sed -e 's|postgresql://||' -e 's|:.*||')
  export DB_PORT=$(echo $DATABASE_URL | sed -e 's|.*:||' -e 's|/.*||')
  export DB_USER=$(echo $DATABASE_URL | sed -e 's|.*://||' -e 's|:.*||')
  export DB_PASSWORD=$(echo $DATABASE_URL | sed -e 's|.*://[^:]*:||' -e 's|@.*||')
  export DB_NAME=$(echo $DATABASE_URL | sed -e 's|.*/||' -e 's|?.*||')
fi

# Run database migrations
echo "Running database migrations..."
if command -v psql &> /dev/null; then
  psql $DATABASE_URL < internal/database/schema.sql || echo "Schema already exists"
  psql $DATABASE_URL < internal/database/auth.sql || echo "Auth schema already exists"
fi

# Start the API server
echo "Starting API server on port ${PORT:-8080}..."
exec ./bin/metabridge-api
```

Make it executable:
```bash
chmod +x start.sh
git add start.sh
git commit -m "Add Render start script"
git push
```

## Step 7: Deploy API Server on Render

1. In Render Dashboard, click **"New +"** → **"Web Service"**
2. Connect your GitHub repository
3. Configure the web service:

**Basic Settings**:
- **Name**: `metabridge-api`
- **Region**: Same as database
- **Branch**: `main`
- **Root Directory**: (leave empty)
- **Runtime**: `Go`
- **Build Command**: `./build.sh`
- **Start Command**: `./start.sh`

**Environment**:
- **Instance Type**: Free (for testing) or Starter ($7/month)

**Environment Variables** (click "Advanced" → "Add Environment Variable"):
Add all variables from your `.env` file above, PLUS:
- Key: `DATABASE_URL`, Value: (copy from PostgreSQL service)
- Key: `PORT`, Value: `8080`

4. Click **"Create Web Service"**

5. Render will:
   - Clone your repo
   - Run `./build.sh` to compile Go binary
   - Run `./start.sh` to start server
   - Assign you a URL like: `https://metabridge-api.onrender.com`

## Step 8: Verify Deployment

Once deployed (5-10 minutes), test your API:

```bash
# Replace with your Render URL
export RENDER_URL=https://metabridge-api.onrender.com

# Test health endpoint
curl $RENDER_URL/health

# Expected response:
# {"status":"ok","version":"1.0.0"}

# Test chain status
curl $RENDER_URL/v1/chains/status

# You should see JSON with chain information
```

## Step 9: Create Admin User (Optional)

If you enabled authentication, create an admin user:

```bash
# Connect to your database via Render shell
# In Render Dashboard → PostgreSQL → "Connect" → Copy PSQL command

# Then run:
psql <your-database-url>

-- Hash your password (use bcrypt online tool or locally)
-- Example: bcrypt hash of "admin123" is $2a$10$...

INSERT INTO users (id, email, name, password_hash, role, active, created_at, updated_at)
VALUES (
  'admin-001',
  'admin@metabridge.local',
  'Admin User',
  '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',  -- "admin123"
  'admin',
  true,
  NOW(),
  NOW()
);
```

## Step 10: Deploy Relayer (Background Worker)

1. In Render Dashboard, click **"New +"** → **"Background Worker"**
2. Select same repository
3. Configure:

**Basic Settings**:
- **Name**: `metabridge-relayer`
- **Region**: Same as API
- **Branch**: `main`
- **Build Command**: `go build -o bin/metabridge-relayer cmd/relayer/main.go`
- **Start Command**: `./bin/metabridge-relayer --config config/config.testnet.yaml`

**Environment Variables**:
Copy all environment variables from API service

4. Click **"Create Background Worker"**

## Testing Your Deployment

### Test 1: Health Check
```bash
curl https://your-app.onrender.com/health
```

### Test 2: Get Chain Status
```bash
curl https://your-app.onrender.com/v1/chains/status
```

### Test 3: Login (if auth enabled)
```bash
curl -X POST https://your-app.onrender.com/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@metabridge.local","password":"admin123"}'
```

### Test 4: Get Bridge Stats
```bash
curl https://your-app.onrender.com/v1/stats
```

## Render Free Tier Limitations

**What's included (FREE)**:
- 750 hours/month of runtime (enough for 1 service 24/7)
- 512 MB RAM
- Shared CPU
- Auto-sleep after 15 minutes of inactivity
- Custom domain support
- Automatic HTTPS

**Limitations**:
- Service sleeps after 15 min inactivity (50s cold start)
- Only 1 free service at a time
- 100 GB bandwidth/month

**For Production** (Starter Plan - $7/month):
- No sleep
- Always on
- 512 MB RAM
- Faster CPU
- Unlimited bandwidth

## Cost Breakdown (Monthly)

### Free Tier (Testing)
- API Server: **$0** (with sleep)
- PostgreSQL: **$0** (25MB limit)
- **Total: $0/month**

### Starter Tier (Production)
- API Server: **$7/month** (no sleep)
- Relayer: **$7/month** (background worker)
- PostgreSQL: **$7/month** (256 MB)
- **Total: $21/month**

### Professional Tier
- API Server: **$25/month** (2 GB RAM)
- Relayer: **$25/month**
- PostgreSQL: **$50/month** (2 GB)
- Redis: **$10/month** (25 MB)
- **Total: $110/month**

## Advantages of Render vs Azure

✅ **Easier**: No SSH, no server management
✅ **Faster**: Deploy in 5 minutes vs 2 hours
✅ **Cheaper**: Free tier available, Azure costs ~$140/month minimum
✅ **Auto-deploy**: Git push = automatic deployment
✅ **Free HTTPS**: Automatic SSL certificates
✅ **Better logs**: Built-in log viewer
✅ **Simpler**: No Docker, systemd, or firewall config needed

## Monitoring on Render

### View Logs
1. Go to Render Dashboard
2. Click on your service
3. Click "Logs" tab
4. See real-time logs

### Metrics
1. Click "Metrics" tab
2. See CPU, memory, and response times

### Custom Domain (Optional)
1. Click "Settings"
2. Scroll to "Custom Domain"
3. Add your domain (e.g., api.yourdomain.com)
4. Update DNS: Add CNAME pointing to your Render URL

## Environment Variables Management

### Option 1: Render Dashboard
1. Service → Settings → Environment
2. Click "Add Environment Variable"
3. Enter key and value
4. Click "Save Changes" (triggers redeploy)

### Option 2: `.env` file (Not Recommended)
- Don't commit `.env` to Git (security risk)
- Use Render dashboard instead

### Option 3: Render Blueprint (Advanced)
Create `render.yaml`:

```yaml
services:
  - type: web
    name: metabridge-api
    runtime: go
    buildCommand: ./build.sh
    startCommand: ./start.sh
    envVars:
      - key: BRIDGE_ENVIRONMENT
        value: testnet
      - key: JWT_SECRET
        generateValue: true
      - key: DATABASE_URL
        fromDatabase:
          name: metabridge-db
          property: connectionString

databases:
  - name: metabridge-db
    databaseName: metabridge
    user: metabridge_user
```

## Troubleshooting

### Service won't start
**Check logs**:
1. Render Dashboard → Your Service → Logs
2. Look for errors during build or start

**Common issues**:
- Missing environment variables
- Database connection failed
- Wrong build command

### Database connection failed
1. Verify `DATABASE_URL` is set correctly
2. Check database is running (green status)
3. Ensure database and service in same region

### Cold starts (free tier)
Service sleeps after 15 min inactivity. First request after sleep takes 30-50 seconds.

**Solutions**:
1. Upgrade to Starter ($7/month) - no sleep
2. Use external uptime monitor (pings every 5 min)
3. Accept the cold start for testing

### Out of memory
Free tier has 512 MB RAM limit.

**Solutions**:
1. Reduce relayer workers in config
2. Upgrade to Starter (512 MB) or Standard (2 GB)
3. Optimize code to use less memory

## Next Steps

1. **Deploy Smart Contracts** (use Hardhat from local machine)
2. **Test Bridge Functionality** (send test transactions)
3. **Add Monitoring** (Render has built-in metrics)
4. **Set Up Custom Domain** (optional)
5. **Upgrade to Starter** (when ready for production)

## Render vs Azure vs Local

| Feature | Render (Free) | Render (Starter) | Azure VM | Local |
|---------|---------------|------------------|----------|-------|
| **Cost** | $0/month | $21/month | $140/month | $0 |
| **Setup Time** | 10 minutes | 10 minutes | 2 hours | 30 minutes |
| **Maintenance** | None | None | High | Medium |
| **Scaling** | Click button | Click button | Manual | N/A |
| **HTTPS** | Automatic | Automatic | Manual | No |
| **Uptime** | 99%* | 99.9% | 99.9% | Variable |
| **Best For** | Testing | Production | Enterprise | Development |

*Service sleeps after 15 min inactivity

## Conclusion

**Use Render if**:
- ✅ You want to test quickly
- ✅ You don't want to manage servers
- ✅ You need automatic deployments
- ✅ Budget is limited ($0-$21/month)

**Use Azure if**:
- ✅ You need full control
- ✅ You want to customize everything
- ✅ You have DevOps experience
- ✅ Budget allows ($140+/month)

**Use Local if**:
- ✅ Just developing/testing
- ✅ Don't need public access
- ✅ Want fastest iteration

For your use case ("see how it works"), **Render is perfect**! Deploy in 10 minutes and start testing immediately.
