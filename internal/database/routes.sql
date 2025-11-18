-- Multi-Hop Routing Database Schema

-- Routes table
CREATE TABLE IF NOT EXISTS routes (
    id VARCHAR(100) PRIMARY KEY,
    source_chain VARCHAR(50) NOT NULL,
    dest_chain VARCHAR(50) NOT NULL,
    total_hops INTEGER NOT NULL,
    total_cost VARCHAR(100) NOT NULL,
    total_time_seconds BIGINT NOT NULL,
    total_fee VARCHAR(100) NOT NULL,
    score NUMERIC(5, 4) NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL CHECK (status IN ('PENDING', 'EXECUTING', 'COMPLETED', 'FAILED', 'PARTIAL')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    executed_at TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX idx_routes_source_dest ON routes(source_chain, dest_chain);
CREATE INDEX idx_routes_status ON routes(status);
CREATE INDEX idx_routes_created_at ON routes(created_at DESC);
CREATE INDEX idx_routes_score ON routes(score DESC);

-- Route hops table
CREATE TABLE IF NOT EXISTS route_hops (
    id SERIAL PRIMARY KEY,
    route_id VARCHAR(100) NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
    step INTEGER NOT NULL,
    source_chain VARCHAR(50) NOT NULL,
    dest_chain VARCHAR(50) NOT NULL,
    bridge_contract VARCHAR(100),
    estimated_cost VARCHAR(100),
    estimated_time_seconds BIGINT,
    gas_price VARCHAR(100),
    liquidity VARCHAR(100),
    message_id VARCHAR(100),
    tx_hash VARCHAR(100),
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING'
);

CREATE INDEX idx_route_hops_route_id ON route_hops(route_id);
CREATE INDEX idx_route_hops_step ON route_hops(route_id, step);

-- Chain pairs table (for tracking connectivity and performance)
CREATE TABLE IF NOT EXISTS chain_pairs (
    id SERIAL PRIMARY KEY,
    source_chain VARCHAR(50) NOT NULL,
    dest_chain VARCHAR(50) NOT NULL,
    available BOOLEAN NOT NULL DEFAULT TRUE,
    total_liquidity VARCHAR(100),
    available_liquidity VARCHAR(100),
    reserved_liquidity VARCHAR(100) DEFAULT '0',
    base_fee VARCHAR(100),
    gas_price VARCHAR(100),
    average_time_seconds BIGINT,
    success_rate NUMERIC(5, 4) DEFAULT 1.0,
    total_transactions INTEGER DEFAULT 0,
    successful_transactions INTEGER DEFAULT 0,
    failed_transactions INTEGER DEFAULT 0,
    last_updated TIMESTAMP NOT NULL DEFAULT NOW(),
    bridge_contract VARCHAR(100),
    UNIQUE(source_chain, dest_chain)
);

CREATE INDEX idx_chain_pairs_source ON chain_pairs(source_chain);
CREATE INDEX idx_chain_pairs_dest ON chain_pairs(dest_chain);
CREATE INDEX idx_chain_pairs_available ON chain_pairs(available);

-- Route executions table (for tracking multi-hop executions)
CREATE TABLE IF NOT EXISTS route_executions (
    id VARCHAR(100) PRIMARY KEY,
    route_id VARCHAR(100) NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
    current_hop INTEGER NOT NULL DEFAULT 0,
    total_hops INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('PENDING', 'EXECUTING', 'COMPLETED', 'FAILED', 'PARTIAL')),
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_update TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    error_message TEXT
);

CREATE INDEX idx_route_executions_route_id ON route_executions(route_id);
CREATE INDEX idx_route_executions_status ON route_executions(status);

-- Route statistics view
CREATE OR REPLACE VIEW route_stats AS
SELECT
    source_chain,
    dest_chain,
    COUNT(*) as total_routes,
    COUNT(*) FILTER (WHERE status = 'COMPLETED') as completed_routes,
    COUNT(*) FILTER (WHERE status = 'FAILED') as failed_routes,
    AVG(total_hops) as avg_hops,
    AVG(CAST(total_cost AS NUMERIC)) as avg_cost,
    AVG(total_time_seconds) as avg_time_seconds,
    AVG(score) as avg_score,
    MAX(created_at) as last_route_created
FROM routes
GROUP BY source_chain, dest_chain;

-- Route performance view
CREATE OR REPLACE VIEW route_performance AS
SELECT
    r.id,
    r.source_chain,
    r.dest_chain,
    r.total_hops,
    r.status,
    r.score,
    EXTRACT(EPOCH FROM (COALESCE(r.completed_at, NOW()) - r.created_at)) as execution_time_seconds,
    CASE
        WHEN r.completed_at IS NOT NULL THEN
            EXTRACT(EPOCH FROM (r.completed_at - r.created_at))
        ELSE NULL
    END as actual_time_seconds,
    r.total_time_seconds as estimated_time_seconds
FROM routes r
WHERE r.created_at > NOW() - INTERVAL '30 days';

-- Active routes view
CREATE OR REPLACE VIEW active_routes AS
SELECT
    r.id,
    r.source_chain,
    r.dest_chain,
    r.total_hops,
    r.status,
    re.current_hop,
    re.started_at,
    re.last_update,
    EXTRACT(EPOCH FROM (NOW() - re.started_at)) as elapsed_seconds
FROM routes r
JOIN route_executions re ON r.id = re.route_id
WHERE r.status IN ('EXECUTING', 'PARTIAL')
ORDER BY re.started_at DESC;

-- Chain connectivity graph view
CREATE OR REPLACE VIEW chain_connectivity AS
SELECT
    cp.source_chain,
    cp.dest_chain,
    cp.available,
    cp.success_rate,
    cp.available_liquidity,
    cp.average_time_seconds,
    cp.last_updated,
    CASE
        WHEN cp.success_rate >= 0.95 THEN 'excellent'
        WHEN cp.success_rate >= 0.90 THEN 'good'
        WHEN cp.success_rate >= 0.80 THEN 'fair'
        ELSE 'poor'
    END as health_status
FROM chain_pairs cp
WHERE cp.available = true
ORDER BY cp.source_chain, cp.dest_chain;

-- Function to update chain pair statistics
CREATE OR REPLACE FUNCTION update_chain_pair_stats()
RETURNS void AS $$
BEGIN
    -- Update chain pair statistics based on message history
    INSERT INTO chain_pairs (
        source_chain,
        dest_chain,
        total_transactions,
        successful_transactions,
        failed_transactions,
        success_rate,
        average_time_seconds,
        last_updated
    )
    SELECT
        source_chain,
        dest_chain,
        COUNT(*) as total,
        COUNT(*) FILTER (WHERE status = 'CONFIRMED') as successful,
        COUNT(*) FILTER (WHERE status = 'FAILED') as failed,
        COALESCE(
            CAST(COUNT(*) FILTER (WHERE status = 'CONFIRMED') AS NUMERIC) / NULLIF(COUNT(*), 0),
            0
        ) as success_rate,
        AVG(EXTRACT(EPOCH FROM (COALESCE(confirmed_at, NOW()) - created_at)))::BIGINT as avg_time,
        NOW()
    FROM messages
    WHERE created_at > NOW() - INTERVAL '7 days'
    GROUP BY source_chain, dest_chain
    ON CONFLICT (source_chain, dest_chain) DO UPDATE SET
        total_transactions = EXCLUDED.total_transactions,
        successful_transactions = EXCLUDED.successful_transactions,
        failed_transactions = EXCLUDED.failed_transactions,
        success_rate = EXCLUDED.success_rate,
        average_time_seconds = EXCLUDED.average_time_seconds,
        last_updated = EXCLUDED.last_updated;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up old route data
CREATE OR REPLACE FUNCTION cleanup_old_routes()
RETURNS void AS $$
BEGIN
    -- Delete completed routes older than 90 days
    DELETE FROM routes
    WHERE status = 'COMPLETED'
      AND completed_at < NOW() - INTERVAL '90 days';

    -- Delete failed routes older than 30 days
    DELETE FROM routes
    WHERE status = 'FAILED'
      AND updated_at < NOW() - INTERVAL '30 days';

    -- Delete orphaned route executions
    DELETE FROM route_executions
    WHERE route_id NOT IN (SELECT id FROM routes);
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update route updated_at timestamp
CREATE OR REPLACE FUNCTION update_route_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_route_updated_at ON routes;
CREATE TRIGGER trigger_update_route_updated_at
    BEFORE UPDATE ON routes
    FOR EACH ROW
    EXECUTE FUNCTION update_route_updated_at();

-- Insert default chain pairs for common routes
INSERT INTO chain_pairs (
    source_chain,
    dest_chain,
    available,
    total_liquidity,
    available_liquidity,
    reserved_liquidity,
    base_fee,
    gas_price,
    average_time_seconds,
    success_rate,
    bridge_contract
) VALUES
    ('polygon', 'ethereum', true, '100000000000000000000', '80000000000000000000', '0', '10000000000000000', '50000000000', 300, 0.98, '0x0000000000000000000000000000000000000000'),
    ('ethereum', 'polygon', true, '100000000000000000000', '80000000000000000000', '0', '10000000000000000', '50000000000', 300, 0.98, '0x0000000000000000000000000000000000000000'),
    ('polygon', 'bsc', true, '100000000000000000000', '80000000000000000000', '0', '10000000000000000', '50000000000', 300, 0.98, '0x0000000000000000000000000000000000000000'),
    ('bsc', 'polygon', true, '100000000000000000000', '80000000000000000000', '0', '10000000000000000', '50000000000', 300, 0.98, '0x0000000000000000000000000000000000000000'),
    ('ethereum', 'bsc', true, '100000000000000000000', '80000000000000000000', '0', '10000000000000000', '50000000000', 300, 0.98, '0x0000000000000000000000000000000000000000'),
    ('bsc', 'ethereum', true, '100000000000000000000', '80000000000000000000', '0', '10000000000000000', '50000000000', 300, 0.98, '0x0000000000000000000000000000000000000000'),
    ('polygon', 'avalanche', true, '100000000000000000000', '80000000000000000000', '0', '10000000000000000', '50000000000', 300, 0.98, '0x0000000000000000000000000000000000000000'),
    ('avalanche', 'polygon', true, '100000000000000000000', '80000000000000000000', '0', '10000000000000000', '50000000000', 300, 0.98, '0x0000000000000000000000000000000000000000'),
    ('ethereum', 'avalanche', true, '100000000000000000000', '80000000000000000000', '0', '10000000000000000', '50000000000', 300, 0.98, '0x0000000000000000000000000000000000000000'),
    ('avalanche', 'ethereum', true, '100000000000000000000', '80000000000000000000', '0', '10000000000000000', '50000000000', 300, 0.98, '0x0000000000000000000000000000000000000000'),
    ('bsc', 'avalanche', true, '100000000000000000000', '80000000000000000000', '0', '10000000000000000', '50000000000', 300, 0.98, '0x0000000000000000000000000000000000000000'),
    ('avalanche', 'bsc', true, '100000000000000000000', '80000000000000000000', '0', '10000000000000000', '50000000000', 300, 0.98, '0x0000000000000000000000000000000000000000')
ON CONFLICT (source_chain, dest_chain) DO NOTHING;

-- Comments for documentation
COMMENT ON TABLE routes IS 'Stores multi-hop routes discovered by the routing engine';
COMMENT ON TABLE route_hops IS 'Individual hops in a multi-hop route';
COMMENT ON TABLE chain_pairs IS 'Chain pair connectivity and performance data';
COMMENT ON TABLE route_executions IS 'Tracking data for route executions';
COMMENT ON VIEW route_stats IS 'Aggregated statistics for routes by chain pair';
COMMENT ON VIEW route_performance IS 'Performance metrics for recent routes';
COMMENT ON VIEW active_routes IS 'Currently executing routes';
COMMENT ON VIEW chain_connectivity IS 'Chain connectivity graph with health status';
