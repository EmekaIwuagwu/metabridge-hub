-- Webhooks and Tracking Database Schema

-- Webhooks table
CREATE TABLE IF NOT EXISTS webhooks (
    id VARCHAR(100) PRIMARY KEY,
    url TEXT NOT NULL,
    secret VARCHAR(255) NOT NULL,
    events TEXT[] NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('ACTIVE', 'PAUSED', 'DISABLED', 'FAILED')),
    description TEXT,
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMP,
    fail_count INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    source_chains TEXT[],
    dest_chains TEXT[],
    min_amount VARCHAR(100),
    max_amount VARCHAR(100)
);

CREATE INDEX idx_webhooks_created_by ON webhooks(created_by);
CREATE INDEX idx_webhooks_status ON webhooks(status);
CREATE INDEX idx_webhooks_events ON webhooks USING GIN(events);

-- Webhook events table
CREATE TABLE IF NOT EXISTS webhook_events (
    id VARCHAR(100) PRIMARY KEY,
    webhook_id VARCHAR(100) NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    signature VARCHAR(255),
    delivery_url TEXT NOT NULL
);

CREATE INDEX idx_webhook_events_webhook_id ON webhook_events(webhook_id);
CREATE INDEX idx_webhook_events_event_type ON webhook_events(event_type);
CREATE INDEX idx_webhook_events_timestamp ON webhook_events(timestamp DESC);

-- Webhook delivery attempts table
CREATE TABLE IF NOT EXISTS webhook_attempts (
    id VARCHAR(100) PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL REFERENCES webhook_events(id) ON DELETE CASCADE,
    webhook_id VARCHAR(100) NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    attempt_number INTEGER NOT NULL,
    status_code INTEGER,
    response_body TEXT,
    error_message TEXT,
    success BOOLEAN NOT NULL DEFAULT FALSE,
    duration_ms BIGINT,
    attempted_at TIMESTAMP NOT NULL DEFAULT NOW(),
    next_retry_at TIMESTAMP
);

CREATE INDEX idx_webhook_attempts_event_id ON webhook_attempts(event_id);
CREATE INDEX idx_webhook_attempts_webhook_id ON webhook_attempts(webhook_id);
CREATE INDEX idx_webhook_attempts_success ON webhook_attempts(success);
CREATE INDEX idx_webhook_attempts_next_retry ON webhook_attempts(next_retry_at) WHERE next_retry_at IS NOT NULL;

-- Message timeline events table
CREATE TABLE IF NOT EXISTS message_timeline_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id VARCHAR(100) NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    description TEXT NOT NULL,
    tx_hash VARCHAR(100),
    block_number BIGINT,
    chain_id VARCHAR(50),
    metadata JSONB
);

CREATE INDEX idx_timeline_message_id ON message_timeline_events(message_id);
CREATE INDEX idx_timeline_timestamp ON message_timeline_events(timestamp DESC);
CREATE INDEX idx_timeline_event_type ON message_timeline_events(event_type);

-- Webhook statistics view
CREATE OR REPLACE VIEW webhook_stats AS
SELECT
    w.id,
    w.url,
    w.status,
    w.created_by,
    w.success_count,
    w.fail_count,
    CASE
        WHEN (w.success_count + w.fail_count) > 0
        THEN ROUND((w.success_count::NUMERIC / (w.success_count + w.fail_count)) * 100, 2)
        ELSE 0
    END as success_rate,
    COUNT(DISTINCT we.id) as total_events,
    COUNT(DISTINCT CASE WHEN wa.success = true THEN wa.id END) as successful_deliveries,
    COUNT(DISTINCT CASE WHEN wa.success = false THEN wa.id END) as failed_deliveries,
    AVG(CASE WHEN wa.success = true THEN wa.duration_ms END) as avg_delivery_time_ms,
    MAX(wa.attempted_at) as last_delivery_attempt
FROM webhooks w
LEFT JOIN webhook_events we ON w.id = we.webhook_id
LEFT JOIN webhook_attempts wa ON we.id = wa.event_id
GROUP BY w.id, w.url, w.status, w.created_by, w.success_count, w.fail_count;

