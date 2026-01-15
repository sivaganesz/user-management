package models

import (
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
