package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)


// MongoUser represents a user in the MongoDB database
// Collection: users
type User struct {
	ID             primitive.ObjectID    `bson:"_id,omitempty" json:"id"`
	Email          string                `bson:"email" json:"email"`
	PasswordHash   string                `bson:"password_hash" json:"-"` // Never expose in JSON
	Name           string                `bson:"name" json:"name"`
	Role           UserRole              `bson:"role" json:"role"`
	Region         string                `bson:"region" json:"region"`
	Team           string                `bson:"team,omitempty" json:"team,omitempty"`
	Permissions    []string              `bson:"permissions,omitempty" json:"permissions,omitempty"`
	Preferences    *UserPreferences      `bson:"preferences,omitempty" json:"preferences,omitempty"`
	EmailSignature string                `bson:"email_signature,omitempty" json:"emailSignature,omitempty"`
	IsActive       bool                  `bson:"is_active" json:"isActive"`
	OTPHash        string                `bson:"otp_hash,omitempty" json:"-"` // Never expose in JSON
	OTPExpiresAt   *time.Time            `bson:"otp_expires_at,omitempty" json:"-"`
	CreatedAt      time.Time             `bson:"created_at" json:"createdAt"`
	UpdatedAt      time.Time             `bson:"updated_at" json:"updatedAt"`
	LastLoginAt  *time.Time         `bson:"last_login_at,omitempty" json:"last_login_at,omitempty"`

}

// MongoUserPreferences stores user-specific preferences (MongoDB version)
type UserPreferences struct {
	Language        string `bson:"language,omitempty" json:"language,omitempty"`
	Timezone        string `bson:"timezone,omitempty" json:"timezone,omitempty"`
	DateFormat      string `bson:"date_format,omitempty" json:"dateFormat,omitempty"`
	TimeFormat      string `bson:"time_format,omitempty" json:"timeFormat,omitempty"`
	Theme           string `bson:"theme,omitempty" json:"theme,omitempty"`
	DashboardLayout string `bson:"dashboard_layout,omitempty" json:"dashboardLayout,omitempty"`
}

type UserRole string

const (
	UserRoleAdmin    UserRole = "admin"     // Full access to all features
	UserRoleSalesRep UserRole = "sales_rep" // Sales operations access
	UserRoleManager  UserRole = "manager"   // Team management access
)

type UserProfile struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email       string             `bson:"email" json:"email"`
	Name        string             `bson:"name" json:"name"`
	Role        string             `bson:"role" json:"role"`
	Team        string             `bson:"team" json:"team"`
	Permissions []string           `bson:"permissions" json:"permissions"`
}

func (u *User) ToProfile() *UserProfile {
	return &UserProfile{
		ID:          u.ID,
		Email:       u.Email,
		Name:        u.Name,
		Role:        string(u.Role),
		Team:        u.Team,
		Permissions: u.Permissions,
	}
}
