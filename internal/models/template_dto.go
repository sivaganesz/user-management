package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateTemplateRequest represents a request to create a new template
type CreateTemplateRequest struct {
	Name         string            `json:"name" validate:"required,min=1,max=200"`
	Description  string            `json:"description,omitempty"`
	Channel      string            `json:"channel" validate:"required,oneof=email sms whatsapp linkedin"`
	Content      map[string]string `json:"content,omitempty"`
	CustomFields map[string]string `json:"customFields,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	// Frontend-compatible fields (camelCase)
	ForStage     []string `json:"forStage,omitempty"`     // Funnel stages
	Industries   []string `json:"industries,omitempty"`   // Industry targeting 
	ApprovalFlag string   `json:"approvalFlag,omitempty"` // green, yellow, red
	AiEnhanced   bool     `json:"aiEnhanced,omitempty"`   // AI-generated flag
	ServiceID    string   `json:"serviceId,omitempty"`    // ObjectID reference to services collection
	// Channel-specific (camelCase)
	Subject          string `json:"subject,omitempty"`          // Email subject (convenience field)
	Message          string `json:"message,omitempty"`          // Message body (convenience field)
	TemplateType     string `json:"type,omitempty"`             // LinkedIn: Connection Request, InMail Message
	MetaTemplateName string `json:"metaTemplateName,omitempty"` // WhatsApp Meta template name
	Category         string `json:"category,omitempty"`         // WhatsApp category
	Language         string `json:"language,omitempty"`         // WhatsApp language
	// Status
	Status string `json:"status,omitempty"`
}

// UpdateTemplateRequest represents a request to update an existing template
type UpdateTemplateRequest struct {
	Name         string            `json:"name,omitempty" validate:"omitempty,min=1,max=200"`
	Description  string            `json:"description,omitempty"`
	Content      map[string]string `json:"content,omitempty"`
	CustomFields map[string]string `json:"customFields,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	// Frontend-compatible fields (camelCase)
	ForStage     []string `json:"forStage,omitempty"`
	Industries   []string `json:"industries,omitempty"`
	ApprovalFlag string   `json:"approvalFlag,omitempty"`
	AiEnhanced   *bool    `json:"aiEnhanced,omitempty"` // Pointer to distinguish unset from false
	ServiceID    string   `json:"serviceId,omitempty"`  // ObjectID reference to services collection
	// Channel-specific (camelCase)
	Subject          string `json:"subject,omitempty"`
	Message          string `json:"message,omitempty"`
	TemplateType     string `json:"type,omitempty"`
	MetaTemplateName string `json:"metaTemplateName,omitempty"`
	Category         string `json:"category,omitempty"`
	Language         string `json:"language,omitempty"`
	// Status
	Status string `json:"status,omitempty"`
}


// TemplateChannel represents a communication channel
type TemplateChannel string

const (
	TemplateChannelEmail    TemplateChannel = "email"
	TemplateChannelSMS      TemplateChannel = "sms"
	TemplateChannelWhatsApp TemplateChannel = "whatsapp"
	TemplateChannelLinkedIn TemplateChannel = "linkedin"
)

// TemplateStatus represents the publication status of a template
type TemplateStatus string

const (
	TemplateStatusDraft     TemplateStatus = "draft"
	TemplateStatusActive    TemplateStatus = "active"    // Published and active
	TemplateStatusArchived  TemplateStatus = "archived"  // Soft-deleted
	TemplateStatusPublished TemplateStatus = "published" // Legacy - maps to active
	// WhatsApp-specific statuses
	TemplateStatusPending  TemplateStatus = "pending"  // Submitted to Meta
	TemplateStatusApproved TemplateStatus = "approved" // Approved by Meta
	TemplateStatusRejected TemplateStatus = "rejected" // Rejected by Meta
)

// ApprovalFlag represents the email approval level
type ApprovalFlag string

const (
	ApprovalFlagGreen  ApprovalFlag = "green"  // Auto-approve
	ApprovalFlagYellow ApprovalFlag = "yellow" // Team lead approval
	ApprovalFlagRed    ApprovalFlag = "red"    // Senior manager approval
)

// Note: Service is now stored as ObjectID reference to the services collection
// The Template struct has been removed - use MongoTemplate instead (mongodb_template.go)

// TemplateMetrics represents template usage and performance metrics
type TemplateMetrics struct {
	Usage             int64   `json:"usage" bson:"usage"`
	OpenRate          float64 `json:"open_rate,omitempty" bson:"open_rate"`
	ClickRate         float64 `json:"click_rate,omitempty" bson:"click_rate"`
	ReplyRate         float64 `json:"reply_rate,omitempty" bson:"reply_rate"`
	DeliveryRate      float64 `json:"delivery_rate,omitempty" bson:"delivery_rate"`
	ReadRate          float64 `json:"read_rate,omitempty" bson:"read_rate"`
	AcceptanceRate    float64 `json:"acceptance_rate,omitempty" bson:"acceptance_rate"`
	ResponseRate      float64 `json:"response_rate,omitempty" bson:"response_rate"`
	MeetingConversion float64 `json:"meeting_conversion,omitempty" bson:"meeting_conversion"`
}

// TemplateVersion represents a historical version snapshot of a template
type TemplateVersion struct {
	TemplateID  primitive.ObjectID `json:"template_id" db:"template_id"`
	Version     int                `json:"version" db:"version"`
	Content     map[string]string  `json:"content" db:"content"`
	MergeTags   []string           `json:"merge_tags" db:"merge_tags"`
	PublishedAt time.Time          `json:"published_at" db:"published_at"`
	PublishedBy primitive.ObjectID `json:"published_by" db:"published_by"`
}

