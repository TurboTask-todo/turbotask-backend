package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net"
	"net/smtp"
	"strconv"

	"macwrite-auth-api/internal/config"
	"macwrite-auth-api/internal/models"
)

// Client represents the email client
type Client struct {
	config config.EmailConfig
}

// NewClient creates a new email client
func NewClient(emailConfig config.EmailConfig) *Client {
	return &Client{
		config: emailConfig,
	}
}

// EmailTemplate represents an email template
type EmailTemplate struct {
	Subject string
	HTML    string
	Text    string
}

// OTPEmailData represents data for OTP email template
type OTPEmailData struct {
	Code        string
	Email       string
	Purpose     string
	ExpiresIn   string
	CompanyName string
	AppName     string
}

// SendOTP sends an OTP email
func (c *Client) SendOTP(email, code string, purpose models.OTPPurpose, expiresIn string) error {
	data := OTPEmailData{
		Code:        code,
		Email:       email,
		Purpose:     string(purpose),
		ExpiresIn:   expiresIn,
		CompanyName: "MacWrite",
		AppName:     "MacWrite Todo App",
	}

	template := c.getOTPTemplate(purpose)

	return c.sendEmail(email, template, data)
}

// sendEmail sends an email using the configured SMTP settings
func (c *Client) sendEmail(to string, emailTemplate EmailTemplate, data interface{}) error {
	// Parse HTML template
	htmlTmpl, err := template.New("html").Parse(emailTemplate.HTML)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	// Parse text template
	textTmpl, err := template.New("text").Parse(emailTemplate.Text)
	if err != nil {
		return fmt.Errorf("failed to parse text template: %w", err)
	}

	// Execute templates
	var htmlBuf, textBuf bytes.Buffer
	if err := htmlTmpl.Execute(&htmlBuf, data); err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}
	if err := textTmpl.Execute(&textBuf, data); err != nil {
		return fmt.Errorf("failed to execute text template: %w", err)
	}

	// Create message
	message := c.buildMessage(to, emailTemplate.Subject, htmlBuf.String(), textBuf.String())

	// Send email
	return c.sendSMTP(to, message)
}

