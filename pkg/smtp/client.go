package smtp

import (
	"encoding/base64"
	"fmt"
	"strings"
	"os"
	"strconv"
	"net/smtp"

	"github.com/white/user-management/internal/models"
)

type EmailAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

type loginAuth struct {
	username string
	password string
	host     string
}

func LoginAuth(username, password, host string) smtp.Auth {
	return &loginAuth{username, password, host}
}

func (a *loginAuth) start(server *smtp.ServerInfo) (string, []byte, error) {
	if server.Name != a.host {
		return "", nil, fmt.Errorf("wrong host name")
	}
	return "LOGIN", nil, nil
}

func (a *loginAuth) next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	command := strings.ToLower(strings.TrimSpace(string(fromServer), ":"))
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

func (c *SMTPClient) SendEmail(msg *model){

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
