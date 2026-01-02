package smtp

import (
	"encoding/base64"
	"fmt"
	"strings"
	"os"
	"strconv"
	"net/smtp"
	"mime/quotedprintable"
	"time"
	"crypto/tls"

	"github.com/white/user-management/internal/models"
)

type EmailAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// loginAuth implements the LOGIN authentication mechanism for SMTP
// Office 365 may require this instead of PLAIN auth
type loginAuth struct {
	username string
	password string
	host     string
}

// LoginAuth returns an Auth that implements the LOGIN authentication mechanism
func LoginAuth(username, password, host string) smtp.Auth {
	return &loginAuth{username, password, host}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if server.Name != a.host {
		return "", nil, fmt.Errorf("wrong host name")
	}
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	command := strings.ToLower(strings.TrimSuffix(string(fromServer), ":"))
	command = strings.TrimSpace(command)

	switch command {
	case "username":
		return []byte(a.username), nil
	case "password":
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("unknown command %s", command)
	}
}

// xoauth2Auth implements XOAUTH2 authentication for OAuth2 tokens
type xoauth2Auth struct {
	username    string
	accessToken string
}

// XOAUTH2Auth returns an Auth that implements the XOAUTH2 authentication mechanism
func XOAUTH2Auth(username, accessToken string) smtp.Auth {
	return &xoauth2Auth{username, accessToken}
}

func (a *xoauth2Auth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	// Build XOAUTH2 string: "user=" + username + "\x01auth=Bearer " + accessToken + "\x01\x01"
	authStr := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", a.username, a.accessToken)
	return "XOAUTH2", []byte(base64.StdEncoding.EncodeToString([]byte(authStr))), nil
}

func (a *xoauth2Auth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		return nil, fmt.Errorf("unexpected server challenge")
	}
	return nil, nil
}

// SMTPClient represents an SMTP email client
type SMTPClient struct {
	host       string
	port       int
	username   string
	password   string
	fromEmail  string
	replyTo    string
	tlsEnabled bool
}

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	Host       string
	Port       int
	Username   string
	Password   string
	FromEmail  string
	ReplyTo    string
	TLSEnabled bool
}

// NewSMTPClientFromEnv creates a new SMTP client from environment variables
func NewSMTPClientFromEnv() (*SMTPClient, error) {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		return nil, fmt.Errorf("SMTP_HOST is not configured")
	}

	portStr := os.Getenv("SMTP_PORT")
	port := 587 // default
	if portStr != "" {
		var err error
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid SMTP_PORT: %w", err)
		}
	}

	username := os.Getenv("SMTP_USER")
	if username == "" {
		return nil, fmt.Errorf("SMTP_USER is not configured")
	}

	password := os.Getenv("SMTP_PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("SMTP_PASSWORD is not configured")
	}

	fromEmail := os.Getenv("SMTP_FROM_EMAIL")
	if fromEmail == "" {
		fromEmail = username // default to username
	}

	replyTo := os.Getenv("SMTP_REPLY_TO")
	if replyTo == "" {
		replyTo = fromEmail // default to fromEmail
	}

	tlsEnabled := true
	tlsStr := os.Getenv("SMTP_TLS_ENABLED")
	if tlsStr != "" {
		tlsEnabled = strings.ToLower(tlsStr) == "true" || tlsStr == "1"
	}

	return &SMTPClient{
		host:       host,
		port:       port,
		username:   username,
		password:   password,
		fromEmail:  fromEmail,
		replyTo:    replyTo,
		tlsEnabled: tlsEnabled,
	}, nil
}

// NewSMTPClient creates a new SMTP client with explicit configuration
func NewSMTPClient(config *SMTPConfig) *SMTPClient {
	return &SMTPClient{
		host:       config.Host,
		port:       config.Port,
		username:   config.Username,
		password:   config.Password,
		fromEmail:  config.FromEmail,
		replyTo:    config.ReplyTo,
		tlsEnabled: config.TLSEnabled,
	}
}

// SendEmail sends a single email via SMTP
func (c *SMTPClient) SendEmail(msg *models.CommMessage) error {
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	// Validate message
	if err := c.validateMessage(msg); err != nil {
		return fmt.Errorf("invalid message: %w", err)
	}

	// Build email message
	emailContent := c.buildEmailContent(msg)

	// Get all recipients
	allRecipients := c.getAllRecipients(msg)

	// Connect and send
	return c.sendViaSMTP(allRecipients, emailContent)
}

