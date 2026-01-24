package models

import (
	"time"
)

// MongoCampaign represents a marketing/sales campaign in MongoDB
// Collection: campaigns
type MongoCampaign struct {
	ID             string         `bson:"_id,omitempty" json:"id"`
	Name           string         `bson:"name" json:"name"`
	Type           string         `bson:"type" json:"type"`                             // email, sms, whatsapp, linkedin, multi-channel
	Status         string         `bson:"status" json:"status"`                         // draft, scheduled, active, paused, completed, failed
	OwnerID        string         `bson:"owner_id,omitempty" json:"ownerId,omitempty"`  // Campaign owner user ID
	Channels       []string       `bson:"channels,omitempty" json:"channels,omitempty"` // For multi-channel campaigns
	Steps          []CampaignStep `bson:"steps,omitempty" json:"steps,omitempty"`
	TargetAudience TargetAudience  `bson:"target_audience" json:"targetAudience"`
	ScheduledAt    *time.Time     `bson:"scheduled_at,omitempty" json:"scheduledAt,omitempty"`
	StartedAt      *time.Time     `bson:"started_at,omitempty" json:"startedAt,omitempty"`
	CompletedAt    *time.Time     `bson:"completed_at,omitempty" json:"completedAt,omitempty"`
}

// CampaignStep represents a step in a campaign sequence
type CampaignStep struct {
	StepNumber int                    `bson:"step_number" json:"stepNumber"`
	Channel    string                 `bson:"channel" json:"channel"`
	TemplateID string                 `bson:"template_id" json:"templateId"`
	DelayDays  int                    `bson:"delay_days" json:"delayDays"` // Delay from previous step
	Status     string                 `bson:"status" json:"status"`        // pending, executing, completed, failed
	ExecutedAt *time.Time             `bson:"executed_at,omitempty" json:"executedAt,omitempty"`
	Metadata   map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// TargetAudience represents the targeting criteria for a campaign
type TargetAudience struct {
	Regions    []string `bson:"regions,omitempty" json:"regions,omitempty"`
	Industries []string `bson:"industries,omitempty" json:"industries,omitempty"`
	Categories []string `bson:"categories,omitempty" json:"categories,omitempty"` // enterprise, mid-market, smb
	Tags       []string `bson:"tags,omitempty" json:"tags,omitempty"`
	Stages     []string `bson:"stages,omitempty" json:"stages,omitempty"` // Deal stages to target
}

// MongoCampaignType represents the type of campaign (MongoDB version to avoid conflicts)
type MongoCampaignType string

const (
	MongoCampaignTypeEmail        MongoCampaignType = "email"
	MongoCampaignTypeSMS          MongoCampaignType = "sms"
	MongoCampaignTypeWhatsApp     MongoCampaignType = "whatsapp"
	MongoCampaignTypeLinkedIn     MongoCampaignType = "linkedin"
	MongoCampaignTypeMultiChannel MongoCampaignType = "multi-channel"
)

// MongoCampaignStatus represents the status of a campaign (MongoDB version to avoid conflicts)
type MongoCampaignStatus string

const (
	MongoCampaignStatusDraft     MongoCampaignStatus = "draft"
	MongoCampaignStatusScheduled MongoCampaignStatus = "scheduled"
	MongoCampaignStatusActive    MongoCampaignStatus = "active"
	MongoCampaignStatusPaused    MongoCampaignStatus = "paused"
	MongoCampaignStatusCompleted MongoCampaignStatus = "completed"
	MongoCampaignStatusFailed    MongoCampaignStatus = "failed"
)

// IsValidMongoCampaignType checks if the campaign type is valid
func IsValidMongoCampaignType(campaignType string) bool {
	validTypes := []MongoCampaignType{
		MongoCampaignTypeEmail,
		MongoCampaignTypeSMS,
		MongoCampaignTypeWhatsApp,
		MongoCampaignTypeLinkedIn,
		MongoCampaignTypeMultiChannel,
	}

	for _, validType := range validTypes {
		if MongoCampaignType(campaignType) == validType {
			return true
		}
	}
	return false
}

// IsValidMongoCampaignStatus checks if the campaign status is valid
func IsValidMongoCampaignStatus(status string) bool {
	validStatuses := []MongoCampaignStatus{
		MongoCampaignStatusDraft,
		MongoCampaignStatusScheduled,
		MongoCampaignStatusActive,
		MongoCampaignStatusPaused,
		MongoCampaignStatusCompleted,
		MongoCampaignStatusFailed,
	}

	for _, validStatus := range validStatuses {
		if MongoCampaignStatus(status) == validStatus {
			return true
		}
	}
	return false
}

// Start marks the campaign as started
func (c *MongoCampaign) Start() {
	now := time.Now()
	c.StartedAt = &now
	c.Status = string(MongoCampaignStatusActive)
}

// Pause pauses the campaign
func (c *MongoCampaign) Pause() {
	c.Status = string(MongoCampaignStatusPaused)
}

// Resume resumes a paused campaign
func (c *MongoCampaign) Resume() {
	c.Status = string(MongoCampaignStatusActive)
}

// Complete marks the campaign as completed
func (c *MongoCampaign) Complete() {
	now := time.Now()
	c.CompletedAt = &now
	c.Status = string(MongoCampaignStatusCompleted)
}

// Fail marks the campaign as failed
func (c *MongoCampaign) Fail() {
	c.Status = string(MongoCampaignStatusFailed)
}

// AddStep adds a step to the campaign
func (c *MongoCampaign) AddStep(step CampaignStep) {
	c.Steps = append(c.Steps, step)
}

// GetCurrentStep returns the current executing step
func (c *MongoCampaign) GetCurrentStep() *CampaignStep {
	for i := range c.Steps {
		if c.Steps[i].Status == "pending" || c.Steps[i].Status == "executing" {
			return &c.Steps[i]
		}
	}
	return nil
}

// Indexes required for Campaigns collection:
// 1. Compound index on status and scheduled_at: db.campaigns.createIndex({ "status": 1, "scheduled_at": 1 })
// 2. Compound index on type and status: db.campaigns.createIndex({ "type": 1, "status": 1 })
