package models

import (
	"encoding/json"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SequenceTemplate represents a reusable sequence template
type SequenceTemplate struct {
	TemplateID  primitive.ObjectID `json:"id" bson:"_id,omitempty" db:"template_id"`
	Name        string             `json:"name" bson:"name" db:"name" validate:"required,min=1,max=200"`
	Description string             `json:"description,omitempty" bson:"description,omitempty" db:"description"`
	ServiceID   primitive.ObjectID `json:"serviceId,omitempty" bson:"service_id,omitempty" db:"service_id"`
	ScheduleID  primitive.ObjectID `json:"scheduleId,omitempty" bson:"schedule_id,omitempty" db:"schedule_id"`
	Version     int                `json:"version" bson:"version" db:"version"`
	IsActive    bool               `json:"isActive" bson:"is_active" db:"is_active"`
	CreatedBy   primitive.ObjectID `json:"createdBy" bson:"created_by,omitempty" db:"created_by" validate:"required"`
	CreatedAt   time.Time          `json:"createdAt" bson:"created_at" db:"created_at"`
	UpdatedAt   time.Time          `json:"updatedAt" bson:"updated_at" db:"updated_at"`
}

// SequenceStepChannel represents a communication channel for a sequence step
type SequenceStepChannel string

const (
	SequenceStepChannelEmail    SequenceStepChannel = "email"
	SequenceStepChannelSMS      SequenceStepChannel = "sms"
	SequenceStepChannelWhatsApp SequenceStepChannel = "whatsapp"
	SequenceStepChannelLinkedIn SequenceStepChannel = "linkedin"
)

// BranchCondition represents branching logic based on recipient behavior
type BranchCondition struct {
	OnOpened  *int `json:"onOpened,omitempty"`  // Next step if message opened
	OnClicked *int `json:"onClicked,omitempty"` // Next step if link clicked
	OnReplied *int `json:"onReplied,omitempty"` // Next step if replied
	OnIgnored *int `json:"onIgnored,omitempty"` // Next step if ignored (no action)
}

// SequenceStepAttachment represents a KOSH document attachment in a sequence step
type SequenceStepAttachment struct {
	ID       string `json:"id" bson:"id"`                             // Document ID (from KOSH)
	Name     string `json:"name" bson:"name"`                         // File name
	Type     string `json:"type" bson:"type"`                         // Document type (PDF, Presentation, etc.)
	Size     string `json:"size" bson:"size"`                         // Formatted size (e.g., "232.3 KB")
	WebURL   string `json:"webUrl,omitempty" bson:"web_url,omitempty"` // Link to document
	Category string `json:"category,omitempty" bson:"category,omitempty"` // KOSH category
}

// CampaignSequenceStep represents a single step in a multi-channel sequence template
type CampaignSequenceStep struct {
	TemplateID        primitive.ObjectID       `json:"sequenceTemplateId" bson:"template_id,omitempty" db:"template_id" validate:"required"`
	StepOrder         int                      `json:"order" bson:"step_order" db:"step_order" validate:"required,min=1"`
	Channel           string                   `json:"communicationType" bson:"channel" db:"channel" validate:"required"`
	ContentTemplateID primitive.ObjectID       `json:"templateId" bson:"content_template_id,omitempty" db:"content_template_id" validate:"required"`
	Subject           string                   `json:"subject,omitempty" bson:"subject,omitempty" db:"subject"` // For email channel
	Body              string                   `json:"message" bson:"body" db:"body" validate:"required"`
	WaitDays          int                      `json:"waitDays" bson:"wait_days" db:"wait_days"`
	WaitHours         int                      `json:"waitHours" bson:"wait_hours" db:"wait_hours"`
	BranchConditions  string                   `json:"branchConditions" bson:"branch_conditions,omitempty" db:"branch_conditions"` // JSON
	Attachments       []SequenceStepAttachment `json:"attachments,omitempty" bson:"attachments,omitempty" db:"attachments"`        // KOSH documents
	CreatedAt         time.Time                `json:"createdAt" bson:"created_at" db:"created_at"`
}

// SequenceTemplateWithSteps combines a template with its ordered steps
type SequenceTemplateWithSteps struct {
	Template SequenceTemplate       `json:"template" bson:",inline"`
	Steps    []CampaignSequenceStep `json:"steps" bson:"steps"`
}

// Helper methods

// IsValidSequenceStepChannel checks if the channel is valid
func IsValidSequenceStepChannel(channel string) bool {
	validChannels := []SequenceStepChannel{
		SequenceStepChannelEmail,
		SequenceStepChannelSMS,
		SequenceStepChannelWhatsApp,
		SequenceStepChannelLinkedIn,
	}

	for _, validChannel := range validChannels {
		if SequenceStepChannel(channel) == validChannel {
			return true
		}
	}
	return false
}

// GetBranchConditions parses the branch conditions JSON string
func (s *CampaignSequenceStep) GetBranchConditions() (*BranchCondition, error) {
	if s.BranchConditions == "" {
		return &BranchCondition{}, nil
	}

	var conditions BranchCondition
	if err := json.Unmarshal([]byte(s.BranchConditions), &conditions); err != nil {
		return nil, err
	}
	return &conditions, nil
}

// SetBranchConditions serializes branch conditions into JSON string
func (s *CampaignSequenceStep) SetBranchConditions(conditions *BranchCondition) error {
	if conditions == nil {
		s.BranchConditions = ""
		return nil
	}

	conditionsJSON, err := json.Marshal(conditions)
	if err != nil {
		return err
	}
	s.BranchConditions = string(conditionsJSON)
	return nil
}

// GetTotalWaitDuration returns the total wait duration in hours
func (s *CampaignSequenceStep) GetTotalWaitDuration() int {
	return (s.WaitDays * 24) + s.WaitHours
}

// ValidateStepOrdering validates that steps are sequentially ordered starting from 1
func ValidateStepOrdering(steps []CampaignSequenceStep) error {
	if len(steps) == 0 {
		return errors.New("sequence must have at least one step")
	}

	// Check for sequential ordering
	for i, step := range steps {
		expectedOrder := i + 1
		if step.StepOrder != expectedOrder {
			return errors.New("steps must be sequentially ordered starting from 1")
		}
	}

	return nil
}

// ValidateBranchTargets validates that branch condition targets reference valid steps
func (t *SequenceTemplateWithSteps) ValidateBranchTargets() error {
	// Create a map of valid step orders
	validSteps := make(map[int]bool)
	for _, step := range t.Steps {
		validSteps[step.StepOrder] = true
	}

	// Validate each step's branch conditions
	for _, step := range t.Steps {
		conditions, err := step.GetBranchConditions()
		if err != nil {
			return err
		}

		// Check that branch targets reference valid steps
		if conditions.OnOpened != nil && !validSteps[*conditions.OnOpened] {
			return errors.New("branch target on_opened references non-existent step")
		}
		if conditions.OnClicked != nil && !validSteps[*conditions.OnClicked] {
			return errors.New("branch target on_clicked references non-existent step")
		}
		if conditions.OnReplied != nil && !validSteps[*conditions.OnReplied] {
			return errors.New("branch target on_replied references non-existent step")
		}
		if conditions.OnIgnored != nil && !validSteps[*conditions.OnIgnored] {
			return errors.New("branch target on_ignored references non-existent step")
		}
	}

	return nil
}

// DetectCircularBranches detects circular branches in the sequence
func (t *SequenceTemplateWithSteps) DetectCircularBranches() error {
	// Build adjacency list for branch graph
	graph := make(map[int][]int)
	for _, step := range t.Steps {
		conditions, err := step.GetBranchConditions()
		if err != nil {
			return err
		}

		// Add edges for each branch condition
		if conditions.OnOpened != nil {
			graph[step.StepOrder] = append(graph[step.StepOrder], *conditions.OnOpened)
		}
		if conditions.OnClicked != nil {
			graph[step.StepOrder] = append(graph[step.StepOrder], *conditions.OnClicked)
		}
		if conditions.OnReplied != nil {
			graph[step.StepOrder] = append(graph[step.StepOrder], *conditions.OnReplied)
		}
		if conditions.OnIgnored != nil {
			graph[step.StepOrder] = append(graph[step.StepOrder], *conditions.OnIgnored)
		}
	}

	// DFS-based cycle detection
	visited := make(map[int]bool)
	recStack := make(map[int]bool)

	var hasCycle func(node int) bool
	hasCycle = func(node int) bool {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if hasCycle(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true // Cycle detected
			}
		}

		recStack[node] = false
		return false
	}

	// Check for cycles starting from each step
	for _, step := range t.Steps {
		if !visited[step.StepOrder] {
			if hasCycle(step.StepOrder) {
				return errors.New("circular branch detected in sequence")
			}
		}
	}

	return nil
}

// Validate performs comprehensive validation on the template with steps
func (t *SequenceTemplateWithSteps) Validate() error {
	// Validate step ordering
	if err := ValidateStepOrdering(t.Steps); err != nil {
		return err
	}

	// Validate branch targets exist
	if err := t.ValidateBranchTargets(); err != nil {
		return err
	}

	// Detect circular branches
	if err := t.DetectCircularBranches(); err != nil {
		return err
	}

	return nil
}
