-- Migration: Create OTP table for email verification
-- Description: Creates table for managing email OTP verification for login and other purposes

-- Create OTP purpose enum
CREATE TYPE otp_purpose AS ENUM ('login', 'register', 'password_reset');

-- Create OTP table
CREATE TABLE otps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL,
    code VARCHAR(6) NOT NULL,
    purpose otp_purpose NOT NULL DEFAULT 'login',
    is_verified BOOLEAN DEFAULT false,
    attempt_count INTEGER DEFAULT 0,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    verified_at TIMESTAMP WITH TIME ZONE NULL
);

-- Indexes for performance
CREATE INDEX idx_otps_email ON otps(email);
CREATE INDEX idx_otps_email_purpose ON otps(email, purpose);
CREATE INDEX idx_otps_code ON otps(code);
CREATE INDEX idx_otps_expires_at ON otps(expires_at);
CREATE INDEX idx_otps_created_at ON otps(created_at);

-- Partial indexes for active OTPs
CREATE INDEX idx_otps_active ON otps(email, purpose, expires_at) 
WHERE is_verified = false;

-- Add comments for documentation
COMMENT ON TABLE otps IS 'Email OTP verification system for authentication';
COMMENT ON COLUMN otps.code IS '6-digit OTP code sent to user email';
COMMENT ON COLUMN otps.purpose IS 'Purpose of OTP: login, register, password_reset';
COMMENT ON COLUMN otps.attempt_count IS 'Number of verification attempts made';
COMMENT ON COLUMN otps.expires_at IS 'When the OTP expires (typically 10 minutes)';

-- Trigger to update attempt count
CREATE OR REPLACE FUNCTION increment_otp_attempt()
RETURNS TRIGGER AS $$
BEGIN
    -- Only increment if verification failed (not verified but attempt was made)
    IF OLD.attempt_count < NEW.attempt_count THEN
        -- Lock account after 5 failed attempts
        IF NEW.attempt_count >= 5 THEN
            NEW.expires_at = NOW(); -- Expire immediately
        END IF;
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER trigger_increment_otp_attempt 
    BEFORE UPDATE ON otps
    FOR EACH ROW EXECUTE FUNCTION increment_otp_attempt();

-- Function to cleanup expired OTPs (run this periodically)
CREATE OR REPLACE FUNCTION cleanup_expired_otps()
RETURNS INTEGER AS $$
DECLARE
    affected_rows INTEGER;
BEGIN
    DELETE FROM otps 
    WHERE expires_at < NOW() - INTERVAL '1 day';
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions (adjust as needed for your user)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON otps TO your_app_user;
-- GRANT USAGE ON SEQUENCE otps_id_seq TO your_app_user;
