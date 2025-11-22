#!/bin/bash

set -e

echo "╔════════════════════════════════════════════════════════════╗"
echo "║          Reset Database - Metabridge                       ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

echo "⚠️  WARNING: This will DROP and recreate the metabridge_prod database!"
echo ""
read -p "Are you sure you want to continue? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "Aborted."
    exit 1
fi

echo ""
echo "1️⃣  Dropping existing database..."
sudo -u postgres psql << EOF
-- Terminate all connections to the database
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname = 'metabridge_prod' AND pid <> pg_backend_pid();

-- Drop and recreate database
DROP DATABASE IF EXISTS metabridge_prod;
CREATE DATABASE metabridge_prod OWNER metabridge;

-- Grant all privileges
GRANT ALL PRIVILEGES ON DATABASE metabridge_prod TO metabridge;
EOF
echo "   ✓ Database dropped and recreated"
echo ""

echo "2️⃣  Granting schema permissions..."
sudo -u postgres psql -d metabridge_prod << EOF
-- Grant all privileges on public schema
GRANT ALL ON SCHEMA public TO metabridge;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO metabridge;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO metabridge;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO metabridge;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO metabridge;
EOF
echo "   ✓ Schema permissions granted"
echo ""

echo "╔════════════════════════════════════════════════════════════╗"
echo "║              ✅ Database Reset Complete!                   ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""
echo "Next step: Run fix-all.sh to apply migrations and start services"
echo "  sudo bash fix-all.sh"
echo ""
