package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

)

// MongoTemplate represents a communication template in MongoDB
// This is the SINGLE source of truth for templates - no conversion needed
// Collection: templates
type MongoTemplate struct {
	ID          string `bson:"_id,omitempty" json:"id"`
	TenantID    string `bson:"tenant_id,omitempty" json:"tenantId,omitempty"`
	Name        string `bson:"name" json:"name"`
	Description string `bson:"description,omitempty" json:"description,omitempty"`
	Type        string `bson:"type,omitempty" json:"type,omitempty"` // Deprecated: use Channel instead
	Channel     string `bson:"channel" json:"channel"`               // email, sms, whatsapp, linkedin
	Status      string `bson:"status" json:"status"`                 // draft, active, archived

	// Content storage - flexible map for channel-specific fields
	// Email: subject, body_html, body_text, pre_header
	// SMS: body
	// WhatsApp: header, body, footer, buttons
	// LinkedIn: body, message_type
	Content      map[string]string `bson:"content,omitempty" json:"content,omitempty"`
	Subject      string            `bson:"subject,omitempty" json:"subject,omitempty"`     // Convenience field for email
	Body         string            `bson:"body,omitempty" json:"message,omitempty"`        // Main body content, JSON as "message" for frontend
	Variables    []string          `bson:"variables,omitempty" json:"variables,omitempty"` // Extracted merge tags
	CustomFields map[string]string `bson:"custom_fields,omitempty" json:"customFields,omitempty"`

	// Organization
	Category string   `bson:"category,omitempty" json:"category,omitempty"` // prospecting, follow-up, nurture, closing
	Tags     []string `bson:"tags,omitempty" json:"tags,omitempty"`

	// Versioning & System
	Version  int  `bson:"version,omitempty" json:"version,omitempty"`
	IsSystem bool `bson:"is_system,omitempty" json:"isSystem,omitempty"` // System templates cannot be deleted

	// Timestamps
	CreatedAt   time.Time  `bson:"created_at" json:"createdAt"`
	UpdatedAt   time.Time  `bson:"updated_at" json:"updatedAt"`
	CreatedBy   string     `bson:"created_by,omitempty" json:"createdBy,omitempty"`
	PublishedAt *time.Time `bson:"published_at,omitempty" json:"publishedAt,omitempty"`
	PublishedBy string     `bson:"published_by,omitempty" json:"publishedBy,omitempty"`

	// Frontend filter fields
	ForStage     []string `bson:"for_stage,omitempty" json:"forStage,omitempty"`         // Funnel stages (prospect, mql, sql, etc.)
	Industries   []string `bson:"industries,omitempty" json:"industries,omitempty"`      // Industry targeting
	ApprovalFlag string   `bson:"approval_flag,omitempty" json:"approvalFlag,omitempty"` // green, yellow, red
	AiEnhanced   bool     `bson:"ai_enhanced,omitempty" json:"aiEnhanced,omitempty"`     // AI-generated content
	ServiceID    string   `bson:"service_id,omitempty" json:"serviceId,omitempty"`       // Reference to services collection

	// WhatsApp Meta approval fields
	MetaTemplateName string     `bson:"meta_template_name,omitempty" json:"metaTemplateName,omitempty"`
	MetaStatus       string     `bson:"meta_status,omitempty" json:"metaStatus,omitempty"` // approved, pending, rejected
	SubmittedDate    *time.Time `bson:"submitted_date,omitempty" json:"submittedDate,omitempty"`
	ExpectedApproval *time.Time `bson:"expected_approval,omitempty" json:"expectedApproval,omitempty"`

	// LinkedIn-specific fields
	TemplateType string `bson:"template_type,omitempty" json:"templateType,omitempty"` // Connection Request, InMail Message

	// KOSH Document attachments - pre-configured documents to attach when using this template
	// Stores references to KOSH documents (item_id from kosh_files collection)
	KoshDocumentIds []string `bson:"kosh_document_ids,omitempty" json:"koshDocumentIds,omitempty"`

}

// TemplateMetrics is defined in template.go - using that definition for consistency

// =============================================================================
// Validation Methods
// =============================================================================

