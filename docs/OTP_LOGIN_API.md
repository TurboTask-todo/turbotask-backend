# Email OTP Login API Documentation

## Overview

The MacWrite application now includes a comprehensive Email OTP (One-Time Password) login system that provides secure, passwordless authentication. Users can log in using only their email address and a 6-digit verification code sent to their email.

## Features

✅ **Passwordless Authentication** - No passwords required  
✅ **Email OTP Verification** - 6-digit codes sent via email  
✅ **Professional Email Templates** - Beautiful HTML and text emails  
✅ **Rate Limiting** - Protection against spam and abuse  
✅ **Security Features** - Attempt limits, expiration, constant-time comparison  
✅ **Gmail SMTP Support** - Ready for Gmail with app passwords  
✅ **Comprehensive Error Handling** - Clear error messages and codes

## API Endpoints

### 1. Initiate Login (Send OTP)

**Endpoint:** `POST /api/v1/auth/login/initiate`

**Description:** Sends a 6-digit OTP code to the user's email address.

**Request Body:**

```json
{
  "email": "user@example.com"
}
```

**Success Response (200):**

```json
{
  "message": "OTP sent successfully to your email",
  "email": "user@example.com",
  "expires_at": "2025-08-08T19:50:00Z"
}
```

**Error Responses:**

- `404` - User not found
- `429` - Too many OTP requests (rate limited)
- `400` - Invalid email format
- `500` - Email sending failed

### 2. Verify OTP and Login

**Endpoint:** `POST /api/v1/auth/login/verify`

**Description:** Verifies the OTP code and returns authentication tokens.

**Request Body:**

```json
{
  "email": "user@example.com",
  "code": "123456"
}
```

**Success Response (200):**

```json
{
  "success": true,
  "message": "Login successful", // OR "Account created and login successful"
  "access_token": "jwt_access_token_here",
  "refresh_token": "jwt_refresh_token_here",
  "is_new_user": false, // true if account was auto-created
  "user": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "email": "user@example.com",
    "username": "john_doe",
    "first_name": "John",
    "last_name": "Doe",
    "created_at": "2023-12-01T10:00:00Z",
    "updated_at": "2023-12-01T10:00:00Z"
  }
}
```

**Error Responses:**

- `401` - Invalid OTP code
- `401` - OTP expired
- `401` - Too many failed attempts
- `400` - Invalid request format

### 3. Resend OTP

**Endpoint:** `POST /api/v1/auth/login/resend`

**Description:** Resends the OTP code to the user's email.

**Request Body:**

```json
{
  "email": "user@example.com"
}
```

**Response:** Same as initiate login endpoint.

### 4. Check Email Existence

**Endpoint:** `GET /api/v1/auth/check-email?email=user@example.com`

**Description:** Checks if an email address is registered in the system.

**Success Response (200):**

```json
{
  "success": true,
  "exists": true,
  "email": "user@example.com",
  "username": "john_doe",
  "first_name": "John",
  "last_name": "Doe"
}
```

### 5. Logout (Protected)

**Endpoint:** `POST /api/v1/auth/logout/otp`

**Headers:** `Authorization: Bearer <access_token>`

**Description:** Logs out the user (invalidates tokens).

**Success Response (200):**

```json
{
  "success": true,
  "message": "Logged out successfully",
  "user_id": "123e4567-e89b-12d3-a456-426614174000"
}
```

## Security Features

### Rate Limiting

- **OTP Initiate:** 5 requests per minute per IP
- **OTP Verify:** 10 attempts per minute per IP
- **OTP Resend:** 3 requests per minute per IP
- **Email Check:** 60 requests per minute per IP

### OTP Security

- **Expiration:** 10 minutes
- **Attempt Limit:** 5 failed attempts per OTP
- **Rate Limiting:** Max 3 OTPs per 10 minutes per email
- **Constant-time Comparison:** Prevents timing attacks
- **Auto-invalidation:** Previous OTPs are invalidated when new ones are generated

### Database Security

- **Automatic Cleanup:** Expired OTPs are automatically cleaned up
- **Attempt Tracking:** Failed attempts are tracked and limited
- **Indexed Queries:** Optimized database queries with proper indexing

## Email Configuration

### Gmail Setup

1. **Enable 2-Factor Authentication** on your Gmail account
2. **Generate App Password:**
   - Go to Google Account settings
   - Security → 2-Step Verification → App passwords
   - Generate password for "Mail"
3. **Set Environment Variables:**
   ```bash
   EMAIL_ENABLED=true
   SMTP_HOST=smtp.gmail.com
   SMTP_PORT=587
   SMTP_USERNAME=your-email@gmail.com
   SMTP_PASSWORD=your-16-character-app-password
   FROM_EMAIL=your-email@gmail.com
   FROM_NAME=MacWrite Team
   ```

### Other Email Providers

The system supports any SMTP provider. Update these variables:

- `SMTP_HOST` - Your SMTP server
- `SMTP_PORT` - Usually 587 (TLS) or 465 (SSL)
- `SMTP_USERNAME` - Your SMTP username
- `SMTP_PASSWORD` - Your SMTP password

