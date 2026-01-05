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
type User1 struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"password_hash" json:"-"` // Never expose password hash in JSON
	Name         string             `bson:"name" json:"name"`
	Role         string             `bson:"role" json:"role"`               // hunting, farming, admin
	Region       string             `bson:"region" json:"region"`           // north, south, east, west, central
	Team         string             `bson:"team" json:"team"`               // sales, marketing, support
	Permissions  []string           `bson:"permissions" json:"permissions"` // Array of permission strings
	IsActive     bool               `bson:"is_active" json:"is_active"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
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
	Region      string             `bson:"region" json:"region"`
	Team        string             `bson:"team" json:"team"`
	Permissions []string           `bson:"permissions" json:"permissions"`
}

// IsValidUserRole checks if the user role is valid
func IsValidUserRole(role string) bool {
	validRoles := []UserRole{
		UserRoleAdmin,
		UserRoleSalesRep,
		UserRoleManager,
	}

	for _, validRole := range validRoles {
		if UserRole(role) == validRole {
			return true
		}
	}
	return false
}

// HasPermission checks if the user has a specific permission
func (u *User) HasPermission(permission string) bool {
	// Admin has all permissions
	if u.Role == UserRoleAdmin {
		return true
	}

	// Check specific permissions
	for _, perm := range u.Permissions {
		if perm == permission {
			return true
		}
	}

	return false
}

// IsOTPValid checks if the OTP is still valid (not expired)
func (u *User) IsOTPValid() bool {
	if u.OTPHash == "" || u.OTPExpiresAt == nil {
		return false
	}

	return time.Now().Before(*u.OTPExpiresAt)
}

// SetOTP sets the OTP hash and expiry time (10 minutes from now)
func (u *User) SetOTP(hash string) {
	u.OTPHash = hash
	expiryTime := time.Now().Add(10 * time.Minute)
	u.OTPExpiresAt = &expiryTime
	u.UpdatedAt = time.Now()
}

// ClearOTP clears the OTP hash and expiry time
func (u *User) ClearOTP() {
	u.OTPHash = ""
	u.OTPExpiresAt = nil
	u.UpdatedAt = time.Now()
}

// UpdatePassword updates the user's password hash
func (u *User) UpdatePassword(hash string) {
	u.PasswordHash = hash
	u.UpdatedAt = time.Now()
}

// ToUser converts MongoUser to User (for service layer compatibility)
func (m *User) ToUser() *User1 {
	if m == nil {
		return nil
	}
	return &User1{
		ID:           m.ID,
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		Name:         m.Name,
		Role:         string(m.Role), // Convert UserRole to string
		Region:       m.Region,
		Team:         m.Team,
		Permissions:  m.Permissions,
		IsActive:     m.IsActive,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// ToProfile converts a User to a UserProfile (safe for API responses)
func (u *User) ToProfile() UserProfile {
	return UserProfile{
		ID:          u.ID,
		Email:       u.Email,
		Name:        u.Name,
		Role:        string(u.Role),
		Region:      u.Region,
		Team:        u.Team,
		Permissions: u.Permissions,
	}
}



type TwoFAOTP struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UserID    primitive.ObjectID `bson:"user_id"`
	TempToken primitive.ObjectID `bson:"temp_token"`
	OTPHash   string             `bson:"otp_hash"`
	ExpiresAt time.Time          `bson:"expires_at"`
	Used      bool               `bson:"used"`
	CreatedAt time.Time          `bson:"created_at"`
}