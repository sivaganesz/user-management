
package models

import (
	"time"
)

// UserActivityLog represents a user activity for audit trail
type UserActivityLog struct {
	UserID       string `json:"user_id"`
	ActivityID   string `json:"activity_id"`
	ActivityType string             `json:"activity_type"`
	Description  string             `json:"description"`
	IPAddress    string             `json:"ip_address,omitempty"`
	UserAgent    string             `json:"user_agent,omitempty"`
	Metadata     string             `json:"metadata,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
}

// UserActivityType constants
const (
	ActivityTypeLogin            = "login"
	ActivityTypeLogout           = "logout"
	ActivityTypePasswordChange   = "password_change"
	ActivityTypeProfileUpdate    = "profile_update"
	ActivityTypePermissionChange = "permission_change"
	ActivityTypeRoleChange       = "role_change"
	ActivityTypeUserStatusChange = "user_status_change"
)

// UserSession represents an active user session
type UserSession struct {
	SessionID      string `json:"session_id"`
	UserID         string `json:"user_id"`
	RefreshToken   string             `json:"-"` // Never expose in JSON
	IPAddress      string             `json:"ip_address"`
	UserAgent      string             `json:"user_agent"`
	DeviceType     string             `json:"device_type"`
	Location       string             `json:"location,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	ExpiresAt      time.Time          `json:"expires_at"`
	LastActivityAt time.Time          `json:"last_activity_at"`
	IsActive       bool               `json:"is_active"`
}

// MongoUserPreferences stores user-specific preferences (MongoDB version)
type MongoUserPreferences struct {
	Language        string `bson:"language,omitempty" json:"language,omitempty"`
	Timezone        string `bson:"timezone,omitempty" json:"timezone,omitempty"`
	DateFormat      string `bson:"date_format,omitempty" json:"dateFormat,omitempty"`
	TimeFormat      string `bson:"time_format,omitempty" json:"timeFormat,omitempty"`
	Theme           string `bson:"theme,omitempty" json:"theme,omitempty"`
	DashboardLayout string `bson:"dashboard_layout,omitempty" json:"dashboardLayout,omitempty"`
}

// UserPreferences represents user-specific preferences
type UserPreferences struct {
	UserID             string `json:"user_id"`
	Language           string             `json:"language"`
	Timezone           string             `json:"timezone"`
	DateFormat         string             `json:"date_format"`
	TimeFormat         string             `json:"time_format"`
	Currency           string             `json:"currency"`
	EmailNotifications bool               `json:"email_notifications"`
	PushNotifications  bool               `json:"push_notifications"`
	SMSNotifications   bool               `json:"sms_notifications"`
	Theme              string             `json:"theme"`
	DashboardLayout    string             `json:"dashboard_layout,omitempty"`
	DefaultView        string             `json:"default_view,omitempty"`
	ItemsPerPage       int                `json:"items_per_page"`
	UpdatedAt          time.Time          `json:"updated_at"`
}