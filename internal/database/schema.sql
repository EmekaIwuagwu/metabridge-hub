-- Articium Database Schema
-- Supports multi-chain bridge operations for EVM, Solana, and NEAR

-- ============================================================
-- CHAINS TABLE
-- ============================================================
CREATE TABLE IF NOT EXISTS chains (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    chain_type VARCHAR(20) NOT NULL CHECK (chain_type IN ('EVM', 'SOLANA', 'NEAR')),
    chain_id VARCHAR(50),
    network_id VARCHAR(50),
    environment VARCHAR(20) NOT NULL CHECK (environment IN ('development', 'testnet', 'mainnet')),
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chains_name ON chains(name);
CREATE INDEX idx_chains_chain_type ON chains(chain_type);
CREATE INDEX idx_chains_environment ON chains(environment);

-- ============================================================
-- MESSAGES TABLE
-- ============================================================
CREATE TABLE IF NOT EXISTS messages (
    id VARCHAR(100) PRIMARY KEY,
    message_type VARCHAR(50) NOT NULL CHECK (message_type IN ('TOKEN_TRANSFER', 'NFT_TRANSFER', 'GENERIC_MESSAGE')),
    nonce BIGINT NOT NULL,

    -- Source chain information
    source_chain_id INTEGER NOT NULL REFERENCES chains(id),
    source_tx_hash VARCHAR(255) NOT NULL,
    source_block BIGINT NOT NULL,

    -- Destination chain information
    dest_chain_id INTEGER NOT NULL REFERENCES chains(id),
    dest_tx_hash VARCHAR(255),
    dest_block BIGINT,

    -- Addresses (stored as strings to support all formats)
    sender_address VARCHAR(255) NOT NULL,
    sender_chain_type VARCHAR(20) NOT NULL,
    recipient_address VARCHAR(255) NOT NULL,
    recipient_chain_type VARCHAR(20) NOT NULL,

    -- Payload
    payload JSONB NOT NULL,
    metadata JSONB,

    -- Processing state
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'VALIDATING', 'PROCESSING', 'COMPLETED', 'FAILED', 'RETRYING')),
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    required_signatures INTEGER NOT NULL DEFAULT 2,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP,

    CONSTRAINT different_chains CHECK (source_chain_id != dest_chain_id)
);

CREATE INDEX idx_messages_status ON messages(status);
CREATE INDEX idx_messages_source_chain ON messages(source_chain_id);
CREATE INDEX idx_messages_dest_chain ON messages(dest_chain_id);
CREATE INDEX idx_messages_source_tx_hash ON messages(source_tx_hash);
CREATE INDEX idx_messages_created_at ON messages(created_at);
CREATE INDEX idx_messages_type ON messages(message_type);
CREATE INDEX idx_messages_sender ON messages(sender_address);
CREATE INDEX idx_messages_recipient ON messages(recipient_address);

