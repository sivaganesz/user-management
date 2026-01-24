package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SettingsUserProfile represents user profile settings for the settings page
type SettingsUserProfile struct {
	ID        string             `bson:"_id,omitempty" json:"id"`
	UserID    string             `bson:"user_id" json:"userId"`
	FirstName string             `bson:"first_name" json:"firstName"`
	LastName  string             `bson:"last_name" json:"lastName"`
	Email     string             `bson:"email" json:"email"`
	Phone     string             `bson:"phone,omitempty" json:"phone,omitempty"`
	JobTitle  string             `bson:"job_title,omitempty" json:"jobTitle,omitempty"`
	Region    string             `bson:"region,omitempty" json:"region,omitempty"`
	Avatar    string             `bson:"avatar,omitempty" json:"avatar,omitempty"`
	CreatedAt time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updatedAt"`
}

// SettingsUpdateProfileRequest represents a profile update request
type SettingsUpdateProfileRequest struct {
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Phone     string `json:"phone,omitempty"`
	JobTitle  string `json:"jobTitle,omitempty"`
	Region    string `json:"region,omitempty"`
	Avatar    string `json:"avatar,omitempty"`
}

// SettingsEmailSignature represents user's email signature settings
type SettingsEmailSignature struct {
	ID        string			 `bson:"_id,omitempty" json:"id"`
	UserID    string			 `bson:"user_id" json:"userId"`
	Signature string             `bson:"signature" json:"signature"`
	Enabled   bool               `bson:"enabled" json:"enabled"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updatedAt"`
}

// SettingsUpdateEmailSignatureRequest represents an email signature update request
type SettingsUpdateEmailSignatureRequest struct {
	Signature string `json:"signature"`
	Enabled   bool   `json:"enabled"`
}

// SettingsUserSecuritySettings represents user's security settings
type SettingsUserSecuritySettings struct {
	ID                 string             `bson:"_id,omitempty" json:"id"`
	UserID             string             `bson:"user_id" json:"userId"`
	TwoFactorEnabled   bool               `bson:"two_factor_enabled" json:"twoFactorEnabled"`
	SessionTimeout     int                `bson:"session_timeout" json:"sessionTimeout"` // minutes
	LastPasswordChange *time.Time         `bson:"last_password_change,omitempty" json:"lastPasswordChange,omitempty"`
	UpdatedAt          time.Time          `bson:"updated_at" json:"updatedAt"`
}

// SettingsUpdateSecuritySettingsRequest represents a security settings update request
type SettingsUpdateSecuritySettingsRequest struct {
	TwoFactorEnabled *bool `json:"twoFactorEnabled,omitempty"`
	SessionTimeout   *int  `json:"sessionTimeout,omitempty"`
}

// SettingsChangePasswordRequest represents a password change request
type SettingsChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
}

// SettingsEmailPreferences represents email communication preferences
type SettingsEmailPreferences struct {
	EnableTracking     bool `bson:"enable_tracking" json:"enableTracking"`
	EnableReadReceipts bool `bson:"enable_read_receipts" json:"enableReadReceipts"`
	AutoFollowUpDays   int  `bson:"auto_follow_up_days,omitempty" json:"autoFollowUpDays,omitempty"`
}

// SettingsWhatsAppPreferences represents WhatsApp communication preferences
type SettingsWhatsAppPreferences struct {
	UseTemplates    bool   `bson:"use_templates" json:"useTemplates"`
	DefaultTemplate string `bson:"default_template,omitempty" json:"defaultTemplate,omitempty"`
}

// SettingsCommunicationPreferences represents user's communication preferences
type SettingsCommunicationPreferences struct {
	ID                  string 			              `bson:"_id,omitempty" json:"id"`
	UserID              string 			              `bson:"user_id" json:"userId"`
	DefaultChannels     []string                      `bson:"default_channels" json:"defaultChannels"`
	EmailPreferences    SettingsEmailPreferences      `bson:"email_preferences" json:"emailPreferences"`
	WhatsAppPreferences SettingsWhatsAppPreferences   `bson:"whatsapp_preferences" json:"whatsappPreferences"`
	UpdatedAt           time.Time                     `bson:"updated_at" json:"updatedAt"`
}