-- Recent webhook deliveries view
CREATE OR REPLACE VIEW recent_webhook_deliveries AS
SELECT
    wa.id as attempt_id,
    wa.webhook_id,
    w.url,
    we.event_type,
    wa.status_code,
    wa.success,
    wa.duration_ms,
    wa.attempted_at,
    wa.error_message
FROM webhook_attempts wa
JOIN webhook_events we ON wa.event_id = we.id
JOIN webhooks w ON wa.webhook_id = w.id
WHERE wa.attempted_at > NOW() - INTERVAL '24 hours'
ORDER BY wa.attempted_at DESC;

-- Failed webhook deliveries pending retry
CREATE OR REPLACE VIEW pending_webhook_retries AS
SELECT
    wa.id as attempt_id,
    wa.event_id,
    wa.webhook_id,
    w.url,
    we.event_type,
    wa.attempt_number,
    wa.next_retry_at,
    wa.error_message,
    wa.attempted_at as last_attempt
FROM webhook_attempts wa
JOIN webhook_events we ON wa.event_id = we.id
JOIN webhooks w ON wa.webhook_id = w.id
WHERE wa.success = false
  AND wa.next_retry_at IS NOT NULL
  AND wa.next_retry_at <= NOW()
ORDER BY wa.next_retry_at ASC;

-- Message tracking summary view
CREATE OR REPLACE VIEW message_tracking_summary AS
SELECT
    m.id,
    m.source_chain,
    m.dest_chain,
    m.sender,
    m.recipient,
    m.amount,
    m.status,
    m.created_at,
    m.updated_at,
    m.source_tx_hash,
    m.dest_tx_hash,
    COUNT(mte.id) as event_count,
    MAX(mte.timestamp) as last_event_time,
    EXTRACT(EPOCH FROM (COALESCE(m.confirmed_at, NOW()) - m.created_at)) as processing_time_seconds
FROM messages m
LEFT JOIN message_timeline_events mte ON m.id = mte.message_id
GROUP BY m.id, m.source_chain, m.dest_chain, m.sender, m.recipient,
         m.amount, m.status, m.created_at, m.updated_at,
         m.source_tx_hash, m.dest_tx_hash, m.confirmed_at;

-- Function to auto-update webhook updated_at timestamp
CREATE OR REPLACE FUNCTION update_webhook_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for auto-updating webhook updated_at
DROP TRIGGER IF EXISTS trigger_update_webhook_updated_at ON webhooks;
CREATE TRIGGER trigger_update_webhook_updated_at
    BEFORE UPDATE ON webhooks
    FOR EACH ROW
    EXECUTE FUNCTION update_webhook_updated_at();

