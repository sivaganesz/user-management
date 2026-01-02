package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Message status
const (
	MessageStatusDraft     = "draft"
	MessageStatusQueued    = "queued"
	MessageStatusSent      = "sent"
	MessageStatusDelivered = "delivered"
	MessageStatusFailed    = "failed"
	MessageStatusBounced   = "bounced"
)

// EmailAccount represents an email account configuration
type EmailAccount struct {
	AccountID      primitive.ObjectID `bson:"_id,omitempty" json:"account_id"`
	UserID         primitive.ObjectID `bson:"user_id" json:"user_id"`
	EmailAddress   string             `bson:"email_address" json:"email_address"`
	DisplayName    string             `bson:"display_name" json:"display_name"`
	AccountType    string             `bson:"account_type" json:"account_type"`
	Provider       string             `bson:"provider" json:"provider"`
	SMTPHost       string             `bson:"smtp_host" json:"smtp_host"`
	SMTPPort       int                `bson:"smtp_port" json:"smtp_port"`
	SMTPUsername   string             `bson:"smtp_username" json:"smtp_username"`
	SMTPPassword   string             `bson:"smtp_password,omitempty" json:"smtp_password,omitempty"` // Encrypted
	SMTPEncryption string             `bson:"smtp_encryption" json:"smtp_encryption"`
	IMAPHost       string             `bson:"imap_host" json:"imap_host"`
	IMAPPort       int                `bson:"imap_port" json:"imap_port"`
	IMAPUsername   string             `bson:"imap_username" json:"imap_username"`
	IMAPPassword   string             `bson:"imap_password,omitempty" json:"imap_password,omitempty"` // Encrypted
	IMAPEncryption string             `bson:"imap_encryption" json:"imap_encryption"`
	IsDefault      bool               `bson:"is_default" json:"is_default"`
	IsActive       bool               `bson:"is_active" json:"is_active"`
	SyncEnabled    bool               `bson:"sync_enabled" json:"sync_enabled"`
	LastSyncAt     *time.Time         `bson:"last_sync_at,omitempty" json:"last_sync_at,omitempty"`
	SyncStatus     string             `bson:"sync_status" json:"sync_status"`
	Signature      string             `bson:"signature,omitempty" json:"signature,omitempty"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}

type CommMessage struct {
	MessageID       primitive.ObjectID `json:"message_id"`
	ThreadID        primitive.ObjectID `json:"thread_id"`
	ParentMessageID primitive.ObjectID `json:"parent_message_id,omitempty"`
	Channel         string             `json:"channel"`
	Direction       string             `json:"direction"`
	Status          string             `json:"status"`
	FromAddress     string             `json:"from_address"`
	FromName        string             `json:"from_name,omitempty"`
	ToAddresses     []string           `json:"to_addresses"`
	CCAddresses     []string           `json:"cc_addresses,omitempty"`
	BCCAddresses    []string           `json:"bcc_addresses,omitempty"`
	Subject         string             `json:"subject,omitempty"`
	BodyText        string             `json:"body_text"`
	BodyHTML        string             `json:"body_html,omitempty"`
	Snippet         string             `json:"snippet,omitempty"`
	HasAttachments  bool                      `json:"has_attachments"`
	AttachmentCount int                       `json:"attachment_count"`
	Attachments     []CommunicationAttachment `json:"attachments,omitempty"` // Full attachment details
	EntityType      string                    `json:"entity_type,omitempty"`
	EntityID        primitive.ObjectID `json:"entity_id,omitempty"`
	UserID          primitive.ObjectID `json:"user_id"`
	AccountID       primitive.ObjectID `json:"account_id,omitempty"`
	ExternalID      string             `json:"external_id,omitempty"`
	InReplyTo       string             `json:"in_reply_to,omitempty"`
	References      string             `json:"references,omitempty"`
	Priority        string             `json:"priority"`
	IsRead          bool               `json:"is_read"`
	ReadAt          *time.Time         `json:"read_at,omitempty"`
	IsStarred       bool               `json:"is_starred"`
	IsArchived      bool               `json:"is_archived"`
	Labels          []string           `json:"labels,omitempty"`
	ScheduledAt     *time.Time         `json:"scheduled_at,omitempty"`
	SentAt          *time.Time         `json:"sent_at,omitempty"`
	DeliveredAt     *time.Time         `json:"delivered_at,omitempty"`
	OpenedAt        *time.Time         `json:"opened_at,omitempty"`
	ClickedAt       *time.Time         `json:"clicked_at,omitempty"`
	BouncedAt       *time.Time         `json:"bounced_at,omitempty"`
	FailedAt        *time.Time         `json:"failed_at,omitempty"`
	ErrorMessage    string             `json:"error_message,omitempty"`
	// AI Analysis Fields
	SentimentScore   float64    `json:"sentiment_score,omitempty"`
	SentimentLabel   string     `json:"sentiment_label,omitempty"`
	Intent           string     `json:"intent,omitempty"`
	Summary          string     `json:"summary,omitempty"`
	SuggestedReplies []string   `json:"suggested_replies,omitempty"`
	AIAnalyzedAt     *time.Time `json:"ai_analyzed_at,omitempty"`
	// WhatsApp-specific Fields
	WhatsAppTemplateID     string `json:"whatsapp_template_id,omitempty"`
	WhatsAppTemplateStatus string `json:"whatsapp_template_status,omitempty"`
	// LinkedIn-specific Fields
	LinkedInConnectionStatus string `json:"linkedin_connection_status,omitempty"`
	InMailCreditsUsed        int    `json:"inmail_credits_used,omitempty"`
	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IsSent checks if message was sent
func (m *CommMessage) IsSent() bool {
	return m.Status == MessageStatusSent || m.Status == MessageStatusDelivered
}

// IsDelivered checks if message was delivered
func (m *CommMessage) IsDelivered() bool {
	return m.Status == MessageStatusDelivered
}

// MessageThread represents a conversation thread
type MessageThread struct {
	ThreadID             primitive.ObjectID   `json:"thread_id"`
	Subject              string               `json:"subject"`
	Channel              string               `json:"channel"`
	EntityType           string               `json:"entity_type,omitempty"`
	EntityID             primitive.ObjectID   `json:"entity_id,omitempty"`
	ParticipantAddresses []string             `json:"participant_addresses"`
	ParticipantIDs       []primitive.ObjectID `json:"participant_ids,omitempty"`
	MessageCount         int                  `json:"message_count"`
	UnreadCount          int                  `json:"unread_count"`
	LastMessageAt        time.Time            `json:"last_message_at"`
	LastMessageSnippet   string               `json:"last_message_snippet,omitempty"`
	IsArchived           bool                 `json:"is_archived"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`
}