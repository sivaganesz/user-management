package repositories

import(
	"context"
	"time"
	// "fmt"

	"github.com/white/user-management/internal/models"
	"github.com/white/user-management/pkg/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SettingsRepository struct {
	client 						*mongodb.Client
	userProfiles 				*mongo.Collection
	emailSignatures 			*mongo.Collection
	securitySettings 			*mongo.Collection
	communicationPrefs          *mongo.Collection
	companyInfo                 *mongo.Collection
	notificationSettings        *mongo.Collection
	auditLogs                   *mongo.Collection
	users                       *mongo.Collection
	systemDefaults              *mongo.Collection
	systemSecurity              *mongo.Collection
	dataPrivacy                 *mongo.Collection
	systemEmailNotifications    *mongo.Collection
}

func NewSettingsRepository(client *mongodb.Client) *SettingsRepository {
	return &SettingsRepository{
		client:                   client,
		userProfiles:             client.Collection("user_profiles"),
		emailSignatures:          client.Collection("email_signatures"),
		securitySettings:         client.Collection("security_settings"),
		communicationPrefs:       client.Collection("communication_preferences"),
		companyInfo:              client.Collection("company_info"),
		notificationSettings:     client.Collection("notification_settings"),
		auditLogs:                client.Collection("audit_logs"),
		users:                    client.Collection("users"),
		systemDefaults:           client.Collection("system_defaults"),
		systemSecurity:           client.Collection("system_security"),
		dataPrivacy:              client.Collection("data_privacy"),
		systemEmailNotifications: client.Collection("system_email_notifications"),
	}
}

// ==================== User Profile ====================

// GetUserProfile retrieves user profile by user ID
func (r *SettingsRepository) GetUserProfile(ctx context.Context, userID primitive.ObjectID) (*models.SettingsUserProfile, error) {
	// First check if a profile exists in user_profiles collection
	var profile models.SettingsUserProfile
	err := r.userProfiles.FindOne(ctx, bson.M{"user_id": userID}).Decode(&profile)
	if err == nil {
		return &profile, nil
	}

	// If not found, create from users collection
	if err == mongo.ErrNoDocuments {
		var user models.MongoUser
		err = r.users.FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
		if err != nil {
			return nil, err
		}

		// Create profile from user data
		profile = models.SettingsUserProfile{
			ID:        primitive.NewObjectID(),
			UserID:    userID,
			FirstName: user.Name, // Use full name as first name for now
			LastName:  "",
			Email:     user.Email,
			Region:    user.Region,
			CreatedAt: user.CreatedAt,
			UpdatedAt: time.Now(),
		}

		// Save the profile
		_, err = r.userProfiles.InsertOne(ctx, profile)
		if err != nil {
			return nil, err
		}

		return &profile, nil
	}

	return nil, err
}

// UpdateUserProfile updates user profile
func (r *SettingsRepository) UpdateUserProfile(ctx context.Context, userId primitive.ObjectID, update models.SettingsUpdateProfileRequest) (*models.SettingsUserProfile, error) {
	filter := bson.M{"user_id": userId}
	updateDoc := bson.M{"$set": bson.M{"updated_at": time.Now()}}

	if update.FirstName != "" {
		updateDoc["$set"].(bson.M)["first_name"] = update.FirstName
	}
	if update.LastName != "" {
		updateDoc["$set"].(bson.M)["last_name"] = update.LastName
	}
	if update.Phone != "" {
		updateDoc["$set"].(bson.M)["phone"] = update.Phone
	}
	if update.JobTitle != "" {
		updateDoc["$set"].(bson.M)["job_title"] = update.JobTitle
	}
	if update.Region != "" {
		updateDoc["$set"].(bson.M)["region"] = update.Region
	}
	if update.Avatar != "" {
		updateDoc["$set"].(bson.M)["avatar"] = update.Avatar
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)
	var profile models.SettingsUserProfile
	err := r.userProfiles.FindOneAndUpdate(ctx, filter, updateDoc, opts).Decode(&profile)
	if err != nil {
		return nil, err
	}

	return &profile, nil
}

// ==================== Email Signature ====================