// Validate validates the template struct
func (t *MongoTemplate) Validate() error {
	if t.Name == "" {
		return errors.New("template name is required")
	}
	if len(t.Name) > 200 {
		return errors.New("template name cannot exceed 200 characters")
	}
	if !IsValidChannel(t.Channel) {
		return fmt.Errorf("invalid channel: %s", t.Channel)
	}
	if !IsValidTemplateStatus(t.Status) {
		return fmt.Errorf("invalid status: %s", t.Status)
	}

	// Build content map from fields if not set
	if t.Content == nil {
		t.Content = make(map[string]string)
	}
	if t.Subject != "" && t.Content["subject"] == "" {
		t.Content["subject"] = t.Subject
	}
	if t.Body != "" && t.Content["body"] == "" {
		t.Content["body"] = t.Body
		t.Content["body_html"] = t.Body // For email compatibility
	}

	// Channel-specific validation
	switch t.Channel {
	case "email":
		return t.validateEmailContent()
	case "sms":
		return t.validateSMSContent()
	}

	return nil
}

func (t *MongoTemplate) validateEmailContent() error {
	subject := t.Content["subject"]
	if subject == "" {
		subject = t.Subject
	}
	if subject == "" {
		return errors.New("email template requires subject")
	}
	if len(subject) > EmailSubjectMaxLength {
		return fmt.Errorf("email subject cannot exceed %d characters", EmailSubjectMaxLength)
	}

	hasHTML := t.Content["body_html"] != ""
	hasText := t.Content["body_text"] != ""
	hasBody := t.Body != ""

	if !hasHTML && !hasText && !hasBody {
		return errors.New("email template requires either body_html or body_text")
	}

	if preHeader := t.Content["pre_header"]; preHeader != "" && len(preHeader) > EmailPreHeaderMaxLength {
		return fmt.Errorf("email pre_header cannot exceed %d characters", EmailPreHeaderMaxLength)
	}

	return nil
}

func (t *MongoTemplate) validateSMSContent() error {
	body := t.Content["body"]
	if body == "" {
		body = t.Body
	}
	if body == "" {
		return errors.New("SMS template requires body")
	}

	segmentCount := (len(body) + SMSSegmentLength - 1) / SMSSegmentLength
	t.Content["segment_count"] = fmt.Sprintf("%d", segmentCount)

	return nil
}


// =============================================================================
// Permission Methods
// =============================================================================

// CanPublish checks if the template can be published
func (t *MongoTemplate) CanPublish() bool {
	return t.Status == string(TemplateStatusDraft) && t.Validate() == nil
}

// CanUnpublish checks if the template can be unpublished
func (t *MongoTemplate) CanUnpublish() bool {
	return t.Status == string(TemplateStatusPublished) || t.Status == string(TemplateStatusActive)
}

// CanEdit checks if the template can be edited
func (t *MongoTemplate) CanEdit() bool {
	return !t.IsSystem
}

// CanDelete checks if the template can be deleted
func (t *MongoTemplate) CanDelete() bool {
	return !t.IsSystem
}

// =============================================================================
// Merge Tag Methods
// =============================================================================

// ExtractMergeTags extracts {{variable}} patterns from content
func (t *MongoTemplate) ExtractMergeTags() []string {
	tags := make(map[string]bool)

	// Check Content map
	for _, value := range t.Content {
		t.extractTagsFromString(value, tags)
	}

	// Check Subject and Body fields
	t.extractTagsFromString(t.Subject, tags)
	t.extractTagsFromString(t.Body, tags)

	result := make([]string, 0, len(tags))
	for tag := range tags {
		result = append(result, tag)
	}

	return result
}

func (t *MongoTemplate) extractTagsFromString(value string, tags map[string]bool) {
	start := 0
	for {
		startIdx := strings.Index(value[start:], "{{")
		if startIdx == -1 {
			break
		}
		startIdx += start
		endIdx := strings.Index(value[startIdx:], "}}")
		if endIdx == -1 {
			break
		}
		endIdx += startIdx

		tagName := strings.TrimSpace(value[startIdx+2 : endIdx])
		if tagName != "" {
			tags[tagName] = true
		}

		start = endIdx + 2
	}
}