## ✨ Auto-Registration Feature

### Seamless User Onboarding

The OTP login system supports **automatic user registration**! When someone tries to login with an email that doesn't exist, we automatically create an account for them upon successful OTP verification.

### How It Works

1. **User enters any email** (existing or new)
2. **System sends OTP** (no user existence check)
3. **User verifies OTP**
4. **If email doesn't exist**: Create user automatically ✨
5. **If email exists**: Login to existing account

### New User Defaults

Auto-created users get sensible defaults:

| Field            | Value                   | Notes                     |
| ---------------- | ----------------------- | ------------------------- |
| `username`       | Email prefix (before @) | `john@gmail.com` → `john` |
| `first_name`     | `"User"`                | Can update later          |
| `last_name`      | `""`                    | Can update later          |
| `email_verified` | `true`                  | Verified via OTP          |
| `status`         | `"active"`              | Ready to use              |

### Response Fields

The verify endpoint returns `is_new_user: true/false` so your frontend can:

- Show welcome flow for new users
- Show "welcome back" for returning users
- Guide users through profile completion

## Email Templates

The system includes beautiful, responsive email templates:

### Login Template Features

- 🎨 Modern, professional design
- 📱 Mobile-responsive layout
- 🔒 Security notices and warnings
- ⏰ Clear expiration information
- 🎯 Large, easy-to-read OTP codes

### Template Types

- **Login OTP** - For user login verification
- **Registration OTP** - For new user email verification (future)
- **Password Reset OTP** - For password reset verification (future)

## Database Schema

### OTPs Table

```sql
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
```

### Indexes

- Email + Purpose for fast lookups
- Expiration time for cleanup
- Active OTPs for quick validation

## Error Handling

### Error Response Format

```json
{
  "success": false,
  "message": "Human-readable error message",
  "error": {
    "code": "ERROR_CODE",
    "message": "Detailed error description"
  },
  "timestamp": "2025-08-08T19:30:00Z"
}
```

### Common Error Codes

- `USER_NOT_FOUND` - Email address not registered
- `RATE_LIMIT_EXCEEDED` - Too many requests
- `OTP_EXPIRED` - OTP has expired
- `INVALID_OTP` - Incorrect OTP code
- `TOO_MANY_ATTEMPTS` - Failed attempt limit reached
- `EMAIL_SEND_FAILED` - Could not send email
- `VALIDATION_ERROR` - Invalid request data

## Usage Examples

### Frontend Integration

```javascript
// 1. Initiate login
const initiateLogin = async (email) => {
  const response = await fetch("/api/v1/auth/login/initiate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email }),
  });
  return response.json();
};

// 2. Verify OTP
const verifyOTP = async (email, code) => {
  const response = await fetch("/api/v1/auth/login/verify", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, code }),
  });
  const data = await response.json();

  if (data.success) {
    // Store tokens
    localStorage.setItem("accessToken", data.access_token);
    localStorage.setItem("refreshToken", data.refresh_token);
    localStorage.setItem("user", JSON.stringify(data.user));
  }

  return data;
};

// 3. Check email existence
const checkEmail = async (email) => {
  const response = await fetch(`/api/v1/auth/check-email?email=${email}`);
  return response.json();
};
```

### Testing with cURL

```bash
# 1. Check if email exists
curl -X GET "http://localhost:8080/api/v1/auth/check-email?email=user@example.com"

# 2. Initiate login
curl -X POST http://localhost:8080/api/v1/auth/login/initiate \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com"}'

# 3. Verify OTP (replace with actual code from email)
curl -X POST http://localhost:8080/api/v1/auth/login/verify \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "code": "123456"}'

# 4. Logout (replace with actual access token)
curl -X POST http://localhost:8080/api/v1/auth/logout/otp \
  -H "Authorization: Bearer your_access_token_here"
```

## Migration and Setup

### 1. Run Database Migration

```bash
# Apply the OTP table migration
psql postgresql://user:password@localhost:5432/database -f migrations/create_otp_table.sql
```

### 2. Set Environment Variables

Copy `env.example` to `.env` and update the email configuration:

```bash
cp env.example .env
# Edit .env with your email settings
```

### 3. Test Email Configuration

```bash
# Start the server
go run ./cmd/server/main.go

# Test email sending
curl -X POST http://localhost:8080/api/v1/auth/login/initiate \
  -H "Content-Type: application/json" \
  -d '{"email": "your-test-email@gmail.com"}'
```

## Production Considerations

### Security

- Use strong, unique JWT secrets
- Enable HTTPS/TLS in production
- Configure proper CORS origins
- Monitor rate limiting logs
- Set up proper logging for security events

### Performance

- Implement Redis for OTP caching (optional)
- Set up database connection pooling
- Monitor email sending performance
- Configure proper database indexes

### Monitoring

- Track OTP success/failure rates
- Monitor email delivery rates
- Set up alerts for failed email sends
- Log authentication events

This OTP login system provides a secure, user-friendly authentication method that's ready for production use with proper configuration and monitoring.