// sendSMTP sends email via SMTP
func (c *Client) sendSMTP(to, message string) error {
	// SMTP server configuration
	host := c.config.SMTPHost + ":" + strconv.Itoa(c.config.SMTPPort)

	// TLS config
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         c.config.SMTPHost,
	}

	var client *smtp.Client
	var err error

	// Different connection methods based on port
	if c.config.SMTPPort == 465 {
		// SSL/TLS connection (port 465)
		conn, err := tls.Dial("tcp", host, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server (SSL): %w", err)
		}
		defer conn.Close()

		client, err = smtp.NewClient(conn, c.config.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
	} else {
		// STARTTLS connection (port 587 or others)
		conn, err := net.Dial("tcp", host)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}
		defer conn.Close()

		client, err = smtp.NewClient(conn, c.config.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}

		// Start TLS (STARTTLS)
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	defer client.Quit()

	// Authenticate
	auth := smtp.PlainAuth("", c.config.SMTPUsername, c.config.SMTPPassword, c.config.SMTPHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	// Set sender
	if err := client.Mail(c.config.FromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipient
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Send message
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	defer writer.Close()

	if _, err := writer.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// buildMessage builds the email message
func (c *Client) buildMessage(to, subject, htmlBody, textBody string) string {
	boundary := "boundary-macwrite-email"

	message := fmt.Sprintf(`From: %s <%s>
To: %s
Subject: %s
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary="%s"

--%s
Content-Type: text/plain; charset=UTF-8
Content-Transfer-Encoding: 7bit

%s

--%s
Content-Type: text/html; charset=UTF-8
Content-Transfer-Encoding: 7bit

%s

--%s--
`, c.config.FromName, c.config.FromEmail, to, subject, boundary, boundary, textBody, boundary, htmlBody, boundary)

	return message
}

// getOTPTemplate returns the appropriate template for OTP emails
func (c *Client) getOTPTemplate(purpose models.OTPPurpose) EmailTemplate {
	switch purpose {
	case models.OTPPurposeLogin:
		return EmailTemplate{
			Subject: "🔐 Your MacWrite Login Code",
			HTML:    c.getLoginOTPHTMLTemplate(),
			Text:    c.getLoginOTPTextTemplate(),
		}
	case models.OTPPurposeRegister:
		return EmailTemplate{
			Subject: "🎉 Welcome to MacWrite - Verify Your Email",
			HTML:    c.getRegisterOTPHTMLTemplate(),
			Text:    c.getRegisterOTPTextTemplate(),
		}
	case models.OTPPurposeReset:
		return EmailTemplate{
			Subject: "🔄 Reset Your MacWrite Password",
			HTML:    c.getResetOTPHTMLTemplate(),
			Text:    c.getResetOTPTextTemplate(),
		}
	default:
		return EmailTemplate{
			Subject: "Your MacWrite Verification Code",
			HTML:    c.getLoginOTPHTMLTemplate(),
			Text:    c.getLoginOTPTextTemplate(),
		}
	}
}

// getLoginOTPHTMLTemplate returns HTML template for login OTP
func (c *Client) getLoginOTPHTMLTemplate() string {
	return `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MacWrite Login Code</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; margin-bottom: 30px; }
        .logo { font-size: 24px; font-weight: bold; color: #3498db; }
        .code-box { background: #f8f9fa; border: 2px solid #3498db; border-radius: 8px; padding: 20px; text-align: center; margin: 20px 0; }
        .code { font-size: 32px; font-weight: bold; color: #3498db; letter-spacing: 8px; font-family: 'Courier New', monospace; }
        .warning { background: #fff3cd; border: 1px solid #ffeaa7; border-radius: 4px; padding: 12px; margin: 20px 0; color: #856404; }
        .footer { text-align: center; margin-top: 30px; color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">🚀 {{.AppName}}</div>
        </div>
        
        <h2>Welcome back!</h2>
        <p>You requested to sign in to your {{.AppName}} account. Use the verification code below:</p>
        
        <div class="code-box">
            <div class="code">{{.Code}}</div>
        </div>
        
        <p><strong>This code will expire in {{.ExpiresIn}}.</strong></p>
        
        <div class="warning">
            <strong>Security Notice:</strong> If you didn't request this code, please ignore this email. Never share your verification code with anyone.
        </div>
        
        <p>Having trouble? Contact our support team.</p>
        
        <div class="footer">
            <p>Best regards,<br>The {{.CompanyName}} Team</p>
            <p style="font-size: 12px; color: #999;">This is an automated message. Please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>`
}

// getLoginOTPTextTemplate returns text template for login OTP
func (c *Client) getLoginOTPTextTemplate() string {
	return `Welcome back to {{.AppName}}!

You requested to sign in to your account. Use this verification code:

CODE: {{.Code}}

This code will expire in {{.ExpiresIn}}.

SECURITY NOTICE: If you didn't request this code, please ignore this email. Never share your verification code with anyone.

Having trouble? Contact our support team.

Best regards,
The {{.CompanyName}} Team

This is an automated message. Please do not reply to this email.`
}

// getRegisterOTPHTMLTemplate returns HTML template for registration OTP
func (c *Client) getRegisterOTPHTMLTemplate() string {
	return `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome to MacWrite</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; margin-bottom: 30px; }
        .logo { font-size: 24px; font-weight: bold; color: #3498db; }
        .code-box { background: #f8f9fa; border: 2px solid #27ae60; border-radius: 8px; padding: 20px; text-align: center; margin: 20px 0; }
        .code { font-size: 32px; font-weight: bold; color: #27ae60; letter-spacing: 8px; font-family: 'Courier New', monospace; }
        .footer { text-align: center; margin-top: 30px; color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">🎉 {{.AppName}}</div>
        </div>
        
        <h2>Welcome to {{.AppName}}!</h2>
        <p>Thank you for joining us! Please verify your email address with the code below:</p>
        
        <div class="code-box">
            <div class="code">{{.Code}}</div>
        </div>
        
        <p><strong>This code will expire in {{.ExpiresIn}}.</strong></p>
        
        <p>Once verified, you'll have access to all our features!</p>
        
        <div class="footer">
            <p>Welcome aboard!<br>The {{.CompanyName}} Team</p>
        </div>
    </div>
</body>
</html>`
}

// getRegisterOTPTextTemplate returns text template for registration OTP
func (c *Client) getRegisterOTPTextTemplate() string {
	return `Welcome to {{.AppName}}!

Thank you for joining us! Please verify your email address with this code:

CODE: {{.Code}}

This code will expire in {{.ExpiresIn}}.

Once verified, you'll have access to all our features!

Welcome aboard!
The {{.CompanyName}} Team`
}

// getResetOTPHTMLTemplate returns HTML template for password reset OTP
func (c *Client) getResetOTPHTMLTemplate() string {
	return `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Password Reset - MacWrite</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; margin-bottom: 30px; }
        .logo { font-size: 24px; font-weight: bold; color: #3498db; }
        .code-box { background: #f8f9fa; border: 2px solid #e74c3c; border-radius: 8px; padding: 20px; text-align: center; margin: 20px 0; }
        .code { font-size: 32px; font-weight: bold; color: #e74c3c; letter-spacing: 8px; font-family: 'Courier New', monospace; }
        .warning { background: #fee; border: 1px solid #e74c3c; border-radius: 4px; padding: 12px; margin: 20px 0; color: #721c24; }
        .footer { text-align: center; margin-top: 30px; color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">🔄 {{.AppName}}</div>
        </div>
        
        <h2>Password Reset Request</h2>
        <p>You requested to reset your password. Use the verification code below:</p>
        
        <div class="code-box">
            <div class="code">{{.Code}}</div>
        </div>
        
        <p><strong>This code will expire in {{.ExpiresIn}}.</strong></p>
        
        <div class="warning">
            <strong>Security Notice:</strong> If you didn't request this password reset, please ignore this email and consider changing your password.
        </div>
        
        <div class="footer">
            <p>Best regards,<br>The {{.CompanyName}} Team</p>
        </div>
    </div>
</body>
</html>`
}

// getResetOTPTextTemplate returns text template for password reset OTP
func (c *Client) getResetOTPTextTemplate() string {
	return `Password Reset Request - {{.AppName}}

You requested to reset your password. Use this verification code:

CODE: {{.Code}}

This code will expire in {{.ExpiresIn}}.

SECURITY NOTICE: If you didn't request this password reset, please ignore this email and consider changing your password.

Best regards,
The {{.CompanyName}} Team`
}
