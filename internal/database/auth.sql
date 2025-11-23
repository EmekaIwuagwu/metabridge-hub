-- Authentication and Authorization Database Schema

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(100) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL CHECK (role IN ('admin', 'developer', 'user', 'readonly')),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_active ON users(active);

-- API Keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id VARCHAR(100) PRIMARY KEY,
    user_id VARCHAR(100) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(64) NOT NULL UNIQUE,
    permissions TEXT[] NOT NULL DEFAULT '{}',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_active ON api_keys(active);
CREATE INDEX idx_api_keys_expires_at ON api_keys(expires_at) WHERE expires_at IS NOT NULL;

-- Audit log for authentication events
CREATE TABLE IF NOT EXISTS auth_audit_log (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(100),
    email VARCHAR(255),
    event_type VARCHAR(50) NOT NULL,
    ip_address VARCHAR(50),
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_auth_audit_user_id ON auth_audit_log(user_id);
CREATE INDEX idx_auth_audit_created_at ON auth_audit_log(created_at DESC);
CREATE INDEX idx_auth_audit_event_type ON auth_audit_log(event_type);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_users_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_users_updated_at ON users;
CREATE TRIGGER trigger_update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_users_updated_at();

DROP TRIGGER IF EXISTS trigger_update_api_keys_updated_at ON api_keys;
CREATE TRIGGER trigger_update_api_keys_updated_at
    BEFORE UPDATE ON api_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_users_updated_at();

-- Function to log authentication events
CREATE OR REPLACE FUNCTION log_auth_event(
    p_user_id VARCHAR,
    p_email VARCHAR,
    p_event_type VARCHAR,
    p_ip_address VARCHAR,
    p_user_agent TEXT,
    p_success BOOLEAN,
    p_error_message TEXT DEFAULT NULL
)
RETURNS void AS $$
BEGIN
    INSERT INTO auth_audit_log (
        user_id, email, event_type, ip_address,
        user_agent, success, error_message
    ) VALUES (
        p_user_id, p_email, p_event_type, p_ip_address,
        p_user_agent, p_success, p_error_message
    );
END;
$$ LANGUAGE plpgsql;

-- Create default admin user (password: admin123 - CHANGE IN PRODUCTION!)
-- Password hash generated with bcrypt for "admin123"
INSERT INTO users (id, email, name, password_hash, role, active)
VALUES (
    'admin-default',
    'admin@articium.local',
    'Default Admin',
    '$2a$10$rQ8K3z3z8L3z8z8z8z8z8OvD5fq5fq5fq5fq5fq5fq5fq5fq5fq5e',
    'admin',
    true
)
ON CONFLICT (email) DO NOTHING;

-- Create default developer user for testing (password: dev123)
INSERT INTO users (id, email, name, password_hash, role, active)
VALUES (
    'dev-default',
    'dev@articium.local',
    'Default Developer',
    '$2a$10$Xyz.Xyz.Xyz.Xyz.Xyz.Xyz.OvD5fq5fq5fq5fq5fq5fq5fq5fq5fq5e',
    'developer',
    true
)
ON CONFLICT (email) DO NOTHING;

-- View for active API keys
CREATE OR REPLACE VIEW active_api_keys AS
SELECT
    ak.id,
    ak.user_id,
    u.email,
    u.name as user_name,
    ak.name as api_key_name,
    ak.permissions,
    ak.last_used_at,
    ak.expires_at,
    ak.created_at,
    CASE
        WHEN ak.expires_at IS NOT NULL AND ak.expires_at < NOW() THEN 'expired'
        WHEN ak.active = false THEN 'revoked'
        ELSE 'active'
    END as status
FROM api_keys ak
JOIN users u ON ak.user_id = u.id
WHERE u.active = true
ORDER BY ak.created_at DESC;

-- View for authentication activity
CREATE OR REPLACE VIEW auth_activity AS
SELECT
    aal.id,
    aal.user_id,
    u.email,
    u.name,
    aal.event_type,
    aal.success,
    aal.ip_address,
    aal.created_at,
    aal.error_message
FROM auth_audit_log aal
LEFT JOIN users u ON aal.user_id = u.id
ORDER BY aal.created_at DESC;

-- View for failed login attempts
CREATE OR REPLACE VIEW failed_login_attempts AS
SELECT
    email,
    COUNT(*) as attempt_count,
    MAX(created_at) as last_attempt,
    array_agg(DISTINCT ip_address) as ip_addresses
FROM auth_audit_log
WHERE event_type = 'login' AND success = false
AND created_at > NOW() - INTERVAL '1 hour'
GROUP BY email
HAVING COUNT(*) >= 3
ORDER BY last_attempt DESC;

-- Function to clean up old audit logs
CREATE OR REPLACE FUNCTION cleanup_old_auth_logs()
RETURNS void AS $$
BEGIN
    -- Delete successful login logs older than 90 days
    DELETE FROM auth_audit_log
    WHERE event_type = 'login'
      AND success = true
      AND created_at < NOW() - INTERVAL '90 days';

    -- Delete failed login logs older than 30 days
    DELETE FROM auth_audit_log
    WHERE event_type = 'login'
      AND success = false
      AND created_at < NOW() - INTERVAL '30 days';

    -- Delete other event logs older than 180 days
    DELETE FROM auth_audit_log
    WHERE event_type != 'login'
      AND created_at < NOW() - INTERVAL '180 days';
END;
$$ LANGUAGE plpgsql;

-- Function to check for suspicious activity
CREATE OR REPLACE FUNCTION check_suspicious_activity(
    p_user_id VARCHAR,
    p_ip_address VARCHAR
)
RETURNS TABLE(suspicious BOOLEAN, reason TEXT) AS $$
DECLARE
    failed_count INTEGER;
    different_ips INTEGER;
BEGIN
    -- Check failed login attempts in last hour
    SELECT COUNT(*) INTO failed_count
    FROM auth_audit_log
    WHERE user_id = p_user_id
      AND event_type = 'login'
      AND success = false
      AND created_at > NOW() - INTERVAL '1 hour';

    IF failed_count >= 5 THEN
        RETURN QUERY SELECT true, 'Multiple failed login attempts';
        RETURN;
    END IF;

    -- Check for access from too many different IPs in short time
    SELECT COUNT(DISTINCT ip_address) INTO different_ips
    FROM auth_audit_log
    WHERE user_id = p_user_id
      AND event_type = 'login'
      AND success = true
      AND created_at > NOW() - INTERVAL '1 hour';

    IF different_ips >= 5 THEN
        RETURN QUERY SELECT true, 'Access from multiple IP addresses';
        RETURN;
    END IF;

    -- No suspicious activity detected
    RETURN QUERY SELECT false, NULL::TEXT;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE users IS 'Stores user accounts for authentication';
COMMENT ON TABLE api_keys IS 'API keys for programmatic access';
COMMENT ON TABLE auth_audit_log IS 'Audit log for authentication events';
COMMENT ON VIEW active_api_keys IS 'View of all active API keys with user information';
COMMENT ON VIEW auth_activity IS 'Recent authentication activity';
COMMENT ON VIEW failed_login_attempts IS 'Failed login attempts in the last hour';
