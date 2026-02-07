package models

import (
	"time"
)

// UserSettings represents complete user settings and preferences
type UserSettings struct {
	UserID string `json:"user_id"`

	// User Profile Information
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	Region      string   `json:"region"`
	Team        string   `json:"team"`
	Permissions []string `json:"permissions"`

	// Preferences
	Language           string `json:"language"`
	Timezone           string `json:"timezone"`
	DateFormat         string `json:"date_format"`
	TimeFormat         string `json:"time_format"`
	Currency           string `json:"currency"`
	EmailNotifications bool   `json:"email_notifications"`
	PushNotifications  bool   `json:"push_notifications"`
	SMSNotifications   bool   `json:"sms_notifications"`
	Theme              string `json:"theme"`            // light, dark
	DashboardLayout    string `json:"dashboard_layout"` // JSON config
	DefaultView        string `json:"default_view"`
	ItemsPerPage       int    `json:"items_per_page"`

	UpdatedAt time.Time `json:"updated_at"`
}

type UpdateProfileRequest struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}


// DefaultUserPreferences returns default preferences for new users
func DefaultUserPreferences(userID string) *UserPreferences {
	return &UserPreferences{
		UserID:             userID,
		Language:           "en",
		Timezone:           "Asia/Kolkata",
		DateFormat:         "DD/MM/YYYY",
		TimeFormat:         "24h",
		Currency:           "INR",
		EmailNotifications: true,
		PushNotifications:  true,
		SMSNotifications:   false,
		Theme:              "light",
		DashboardLayout:    "{}",
		DefaultView:        "dashboard",
		ItemsPerPage:       50,
		UpdatedAt:          time.Now(),
	}
}