// GetEmailSignature retrieves email signature by user ID
func (r *SettingsRepository) GetEmailSignature(ctx context.Context, userID primitive.ObjectID) (*models.SettingsEmailSignature, error) {
	var signature models.SettingsEmailSignature
	err := r.emailSignatures.FindOne(ctx, bson.M{"user_id": userID}).Decode(&signature)
	if err == mongo.ErrNoDocuments {
		// Return default signature
		return &models.SettingsEmailSignature{
			UserID:    userID,
			Signature: "",
			Enabled:   false,
			UpdatedAt: time.Now(),
		}, nil
	}
	return &signature, err
}

// UpdateEmailSignature updates email signature
func (r *SettingsRepository) UpdateEmailSignature(ctx context.Context, userID primitive.ObjectID, update models.SettingsUpdateEmailSignatureRequest) (*models.SettingsEmailSignature, error) {
	filter := bson.M{"user_id": userID}
	updateDoc := bson.M{
		"$set": bson.M{
			"signature":  update.Signature,
			"enabled":    update.Enabled,
			"updated_at": time.Now(),
		},
		"$setOnInsert": bson.M{
			"user_id": userID,
		},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)
	var signature models.SettingsEmailSignature
	err := r.emailSignatures.FindOneAndUpdate(ctx, filter, updateDoc, opts).Decode(&signature)
	if err != nil {
		return nil, err
	}
	return &signature, nil
}

// ==================== Security Settings ====================
// GetSecuritySettings retrieves security settings by user ID
func (r *SettingsRepository) GetSecuritySettings(ctx context.Context, userID primitive.ObjectID) (*models.SettingsUserSecuritySettings, error) {
	var settings models.SettingsUserSecuritySettings
	err := r.securitySettings.FindOne(ctx, bson.M{"user_id": userID}).Decode(&settings)
	if err == mongo.ErrNoDocuments {
		// Return default settings
		return &models.SettingsUserSecuritySettings{
			UserID:           userID,
			TwoFactorEnabled: false,
			SessionTimeout:   30, // 30 minutes default
			UpdatedAt:        time.Now(),
		}, nil
	}
	return &settings, err
}