// SettingsUpdateCommunicationPreferencesRequest represents a communication preferences update request
type SettingsUpdateCommunicationPreferencesRequest struct {
	DefaultChannels     []string                       `json:"defaultChannels,omitempty"`
	EmailPreferences    *SettingsEmailPreferences      `json:"emailPreferences,omitempty"`
	WhatsAppPreferences *SettingsWhatsAppPreferences   `json:"whatsappPreferences,omitempty"`
}

// SettingsCompanyInfo represents company settings
type SettingsCompanyInfo struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`
	Logo      string             `bson:"logo,omitempty" json:"logo,omitempty"`
	Industry  string             `bson:"industry,omitempty" json:"industry,omitempty"`
	Size      string             `bson:"size,omitempty" json:"size,omitempty"`
	Website   string             `bson:"website,omitempty" json:"website,omitempty"`
	Address   string             `bson:"address,omitempty" json:"address,omitempty"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updatedAt"`
}

// SettingsUpdateCompanyInfoRequest represents a company info update request
type SettingsUpdateCompanyInfoRequest struct {
	Name     string `json:"name,omitempty"`
	Logo     string `json:"logo,omitempty"`
	Industry string `json:"industry,omitempty"`
	Size     string `json:"size,omitempty"`
	Website  string `json:"website,omitempty"`
	Address  string `json:"address,omitempty"`
}

// SettingsEmailNotificationSettings represents email notification preferences
type SettingsEmailNotificationSettings struct {
	TaskReminder    bool `bson:"task_reminder" json:"taskReminder"`
	WeeklyReport    bool `bson:"weekly_report" json:"weeklyReport"`
}

// SettingsBrowserNotificationSettings represents browser notification preferences
type SettingsBrowserNotificationSettings struct {
	Enabled    bool `bson:"enabled" json:"enabled"`
	NewMessage bool `bson:"new_message" json:"newMessage"`
	TaskDue    bool `bson:"task_due" json:"taskDue"`
}

// SettingsNotificationSettings represents user's notification settings
type SettingsNotificationSettings struct {
	ID                   string                  `bson:"_id,omitempty" json:"id"`
	UserID               string                  `bson:"user_id" json:"userId"`
	EmailNotifications   SettingsEmailNotificationSettings   `bson:"email_notifications" json:"emailNotifications"`
	BrowserNotifications SettingsBrowserNotificationSettings `bson:"browser_notifications" json:"browserNotifications"`
	UpdatedAt            time.Time                           `bson:"updated_at" json:"updatedAt"`
}

// SettingsUpdateNotificationSettingsRequest represents a notification settings update request
type SettingsUpdateNotificationSettingsRequest struct {
	EmailNotifications   *SettingsEmailNotificationSettings   `json:"emailNotifications,omitempty"`
	BrowserNotifications *SettingsBrowserNotificationSettings `json:"browserNotifications,omitempty"`
}

// SettingsAuditLog represents an audit log entry for the settings page
type SettingsAuditLog struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Timestamp time.Time          `bson:"timestamp" json:"timestamp"`
	UserID    primitive.ObjectID `bson:"user_id" json:"userId"`
	UserName  string             `bson:"user_name" json:"userName"`
	Action    string             `bson:"action" json:"action"`
	Resource  string             `bson:"resource" json:"resource"`
	Details   string             `bson:"details" json:"details"`
	IPAddress string             `bson:"ip_address,omitempty" json:"ipAddress,omitempty"`
}

// ==================== System Default Settings ====================

// SystemDefaultSettings represents system-wide default settings
type SystemDefaultSettings struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Timezone        string             `bson:"timezone" json:"timezone"`
	Currency        string             `bson:"currency" json:"currency"`
	Language        string             `bson:"language" json:"language"`
	DateFormat      string             `bson:"date_format" json:"dateFormat"`
	WorkingHoursStart string           `bson:"working_hours_start" json:"workingHoursStart"`
	WorkingHoursEnd   string           `bson:"working_hours_end" json:"workingHoursEnd"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updatedAt"`
}

