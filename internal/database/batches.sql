-- Batch Management Schema
-- Add this to the main schema.sql or run separately

-- ============================================================
-- BATCHES TABLE
-- ============================================================
CREATE TABLE IF NOT EXISTS batches (
    id VARCHAR(100) PRIMARY KEY,
    merkle_root VARCHAR(66) NOT NULL UNIQUE,
    message_count INTEGER NOT NULL DEFAULT 0,
    total_value NUMERIC(78, 0) NOT NULL DEFAULT 0,

    -- Chain information
    source_chain_id INTEGER NOT NULL REFERENCES chains(id),
    dest_chain_id INTEGER NOT NULL REFERENCES chains(id),

    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'READY', 'SUBMITTED', 'CONFIRMED', 'FAILED')),

    -- Transaction details
    tx_hash VARCHAR(255),
    block_number BIGINT,

    -- Gas savings
    gas_cost_saved NUMERIC(78, 0),
    gas_cost_actual NUMERIC(78, 0),
    savings_percentage NUMERIC(5, 2),

    -- Submitter
    submitter_address VARCHAR(255),

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    submitted_at TIMESTAMP,
    confirmed_at TIMESTAMP,

    -- Metadata
    metadata JSONB,

    CONSTRAINT different_chains_batch CHECK (source_chain_id != dest_chain_id)
);

CREATE INDEX idx_batches_status ON batches(status);
CREATE INDEX idx_batches_merkle_root ON batches(merkle_root);
CREATE INDEX idx_batches_source_chain ON batches(source_chain_id);
CREATE INDEX idx_batches_dest_chain ON batches(dest_chain_id);
CREATE INDEX idx_batches_created_at ON batches(created_at);
CREATE INDEX idx_batches_confirmed_at ON batches(confirmed_at);

-- ============================================================
-- BATCH_MESSAGES TABLE (Junction table)
-- ============================================================
CREATE TABLE IF NOT EXISTS batch_messages (
    id SERIAL PRIMARY KEY,
    batch_id VARCHAR(100) NOT NULL REFERENCES batches(id) ON DELETE CASCADE,
    message_id VARCHAR(100) NOT NULL REFERENCES messages(id) ON DELETE CASCADE,

    -- Merkle proof for this message
    merkle_proof JSONB,
    proof_index INTEGER NOT NULL,

    -- Processing status within batch
    settled BOOLEAN NOT NULL DEFAULT FALSE,
    settled_at TIMESTAMP,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(batch_id, message_id)
);

CREATE INDEX idx_batch_messages_batch ON batch_messages(batch_id);
CREATE INDEX idx_batch_messages_message ON batch_messages(message_id);
CREATE INDEX idx_batch_messages_settled ON batch_messages(settled);

-- ============================================================
-- BATCH_STATS TABLE (Aggregated statistics)
-- ============================================================
CREATE TABLE IF NOT EXISTS batch_stats (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL UNIQUE,

    -- Counts
    batches_submitted INTEGER NOT NULL DEFAULT 0,
    batches_confirmed INTEGER NOT NULL DEFAULT 0,
    total_messages_batched INTEGER NOT NULL DEFAULT 0,

    -- Savings
    total_gas_saved NUMERIC(78, 0) NOT NULL DEFAULT 0,
    total_gas_cost NUMERIC(78, 0) NOT NULL DEFAULT 0,
    average_savings_percentage NUMERIC(5, 2),

    -- Efficiency
    average_batch_size NUMERIC(10, 2),
    average_wait_time_seconds INTEGER,

    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_batch_stats_date ON batch_stats(date);

-- ============================================================
-- VIEWS
-- ============================================================

-- View for pending batches with message count
CREATE OR REPLACE VIEW pending_batches_summary AS
SELECT
    b.id,
    b.merkle_root,
    b.source_chain_id,
    b.dest_chain_id,
    COUNT(bm.message_id) as message_count,
    b.total_value,
    b.status,
    b.created_at,
    EXTRACT(EPOCH FROM (NOW() - b.created_at)) as age_seconds
FROM batches b
LEFT JOIN batch_messages bm ON b.id = bm.batch_id
WHERE b.status IN ('PENDING', 'READY')
GROUP BY b.id, b.merkle_root, b.source_chain_id, b.dest_chain_id,
         b.total_value, b.status, b.created_at;

-- View for batch efficiency metrics
CREATE OR REPLACE VIEW batch_efficiency_metrics AS
SELECT
    b.id,
    b.message_count,
    b.gas_cost_saved,
    b.savings_percentage,
    CASE
        WHEN b.message_count > 0 THEN b.gas_cost_saved / b.message_count
        ELSE 0
    END as gas_saved_per_message,
    b.created_at,
    b.confirmed_at,
    EXTRACT(EPOCH FROM (b.confirmed_at - b.created_at)) as processing_time_seconds
FROM batches b
WHERE b.status = 'CONFIRMED';

-- ============================================================
-- FUNCTIONS
-- ============================================================

-- Function to update batch stats
CREATE OR REPLACE FUNCTION update_batch_stats()
RETURNS TRIGGER AS $$
BEGIN
    -- Update daily stats when batch is confirmed
    IF NEW.status = 'CONFIRMED' AND (OLD.status IS NULL OR OLD.status != 'CONFIRMED') THEN
        INSERT INTO batch_stats (
            date,
            batches_confirmed,
            total_messages_batched,
            total_gas_saved,
            total_gas_cost
        ) VALUES (
            CURRENT_DATE,
            1,
            NEW.message_count,
            COALESCE(NEW.gas_cost_saved, 0),
            COALESCE(NEW.gas_cost_actual, 0)
        )
        ON CONFLICT (date) DO UPDATE SET
            batches_confirmed = batch_stats.batches_confirmed + 1,
            total_messages_batched = batch_stats.total_messages_batched + NEW.message_count,
            total_gas_saved = batch_stats.total_gas_saved + COALESCE(NEW.gas_cost_saved, 0),
            total_gas_cost = batch_stats.total_gas_cost + COALESCE(NEW.gas_cost_actual, 0),
            updated_at = NOW();
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update stats
CREATE TRIGGER update_batch_stats_trigger
    AFTER INSERT OR UPDATE ON batches
    FOR EACH ROW
    EXECUTE FUNCTION update_batch_stats();
