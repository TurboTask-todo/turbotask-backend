-- Create a test user for OTP login testing
-- Run this if you don't have any users in your database

-- Insert a test user (update the email to your actual email)
INSERT INTO users (
    id, 
    email, 
    username, 
    first_name, 
    last_name, 
    email_verified,
    status,
    created_at, 
    updated_at
) VALUES (
    uuid_generate_v4(),
    'your-email@gmail.com',  -- CHANGE THIS TO YOUR EMAIL
    'testuser',
    'Test',
    'User',
    true,
    'active',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT (email) DO NOTHING;

-- Verify the user was created
SELECT id, email, username, first_name, last_name, created_at 
FROM users 
WHERE email = 'your-email@gmail.com';  -- CHANGE THIS TO YOUR EMAIL