// UpdateSecuritySettings updates security settings
func (r *SettingsRepository) UpdateSecuritySettings(ctx context.Context, userID primitive.ObjectID, update models.SettingsUpdateSecuritySettingsRequest) (*models.SettingsUserSecuritySettings, error) {
	filter := bson.M{"user_id": userID}
	updateDoc := bson.M{"$set": bson.M{"updated_at": time.Now()}}

	if update.TwoFactorEnabled != nil {
		updateDoc["$set"].(bson.M)["two_factor_enabled"] = *update.TwoFactorEnabled
	}
	if update.SessionTimeout != nil {
		updateDoc["$set"].(bson.M)["session_timeout"] = *update.SessionTimeout
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)
	var settings models.SettingsUserSecuritySettings
	err := r.securitySettings.FindOneAndUpdate(ctx, filter, updateDoc, opts).Decode(&settings)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

// UpdateLastPasswordChange updates the last password change timestamp
func (r *SettingsRepository) UpdateLastPasswordChange(ctx context.Context, userID primitive.ObjectID) error {
	filter := bson.M{"user_id": userID}
	now := time.Now()
	updateDoc := bson.M{
		"$set": bson.M{
			"last_password_change": now,
			"updated_at":           now,
		},
		"$setOnInsert": bson.M{
			"user_id": userID,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.securitySettings.UpdateOne(ctx, filter, updateDoc, opts)
	return err
}

// ==================== Communication Preferences ====================

// GetCommunicationPreferences retrieves communication preferences by user ID
func (r *SettingsRepository) GetCommunicationPreferences(ctx context.Context, userID primitive.ObjectID) (*models.SettingsCommunicationPreferences, error) {
	var prefs models.SettingsCommunicationPreferences
	err := r.communicationPrefs.FindOne(ctx, bson.M{"user_id": userID}).Decode(&prefs)
	if err == mongo.ErrNoDocuments {
		// Return default preferences
		return &models.SettingsCommunicationPreferences{
			UserID:          userID,
			DefaultChannels: []string{"email"},
			EmailPreferences: models.SettingsEmailPreferences{
				EnableTracking:     true,
				EnableReadReceipts: true,
				AutoFollowUpDays:   3,
			},
			WhatsAppPreferences: models.SettingsWhatsAppPreferences{
				UseTemplates: true,
			},
			UpdatedAt: time.Now(),
		}, nil
	}
	return &prefs, err
}

// UpdateCommunicationPreferences updates communication preferences
func (r *SettingsRepository) UpdateCommunicationPreferences(ctx context.Context, userID primitive.ObjectID, update *models.SettingsUpdateCommunicationPreferencesRequest) (*models.SettingsCommunicationPreferences, error) {
	filter := bson.M{"user_id": userID}
	setFields := bson.M{"updated_at": time.Now()}

	if update.DefaultChannels != nil {
		setFields["default_channels"] = update.DefaultChannels
	}
	if update.EmailPreferences != nil {
		setFields["email_preferences"] = update.EmailPreferences
	}
	if update.WhatsAppPreferences != nil {
		setFields["whatsapp_preferences"] = update.WhatsAppPreferences
	}

	updateDoc := bson.M{
		"$set": setFields,
		"$setOnInsert": bson.M{
			"user_id": userID,
		},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)
	var prefs models.SettingsCommunicationPreferences
	err := r.communicationPrefs.FindOneAndUpdate(ctx, filter, updateDoc, opts).Decode(&prefs)
	if err != nil {
		return nil, err
	}

	return &prefs, nil
}

// ==================== Company Info ====================

// GetCompanyInfo retrieves company info (singleton)
func (r *SettingsRepository) GetCompanyInfo(ctx context.Context) (*models.SettingsCompanyInfo, error) {
	var info models.SettingsCompanyInfo
	err := r.companyInfo.FindOne(ctx, bson.M{}).Decode(&info)
	if err == mongo.ErrNoDocuments {
		// Return default company info
		return &models.SettingsCompanyInfo{
			ID:        primitive.NewObjectID(),
			Name:      "SkillMine Technologies",
			Industry:  "Technology",
			Size:      "50-200",
			Website:   "https://skillmine.com",
			UpdatedAt: time.Now(),
		}, nil
	}
	return &info, err
}

// UpdateCompanyInfo updates company info
func (r *SettingsRepository) UpdateCompanyInfo(ctx context.Context, update *models.SettingsUpdateCompanyInfoRequest) (*models.SettingsCompanyInfo, error) {
	filter := bson.M{}
	setFields := bson.M{"updated_at": time.Now()}

	if update.Name != "" {
		setFields["name"] = update.Name
	}
	if update.Logo != "" {
		setFields["logo"] = update.Logo
	}
	if update.Industry != "" {
		setFields["industry"] = update.Industry
	}
	if update.Size != "" {
		setFields["size"] = update.Size
	}
	if update.Website != "" {
		setFields["website"] = update.Website
	}
	if update.Address != "" {
		setFields["address"] = update.Address
	}

	updateDoc := bson.M{"$set": setFields}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)
	var info models.SettingsCompanyInfo
	err := r.companyInfo.FindOneAndUpdate(ctx, filter, updateDoc, opts).Decode(&info)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

// ==================== Notification Settings ====================

// GetNotificationSettings retrieves notification settings by user ID
func (r *SettingsRepository) GetNotificationSettings(ctx context.Context, userID primitive.ObjectID) (*models.SettingsNotificationSettings, error) {
	var settings models.SettingsNotificationSettings
	err := r.notificationSettings.FindOne(ctx, bson.M{"user_id": userID}).Decode(&settings)
	if err == mongo.ErrNoDocuments {
		// Return default settings
		return &models.SettingsNotificationSettings{
			UserID: userID,
			EmailNotifications: models.SettingsEmailNotificationSettings{
				TaskReminder:    true,
				WeeklyReport:    true,
			},
			BrowserNotifications: models.SettingsBrowserNotificationSettings{
				Enabled:    true,
				NewMessage: true,
				TaskDue:    true,
			},
			UpdatedAt: time.Now(),
		}, nil
	}
	return &settings, err
}


// UpdateNotificationSettings updates notification settings
func (r *SettingsRepository) UpdateNotificationSettings(ctx context.Context, userID primitive.ObjectID, update *models.SettingsUpdateNotificationSettingsRequest) (*models.SettingsNotificationSettings, error) {
	filter := bson.M{"user_id": userID}
	setFields := bson.M{"updated_at": time.Now()}

	if update.EmailNotifications != nil {
		setFields["email_notifications"] = update.EmailNotifications
	}
	if update.BrowserNotifications != nil {
		setFields["browser_notifications"] = update.BrowserNotifications
	}

	updateDoc := bson.M{
		"$set": setFields,
		"$setOnInsert": bson.M{
			"user_id": userID,
		},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)
	var settings models.SettingsNotificationSettings
	err := r.notificationSettings.FindOneAndUpdate(ctx, filter, updateDoc, opts).Decode(&settings)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

// ==================== Audit Logs ====================

// GetAuditLogs retrieves audit logs with pagination
func (r *SettingsRepository) GetAuditLogs(ctx context.Context, limit, offset int) ([]models.SettingsAuditLog, int64, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// Get total count
	total, err := r.auditLogs.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	// Get logs with pagination
	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.auditLogs.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var logs []models.SettingsAuditLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, 0, err
	}

	if logs == nil {
		logs = []models.SettingsAuditLog{}
	}

	return logs, total, nil
}

// CreateAuditLog creates a new audit log entry
func (r *SettingsRepository) CreateAuditLog(ctx context.Context, log *models.SettingsAuditLog) error {
	log.ID = primitive.NewObjectID()
	log.Timestamp = time.Now()
	_, err := r.auditLogs.InsertOne(ctx, log)
	return err
}

// ==================== System Default Settings ====================

// GetSystemDefaultSettings retrieves system default settings (singleton)
func (r *SettingsRepository) GetSystemDefaultSettings(ctx context.Context) (*models.SystemDefaultSettings, error) {
	var settings models.SystemDefaultSettings
	err := r.systemDefaults.FindOne(ctx, bson.M{}).Decode(&settings)
	if err == mongo.ErrNoDocuments {
		// Return default settings
		return &models.SystemDefaultSettings{
			ID:                primitive.NewObjectID(),
			Timezone:          "Asia/Kolkata",
			Currency:          "INR",
			Language:          "english",
			DateFormat:        "dd-mm-yyyy",
			WorkingHoursStart: "9am",
			WorkingHoursEnd:   "6pm",
			UpdatedAt:         time.Now(),
		}, nil
	}
	return &settings, err
}

// UpdateSystemDefaultSettings updates system default settings
func (r *SettingsRepository) UpdateSystemDefaultSettings(ctx context.Context, update *models.UpdateSystemDefaultSettingsRequest) (*models.SystemDefaultSettings, error) {
	filter := bson.M{}
	setFields := bson.M{"updated_at": time.Now()}

	if update.Timezone != "" {
		setFields["timezone"] = update.Timezone
	}
	if update.Currency != "" {
		setFields["currency"] = update.Currency
	}
	if update.Language != "" {
		setFields["language"] = update.Language
	}
	if update.DateFormat != "" {
		setFields["date_format"] = update.DateFormat
	}
	if update.WorkingHoursStart != "" {
		setFields["working_hours_start"] = update.WorkingHoursStart
	}
	if update.WorkingHoursEnd != "" {
		setFields["working_hours_end"] = update.WorkingHoursEnd
	}

	updateDoc := bson.M{"$set": setFields}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)
	var settings models.SystemDefaultSettings
	err := r.systemDefaults.FindOneAndUpdate(ctx, filter, updateDoc, opts).Decode(&settings)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

// ==================== System Security Settings ====================

// GetSystemSecuritySettings retrieves system security settings (singleton)
func (r *SettingsRepository) GetSystemSecuritySettings(ctx context.Context) (*models.SystemSecuritySettings, error) {
	var settings models.SystemSecuritySettings
	err := r.systemSecurity.FindOne(ctx, bson.M{}).Decode(&settings)
	if err == mongo.ErrNoDocuments {
		// Return default settings
		return &models.SystemSecuritySettings{
			ID:                    primitive.NewObjectID(),
			TwoFactorRequired:     true,
			MinPasswordLength:     12,
			PasswordExpiryDays:    90,
			RequireSpecialChars:   true,
			SessionTimeoutMinutes: 30,
			IPWhitelist:           "",
			SSOEnabled:            false,
			UpdatedAt:             time.Now(),
		}, nil
	}
	return &settings, err
}

// UpdateSystemSecuritySettings updates system security settings
func (r *SettingsRepository) UpdateSystemSecuritySettings(ctx context.Context, update *models.UpdateSystemSecuritySettingsRequest) (*models.SystemSecuritySettings, error) {
	filter := bson.M{}
	setFields := bson.M{"updated_at": time.Now()}

	if update.TwoFactorRequired != nil {
		setFields["two_factor_required"] = *update.TwoFactorRequired
	}
	if update.MinPasswordLength != nil {
		setFields["min_password_length"] = *update.MinPasswordLength
	}
	if update.PasswordExpiryDays != nil {
		setFields["password_expiry_days"] = *update.PasswordExpiryDays
	}
	if update.RequireSpecialChars != nil {
		setFields["require_special_chars"] = *update.RequireSpecialChars
	}
	if update.SessionTimeoutMinutes != nil {
		setFields["session_timeout_minutes"] = *update.SessionTimeoutMinutes
	}
	if update.IPWhitelist != nil {
		setFields["ip_whitelist"] = *update.IPWhitelist
	}
	if update.SSOEnabled != nil {
		setFields["sso_enabled"] = *update.SSOEnabled
	}

	updateDoc := bson.M{"$set": setFields}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)
	var settings models.SystemSecuritySettings
	err := r.systemSecurity.FindOneAndUpdate(ctx, filter, updateDoc, opts).Decode(&settings)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

// ==================== Data & Privacy Settings ====================

// GetDataPrivacySettings retrieves data privacy settings (singleton)
func (r *SettingsRepository) GetDataPrivacySettings(ctx context.Context) (*models.DataPrivacySettings, error) {
	var settings models.DataPrivacySettings
	err := r.dataPrivacy.FindOne(ctx, bson.M{}).Decode(&settings)
	if err == mongo.ErrNoDocuments {
		// Return default settings
		return &models.DataPrivacySettings{
			ID:                   primitive.NewObjectID(),
			DataRetentionDays:    90,
			AutomaticDataCleanup: true,
			UpdatedAt:            time.Now(),
		}, nil
	}
	return &settings, err
}

// UpdateDataPrivacySettings updates data privacy settings
func (r *SettingsRepository) UpdateDataPrivacySettings(ctx context.Context, update *models.UpdateDataPrivacySettingsRequest) (*models.DataPrivacySettings, error) {
	filter := bson.M{}
	setFields := bson.M{"updated_at": time.Now()}

	if update.DataRetentionDays != nil {
		setFields["data_retention_days"] = *update.DataRetentionDays
	}
	if update.AutomaticDataCleanup != nil {
		setFields["automatic_data_cleanup"] = *update.AutomaticDataCleanup
	}

	updateDoc := bson.M{"$set": setFields}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)
	var settings models.DataPrivacySettings
	err := r.dataPrivacy.FindOneAndUpdate(ctx, filter, updateDoc, opts).Decode(&settings)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

// ==================== System Email & Notification Settings ====================

// GetSystemEmailNotificationSettings retrieves system email notification settings (singleton)
func (r *SettingsRepository) GetSystemEmailNotificationSettings(ctx context.Context) (*models.SystemEmailNotificationSettings, error) {
	var settings models.SystemEmailNotificationSettings
	err := r.systemEmailNotifications.FindOne(ctx, bson.M{}).Decode(&settings)
	if err == mongo.ErrNoDocuments {
		// Return default settings
		return &models.SystemEmailNotificationSettings{
			ID:                         primitive.NewObjectID(),
			SystemNotificationEmail:    "sivaganesz7482@skillmine.com",
			WeeklyReportSchedule:       "monday",
			EmailSendLimitAlertPercent: 90,
			UpdatedAt:                  time.Now(),
		}, nil
	}
	return &settings, err
}

// UpdateSystemEmailNotificationSettings updates system email notification settings
func (r *SettingsRepository) UpdateSystemEmailNotificationSettings(ctx context.Context, update *models.UpdateSystemEmailNotificationSettingsRequest) (*models.SystemEmailNotificationSettings, error) {
	filter := bson.M{}
	setFields := bson.M{"updated_at": time.Now()}

	if update.SystemNotificationEmail != nil {
		setFields["system_notification_email"] = *update.SystemNotificationEmail
	}
	if update.WeeklyReportSchedule != nil {
		setFields["weekly_report_schedule"] = *update.WeeklyReportSchedule
	}
	if update.EmailSendLimitAlertPercent != nil {
		setFields["email_send_limit_alert_percent"] = *update.EmailSendLimitAlertPercent
	}

	updateDoc := bson.M{"$set": setFields}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)
	var settings models.SystemEmailNotificationSettings
	err := r.systemEmailNotifications.FindOneAndUpdate(ctx, filter, updateDoc, opts).Decode(&settings)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}
