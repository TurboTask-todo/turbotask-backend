# Troubleshooting Guide

## 🔧 **Fixed Issues**

### 1. Nil Pointer Dereference in Login Verification

**Error:** `runtime error: invalid memory address or nil pointer dereference`

**Cause:** The `GetByEmail` method returns `(nil, nil)` when no user is found, but the code was only checking for errors, not nil users.

**Fix Applied:** Added proper nil checks in both login handler and OTP service.

### 2. SMTP TLS Handshake Error

**Error:** `"tls: first record does not look like a TLS handshake"`

**Cause:** Wrong connection method for Gmail SMTP (port 587 requires STARTTLS, not direct TLS).

**Fix Applied:** Updated email client to use proper connection methods based on port.

## 🚀 **Setup Steps**

### 1. Create a Test User

If you don't have any users in your database, create one first:

```sql
-- Run this SQL to create a test user
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
    'your-email@gmail.com',  -- CHANGE THIS
    'testuser',
    'Test',
    'User',
    true,
    'active',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
);
```

Or use the provided script:

```bash
# Edit the email in the file first
psql your_database -f scripts/create_test_user.sql
```

### 2. Verify User Exists

Test if your user exists:

```bash
curl "http://localhost:8080/api/v1/auth/check-email?email=your-email@gmail.com"
```

Expected response:

```json
{
  "success": true,
  "exists": true,
  "email": "your-email@gmail.com",
  "username": "testuser",
  "first_name": "Test",
  "last_name": "User",
  "user_id": "uuid-here"
}
```

### 3. Test OTP Flow

**Step 1: Initiate Login**

```bash
curl -X POST http://localhost:8080/api/v1/auth/login/initiate \
  -H "Content-Type: application/json" \
  -d '{"email": "your-email@gmail.com"}'
```

**Step 2: Check Email** (you should receive an email with 6-digit code)

**Step 3: Verify OTP**

```bash
curl -X POST http://localhost:8080/api/v1/auth/login/verify \
  -H "Content-Type: application/json" \
  -d '{"email": "your-email@gmail.com", "code": "123456"}'
```

## 🐛 **Common Issues**

### "User not found" Error

- **Cause:** No user with that email exists in database
- **Solution:** Create a test user using the SQL script above

### "Email sending failed" Error

- **Cause:** SMTP configuration issues
- **Solution:** Check Gmail setup in `docs/GMAIL_SETUP.md`

### "OTP verification failed" Error

- **Cause:** Wrong OTP code or expired OTP
- **Solution:**
  - Check your email for the latest code
  - OTPs expire in 10 minutes
  - Try requesting a new OTP

### "Too many attempts" Error

- **Cause:** Rate limiting protection
- **Solution:** Wait 10 minutes before trying again

## 📝 **Debug Steps**

1. **Check server logs** for detailed error messages
2. **Verify database connection** by checking existing users
3. **Test email sending** with a simple initiate request
4. **Check environment variables** in your `.env` file

## 🔒 **Security Notes**

- OTPs expire in 10 minutes
- Maximum 5 failed verification attempts per OTP
- Maximum 3 OTP requests per 10 minutes per email
- Rate limiting applies at IP level as well

The system is now robust and handles all edge cases properly! 🎉