// SendBulkEmail sends multiple emails
func (c *SMTPClient) SendBulkEmail(msgs []*models.CommMessage) error {
	if len(msgs) == 0 {
		return nil
	}

	var errors []error
	for i, msg := range msgs {
		if err := c.SendEmail(msg); err != nil {
			errors = append(errors, fmt.Errorf("email %d failed: %w", i, err))
		}
		// Small delay between emails to avoid rate limiting
		if i < len(msgs)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("bulk send completed with %d errors: %v", len(errors), errors)
	}

	return nil
}


// buildEmailContent builds the MIME email content
func (c *SMTPClient) buildEmailContent(msg *models.CommMessage) []byte {
	var builder strings.Builder

	// From header - ALWAYS use configured SMTP sender to avoid SendAsDenied errors
	fromAddr := c.fromEmail
	fromName := msg.FromName
	if fromName == "" {
		fromName = "White Platform"
	}

	builder.WriteString(fmt.Sprintf("From: %s <%s>\r\n", fromName, fromAddr))

	// To header
	builder.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.ToAddresses, ", ")))

	// CC header
	if len(msg.CCAddresses) > 0 {
		builder.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.CCAddresses, ", ")))
	}

	// Reply-To header - use original sender's email so replies go to them
	replyTo := c.replyTo
	if msg.FromAddress != "" && msg.FromAddress != c.fromEmail {
		replyTo = msg.FromAddress // Use original sender for replies
	}
	if replyTo != "" {
		builder.WriteString(fmt.Sprintf("Reply-To: %s\r\n", replyTo))
	}

	// Subject header
	builder.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))

	// Message-ID header
	if msg.MessageID.Hex() != "" && msg.MessageID.Hex() != "000000000000000000000000" {
		builder.WriteString(fmt.Sprintf("Message-ID: <%s@csa.skillzen.ai>\r\n", msg.MessageID.Hex()))
	}

	// In-Reply-To and References headers for threading
	if msg.InReplyTo != "" {
		builder.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", msg.InReplyTo))
	}
	if msg.References != "" {
		builder.WriteString(fmt.Sprintf("References: %s\r\n", msg.References))
	}

	// Date header
	builder.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))

	// MIME headers
	builder.WriteString("MIME-Version: 1.0\r\n")

	// Determine content type
	if msg.BodyHTML != "" && msg.BodyText != "" {
		// Multipart alternative
		boundary := fmt.Sprintf("----=_Part_%d", time.Now().UnixNano())
		builder.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		builder.WriteString("\r\n")

		// Plain text part - use quoted-printable encoding
		builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		builder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(encodeQuotedPrintable(msg.BodyText))
		builder.WriteString("\r\n")

		// HTML part - use quoted-printable encoding
		builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		builder.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(encodeQuotedPrintable(msg.BodyHTML))
		builder.WriteString("\r\n")

		// End boundary
		builder.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if msg.BodyHTML != "" {
		// HTML only - use quoted-printable encoding
		builder.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(encodeQuotedPrintable(msg.BodyHTML))
	} else {
		// Plain text only - use quoted-printable encoding
		builder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		builder.WriteString("\r\n")
		builder.WriteString(encodeQuotedPrintable(msg.BodyText))
	}

	return []byte(builder.String())
}

// getAllRecipients returns all recipients (To, CC, BCC)
func (c *SMTPClient) getAllRecipients(msg *models.CommMessage) []string {
	recipients := make([]string, 0, len(msg.ToAddresses)+len(msg.CCAddresses)+len(msg.BCCAddresses))
	recipients = append(recipients, msg.ToAddresses...)
	recipients = append(recipients, msg.CCAddresses...)
	recipients = append(recipients, msg.BCCAddresses...)
	return recipients
}

// validateMessage validates message before sending
func (c *SMTPClient) validateMessage(msg *models.CommMessage) error {
	if len(msg.ToAddresses) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	if msg.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	if msg.BodyText == "" && msg.BodyHTML == "" {
		return fmt.Errorf("message body is required")
	}
	return nil
}

