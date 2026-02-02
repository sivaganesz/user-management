package events

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/white/user-management/pkg/kafka"
)

// AuditEventTopic returns the configured Kafka topic for audit log events
// This uses kafka.TopicAudit which supports environment-based prefixes
// func AuditEventTopic() string {
// 	return kafka.TopicAudit
// }

// AuditAction represents the type of action being audited
type AuditAction string

const (
	// Authentication actions
	ActionLogin                  AuditAction = "LOGIN"
	ActionLogout                 AuditAction = "LOGOUT"
	ActionLoginFailed            AuditAction = "LOGIN_FAILED"
	ActionPasswordReset          AuditAction = "PASSWORD_RESET"
	ActionPasswordChanged        AuditAction = "PASSWORD_CHANGED"
	ActionPasswordResetRequested AuditAction = "PASSWORD_RESET_REQUESTED"
	Action2FAEnabled             AuditAction = "2FA_ENABLED"
	Action2FADisabled            AuditAction = "2FA_DISABLED"

	// Communication actions
	ActionEmailSent     AuditAction = "EMAIL_SENT"
	ActionEmailReceived AuditAction = "EMAIL_RECEIVED"
	ActionSMSSent       AuditAction = "SMS_SENT"
	ActionCallLogged    AuditAction = "CALL_LOGGED"

	// Campaign actions
	ActionCampaignCreated   AuditAction = "CAMPAIGN_CREATED"
	ActionCampaignUpdated   AuditAction = "CAMPAIGN_UPDATED"
	ActionCampaignDeleted   AuditAction = "CAMPAIGN_DELETED"
	ActionCampaignLaunched  AuditAction = "CAMPAIGN_LAUNCHED"
	ActionCampaignPaused    AuditAction = "CAMPAIGN_PAUSED"
	ActionCampaignCompleted AuditAction = "CAMPAIGN_COMPLETED"

	// Template actions
	ActionTemplateCreated AuditAction = "TEMPLATE_CREATED"
	ActionTemplateUpdated AuditAction = "TEMPLATE_UPDATED"
	ActionTemplateDeleted AuditAction = "TEMPLATE_DELETED"

	// Settings actions
	ActionSettingsUpdated       AuditAction = "SETTINGS_UPDATED"
	ActionCompanyInfoUpdated    AuditAction = "COMPANY_INFO_UPDATED"
	ActionTeamMemberAdded       AuditAction = "TEAM_MEMBER_ADDED"
	ActionTeamMemberRemoved     AuditAction = "TEAM_MEMBER_REMOVED"
	ActionTeamMemberUpdated     AuditAction = "TEAM_MEMBER_UPDATED"
	ActionTeamMemberActivated   AuditAction = "TEAM_MEMBER_ACTIVATED"
	ActionTeamMemberDeactivated AuditAction = "TEAM_MEMBER_DEACTIVATED"
	ActionRoleChanged           AuditAction = "ROLE_CHANGED"

	// Document actions
	ActionDocumentUploaded AuditAction = "DOCUMENT_UPLOADED"
	ActionDocumentDeleted  AuditAction = "DOCUMENT_DELETED"
	ActionDocumentShared   AuditAction = "DOCUMENT_SHARED"

	// RBAC actions
	ActionRoleCreated            AuditAction = "ROLE_CREATED"
	ActionRoleDeleted            AuditAction = "ROLE_DELETED"
	ActionRolePermissionsUpdated AuditAction = "ROLE_PERMISSIONS_UPDATED"
)

// AuditResource represents the type of resource being audited
type AuditResource string

const (
	ResourceAuth     AuditResource = "AUTH"
	ResourceUser     AuditResource = "USER"
	ResourceCampaign AuditResource = "CAMPAIGN"
	ResourceTemplate AuditResource = "TEMPLATE"
	ResourceSettings AuditResource = "SETTINGS"
	ResourceTeam     AuditResource = "TEAM"
	ResourceRole     AuditResource = "ROLE"
	ResourceAdmin    AuditResource = "ADMIN"
	ResourceSession  AuditResource = "SESSION"
	ResourceWorkflow AuditResource = "WORKFLOW"
)

// AuditEvent represents an audit log event published to Kafka
type AuditEvent struct {
	EventID    string                 `json:"event_id"`
	Timestamp  int64                  `json:"timestamp"`
	UserID     string                 `json:"user_id"`
	UserName   string                 `json:"user_name"`
	UserEmail  string                 `json:"user_email,omitempty"`
	Action     AuditAction            `json:"action"`
	Resource   AuditResource          `json:"resource"`
	ResourceID string                 `json:"resource_id,omitempty"`
	Details    string                 `json:"details"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	SessionID  string                 `json:"session_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	OldValue   string                 `json:"old_value,omitempty"`
	NewValue   string                 `json:"new_value,omitempty"`
	Success    bool                   `json:"success"`
	ErrorMsg   string                 `json:"error_msg,omitempty"`
}

