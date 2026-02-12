package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/white/user-management/internal/middleware"
	"github.com/white/user-management/internal/models"
	"github.com/white/user-management/internal/repositories"
	"github.com/white/user-management/internal/services"
	"github.com/white/user-management/pkg/kafka"
	"github.com/white/user-management/pkg/uuid"
)

// SequenceTemplateHandler handles sequence template operations
type SequenceTemplateHandler struct {
	sequenceRepo  *repositories.SequenceTemplateRepository
	activityRepo  *repositories.ActivityRepository
	userRepo      *repositories.MongoUserRepository
	kafkaProducer *kafka.Producer
}

// NewSequenceTemplateHandler creates a new sequence template handler
func NewSequenceTemplateHandler(
	sequenceRepo *repositories.SequenceTemplateRepository,
	activityRepo *repositories.ActivityRepository,
	userRepo *repositories.MongoUserRepository,
	kafkaProducer *kafka.Producer,
) *SequenceTemplateHandler {
	return &SequenceTemplateHandler{
		sequenceRepo:  sequenceRepo,
		activityRepo:  activityRepo,
		userRepo:      userRepo,
		kafkaProducer: kafkaProducer,
	}
}


// CreateSequenceTemplate godoc
// @Summary Create a new sequence template
// @Description Creates a new multi-step sequence template. Steps use delayDays (days after previous step; 0 for first) and sendAt (required, HH:MM 24h). Legacy waitDays/waitHours/sendTime are accepted but deprecated; scheduling uses delay_days + send_at only.
// @Tags Sequences
// @Accept json
// @Produce json
// @Param template body models.SequenceTemplateWithSteps true "Sequence template with steps"
// @Success 201 {object} models.SequenceTemplateWithSteps
// @Failure 400 {object} map[string]interface{} "Invalid request payload or validation error"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/sequences [post]
// @Security BearerAuth
func (h *SequenceTemplateHandler) CreateSequenceTemplate(w http.ResponseWriter, r *http.Request) {
	// Parse request body - support both flat and nested structures
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request payload: " + err.Error(),
			},
		})
		return
	}

	// Build the template structure
	var template models.SequenceTemplateWithSteps

	// Check if payload has nested 'template' field or flat structure
	if _, hasTemplate := payload["template"]; hasTemplate {
		// Nested structure with 'template' and 'steps'
		payloadBytes, _ := json.Marshal(payload)
		if err := json.Unmarshal(payloadBytes, &template); err != nil {
			respondWithJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"error": map[string]interface{}{
					"code":    "INVALID_REQUEST",
					"message": "Invalid template structure: " + err.Error(),
				},
			})
			return
		}
	} else {
		// Flat structure - convert to nested
		template.Template.Name, _ = payload["name"].(string)
		template.Template.Description, _ = payload["description"].(string)

		// Parse service_id (supports serviceId, service_id)
		if serviceID, ok := payload["serviceId"].(string); ok && serviceID != "" {
			template.Template.ServiceID = serviceID
		} else if serviceID, ok := payload["service_id"].(string); ok && serviceID != "" {
			template.Template.ServiceID = serviceID
		}

		// Parse schedule_id (supports scheduleId, schedule_id)
		if scheduleID, ok := payload["scheduleId"].(string); ok && scheduleID != "" {
			template.Template.ScheduleID = scheduleID
		} else if scheduleID, ok := payload["schedule_id"].(string); ok && scheduleID != "" {
			template.Template.ScheduleID = scheduleID
		}

		// Parse isActive (supports isActive, is_active)
		if isActive, ok := payload["isActive"].(bool); ok {
			template.Template.IsActive = isActive
		} else if isActive, ok := payload["is_active"].(bool); ok {
			template.Template.IsActive = isActive
		} else {
			template.Template.IsActive = true // Default to active
		}

		// Parse steps array and map field names
		if stepsData, hasSteps := payload["steps"]; hasSteps {
			if stepsArray, ok := stepsData.([]interface{}); ok {
				template.Steps = make([]models.CampaignSequenceStep, len(stepsArray))
				for i, stepData := range stepsArray {
					stepMap, ok := stepData.(map[string]interface{})
					if !ok {
						continue
					}

					// Map field names to model field names
					step := &template.Steps[i]

					// order / step_number / step_order -> StepOrder
					if order, ok := stepMap["order"].(float64); ok {
						step.StepOrder = int(order)
					} else if stepNum, ok := stepMap["step_number"].(float64); ok {
						step.StepOrder = int(stepNum)
					} else if stepOrder, ok := stepMap["step_order"].(float64); ok {
						step.StepOrder = int(stepOrder)
					}

					// communicationType / channel -> Channel
					if commType, ok := stepMap["communicationType"].(string); ok {
						step.Channel = commType
					} else if channel, ok := stepMap["channel"].(string); ok {
						step.Channel = channel
					}

					// subject (for email)
					if subject, ok := stepMap["subject"].(string); ok {
						step.Subject = subject
					}

					// message / body -> Body
					if message, ok := stepMap["message"].(string); ok {
						step.Body = message
					} else if body, ok := stepMap["body"].(string); ok {
						step.Body = body
					}

					// delayDays / delay_days / waitDays / wait_days -> DelayDays and WaitDays (proposed + legacy)
					if delayDays, ok := stepMap["delayDays"].(float64); ok {
						step.DelayDays = int(delayDays)
						step.WaitDays = int(delayDays)
					} else if delayDays, ok := stepMap["delay_days"].(float64); ok {
						step.DelayDays = int(delayDays)
						step.WaitDays = int(delayDays)
					} else if waitDays, ok := stepMap["waitDays"].(float64); ok {
						step.WaitDays = int(waitDays)
						if step.DelayDays == 0 {
							step.DelayDays = int(waitDays)
						}
					} else if waitDays, ok := stepMap["wait_days"].(float64); ok {
						step.WaitDays = int(waitDays)
						if step.DelayDays == 0 {
							step.DelayDays = int(waitDays)
						}
					}

					// waitHours / wait_hours (deprecated; ignored for new model)
					if waitHours, ok := stepMap["waitHours"].(float64); ok {
						step.WaitHours = int(waitHours)
					} else if waitHours, ok := stepMap["wait_hours"].(float64); ok {
						step.WaitHours = int(waitHours)
					}

					// sendAt / send_at / sendTime / send_time -> SendAt and SendTime (required HH:MM)
					if sendAt, ok := stepMap["sendAt"].(string); ok && sendAt != "" {
						step.SendAt = sendAt
						step.SendTime = sendAt
					} else if sendAt, ok := stepMap["send_at"].(string); ok && sendAt != "" {
						step.SendAt = sendAt
						step.SendTime = sendAt
					} else if sendTime, ok := stepMap["sendTime"].(string); ok && sendTime != "" {
						step.SendTime = sendTime
						step.SendAt = sendTime
					} else if sendTime, ok := stepMap["send_time"].(string); ok && sendTime != "" {
						step.SendTime = sendTime
						step.SendAt = sendTime
					}

					// templateId / content_template_id -> ContentTemplateID
					if templateIDStr, ok := stepMap["templateId"].(string); ok && templateIDStr != "" {
						step.ContentTemplateID = templateIDStr
					} else if templateIDStr, ok := stepMap["content_template_id"].(string); ok && templateIDStr != "" {
						step.ContentTemplateID = templateIDStr
					}

					// Generate content_template_id if not provided
					if step.ContentTemplateID == "" {
						step.ContentTemplateID = uuid.MustNewUUID()
					}

					// Parse attachments array
					if attachmentsData, hasAttachments := stepMap["attachments"]; hasAttachments {
						if attachmentsArray, ok := attachmentsData.([]interface{}); ok {
							step.Attachments = make([]models.SequenceStepAttachment, 0, len(attachmentsArray))
							for _, attachData := range attachmentsArray {
								attachMap, ok := attachData.(map[string]interface{})
								if !ok {
									continue
								}
								attachment := models.SequenceStepAttachment{}
								if id, ok := attachMap["id"].(string); ok {
									attachment.ID = id
								}
								if name, ok := attachMap["name"].(string); ok {
									attachment.Name = name
								}
								if typ, ok := attachMap["type"].(string); ok {
									attachment.Type = typ
								}
								if size, ok := attachMap["size"].(string); ok {
									attachment.Size = size
								}
								if webUrl, ok := attachMap["webUrl"].(string); ok {
									attachment.WebURL = webUrl
								} else if webUrl, ok := attachMap["web_url"].(string); ok {
									attachment.WebURL = webUrl
								}
								if category, ok := attachMap["category"].(string); ok {
									attachment.Category = category
								}
								step.Attachments = append(step.Attachments, attachment)
							}
						}
					}
				}
			}
		}
	}

	// Validate required fields
	if template.Template.Name == "" {
		respondWithJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "VALIDATION_ERROR",
				"message": "Template name is required",
			},
		})
		return
	}

	if len(template.Steps) == 0 {
		respondWithJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "VALIDATION_ERROR",
				"message": "At least one step is required",
			},
		})
		return
	}

	// Validate step ordering
	if err := models.ValidateStepOrdering(template.Steps); err != nil {
		respondWithJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid step ordering: " + err.Error(),
			},
		})
		return
	}

	// Normalize: ensure DelayDays and SendAt are set from legacy fields for persistence
	for i := range template.Steps {
		if template.Steps[i].DelayDays == 0 && template.Steps[i].WaitDays != 0 {
			template.Steps[i].DelayDays = template.Steps[i].WaitDays
		}
		if template.Steps[i].SendAt == "" && template.Steps[i].SendTime != "" {
			template.Steps[i].SendAt = template.Steps[i].SendTime
		}
	}

	// Validate sequence step timing: send_at required, format HH:MM
	if err := models.ValidateSequenceStepTiming(template.Steps); err != nil {
		respondWithJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	// Validate branch conditions
	if err := template.Validate(); err != nil {
		respondWithJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "VALIDATION_ERROR",
				"message": "Validation failed: " + err.Error(),
			},
		})
		return
	}

	// Get user ID from JWT context (set by authMiddleware)
	userID := middleware.GetUserID(r)
	if userID == "" {
		// If user ID not in context, return unauthorized
		respondWithJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "User not authenticated",
			},
		})
		return
	}
	template.Template.CreatedBy = userID

	// Generate template ID
	template.Template.TemplateID = uuid.MustNewUUID()

	// Set template ID on all steps
	for i := range template.Steps {
		template.Steps[i].TemplateID = template.Template.TemplateID
	}

	// Create sequence template
	if err := h.sequenceRepo.CreateSequenceTemplate(&template); err != nil {
		respondWithJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "DATABASE_ERROR",
				"message": "Failed to create sequence template: " + err.Error(),
			},
		})
		return
	}

	// Publish Kafka event (fire-and-forget)
	if h.kafkaProducer != nil {
		event := map[string]interface{}{
			"event_type":  "sequence_template_created",
			"template_id": template.Template.TemplateID,
			"name":        template.Template.Name,
			"version":     template.Template.Version,
			"step_count":  len(template.Steps),
			"created_by":  template.Template.CreatedBy,
			"timestamp":   time.Now().Format(time.RFC3339),
		}
		_ = h.kafkaProducer.PublishJSON(ctx, "sequence-events", event)
	}

	// Log activity
	if h.activityRepo != nil {
		activity := &models.Activity{
			ID:            uuid.MustNewUUID(),
			ActivityType:  "sequence_template_created",
			Title:         "Sequence Template Created",
			Description:   "Created sequence template: " + template.Template.Name,
			Owner:         userID,
			RelatedToType: "sequence_template",
			RelatedToID:   template.Template.TemplateID,
			Status:        "completed",
			Priority:      "normal",
			CreatedAt:     time.Now(),
		}
		_ = h.activityRepo.CreateActivityCompat(activity)
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"success":  true,
		"id":       template.Template.TemplateID,
		"template": template,
	})
}