// TemplateTag represents a tag for organizing templates
type TemplateTag struct {
	TenantID      primitive.ObjectID `json:"tenant_id" db:"tenant_id"`
	Name          string             `json:"name" db:"name" validate:"required,min=1,max=50"`
	TemplateCount int64              `json:"template_count" db:"template_count"` // Counter field
	IsSystem      bool               `json:"is_system" db:"is_system"`           // Reserved funnel stage tags
	CreatedAt     time.Time          `json:"created_at" db:"created_at"`
}

// TemplateTagAssociation represents the many-to-many relationship between templates and tags
type TemplateTagAssociation struct {
	TenantID   primitive.ObjectID `json:"tenant_id" db:"tenant_id"`
	TemplateID primitive.ObjectID `json:"template_id" db:"template_id"`
	TagName    string             `json:"tag_name" db:"tag_name"`
	CreatedAt  time.Time          `json:"created_at" db:"created_at"`
}

// TemplateAnalytics represents usage analytics for a template
type TemplateAnalytics struct {
	TemplateID    primitive.ObjectID `json:"template_id" db:"template_id"`
	Period        string             `json:"period" db:"period"`                 // daily, weekly, monthly, all-time
	SendCount     int64              `json:"send_count" db:"send_count"`         // Counter
	OpenCount     int64              `json:"open_count" db:"open_count"`         // Counter
	ClickCount    int64              `json:"click_count" db:"click_count"`       // Counter
	ResponseCount int64              `json:"response_count" db:"response_count"` // Counter
}

// TemplateLibraryItem represents a pre-built system template
type TemplateLibraryItem struct {
	ID           primitive.ObjectID `json:"id" db:"id"`
	Name         string             `json:"name" db:"name"`
	Description  string             `json:"description,omitempty" db:"description"`
	Category     string             `json:"category" db:"category"` // Funnel stage: prospect, mql, sql, etc.
	Channel      TemplateChannel    `json:"channel" db:"channel"`
	Content      map[string]string  `json:"content" db:"content"`
	MergeTags    []string           `json:"merge_tags" db:"merge_tags"`
	CustomFields map[string]string  `json:"custom_fields,omitempty" db:"custom_fields"`
	IsSystem     bool               `json:"is_system" db:"is_system"`
	CreatedAt    time.Time          `json:"created_at" db:"created_at"`
}

// Standard merge tags available in all templates (17 standard variables)
var StandardMergeTags = []string{
	"first_name",
	"last_name",
	"full_name",
	"email",
	"phone",
	"company_name",
	"company_domain",
	"job_title",
	"industry",
	"company_size",
	"region",
	"deal_value",
	"deal_stage",
	"expected_close_date",
	"assigned_rep_name",
	"assigned_rep_email",
	"custom_field",
}

// Channel-specific content field requirements and limits
const (
	// Email limits
	EmailSubjectMaxLength   = 200
	EmailPreHeaderMaxLength = 100

	// SMS limits
	SMSSegmentLength = 160

	// WhatsApp limits
	WhatsAppHeaderMaxLength = 60
	WhatsAppBodyMaxLength   = 1024
	WhatsAppFooterMaxLength = 60
	WhatsAppMaxButtons      = 3

	// LinkedIn limits
	LinkedInInMailMaxLength     = 8000
	LinkedInConnectionMaxLength = 300
)

// IsValidChannel checks if the template channel is valid
func IsValidChannel(channel string) bool {
	validChannels := []TemplateChannel{
		TemplateChannelEmail,
		TemplateChannelSMS,
		TemplateChannelWhatsApp,
		TemplateChannelLinkedIn,
	}

	for _, validChannel := range validChannels {
		if TemplateChannel(channel) == validChannel {
			return true
		}
	}
	return false
}

// IsValidTemplateStatus checks if the template status is valid
func IsValidTemplateStatus(status string) bool {
	validStatuses := []TemplateStatus{
		TemplateStatusDraft,
		TemplateStatusActive,
		TemplateStatusArchived,
		TemplateStatusPublished,
		TemplateStatusPending,
		TemplateStatusApproved,
		TemplateStatusRejected,
	}

	for _, validStatus := range validStatuses {
		if TemplateStatus(status) == validStatus {
			return true
		}
	}
	return false
}

// IsValidApprovalFlag checks if the approval flag is valid
func IsValidApprovalFlag(flag string) bool {
	validFlags := []ApprovalFlag{
		ApprovalFlagGreen,
		ApprovalFlagYellow,
		ApprovalFlagRed,
	}

	for _, validFlag := range validFlags {
		if ApprovalFlag(flag) == validFlag {
			return true
		}
	}
	return flag == "" // Empty is valid (optional field)
}

// Note: Template methods (Validate, ExtractMergeTags, etc.) are now on MongoTemplate in mongodb_template.go

// CalculateRates calculates analytics rates
func (a *TemplateAnalytics) CalculateRates() (openRate, clickRate, responseRate float64) {
	if a.SendCount == 0 {
		return 0, 0, 0
	}

	openRate = float64(a.OpenCount) / float64(a.SendCount) * 100
	clickRate = float64(a.ClickCount) / float64(a.SendCount) * 100
	responseRate = float64(a.ResponseCount) / float64(a.SendCount) * 100

	return openRate, clickRate, responseRate
}
