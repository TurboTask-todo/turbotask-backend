-- =====================================================
-- ENHANCED AUTHENTICATION SCHEMA
-- High-Performance, Scalable, Professional Design
-- =====================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =====================================================
-- CORE USER MANAGEMENT
-- =====================================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(320) NOT NULL, -- RFC 5322 compliant max length
    username VARCHAR(50) UNIQUE,
    email_verified BOOLEAN DEFAULT false NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    avatar_url TEXT,
    timezone VARCHAR(50) DEFAULT 'UTC' NOT NULL,
    locale VARCHAR(10) DEFAULT 'en-US' NOT NULL,
    preferences JSONB DEFAULT '{}' NOT NULL,
    status VARCHAR(20) DEFAULT 'active' NOT NULL 
        CHECK (status IN ('active', 'inactive', 'suspended', 'pending', 'deleted')),
    last_login_at TIMESTAMP WITH TIME ZONE,
    failed_login_attempts INTEGER DEFAULT 0 NOT NULL,
    account_locked_until TIMESTAMP WITH TIME ZONE,
    password_changed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE -- Soft delete implementation
);

-- Unique constraint for active users only (allows reuse of email/username after soft delete)
CREATE UNIQUE INDEX users_email_active_unique 
    ON users (email) 
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX users_username_active_unique 
    ON users (username) 
    WHERE deleted_at IS NULL AND username IS NOT NULL;

-- =====================================================
-- PASSWORD AUTHENTICATION
-- =====================================================

CREATE TABLE user_passwords (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    password_hash VARCHAR(255) NOT NULL, -- Argon2id recommended over bcrypt
    salt VARCHAR(64), -- 64 bytes for Argon2id
    algorithm VARCHAR(20) DEFAULT 'argon2id' NOT NULL,
    cost_params JSONB DEFAULT '{}', -- Store algorithm-specific parameters
    version INTEGER DEFAULT 1 NOT NULL, -- For password policy versioning
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE, -- For password expiry policies
    is_temporary BOOLEAN DEFAULT false NOT NULL,
    is_active BOOLEAN DEFAULT true NOT NULL
);

-- Ensure one active password per user
CREATE UNIQUE INDEX user_passwords_user_active_unique 
    ON user_passwords (user_id) 
    WHERE is_active = true;

-- =====================================================
-- OAUTH PROVIDERS
-- =====================================================

CREATE TABLE oauth_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL CHECK (provider IN (
        'google', 'github', 'facebook', 'twitter', 'apple', 
        'linkedin', 'microsoft', 'discord', 'slack', 'zoom',
        'spotify', 'twitch', 'reddit', 'gitlab', 'bitbucket'
    )),
    provider_user_id VARCHAR(255) NOT NULL, -- ID from the OAuth provider
    provider_email VARCHAR(320),
    provider_username VARCHAR(100),
    provider_data JSONB DEFAULT '{}' NOT NULL, -- Store additional provider data
    access_token TEXT, -- Encrypted OAuth access token
    refresh_token TEXT, -- Encrypted OAuth refresh token
    token_type VARCHAR(20) DEFAULT 'Bearer',
    scope TEXT[], -- OAuth scopes granted
    expires_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    UNIQUE(provider, provider_user_id),
    UNIQUE(user_id, provider) -- One account per provider per user
);

-- =====================================================
-- JWT REFRESH TOKENS
-- =====================================================

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE, -- SHA-256 hash (32 bytes = 64 hex chars)
    token_family UUID DEFAULT gen_random_uuid(), -- For token rotation detection
    device_info JSONB DEFAULT '{}' NOT NULL, -- Browser, OS, device type, etc.
    device_fingerprint VARCHAR(64), -- Unique device identifier
    ip_address INET,
    user_agent TEXT,
    location_data JSONB DEFAULT '{}', -- Geolocation info
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    usage_count INTEGER DEFAULT 0 NOT NULL,
    is_revoked BOOLEAN DEFAULT false NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_reason VARCHAR(50) CHECK (revoked_reason IN (
        'manual', 'suspicious_activity', 'password_change', 
        'logout', 'token_rotation', 'expired', 'admin_revoke'
    )),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- =====================================================
-- VERIFICATION TOKENS
-- =====================================================