// sendViaSMTP connects to SMTP server and sends the email
func (c *SMTPClient) sendViaSMTP(recipients []string, content []byte) error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)

	// Create authentication
	auth := smtp.PlainAuth("", c.username, c.password, c.host)

	if c.tlsEnabled && c.port == 587 {
		// Use STARTTLS for port 587
		return c.sendWithSTARTTLS(addr, auth, recipients, content)
	} else if c.port == 465 {
		// Use implicit TLS for port 465
		return c.sendWithImplicitTLS(addr, auth, recipients, content)
	}

	// Plain SMTP (not recommended)
	return smtp.SendMail(addr, auth, c.fromEmail, recipients, content)
}

// sendWithSTARTTLS sends email using STARTTLS (port 587)
// Tries LOGIN auth first (for Office 365), then falls back to PLAIN auth
func (c *SMTPClient) sendWithSTARTTLS(addr string, auth smtp.Auth, recipients []string, content []byte) error {
	// Connect to server
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	// Say hello
	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("failed to send HELO: %w", err)
	}

	// Start TLS
	tlsConfig := &tls.Config{
		ServerName: c.host,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Try LOGIN auth first (Office 365 requires this)
	loginAuth := LoginAuth(c.username, c.password, c.host)
	if err := client.Auth(loginAuth); err != nil {
		// If LOGIN fails, try PLAIN auth as fallback
		plainAuth := smtp.PlainAuth("", c.username, c.password, c.host)
		if err := client.Auth(plainAuth); err != nil {
			return fmt.Errorf("failed to authenticate (tried LOGIN and PLAIN): %w", err)
		}
	}

	// Set sender
	if err := client.Mail(c.fromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send message body
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open data connection: %w", err)
	}

	_, err = writer.Write(content)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close data connection: %w", err)
	}

	// Quit
	return client.Quit()
}

// sendWithImplicitTLS sends email using implicit TLS (port 465)
// Tries LOGIN auth first (for Office 365), then falls back to PLAIN auth
func (c *SMTPClient) sendWithImplicitTLS(addr string, auth smtp.Auth, recipients []string, content []byte) error {
	// Create TLS connection
	tlsConfig := &tls.Config{
		ServerName: c.host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect with TLS: %w", err)
	}
	defer conn.Close()

	// Create SMTP client on TLS connection
	client, err := smtp.NewClient(conn, c.host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Try LOGIN auth first (Office 365 requires this)
	loginAuth := LoginAuth(c.username, c.password, c.host)
	if err := client.Auth(loginAuth); err != nil {
		// If LOGIN fails, try PLAIN auth as fallback
		plainAuth := smtp.PlainAuth("", c.username, c.password, c.host)
		if err := client.Auth(plainAuth); err != nil {
			return fmt.Errorf("failed to authenticate (tried LOGIN and PLAIN): %w", err)
		}
	}

	// Set sender
	if err := client.Mail(c.fromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send message body
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open data connection: %w", err)
	}

	_, err = writer.Write(content)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close data connection: %w", err)
	}

	// Quit
	return client.Quit()
}

// TestConnection tests the SMTP connection
// Tries LOGIN auth first (for Office 365), then falls back to PLAIN auth
func (c *SMTPClient) TestConnection() error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)

	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("failed HELO: %w", err)
	}

	if c.tlsEnabled {
		tlsConfig := &tls.Config{
			ServerName: c.host,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed STARTTLS: %w", err)
		}
	}

	// Try LOGIN auth first (Office 365 requires this)
	loginAuth := LoginAuth(c.username, c.password, c.host)
	if err := client.Auth(loginAuth); err != nil {
		// If LOGIN fails, try PLAIN auth as fallback
		plainAuth := smtp.PlainAuth("", c.username, c.password, c.host)
		if err := client.Auth(plainAuth); err != nil {
			return fmt.Errorf("authentication failed (tried LOGIN and PLAIN): %w", err)
		}
	}

	return client.Quit()
}

// GetFromEmail returns the configured from email address
func (c *SMTPClient) GetFromEmail() string {
	return c.fromEmail
}

// GetReplyTo returns the configured reply-to address
func (c *SMTPClient) GetReplyTo() string {
	return c.replyTo
}

// encodeQuotedPrintable encodes a string in quoted-printable format
func encodeQuotedPrintable(s string) string {
	var buf strings.Builder
	writer := quotedprintable.NewWriter(&buf)
	writer.Write([]byte(s))
	writer.Close()
	return buf.String()
}