// UpdateSystemDefaultSettingsRequest represents an update request for system default settings
type UpdateSystemDefaultSettingsRequest struct {
	Timezone        string `json:"timezone,omitempty"`
	Currency        string `json:"currency,omitempty"`
	Language        string `json:"language,omitempty"`
	DateFormat      string `json:"dateFormat,omitempty"`
	WorkingHoursStart string `json:"workingHoursStart,omitempty"`
	WorkingHoursEnd   string `json:"workingHoursEnd,omitempty"`
}

// ==================== System Security Settings ====================

// SystemSecuritySettings represents system-wide security settings
type SystemSecuritySettings struct {
	ID                     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TwoFactorRequired      bool               `bson:"two_factor_required" json:"twoFactorRequired"`
	MinPasswordLength      int                `bson:"min_password_length" json:"minPasswordLength"`
	PasswordExpiryDays     int                `bson:"password_expiry_days" json:"passwordExpiryDays"`
	RequireSpecialChars    bool               `bson:"require_special_chars" json:"requireSpecialChars"`
	SessionTimeoutMinutes  int                `bson:"session_timeout_minutes" json:"sessionTimeoutMinutes"`
	IPWhitelist            string             `bson:"ip_whitelist,omitempty" json:"ipWhitelist,omitempty"`
	SSOEnabled             bool               `bson:"sso_enabled" json:"ssoEnabled"`
	UpdatedAt              time.Time          `bson:"updated_at" json:"updatedAt"`
}

// UpdateSystemSecuritySettingsRequest represents an update request for system security settings
type UpdateSystemSecuritySettingsRequest struct {
	TwoFactorRequired      *bool   `json:"twoFactorRequired,omitempty"`
	MinPasswordLength      *int    `json:"minPasswordLength,omitempty"`
	PasswordExpiryDays     *int    `json:"passwordExpiryDays,omitempty"`
	RequireSpecialChars    *bool   `json:"requireSpecialChars,omitempty"`
	SessionTimeoutMinutes  *int    `json:"sessionTimeoutMinutes,omitempty"`
	IPWhitelist            *string `json:"ipWhitelist,omitempty"`
	SSOEnabled             *bool   `json:"ssoEnabled,omitempty"`
}

// ==================== Data & Privacy Settings ====================

// DataPrivacySettings represents system-wide data and privacy settings
type DataPrivacySettings struct {
	ID                     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	DataRetentionDays      int                `bson:"data_retention_days" json:"dataRetentionDays"`
	AutomaticDataCleanup   bool               `bson:"automatic_data_cleanup" json:"automaticDataCleanup"`
	UpdatedAt              time.Time          `bson:"updated_at" json:"updatedAt"`
}

// UpdateDataPrivacySettingsRequest represents an update request for data privacy settings
type UpdateDataPrivacySettingsRequest struct {
	DataRetentionDays    *int  `json:"dataRetentionDays,omitempty"`
	AutomaticDataCleanup *bool `json:"automaticDataCleanup,omitempty"`
}

// ==================== System Email & Notification Settings ====================

// SystemEmailNotificationSettings represents system-wide email and notification settings
type SystemEmailNotificationSettings struct {
	ID                      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SystemNotificationEmail string             `bson:"system_notification_email" json:"systemNotificationEmail"`
	WeeklyReportSchedule    string             `bson:"weekly_report_schedule" json:"weeklyReportSchedule"`
	EmailSendLimitAlertPercent int             `bson:"email_send_limit_alert_percent" json:"emailSendLimitAlertPercent"`
	UpdatedAt               time.Time          `bson:"updated_at" json:"updatedAt"`
}

// UpdateSystemEmailNotificationSettingsRequest represents an update request for system email notification settings
type UpdateSystemEmailNotificationSettingsRequest struct {
	SystemNotificationEmail    *string `json:"systemNotificationEmail,omitempty"`
	WeeklyReportSchedule       *string `json:"weeklyReportSchedule,omitempty"`
	EmailSendLimitAlertPercent *int    `json:"emailSendLimitAlertPercent,omitempty"`
}