// AuditPublisher handles publishing audit events to Kafka
type AuditPublisher struct {
	producer *kafka.Producer
	enabled  bool
}

// NewAuditPublisher creates a new audit publisher
func NewAuditPublisher(producer *kafka.Producer) *AuditPublisher {
	enabled := producer != nil
	if enabled {
		log.Println("Audit event publisher initialized (Kafka enabled)")
	} else {
		log.Println("Audit event publisher initialized (Kafka disabled - events will be logged only)")
	}
	return &AuditPublisher{
		producer: producer,
		enabled:  enabled,
	}
}

// Publish sends an audit event to Kafka (fire-and-forget)
func (p *AuditPublisher) Publish(event *AuditEvent) {
	ctx := context.Background()
	// Set defaults
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}

	// Always log the event for debugging
	eventJSON, _ := json.Marshal(event)
	log.Printf("AUDIT: %s", string(eventJSON))

	// If Kafka is not available, just log
	if !p.enabled || p.producer == nil {
		return
	}

	// Fire-and-forget publish to Kafka
	go func() {
		if err := p.producer.PublishJSON(ctx, "audit.events", event); err != nil {
			log.Printf("Failed to publish audit event: %v", err)
		}
	}()
}

// PublishFromRequest creates and publishes an audit event from HTTP request context
func (p *AuditPublisher) PublishFromRequest(
	r *http.Request,
	userID, userName, userEmail string,
	action AuditAction,
	resource AuditResource,
	resourceID string,
	details string,
	success bool,
	errorMsg string,
	metadata map[string]interface{},
) {
	event := &AuditEvent{
		EventID:    uuid.New().String(),
		Timestamp:  time.Now().Unix(),
		UserID:     userID,
		UserName:   userName,
		UserEmail:  userEmail,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
		IPAddress:  getClientIP(r),
		UserAgent:  r.UserAgent(),
		Metadata:   metadata,
		Success:    success,
		ErrorMsg:   errorMsg,
	}
	p.Publish(event)
}

// Helper to get client IP address
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// Convenience methods for common audit events

// PublishAuthEvent publishes an authentication-related audit event
func (p *AuditPublisher) PublishAuthEvent(r *http.Request, userID, userName, email string, action AuditAction, success bool, details string) {
	p.PublishFromRequest(r, userID, userName, email, action, ResourceAuth, "", details, success, "", nil)
}

// PublishSettingsEvent publishes a settings-related audit event
func (p *AuditPublisher) PublishSettingsEvent(r *http.Request, userID, userName string, action AuditAction, details string) {
	p.PublishFromRequest(r, userID, userName, "", action, ResourceSettings, "", details, true, "", nil)
}

// PublishTeamEvent publishes a team-related audit event
func (p *AuditPublisher) PublishTeamEvent(r *http.Request, userID, userName string, action AuditAction, targetUserID, details string) {
	p.PublishFromRequest(r, userID, userName, "", action, ResourceTeam, targetUserID, details, true, "", nil)
}

// PublishCampaignEvent publishes a campaign-related audit event
func (p *AuditPublisher) PublishCampaignEvent(r *http.Request, userID, userName string, action AuditAction, campaignID, details string, metadata map[string]interface{}) {
	p.PublishFromRequest(r, userID, userName, "", action, ResourceCampaign, campaignID, details, true, "", metadata)
}

// PublishTemplateEvent publishes a template-related audit event
func (p *AuditPublisher) PublishTemplateEvent(r *http.Request, userID, userName string, action AuditAction, templateID, details string) {
	p.PublishFromRequest(r, userID, userName, "", action, ResourceTemplate, templateID, details, true, "", nil)
}

// PublishSessionEvent publishes a session-related audit event
func (p *AuditPublisher) PublishSessionEvent(r *http.Request, userID, userName string, action AuditAction, sessionID, targetUserID, details string) {
	metadata := map[string]interface{}{
		"target_user_id": targetUserID,
	}
	p.PublishFromRequest(r, userID, userName, "", action, ResourceSession, sessionID, details, true, "", metadata)
}

// PublishRoleEvent publishes a role/RBAC-related audit event
func (p *AuditPublisher) PublishRoleEvent(r *http.Request, userID, userName string, action AuditAction, roleCode, details string, metadata map[string]interface{}) {
	p.PublishFromRequest(r, userID, userName, "", action, ResourceRole, roleCode, details, true, "", metadata)
}

// PublishAdminEvent publishes an admin action audit event (bulk operations, data management)
func (p *AuditPublisher) PublishAdminEvent(r *http.Request, userID, userName string, action AuditAction, details string, metadata map[string]interface{}) {
	p.PublishFromRequest(r, userID, userName, "", action, ResourceAdmin, "", details, true, "", metadata)
}