// ValidateMergeTags returns warnings for undefined merge tags
func (t *MongoTemplate) ValidateMergeTags() []string {
	warnings := []string{}
	extractedTags := t.ExtractMergeTags()

	standardTagsMap := make(map[string]bool)
	for _, tag := range StandardMergeTags {
		standardTagsMap[tag] = true
	}

	for _, tag := range extractedTags {
		if standardTagsMap[tag] {
			continue
		}

		if t.CustomFields != nil {
			if _, exists := t.CustomFields[tag]; exists {
				continue
			}
		}

		warnings = append(warnings, fmt.Sprintf("merge tag '{{%s}}' is not defined", tag))
	}

	return warnings
}

// =============================================================================
// Tag Methods
// =============================================================================

// AddTag adds a tag to the template
func (t *MongoTemplate) AddTag(tag string) {
	for _, existingTag := range t.Tags {
		if existingTag == tag {
			return
		}
	}
	t.Tags = append(t.Tags, tag)
}

// RemoveTag removes a tag from the template
func (t *MongoTemplate) RemoveTag(tag string) {
	for i, existingTag := range t.Tags {
		if existingTag == tag {
			t.Tags = append(t.Tags[:i], t.Tags[i+1:]...)
			return
		}
	}
}

// HasTag checks if the template has a specific tag
func (t *MongoTemplate) HasTag(tag string) bool {
	for _, existingTag := range t.Tags {
		if existingTag == tag {
			return true
		}
	}
	return false
}

// AddVariable adds a variable to the template's variable list
func (t *MongoTemplate) AddVariable(variable string) {
	for _, v := range t.Variables {
		if v == variable {
			return
		}
	}
	t.Variables = append(t.Variables, variable)
}

// RemoveVariable removes a variable from the template
func (t *MongoTemplate) RemoveVariable(variable string) {
	for i, v := range t.Variables {
		if v == variable {
			t.Variables = append(t.Variables[:i], t.Variables[i+1:]...)
			return
		}
	}
}

// =============================================================================
// Response Helpers
// =============================================================================

// GetApprovalText returns human-readable approval text
func (t *MongoTemplate) GetApprovalText() string {
	switch t.ApprovalFlag {
	case "green":
		return "Auto-Approve"
	case "yellow":
		return "Team Lead Approval Required"
	case "red":
		return "Senior Manager Approval Required"
	default:
		return "Auto-Approve"
	}
}

// GetLastModified returns formatted last modified date
func (t *MongoTemplate) GetLastModified() string {
	return t.UpdatedAt.Format("2006-01-02")
}

// GetMessage returns the body content for frontend
func (t *MongoTemplate) GetMessage() string {
	if t.Body != "" {
		return t.Body
	}
	if body := t.Content["body_html"]; body != "" {
		return body
	}
	if body := t.Content["body_text"]; body != "" {
		return body
	}
	return t.Content["body"]
}

// Note: ToFrontendResponse has been removed
// MongoTemplate now has proper JSON tags and is returned directly to frontend

// =============================================================================
// Type Definitions (kept for compatibility)
// =============================================================================

// MongoTemplateCategory represents template categories
type MongoTemplateCategory string

const (
	MongoTemplateCategoryProspecting MongoTemplateCategory = "prospecting"
	MongoTemplateCategoryFollowUp    MongoTemplateCategory = "follow-up"
	MongoTemplateCategoryNurture     MongoTemplateCategory = "nurture"
	MongoTemplateCategoryClosing     MongoTemplateCategory = "closing"
	MongoTemplateCategoryOnboarding  MongoTemplateCategory = "onboarding"
	MongoTemplateCategoryRenewal     MongoTemplateCategory = "renewal"
)

// IsValidMongoTemplateCategory checks if the category is valid
func IsValidMongoTemplateCategory(category string) bool {
	validCategories := []MongoTemplateCategory{
		MongoTemplateCategoryProspecting,
		MongoTemplateCategoryFollowUp,
		MongoTemplateCategoryNurture,
		MongoTemplateCategoryClosing,
		MongoTemplateCategoryOnboarding,
		MongoTemplateCategoryRenewal,
	}

	for _, validCategory := range validCategories {
		if MongoTemplateCategory(category) == validCategory {
			return true
		}
	}
	return false
}
