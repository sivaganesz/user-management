package models

import (
	"time"
)

// Activity represents a sales activity
type Activity struct {
	ID              string     `bson:"_id,omitempty" json:"id"`
	ActivityType    string     `bson:"activity_type" json:"activity_type"`
	Title           string     `bson:"title" json:"title"`
	Description     string     `bson:"description,omitempty" json:"description,omitempty"`
	Owner           string     `bson:"owner" json:"owner"`
	RelatedToType   string     `bson:"related_to_type,omitempty" json:"related_to_type,omitempty"`
	RelatedToID     string     `bson:"related_to_id,omitempty" json:"related_to_id,omitempty"`
	Status          string     `bson:"status" json:"status"`
	Priority        string     `bson:"priority" json:"priority"`
	Outcome         string     `bson:"outcome,omitempty" json:"outcome,omitempty"`
	DurationMinutes int        `bson:"duration_minutes,omitempty" json:"duration_minutes,omitempty"`
	ScheduledAt     *time.Time `bson:"scheduled_at,omitempty" json:"scheduled_at,omitempty"`
	CompletedAt     *time.Time `bson:"completed_at,omitempty" json:"completed_at,omitempty"`
	DueDate         *time.Time `bson:"due_date,omitempty" json:"due_date,omitempty"`
	Participants    string     `bson:"participants,omitempty" json:"participants,omitempty"`
	Location        string     `bson:"location,omitempty" json:"location,omitempty"`
	CallDirection   string     `bson:"call_direction,omitempty" json:"call_direction,omitempty"`
	CallResult      string     `bson:"call_result,omitempty" json:"call_result,omitempty"`
	EmailSubject    string     `bson:"email_subject,omitempty" json:"email_subject,omitempty"`
	EmailSentTo     string     `bson:"email_sent_to,omitempty" json:"email_sent_to,omitempty"`
	EmailCC         string     `bson:"email_cc,omitempty" json:"email_cc,omitempty"`
	MeetingType     string     `bson:"meeting_type,omitempty" json:"meeting_type,omitempty"`
	MeetingLink     string     `bson:"meeting_link,omitempty" json:"meeting_link,omitempty"`
	Attachments     string     `bson:"attachments,omitempty" json:"attachments,omitempty"`
	Tags            string     `bson:"tags,omitempty" json:"tags,omitempty"`
	Region          string     `bson:"region" json:"region"`
	Team            string     `bson:"team" json:"team"`
	CreatedBy       string     `bson:"created_by" json:"created_by"`
	CreatedAt       time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt       *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}