CREATE TABLE verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE, -- SHA-256 hash
    token_type VARCHAR(30) NOT NULL CHECK (token_type IN (
        'email_verification', 'password_reset', 'phone_verification',
        'email_change', 'account_recovery', 'magic_link'
    )),
    target_value VARCHAR(320), -- Email or phone for verification
    attempts_count INTEGER DEFAULT 0 NOT NULL,
    max_attempts INTEGER DEFAULT 3 NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    ip_address INET,
    user_agent TEXT,
    is_active BOOLEAN DEFAULT true NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- =====================================================
-- SECURITY EVENTS & AUDIT LOG
-- =====================================================

CREATE TABLE user_security_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL, -- Keep logs even if user deleted
    session_id UUID, -- Link to session if applicable
    event_type VARCHAR(30) NOT NULL CHECK (event_type IN (
        'login_success', 'login_failed', 'logout', 'password_change',
        'password_reset', 'email_verification', 'phone_verification',
        'account_locked', 'account_unlocked', 'mfa_enabled', 'mfa_disabled',
        'api_key_created', 'api_key_revoked', 'oauth_linked', 'oauth_unlinked',
        'suspicious_activity', 'admin_action', 'token_refresh', 'session_expired'
    )),
    event_category VARCHAR(20) DEFAULT 'authentication' CHECK (event_category IN (
        'authentication', 'authorization', 'account_management', 
        'security', 'api_access', 'admin'
    )),
    ip_address INET,
    user_agent TEXT,
    device_info JSONB DEFAULT '{}',
    location_data JSONB DEFAULT '{}',
    success BOOLEAN NOT NULL,
    failure_reason VARCHAR(100),
    risk_score INTEGER DEFAULT 0 CHECK (risk_score BETWEEN 0 AND 100),
    metadata JSONB DEFAULT '{}' NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- =====================================================
-- TWO-FACTOR AUTHENTICATION
-- =====================================================
CREATE TABLE user_2fa (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    method VARCHAR(20) NOT NULL CHECK (method IN (
        'totp', 'sms', 'email', 'webauthn', 'hardware_key', 'backup_codes'
    )),
    label VARCHAR(100), -- User-friendly name for the 2FA method
    secret_encrypted TEXT, -- Encrypted TOTP secret or key data
    phone_number VARCHAR(20), -- For SMS 2FA (E.164 format)
    email_address VARCHAR(320), -- For email 2FA
    backup_codes_encrypted TEXT[], -- Encrypted backup codes
    device_data JSONB DEFAULT '{}', -- WebAuthn device info
    usage_count INTEGER DEFAULT 0 NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE,
    is_enabled BOOLEAN DEFAULT false NOT NULL,
    is_primary BOOLEAN DEFAULT false NOT NULL, -- Primary 2FA method
    verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    UNIQUE(user_id, method)
);

-- =====================================================
-- SESSION MANAGEMENT
-- =====================================================

CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_token_hash VARCHAR(64) NOT NULL UNIQUE, -- SHA-256 hash
    session_name VARCHAR(100), -- User-friendly session name
    device_info JSONB DEFAULT '{}' NOT NULL,
    device_fingerprint VARCHAR(64), -- Unique device identifier
    ip_address INET,
    user_agent TEXT,
    location_data JSONB DEFAULT '{}',
    mfa_verified BOOLEAN DEFAULT false NOT NULL,
    risk_score INTEGER DEFAULT 0 CHECK (risk_score BETWEEN 0 AND 100),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    activity_count INTEGER DEFAULT 0 NOT NULL,
    is_active BOOLEAN DEFAULT true NOT NULL,
    terminated_at TIMESTAMP WITH TIME ZONE,
    termination_reason VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- =====================================================
-- API KEYS FOR AI SERVICES
-- =====================================================

CREATE TABLE api_service_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_name VARCHAR(50) NOT NULL UNIQUE,
    service_display_name VARCHAR(100) NOT NULL,
    provider VARCHAR(50) NOT NULL, -- 'openai', 'anthropic', 'google', etc.
    api_endpoint TEXT,
    documentation_url TEXT,
    pricing_model VARCHAR(20) DEFAULT 'token-based',
    is_active BOOLEAN DEFAULT true NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- Pre-populate with popular AI services
