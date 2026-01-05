package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Session represents an authenticated user session
type Session struct {
	TokenID      primitive.ObjectID `json:"token_id" bson:"token_id"`
	UserID       primitive.ObjectID `json:"user_id" bson:"user_id"`
	RefreshToken string             `json:"refresh_token" bson:"refresh_token"`
	IssuedAt     time.Time          `json:"issued_at" bson:"issued_at"`
	ExpiresAt    time.Time          `json:"expires_at" bson:"expires_at"`
	IPAddress    string             `json:"ip_address" bson:"ip_address"`
	UserAgent    string             `json:"user_agent" bson:"user_agent"`
	IsRevoked    bool               `json:"is_revoked" bson:"is_revoked"`
	RevokedAt    *time.Time         `json:"revoked_at,omitempty" bson:"revoked_at,omitempty"`

}

// IsValid checks if the session is still valid
func (s *Session) IsVaild() bool {
	return !s.IsRevoked && time.Now().Before(s.ExpiresAt);
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}