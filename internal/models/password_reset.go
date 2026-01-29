package models

import (
	"time"
)

// PasswordReset represents a password reset token
type PasswordReset struct {
	ResetToken string             `json:"reset_token"`
	UserID     string             `json:"user_id"`
	Email      string             `json:"email"`
	CreatedAt  time.Time          `json:"created_at"`
	ExpiresAt  time.Time          `json:"expires_at"`
	IsUsed     bool               `json:"is_used"`
	IPAddress  string             `json:"ip_address"`
	UserAgent  string             `json:"user_agent"`
}

// IsValid checks if the reset token is still valid
func (pr *PasswordReset) IsValid() bool {
	return !pr.IsUsed && time.Now().Before(pr.ExpiresAt)
}