INSERT INTO api_service_types (service_name, service_display_name, provider, api_endpoint) VALUES
('openai_gpt', 'ChatGPT / GPT Models', 'openai', 'https://api.openai.com/v1'),
('anthropic_claude', 'Claude (Anthropic)', 'anthropic', 'https://api.anthropic.com/v1'),
('google_gemini', 'Gemini (Google)', 'google', 'https://generativelanguage.googleapis.com/v1'),
('google_palm', 'PaLM (Google)', 'google', 'https://generativelanguage.googleapis.com/v1'),
('cohere_ai', 'Cohere AI', 'cohere', 'https://api.cohere.ai/v1'),
('huggingface', 'Hugging Face', 'huggingface', 'https://api-inference.huggingface.co'),
('replicate', 'Replicate', 'replicate', 'https://api.replicate.com/v1'),
('stability_ai', 'Stability AI', 'stability', 'https://api.stability.ai/v1');

CREATE TABLE user_api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    service_type_id UUID NOT NULL REFERENCES api_service_types(id) ON DELETE CASCADE,
    key_name VARCHAR(100), -- User-friendly name for the key
    key_hash VARCHAR(64) NOT NULL UNIQUE, -- SHA-256 hash of the actual key
    key_prefix VARCHAR(20), -- First few characters for identification
    permissions JSONB DEFAULT '{}' NOT NULL, -- Service-specific permissions
    usage_stats JSONB DEFAULT '{}' NOT NULL, -- Usage tracking
    rate_limit_per_hour INTEGER DEFAULT 1000,
    rate_limit_per_day INTEGER DEFAULT 10000,
    rate_limit_per_month INTEGER DEFAULT 100000,
    current_usage JSONB DEFAULT '{}' NOT NULL, -- Current period usage
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    usage_count INTEGER DEFAULT 0 NOT NULL,
    is_active BOOLEAN DEFAULT true NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    -- Ensure one API key per service per user
    UNIQUE(user_id, service_type_id)
);

-- =====================================================
-- PERFORMANCE INDEXES
-- =====================================================

