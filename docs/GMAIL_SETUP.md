# Gmail SMTP Setup Guide

## 🔧 **SMTP Connection Error Fixed**

The "TLS handshake" error has been resolved! The issue was that the original code was trying to connect directly with TLS to port 587, but Gmail's SMTP server requires STARTTLS (plain connection first, then upgrade to TLS).

## ✅ **Gmail Setup Instructions**

### 1. Enable 2-Factor Authentication

- Go to your [Google Account Settings](https://myaccount.google.com/)
- Click **Security** in the left sidebar
- Enable **2-Step Verification** if not already enabled

### 2. Generate App Password

- In Google Account Settings → **Security**
- Click **2-Step Verification**
- Scroll down to **App passwords**
- Select app: **Mail**
- Select device: **Other (Custom name)** → Enter "MacWrite API"
- Copy the generated 16-character password

### 3. Configure Environment Variables

Update your `.env` file with these settings:

```bash
# Email Configuration (Gmail SMTP)
EMAIL_ENABLED=true
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-16-character-app-password
FROM_EMAIL=your-email@gmail.com
FROM_NAME=MacWrite Team
```

**Important Notes:**

- Use port **587** for STARTTLS (recommended)
- Use port **465** for SSL/TLS (alternative)
- The password should be the 16-character app password, NOT your regular Gmail password
- Remove spaces from the app password

### 4. Test the Configuration

Start your server and test:

```bash
# Start the server
go run ./cmd/server/main.go

# Test OTP sending
curl -X POST http://localhost:8080/api/v1/auth/login/initiate \
  -H "Content-Type: application/json" \
  -d '{"email": "your-email@gmail.com"}'
```

## 🛠 **Alternative SMTP Providers**

The system now supports both connection methods:

### Gmail (Recommended)

```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587  # STARTTLS
# OR
SMTP_PORT=465  # SSL/TLS
```

### Outlook/Hotmail

```bash
SMTP_HOST=smtp-mail.outlook.com
SMTP_PORT=587
```

### Yahoo Mail

```bash
SMTP_HOST=smtp.mail.yahoo.com
SMTP_PORT=587
```

### Custom SMTP Server

```bash
SMTP_HOST=your-smtp-server.com
SMTP_PORT=587  # or 465
```

## 🔍 **Troubleshooting**

### Common Issues:

1. **"Username and Password not accepted"**

   - Make sure you're using the App Password, not your regular password
   - Verify 2FA is enabled on your Google account

2. **"Connection refused"**

   - Check your internet connection
   - Verify the SMTP_HOST and SMTP_PORT values
   - Make sure your firewall isn't blocking outbound connections

3. **"Authentication failed"**

   - Double-check your email and app password
   - Make sure there are no extra spaces in the credentials

4. **"TLS handshake error"** (Fixed!)
   - This should no longer occur with the updated code
   - The system automatically uses the correct connection method

### Debug Steps:

1. **Verify Gmail credentials manually:**

   ```bash
   # Test with telnet (if available)
   telnet smtp.gmail.com 587
   ```

2. **Check server logs:**

   ```bash
   # Look for detailed error messages in the console output
   go run ./cmd/server/main.go
   ```

3. **Test with minimal example:**
   ```bash
   # Send to your own email first
   curl -X POST http://localhost:8080/api/v1/auth/login/initiate \
     -H "Content-Type: application/json" \
     -d '{"email": "your-own-email@gmail.com"}'
   ```

## 📧 **Email Template Preview**

When working correctly, you'll receive professional emails like this:

```
Subject: 🔐 Your MacWrite Login Code

🚀 MacWrite Todo App

Welcome back!

You requested to sign in to your MacWrite Todo App account.
Use the verification code below:

[  1  2  3  4  5  6  ]

This code will expire in 10 minutes.

SECURITY NOTICE: If you didn't request this code, please ignore
this email. Never share your verification code with anyone.
```

## 🔒 **Security Best Practices**

1. **Use App Passwords**: Never use your main Gmail password
2. **Rotate Passwords**: Generate new app passwords periodically
3. **Monitor Usage**: Check your Gmail's "Recent security activity"
4. **Limit Scope**: Create separate app passwords for different applications

The SMTP connection is now properly configured to work with Gmail and other major email providers! 🚀
