package models

import (
	"time"
)

// MongoCommunication represents a communication record in MongoDB
// Collection: communications
type MongoCommunication struct {
	ID          string        `bson:"_id,omitempty" json:"id"`
	ThreadID    string        `bson:"thread_id,omitempty" json:"threadId,omitempty"` // Thread/conversation ID for grouping
	Channel     string                    `bson:"channel" json:"channel"`                        // email, sms, whatsapp, linkedin
	Direction   string                    `bson:"direction" json:"direction"`                    // inbound, outbound
	From        string                    `bson:"from" json:"from"`                              // Sender name for display
	FromEmail   string                    `bson:"from_email,omitempty" json:"fromEmail,omitempty"` // Sender email address
	To          string                    `bson:"to" json:"to"`                                  // Primary recipient (name or email)
	ToEmail     string                    `bson:"to_email,omitempty" json:"toEmail,omitempty"`   // Primary recipient email
	CC          []string                  `bson:"cc,omitempty" json:"cc,omitempty"`              // CC recipients
	BCC         []string                  `bson:"bcc,omitempty" json:"bcc,omitempty"`            // BCC recipients
	Company     string                    `bson:"company,omitempty" json:"company,omitempty"`    // Company name (for display)
	Subject     string                    `bson:"subject,omitempty" json:"subject,omitempty"`    // For emails
	Body        string                    `bson:"body" json:"message"`                           // Plain text body
	BodyHTML    string                    `bson:"body_html,omitempty" json:"bodyHtml,omitempty"` // HTML body for rich formatting
	Snippet     string                    `bson:"snippet,omitempty" json:"snippet,omitempty"`    // Short preview of message
	Attachments []CommunicationAttachment `bson:"attachments,omitempty" json:"attachments,omitempty"` // File attachments
	Priority    string                    `bson:"priority,omitempty" json:"priority,omitempty"`  // normal, high, urgent
	CustomerID  string        `bson:"customer_id,omitempty" json:"customerId,omitempty"`
	CampaignID  string        `bson:"campaign_id,omitempty" json:"campaignId,omitempty"`
	UserID      string        `bson:"user_id,omitempty" json:"userId,omitempty"`     // Sender/owner user ID
	Status      string                    `bson:"status" json:"status"`                          // pending, queued, sending, sent, delivered, opened, clicked, bounced, failed, spam, unsubscribed

	// External provider tracking
	ExternalMessageID string                    `bson:"external_message_id,omitempty" json:"externalMessageId,omitempty"` // Provider-specific message ID (e.g., SendGrid sg_message_id)
	SGMessageID       string                    `bson:"sg_message_id,omitempty" json:"sgMessageId,omitempty"`             // SendGrid message ID for webhook matching

	// User interaction flags
	IsRead      bool                      `bson:"is_read" json:"isRead"`
	IsStarred   bool                      `bson:"is_starred" json:"isStarred"`
	IsArchived  bool                      `bson:"is_archived" json:"isArchived"`

	// Scheduling
	ScheduledAt   *time.Time              `bson:"scheduled_at,omitempty" json:"scheduledAt,omitempty"` // When the message is scheduled to be sent

	// Email delivery tracking timestamps
	SentAt        time.Time               `bson:"sent_at,omitempty" json:"sentAt,omitempty"`
	DeliveredAt   *time.Time              `bson:"delivered_at,omitempty" json:"deliveredAt,omitempty"`
	OpenedAt      *time.Time              `bson:"opened_at,omitempty" json:"openedAt,omitempty"`
	ClickedAt     *time.Time              `bson:"clicked_at,omitempty" json:"clickedAt,omitempty"`
	BouncedAt     *time.Time              `bson:"bounced_at,omitempty" json:"bouncedAt,omitempty"`
	FailedAt      *time.Time              `bson:"failed_at,omitempty" json:"failedAt,omitempty"`
	SpamAt        *time.Time              `bson:"spam_at,omitempty" json:"spamAt,omitempty"`              // Marked as spam
	UnsubscribedAt *time.Time             `bson:"unsubscribed_at,omitempty" json:"unsubscribedAt,omitempty"`
	ReadAt        *time.Time              `bson:"read_at,omitempty" json:"readAt,omitempty"`              // User marked as read in inbox

	// Email engagement metrics
	OpenCount     int                     `bson:"open_count" json:"openCount"`                   // Number of times opened
	ClickCount    int                     `bson:"click_count" json:"clickCount"`                 // Number of link clicks
	ClickedLinks  []string                `bson:"clicked_links,omitempty" json:"clickedLinks,omitempty"` // Which links were clicked

	// Bounce/failure details
	BounceType    string                  `bson:"bounce_type,omitempty" json:"bounceType,omitempty"`     // hard, soft
	BounceReason  string                  `bson:"bounce_reason,omitempty" json:"bounceReason,omitempty"` // Detailed bounce reason
	FailureReason string                  `bson:"failure_reason,omitempty" json:"failureReason,omitempty"` // Why sending failed

	// Timestamps
	CreatedAt   time.Time                 `bson:"created_at" json:"createdAt"`
	UpdatedAt   time.Time                 `bson:"updated_at" json:"updatedAt"`
}