-- Function to automatically create timeline events for message status changes
CREATE OR REPLACE FUNCTION create_message_timeline_event()
RETURNS TRIGGER AS $$
BEGIN
    -- Only create timeline events when status changes
    IF (TG_OP = 'UPDATE' AND NEW.status != OLD.status) OR TG_OP = 'INSERT' THEN
        INSERT INTO message_timeline_events (
            message_id,
            event_type,
            timestamp,
            description,
            tx_hash,
            chain_id
        ) VALUES (
            NEW.id,
            'status_change',
            NOW(),
            CASE
                WHEN TG_OP = 'INSERT' THEN 'Message created with status: ' || NEW.status
                ELSE 'Status changed from ' || OLD.status || ' to ' || NEW.status
            END,
            COALESCE(NEW.dest_tx_hash, NEW.source_tx_hash),
            NEW.dest_chain
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for auto-creating timeline events on message changes
DROP TRIGGER IF EXISTS trigger_message_timeline_event ON messages;
CREATE TRIGGER trigger_message_timeline_event
    AFTER INSERT OR UPDATE ON messages
    FOR EACH ROW
    EXECUTE FUNCTION create_message_timeline_event();

-- Function to clean up old webhook data
CREATE OR REPLACE FUNCTION cleanup_old_webhook_data()
RETURNS void AS $$
BEGIN
    -- Delete webhook events older than 90 days
    DELETE FROM webhook_events
    WHERE timestamp < NOW() - INTERVAL '90 days';

    -- Delete successful webhook attempts older than 30 days
    DELETE FROM webhook_attempts
    WHERE success = true
      AND attempted_at < NOW() - INTERVAL '30 days';

    -- Delete failed webhook attempts older than 7 days (keep recent failures for analysis)
    DELETE FROM webhook_attempts
    WHERE success = false
      AND next_retry_at IS NULL  -- Already exhausted retries
      AND attempted_at < NOW() - INTERVAL '7 days';

    -- Delete timeline events older than 180 days
    DELETE FROM message_timeline_events
    WHERE timestamp < NOW() - INTERVAL '180 days';
END;
$$ LANGUAGE plpgsql;

-- Create a scheduled job to run cleanup (requires pg_cron extension)
-- Uncomment if pg_cron is available:
-- SELECT cron.schedule('cleanup-webhook-data', '0 2 * * *', 'SELECT cleanup_old_webhook_data()');

-- Insert initial timeline events for existing messages
INSERT INTO message_timeline_events (message_id, event_type, timestamp, description, tx_hash, chain_id)
SELECT
    id,
    'message_created',
    created_at,
    'Message created',
    source_tx_hash,
    source_chain
FROM messages
WHERE id NOT IN (SELECT DISTINCT message_id FROM message_timeline_events)
ON CONFLICT DO NOTHING;

-- Webhook delivery metrics (daily aggregation)
CREATE TABLE IF NOT EXISTS webhook_delivery_metrics (
    date DATE NOT NULL,
    webhook_id VARCHAR(100) NOT NULL,
    total_events INTEGER NOT NULL DEFAULT 0,
    successful_deliveries INTEGER NOT NULL DEFAULT 0,
    failed_deliveries INTEGER NOT NULL DEFAULT 0,
    avg_delivery_time_ms NUMERIC(10, 2),
    PRIMARY KEY (date, webhook_id),
    FOREIGN KEY (webhook_id) REFERENCES webhooks(id) ON DELETE CASCADE
);

CREATE INDEX idx_webhook_metrics_date ON webhook_delivery_metrics(date DESC);
CREATE INDEX idx_webhook_metrics_webhook_id ON webhook_delivery_metrics(webhook_id);

-- Function to aggregate webhook metrics daily
CREATE OR REPLACE FUNCTION aggregate_webhook_metrics()
RETURNS void AS $$
BEGIN
    INSERT INTO webhook_delivery_metrics (
        date,
        webhook_id,
        total_events,
        successful_deliveries,
        failed_deliveries,
        avg_delivery_time_ms
    )
    SELECT
        DATE(wa.attempted_at) as date,
        wa.webhook_id,
        COUNT(*) as total_events,
        COUNT(*) FILTER (WHERE wa.success = true) as successful_deliveries,
        COUNT(*) FILTER (WHERE wa.success = false) as failed_deliveries,
        AVG(wa.duration_ms) FILTER (WHERE wa.success = true) as avg_delivery_time_ms
    FROM webhook_attempts wa
    WHERE DATE(wa.attempted_at) = CURRENT_DATE - INTERVAL '1 day'
    GROUP BY DATE(wa.attempted_at), wa.webhook_id
    ON CONFLICT (date, webhook_id) DO UPDATE SET
        total_events = EXCLUDED.total_events,
        successful_deliveries = EXCLUDED.successful_deliveries,
        failed_deliveries = EXCLUDED.failed_deliveries,
        avg_delivery_time_ms = EXCLUDED.avg_delivery_time_ms;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE webhooks IS 'Stores webhook registrations for real-time event notifications';
COMMENT ON TABLE webhook_events IS 'Stores individual webhook events to be delivered';
COMMENT ON TABLE webhook_attempts IS 'Tracks delivery attempts for webhook events with retry logic';
COMMENT ON TABLE message_timeline_events IS 'Timeline of events for cross-chain messages';
COMMENT ON TABLE webhook_delivery_metrics IS 'Daily aggregated metrics for webhook deliveries';
COMMENT ON VIEW webhook_stats IS 'Statistics and performance metrics for each webhook';
COMMENT ON VIEW message_tracking_summary IS 'Summary view for message tracking with event counts';
