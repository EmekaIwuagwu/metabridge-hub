#!/bin/bash

# Redis Troubleshooter and Fixer

echo "╔════════════════════════════════════════════════════════════╗"
echo "║              Redis Troubleshooter                          ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# Check Redis service status
echo "1️⃣  Checking Redis service status..."
systemctl status redis-server --no-pager -l || true
echo ""

# Check Redis logs
echo "2️⃣  Checking Redis error logs..."
journalctl -u redis-server -n 50 --no-pager || true
echo ""

# Check if port 6379 is already in use
echo "3️⃣  Checking if port 6379 is in use..."
if netstat -tuln | grep -q ":6379"; then
    echo "   ⚠️  Port 6379 is already in use!"
    echo "   Process using port 6379:"
    lsof -i :6379 || netstat -tuln | grep ":6379"
    echo ""
    echo "   Killing process on port 6379..."
    fuser -k 6379/tcp 2>/dev/null || true
    sleep 2
else
    echo "   ✓ Port 6379 is available"
fi
echo ""

# Check Redis configuration
echo "4️⃣  Checking Redis configuration..."
if [ -f /etc/redis/redis.conf ]; then
    echo "   ✓ Redis config exists"

    # Check for common issues
    if grep -q "^bind 0.0.0.0" /etc/redis/redis.conf; then
        echo "   ⚠️  Redis is configured to bind to all interfaces"
        echo "   Changing to localhost only for security..."
        sed -i 's/^bind 0.0.0.0/bind 127.0.0.1/' /etc/redis/redis.conf
    fi

    # Ensure supervised is set to systemd
    if ! grep -q "^supervised systemd" /etc/redis/redis.conf; then
        echo "   Fixing supervised mode..."
        sed -i 's/^supervised no/supervised systemd/' /etc/redis/redis.conf
        sed -i 's/^supervised auto/supervised systemd/' /etc/redis/redis.conf
    fi
else
    echo "   ✗ Redis config not found at /etc/redis/redis.conf"
fi
echo ""

# Check Redis data directory permissions
echo "5️⃣  Checking Redis data directory permissions..."
REDIS_DIR="/var/lib/redis"
if [ -d "$REDIS_DIR" ]; then
    echo "   ✓ Redis data directory exists"
    chown -R redis:redis "$REDIS_DIR" 2>/dev/null || true
    chmod 755 "$REDIS_DIR" 2>/dev/null || true
    echo "   ✓ Permissions fixed"
else
    echo "   Creating Redis data directory..."
    mkdir -p "$REDIS_DIR"
    chown -R redis:redis "$REDIS_DIR" 2>/dev/null || true
    chmod 755 "$REDIS_DIR"
    echo "   ✓ Directory created"
fi
echo ""

# Check Redis log directory permissions
echo "6️⃣  Checking Redis log directory permissions..."
REDIS_LOG_DIR="/var/log/redis"
if [ -d "$REDIS_LOG_DIR" ]; then
    chown -R redis:redis "$REDIS_LOG_DIR" 2>/dev/null || true
    chmod 755 "$REDIS_LOG_DIR" 2>/dev/null || true
    echo "   ✓ Log directory permissions fixed"
else
    mkdir -p "$REDIS_LOG_DIR"
    chown -R redis:redis "$REDIS_LOG_DIR" 2>/dev/null || true
    chmod 755 "$REDIS_LOG_DIR"
    echo "   ✓ Log directory created"
fi
echo ""

# Disable protected mode for local development
echo "7️⃣  Configuring Redis for local development..."
if [ -f /etc/redis/redis.conf ]; then
    # Set protected mode to no for easier local development
    sed -i 's/^protected-mode yes/protected-mode no/' /etc/redis/redis.conf

    # Ensure bind is set to localhost
    if ! grep -q "^bind 127.0.0.1" /etc/redis/redis.conf; then
        sed -i '/^bind /d' /etc/redis/redis.conf
        echo "bind 127.0.0.1" >> /etc/redis/redis.conf
    fi

    echo "   ✓ Redis configured"
fi
echo ""

# Try to start Redis
echo "8️⃣  Attempting to start Redis..."
systemctl stop redis-server 2>/dev/null || true
sleep 2
systemctl start redis-server

if systemctl is-active --quiet redis-server; then
    echo "   ✅ Redis started successfully!"

    # Test connection
    if redis-cli ping | grep -q "PONG"; then
        echo "   ✅ Redis is responding to PING"
    else
        echo "   ⚠️  Redis started but not responding to PING"
    fi
else
    echo "   ❌ Redis failed to start"
    echo ""
    echo "Detailed error:"
    journalctl -u redis-server -n 20 --no-pager
    exit 1
fi

echo ""
echo "╔════════════════════════════════════════════════════════════╗"
echo "║              ✅ Redis is now running!                      ║"
echo "╚════════════════════════════════════════════════════════════╝"