-- Core user indexes
CREATE INDEX idx_users_email ON users (email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_username ON users (username) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_status ON users (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_created_at ON users (created_at);
CREATE INDEX idx_users_last_login ON users (last_login_at) WHERE last_login_at IS NOT NULL;
CREATE INDEX idx_users_account_locked ON users (account_locked_until) WHERE account_locked_until IS NOT NULL;

-- Password indexes
CREATE INDEX idx_user_passwords_user_id ON user_passwords (user_id);
CREATE INDEX idx_user_passwords_created_at ON user_passwords (created_at);

-- OAuth provider indexes
CREATE INDEX idx_oauth_providers_user_id ON oauth_providers (user_id);
CREATE INDEX idx_oauth_providers_provider ON oauth_providers (provider);
CREATE INDEX idx_oauth_providers_email ON oauth_providers (provider_email);

-- Refresh token indexes
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens (expires_at);
CREATE INDEX idx_refresh_tokens_family ON refresh_tokens (token_family);
CREATE INDEX idx_refresh_tokens_device ON refresh_tokens (device_fingerprint) WHERE device_fingerprint IS NOT NULL;
CREATE INDEX idx_refresh_tokens_revoked ON refresh_tokens (is_revoked, created_at);

-- Verification token indexes
CREATE INDEX idx_verification_tokens_user_id ON verification_tokens (user_id);
CREATE INDEX idx_verification_tokens_type ON verification_tokens (token_type);
CREATE INDEX idx_verification_tokens_expires_at ON verification_tokens (expires_at);
CREATE INDEX idx_verification_tokens_active ON verification_tokens (is_active, expires_at);

-- Security event indexes (for audit and analysis)
CREATE INDEX idx_security_events_user_id ON user_security_events (user_id, created_at DESC);
CREATE INDEX idx_security_events_type ON user_security_events (event_type, created_at DESC);
CREATE INDEX idx_security_events_ip ON user_security_events (ip_address, created_at DESC);
CREATE INDEX idx_security_events_success ON user_security_events (success, event_type, created_at DESC);
CREATE INDEX idx_security_events_risk ON user_security_events (risk_score, created_at DESC) WHERE risk_score > 50;

-- 2FA indexes
CREATE INDEX idx_user_2fa_user_id ON user_2fa (user_id);
CREATE INDEX idx_user_2fa_enabled ON user_2fa (user_id, is_enabled);
CREATE INDEX idx_user_2fa_primary ON user_2fa (user_id, is_primary) WHERE is_primary = true;

-- Session indexes
CREATE INDEX idx_user_sessions_user_id ON user_sessions (user_id, last_activity_at DESC);
CREATE INDEX idx_user_sessions_active ON user_sessions (is_active, expires_at);
CREATE INDEX idx_user_sessions_device ON user_sessions (device_fingerprint) WHERE device_fingerprint IS NOT NULL;
CREATE INDEX idx_user_sessions_ip ON user_sessions (ip_address, created_at DESC);

-- API key indexes
CREATE INDEX idx_api_service_types_name ON api_service_types (service_name);
CREATE INDEX idx_api_service_types_provider ON api_service_types (provider);
CREATE INDEX idx_user_api_keys_user_id ON user_api_keys (user_id);
CREATE INDEX idx_user_api_keys_service ON user_api_keys (service_type_id);
CREATE INDEX idx_user_api_keys_active ON user_api_keys (is_active, last_used_at DESC);
CREATE INDEX idx_user_api_keys_usage ON user_api_keys (usage_count, last_used_at DESC);

-- Composite indexes for common queries
CREATE INDEX idx_users_status_email ON users (status, email) WHERE deleted_at IS NULL;
CREATE INDEX idx_oauth_providers_composite ON oauth_providers (user_id, provider, is_active);
CREATE INDEX idx_refresh_tokens_composite ON refresh_tokens (user_id, is_revoked, expires_at);
CREATE INDEX idx_security_events_composite ON user_security_events (user_id, event_type, success, created_at DESC);

-- =====================================================
-- TRIGGERS FOR UPDATED_AT TIMESTAMPS
-- =====================================================

-- Generic trigger function for updating timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers to all tables with updated_at columns
CREATE TRIGGER trigger_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_user_passwords_updated_at 
    BEFORE UPDATE ON user_passwords 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_oauth_providers_updated_at 
    BEFORE UPDATE ON oauth_providers 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_refresh_tokens_updated_at 
    BEFORE UPDATE ON refresh_tokens 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_verification_tokens_updated_at 
    BEFORE UPDATE ON verification_tokens 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_user_2fa_updated_at 
    BEFORE UPDATE ON user_2fa 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_user_sessions_updated_at 
    BEFORE UPDATE ON user_sessions 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_api_service_types_updated_at 
    BEFORE UPDATE ON api_service_types 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_user_api_keys_updated_at 
    BEFORE UPDATE ON user_api_keys 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =====================================================
-- SOFT DELETE FUNCTIONS
-- =====================================================

-- Function to soft delete a user and cascade to related records
CREATE OR REPLACE FUNCTION soft_delete_user(user_uuid UUID)
RETURNS BOOLEAN AS $$
BEGIN
    -- Soft delete the user
    UPDATE users 
    SET 
        deleted_at = NOW(),
        status = 'deleted',
        email = email || '_deleted_' || EXTRACT(epoch FROM NOW())::text,
        username = CASE 
            WHEN username IS NOT NULL 
            THEN username || '_deleted_' || EXTRACT(epoch FROM NOW())::text 
            ELSE NULL 
        END
    WHERE id = user_uuid AND deleted_at IS NULL;
    
    -- Check if user was found and updated
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;
    
    -- Deactivate related records (but don't delete them for audit purposes)
    UPDATE user_passwords SET is_active = FALSE WHERE user_id = user_uuid;
    UPDATE oauth_providers SET is_active = FALSE WHERE user_id = user_uuid;
    UPDATE refresh_tokens SET is_revoked = TRUE, revoked_reason = 'user_deleted', revoked_at = NOW() WHERE user_id = user_uuid;
    UPDATE verification_tokens SET is_active = FALSE WHERE user_id = user_uuid;
    UPDATE user_2fa SET is_enabled = FALSE WHERE user_id = user_uuid;
    UPDATE user_sessions SET is_active = FALSE, terminated_at = NOW(), termination_reason = 'user_deleted' WHERE user_id = user_uuid;
    UPDATE user_api_keys SET is_active = FALSE WHERE user_id = user_uuid;
    
    -- Log the deletion
    INSERT INTO user_security_events (user_id, event_type, event_category, success, metadata)
    VALUES (user_uuid, 'account_deleted', 'account_management', TRUE, '{"deleted_at": "' || NOW() || '"}');
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Function to restore a soft-deleted user
CREATE OR REPLACE FUNCTION restore_user(user_uuid UUID)
RETURNS BOOLEAN AS $$
BEGIN
    UPDATE users 
    SET 
        deleted_at = NULL,
        status = 'inactive' -- Set to inactive, admin can activate manually
    WHERE id = user_uuid AND deleted_at IS NOT NULL;
    
    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;
    
    -- Log the restoration
    INSERT INTO user_security_events (user_id, event_type, event_category, success, metadata)
    VALUES (user_uuid, 'account_restored', 'account_management', TRUE, '{"restored_at": "' || NOW() || '"}');
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- CLEANUP AND MAINTENANCE FUNCTIONS
-- =====================================================

-- Function to cleanup expired tokens and sessions
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS INTEGER AS $$
DECLARE
    cleanup_count INTEGER := 0;
BEGIN
    -- Delete expired verification tokens
    DELETE FROM verification_tokens 
    WHERE expires_at < NOW() - INTERVAL '7 days';
    GET DIAGNOSTICS cleanup_count = ROW_COUNT;
    
    -- Revoke expired refresh tokens
    UPDATE refresh_tokens 
    SET is_revoked = TRUE, revoked_reason = 'expired', revoked_at = NOW()
    WHERE expires_at < NOW() AND is_revoked = FALSE;
    
    -- Deactivate expired sessions
    UPDATE user_sessions 
    SET is_active = FALSE, terminated_at = NOW(), termination_reason = 'expired'
    WHERE expires_at < NOW() AND is_active = TRUE;
    
    -- Deactivate expired API keys
    UPDATE user_api_keys 
    SET is_active = FALSE
    WHERE expires_at < NOW() AND is_active = TRUE;
    
    RETURN cleanup_count;
END;
$$ LANGUAGE plpgsql;

-- Function to cleanup old security events (keep last 90 days)
CREATE OR REPLACE FUNCTION cleanup_old_security_events()
RETURNS INTEGER AS $$
DECLARE
    cleanup_count INTEGER := 0;
BEGIN
    DELETE FROM user_security_events 
    WHERE created_at < NOW() - INTERVAL '90 days'
    AND event_category NOT IN ('security', 'admin'); -- Keep important events longer
    
    GET DIAGNOSTICS cleanup_count = ROW_COUNT;
    RETURN cleanup_count;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- HELPER VIEWS FOR COMMON QUERIES
-- =====================================================

-- View for active users with their authentication methods
CREATE VIEW v_user_auth_summary AS
SELECT 
    u.id,
    u.email,
    u.username,
    u.status,
    u.email_verified,
    u.last_login_at,
    u.created_at,
    CASE WHEN up.id IS NOT NULL THEN TRUE ELSE FALSE END as has_password,
    ARRAY_AGG(DISTINCT op.provider) FILTER (WHERE op.provider IS NOT NULL) as oauth_providers,
    ARRAY_AGG(DISTINCT mfa.method) FILTER (WHERE mfa.method IS NOT NULL AND mfa.is_enabled) as mfa_methods,
    COUNT(DISTINCT ak.id) FILTER (WHERE ak.is_active) as active_api_keys
FROM users u
LEFT JOIN user_passwords up ON u.id = up.user_id AND up.is_active = TRUE
LEFT JOIN oauth_providers op ON u.id = op.user_id AND op.is_active = TRUE
LEFT JOIN user_2fa mfa ON u.id = mfa.user_id AND mfa.is_enabled = TRUE
LEFT JOIN user_api_keys ak ON u.id = ak.user_id AND ak.is_active = TRUE
WHERE u.deleted_at IS NULL
GROUP BY u.id, u.email, u.username, u.status, u.email_verified, u.last_login_at, u.created_at, up.id;

-- View for API key usage summary
CREATE VIEW v_api_key_usage AS
SELECT 
    ak.id,
    u.email as user_email,
    ast.service_display_name,
    ast.provider,
    ak.key_name,
    ak.usage_count,
    ak.last_used_at,
    ak.rate_limit_per_hour,
    ak.current_usage,
    ak.is_active,
    ak.created_at
FROM user_api_keys ak
JOIN users u ON ak.user_id = u.id
JOIN api_service_types ast ON ak.service_type_id = ast.id
WHERE u.deleted_at IS NULL;

-- =====================================================
-- INITIAL ADMIN SETUP (OPTIONAL)
-- =====================================================

-- Create initial admin user (commented out - uncomment and modify as needed)

INSERT INTO users (email, username, email_verified, first_name, last_name, status)
VALUES ('admin@example.com', 'admin', TRUE, 'System', 'Administrator', 'active');