-- ============================================================
-- VALIDATOR SIGNATURES TABLE
-- ============================================================
CREATE TABLE IF NOT EXISTS validator_signatures (
    id SERIAL PRIMARY KEY,
    message_id VARCHAR(100) NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    validator_address VARCHAR(255) NOT NULL,
    signature BYTEA NOT NULL,
    signature_scheme VARCHAR(20) NOT NULL CHECK (signature_scheme IN ('ECDSA', 'Ed25519')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(message_id, validator_address)
);

CREATE INDEX idx_validator_signatures_message ON validator_signatures(message_id);
CREATE INDEX idx_validator_signatures_validator ON validator_signatures(validator_address);

-- ============================================================
-- VALIDATORS TABLE
-- ============================================================
CREATE TABLE IF NOT EXISTS validators (
    id SERIAL PRIMARY KEY,
    address VARCHAR(255) NOT NULL,
    chain_type VARCHAR(20) NOT NULL CHECK (chain_type IN ('EVM', 'SOLANA', 'NEAR')),
    public_key BYTEA,
    active BOOLEAN NOT NULL DEFAULT true,
    environment VARCHAR(20) NOT NULL CHECK (environment IN ('development', 'testnet', 'mainnet')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(address, chain_type, environment)
);

CREATE INDEX idx_validators_address ON validators(address);
CREATE INDEX idx_validators_active ON validators(active);
CREATE INDEX idx_validators_chain_type ON validators(chain_type);
CREATE INDEX idx_validators_environment ON validators(environment);

-- ============================================================
-- TRANSACTIONS TABLE (for tracking all blockchain transactions)
-- ============================================================
CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    tx_hash VARCHAR(255) NOT NULL,
    chain_id INTEGER NOT NULL REFERENCES chains(id),
    block_number BIGINT,
    from_address VARCHAR(255),
    to_address VARCHAR(255),
    amount VARCHAR(100),
    gas_used BIGINT,
    status VARCHAR(50) NOT NULL CHECK (status IN ('PENDING', 'CONFIRMED', 'FAILED', 'FINALIZED')),
    tx_type VARCHAR(50) CHECK (tx_type IN ('LOCK', 'RELEASE', 'BURN', 'MINT')),
    message_id VARCHAR(100) REFERENCES messages(id),
    error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    confirmed_at TIMESTAMP,
    finalized_at TIMESTAMP,

    UNIQUE(tx_hash, chain_id)
);

CREATE INDEX idx_transactions_hash ON transactions(tx_hash);
CREATE INDEX idx_transactions_chain ON transactions(chain_id);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_message ON transactions(message_id);
CREATE INDEX idx_transactions_created_at ON transactions(created_at);

-- ============================================================
-- TOKEN MAPPINGS TABLE
-- ============================================================
CREATE TABLE IF NOT EXISTS token_mappings (
    id SERIAL PRIMARY KEY,
    source_chain_id INTEGER NOT NULL REFERENCES chains(id),
    source_token_address VARCHAR(255) NOT NULL,
    dest_chain_id INTEGER NOT NULL REFERENCES chains(id),
    dest_token_address VARCHAR(255) NOT NULL,
    token_standard VARCHAR(50) NOT NULL, -- ERC20, SPL, NEP141, etc.
    decimals INTEGER NOT NULL,
    symbol VARCHAR(20),
    name VARCHAR(100),
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(source_chain_id, source_token_address, dest_chain_id)
);

CREATE INDEX idx_token_mappings_source ON token_mappings(source_chain_id, source_token_address);
CREATE INDEX idx_token_mappings_dest ON token_mappings(dest_chain_id, dest_token_address);
CREATE INDEX idx_token_mappings_enabled ON token_mappings(enabled);

-- ============================================================
-- BRIDGE STATISTICS TABLE
-- ============================================================
CREATE TABLE IF NOT EXISTS bridge_stats (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    chain_id INTEGER NOT NULL REFERENCES chains(id),
    total_messages BIGINT NOT NULL DEFAULT 0,
    completed_messages BIGINT NOT NULL DEFAULT 0,
    failed_messages BIGINT NOT NULL DEFAULT 0,
    total_volume_usd NUMERIC(30, 6) NOT NULL DEFAULT 0,
    unique_users BIGINT NOT NULL DEFAULT 0,
    avg_processing_time_seconds INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(date, chain_id)
);

CREATE INDEX idx_bridge_stats_date ON bridge_stats(date);
CREATE INDEX idx_bridge_stats_chain ON bridge_stats(chain_id);

-- ============================================================
-- RATE LIMITS TABLE
-- ============================================================
CREATE TABLE IF NOT EXISTS rate_limits (
    id SERIAL PRIMARY KEY,
    address VARCHAR(255) NOT NULL,
    chain_id INTEGER NOT NULL REFERENCES chains(id),
    hourly_count INTEGER NOT NULL DEFAULT 0,
    daily_count INTEGER NOT NULL DEFAULT 0,
    hourly_volume_usd NUMERIC(30, 6) NOT NULL DEFAULT 0,
    daily_volume_usd NUMERIC(30, 6) NOT NULL DEFAULT 0,
    last_hourly_reset TIMESTAMP NOT NULL DEFAULT NOW(),
    last_daily_reset TIMESTAMP NOT NULL DEFAULT NOW(),
    blocked BOOLEAN NOT NULL DEFAULT false,
    block_reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(address, chain_id)
);

CREATE INDEX idx_rate_limits_address ON rate_limits(address);
CREATE INDEX idx_rate_limits_blocked ON rate_limits(blocked);

-- ============================================================
-- AUDIT LOG TABLE
-- ============================================================
CREATE TABLE IF NOT EXISTS audit_log (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,
    actor VARCHAR(255),
    message_id VARCHAR(100) REFERENCES messages(id),
    chain_id INTEGER REFERENCES chains(id),
    details JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_event_type ON audit_log(event_type);
CREATE INDEX idx_audit_log_message ON audit_log(message_id);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);

-- ============================================================
-- EMERGENCY PAUSE TABLE
-- ============================================================
CREATE TABLE IF NOT EXISTS emergency_pause (
    id SERIAL PRIMARY KEY,
    chain_id INTEGER REFERENCES chains(id),
    is_paused BOOLEAN NOT NULL DEFAULT false,
    reason TEXT,
    paused_by VARCHAR(255),
    paused_at TIMESTAMP,
    resumed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_emergency_pause_chain ON emergency_pause(chain_id);
CREATE INDEX idx_emergency_pause_status ON emergency_pause(is_paused);

-- ============================================================
-- FUNCTIONS AND TRIGGERS
-- ============================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_chains_updated_at BEFORE UPDATE ON chains FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_messages_updated_at BEFORE UPDATE ON messages FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_validators_updated_at BEFORE UPDATE ON validators FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_token_mappings_updated_at BEFORE UPDATE ON token_mappings FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_bridge_stats_updated_at BEFORE UPDATE ON bridge_stats FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_rate_limits_updated_at BEFORE UPDATE ON rate_limits FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_emergency_pause_updated_at BEFORE UPDATE ON emergency_pause FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- VIEWS
-- ============================================================

-- View for pending messages
CREATE OR REPLACE VIEW pending_messages AS
SELECT
    m.*,
    sc.name as source_chain_name,
    sc.chain_type as source_chain_type,
    dc.name as dest_chain_name,
    dc.chain_type as dest_chain_type,
    COUNT(vs.id) as signature_count
FROM messages m
JOIN chains sc ON m.source_chain_id = sc.id
JOIN chains dc ON m.dest_chain_id = dc.id
LEFT JOIN validator_signatures vs ON m.id = vs.message_id
WHERE m.status IN ('PENDING', 'VALIDATING', 'PROCESSING')
GROUP BY m.id, sc.name, sc.chain_type, dc.name, dc.chain_type;

-- View for daily statistics
CREATE OR REPLACE VIEW daily_statistics AS
SELECT
    date,
    c.name as chain_name,
    c.chain_type,
    c.environment,
    total_messages,
    completed_messages,
    failed_messages,
    total_volume_usd,
    unique_users,
    CASE
        WHEN total_messages > 0 THEN ROUND((completed_messages::NUMERIC / total_messages) * 100, 2)
        ELSE 0
    END as success_rate
FROM bridge_stats bs
JOIN chains c ON bs.chain_id = c.id
ORDER BY date DESC, chain_name;

-- ============================================================
-- SEED DATA (for development/testing)
-- ============================================================

-- Insert chain configurations
INSERT INTO chains (name, chain_type, chain_id, network_id, environment, enabled) VALUES
-- Testnet chains
('polygon-amoy', 'EVM', '80002', NULL, 'testnet', true),
('bnb-testnet', 'EVM', '97', NULL, 'testnet', true),
('avalanche-fuji', 'EVM', '43113', NULL, 'testnet', true),
('ethereum-sepolia', 'EVM', '11155111', NULL, 'testnet', true),
('solana-devnet', 'SOLANA', NULL, 'devnet', 'testnet', true),
('near-testnet', 'NEAR', NULL, 'testnet', 'testnet', true),

-- Mainnet chains
('polygon-mainnet', 'EVM', '137', NULL, 'mainnet', false), -- Start disabled
('bnb-mainnet', 'EVM', '56', NULL, 'mainnet', false),
('avalanche-mainnet', 'EVM', '43114', NULL, 'mainnet', false),
('ethereum-mainnet', 'EVM', '1', NULL, 'mainnet', false),
('solana-mainnet', 'SOLANA', NULL, 'mainnet-beta', 'mainnet', false),
('near-mainnet', 'NEAR', NULL, 'mainnet', 'mainnet', false)
ON CONFLICT (name) DO NOTHING;
