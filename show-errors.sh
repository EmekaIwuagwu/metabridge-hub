#!/bin/bash

echo "=== API Service Logs (last 50 lines) ==="
sudo journalctl -u articium-api -n 50 --no-pager
echo ""

echo "=== Listener Service Logs (last 50 lines) ==="
sudo journalctl -u articium-listener -n 50 --no-pager
echo ""

echo "=== Batcher Service Logs (last 50 lines) ==="
sudo journalctl -u articium-batcher -n 50 --no-pager
echo ""

echo "=== NATS Status ==="
systemctl status nats --no-pager || echo "NATS service not found"
echo ""

echo "=== Redis Status ==="
systemctl status redis-server --no-pager || echo "Redis service not found"
echo ""

echo "=== Test /health endpoint ==="
curl -v http://localhost:8080/health
echo ""

echo "=== Test /ready endpoint ==="
curl -v http://localhost:8080/ready
echo ""

echo "=== List all listening ports ==="
ss -tlnp | grep -E ":(8080|4222|6379|5433)"