// CommunicationAttachment represents a file attachment in a communication
type CommunicationAttachment struct {
	FileName    string `bson:"file_name" json:"fileName"`
	FileType    string `bson:"file_type" json:"fileType"`    // MIME type
	FileSize    int64  `bson:"file_size" json:"fileSize"`    // Size in bytes
	FileURL     string `bson:"file_url" json:"fileUrl"`      // URL or path to the file
	ContentID   string `bson:"content_id,omitempty" json:"contentId,omitempty"` // For inline attachments (cid:)
	ItemID      string `bson:"item_id,omitempty" json:"itemId,omitempty"`       // OneDrive item ID for downloading
	WebURL      string `bson:"web_url,omitempty" json:"webUrl,omitempty"`       // OneDrive web URL for preview
}

// CommunicationChannel represents the channel of communication
type CommunicationChannel string

const (
	CommunicationChannelEmail    CommunicationChannel = "email"
	CommunicationChannelWhatsApp CommunicationChannel = "whatsapp"
)

// CommunicationDirection represents the direction of communication
type CommunicationDirection string

const (
	CommunicationDirectionInbound  CommunicationDirection = "inbound"
	CommunicationDirectionOutbound CommunicationDirection = "outbound"
)

// CommunicationStatus represents the status of a communication
type CommunicationStatus string

const (
	CommunicationStatusPending   CommunicationStatus = "pending"
	CommunicationStatusScheduled CommunicationStatus = "scheduled" // Scheduled for future delivery
	CommunicationStatusQueued    CommunicationStatus = "queued"    // In queue for immediate sending
	CommunicationStatusSent      CommunicationStatus = "sent"
	CommunicationStatusDelivered CommunicationStatus = "delivered"
	CommunicationStatusRead      CommunicationStatus = "read"
	CommunicationStatusFailed    CommunicationStatus = "failed"
)

// IsValidCommunicationChannel checks if the communication channel is valid
func IsValidCommunicationChannel(channel string) bool {
	validChannels := []CommunicationChannel{
		CommunicationChannelEmail,
		CommunicationChannelWhatsApp,
	}

	for _, validChannel := range validChannels {
		if CommunicationChannel(channel) == validChannel {
			return true
		}
	}
	return false
}

// IsValidCommunicationDirection checks if the communication direction is valid
func IsValidCommunicationDirection(direction string) bool {
	validDirections := []CommunicationDirection{
		CommunicationDirectionInbound,
		CommunicationDirectionOutbound,
	}

	for _, validDirection := range validDirections {
		if CommunicationDirection(direction) == validDirection {
			return true
		}
	}
	return false
}

// IsValidCommunicationStatus checks if the communication status is valid
func IsValidCommunicationStatus(status string) bool {
	validStatuses := []CommunicationStatus{
		CommunicationStatusPending,
		CommunicationStatusScheduled,
		CommunicationStatusQueued,
		CommunicationStatusSent,
		CommunicationStatusDelivered,
		CommunicationStatusRead,
		CommunicationStatusFailed,
	}

	for _, validStatus := range validStatuses {
		if CommunicationStatus(status) == validStatus {
			return true
		}
	}
	return false
}

// IsScheduled checks if the communication is scheduled for future delivery
func (c *MongoCommunication) IsScheduled() bool {
	return c.ScheduledAt != nil && c.ScheduledAt.After(time.Now())
}

// MarkAsRead marks the communication as read with the current timestamp
func (c *MongoCommunication) MarkAsRead() {
	now := time.Now()
	c.ReadAt = &now
	c.Status = string(CommunicationStatusRead)
}

// MarkAsSent marks the communication as sent with the current timestamp
func (c *MongoCommunication) MarkAsSent() {
	c.SentAt = time.Now()
	c.Status = string(CommunicationStatusSent)
}

// MarkAsDelivered marks the communication as delivered
func (c *MongoCommunication) MarkAsDelivered() {
	c.Status = string(CommunicationStatusDelivered)
}

// MarkAsFailed marks the communication as failed
func (c *MongoCommunication) MarkAsFailed() {
	c.Status = string(CommunicationStatusFailed)
}

// HasBeenRead checks if the communication has been read (based on ReadAt timestamp)
func (c *MongoCommunication) HasBeenRead() bool {
	return c.ReadAt != nil
}

// Indexes required for Communications collection:
// 1. Compound index on customer_id and sent_at: db.communications.createIndex({ "customer_id": 1, "sent_at": -1 })
// 3. Index on status: db.communications.createIndex({ "status": 1 })
