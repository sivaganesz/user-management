package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/white/user-management/internal/cache"
	"github.com/white/user-management/internal/models"
	"github.com/white/user-management/internal/repositories"
	"github.com/white/user-management/pkg/kafka"
	"github.com/white/user-management/pkg/uuid"
)

// TemplateHandler handles template-related HTTP requests
type TemplateHandler struct {
	templateRepo       *repositories.TemplateRepository
	activityRepo       *repositories.ActivityRepository
	userRepo           *repositories.MongoUserRepository
	kafkaProducer      *kafka.Producer
	// geminiClient       *gemini.GeminiClient
	// rateLimiter        *utils.RateLimiter
	cache              *cache.TemplateCache      // Redis cache for templates
	// integrationHandler *IntegrationHandler       // For Exotel template submission
}

// NewTemplateHandler creates a new template handler
func NewTemplateHandler(templateRepo *repositories.TemplateRepository, activityRepo *repositories.ActivityRepository, kafkaProducer *kafka.Producer, userRepo *repositories.MongoUserRepository, templateCache *cache.TemplateCache) *TemplateHandler {
	return &TemplateHandler{
		templateRepo:  templateRepo,
		activityRepo:  activityRepo,
		userRepo:      userRepo,
		kafkaProducer: kafkaProducer,
		// geminiClient:  geminiClient,
		// rateLimiter:   utils.NewRateLimiter(10, 1*time.Hour), // 10 requests per hour
		cache:         templateCache,
	}
}


// getTenantID gets tenant ID from context, falling back to user ID if not set
// This allows the template system to work in single-tenant mode where tenant_id
// is not explicitly set in the JWT token
func (h *TemplateHandler) getTenantID(r *http.Request) (string, error) {
	// First try to get tenant_id from context
	if tenantIDStr, ok := r.Context().Value("tenant_id").(string); ok {
		return tenantIDStr, nil
	}
	// Fall back to user_id as tenant_id (single-tenant mode)
	if userID, ok := r.Context().Value("user_id").(string); ok {
		return userID, nil
	}
	// Generate a default tenant ID for testing/development
	return uuid.MustNewUUID(), nil
}

// =====================================================
// Template CRUD Operations
// =====================================================

// CreateTemplate godoc
// @Summary Create a new template
// @Description Creates a new communication template with channel-specific validation. Sets status to draft and version to 1.
// @Tags Templates
// @Accept json
// @Produce json
// @Param template body models.CreateTemplateRequest true "Template creation request"
// @Success 201 {object} models.MongoTemplate
// @Failure 400 {object} map[string]string "Invalid request payload or validation error"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/templates [post]
// @Security BearerAuth
func (h *TemplateHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	var req models.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Get user ID from context
	var createdBy string
	if userID, ok := r.Context().Value("user_id").(string); ok {
		createdBy = userID
	} else {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get tenant ID from context (with fallback to user ID)
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid tenant ID")
		return
	}

	// Validate channel
	if !models.IsValidChannel(req.Channel) {
		respondWithError(w, http.StatusBadRequest, "Invalid channel: must be email, sms, whatsapp, or linkedin")
		return
	}

	// Build content from request (merge convenience fields into content map)
	content := req.Content
	if content == nil {
		content = make(map[string]string)
	}

	// Extract subject and body for convenience fields
	subject := req.Subject
	body := req.Message
	if subject == "" && content != nil {
		subject = content["subject"]
	}
	if body == "" && content != nil {
		if b, ok := content["body"]; ok {
			body = b
		} else if b, ok := content["body_html"]; ok {
			body = b
		}
	}

	// Merge convenience fields into content if provided
	if req.Subject != "" {
		content["subject"] = req.Subject
	}
	if req.Message != "" {
		// Determine which content field to use based on channel
		switch req.Channel {
		case "email":
			if _, exists := content["body_html"]; !exists {
				content["body_html"] = req.Message
			}
		default:
			content["body"] = req.Message
		}
	}

	// Determine status - use requested status or default to draft
	status := "draft"
	if req.Status != "" {
		status = req.Status
	}

	// Parse ServiceID if provided
	var serviceID string
	if req.ServiceID != "" {
		// Validate UUID format if needed
		if err := uuid.ValidateUUID(req.ServiceID); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid service ID format")
			return
		}
		serviceID = req.ServiceID
	}

	// Create template object
	now := time.Now()
	template := &models.MongoTemplate{
		ID:           uuid.MustNewUUID(),
		TenantID:     tenantID,
		Name:         req.Name,
		Description:  req.Description,
		Channel:      req.Channel,
		Status:       status,
		Content:      content,
		Subject:      subject,
		Body:         body,
		CustomFields: req.CustomFields,
		Tags:         req.Tags,
		Version:      1,
		IsSystem:     false,
		CreatedBy:    createdBy,
		CreatedAt:    now,
		UpdatedAt:    now,
		// Frontend-compatible fields
		ForStage:     req.ForStage,
		Industries:   req.Industries,
		ApprovalFlag: req.ApprovalFlag,
		AiEnhanced:   req.AiEnhanced,
		ServiceID:    serviceID,
		// Channel-specific fields
		TemplateType:     req.TemplateType,
		MetaTemplateName: req.MetaTemplateName,
	}

	// Extract merge tags from content
	template.Variables = template.ExtractMergeTags()

	// Validate merge tags (returns warnings for undefined tags)
	mergeTagWarnings := template.ValidateMergeTags()
	if len(mergeTagWarnings) > 0 {
		// Add warnings to response header (non-fatal)
		w.Header().Set("X-Merge-Tag-Warnings", strings.Join(mergeTagWarnings, "; "))
	}

	// Validate template (channel-specific validation)
	if err := template.Validate(); err != nil {
		respondWithError(w, http.StatusBadRequest, "Template validation failed: "+err.Error())
		return
	}

	// Create template in database
	if err := h.templateRepo.CreateTemplateCompat(template); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create template: "+err.Error())
		return
	}

	// Publish Kafka event (fire-and-forget)
	if h.kafkaProducer != nil {
		event := map[string]interface{}{
			"event_type":  "template.created",
			"template_id": template.ID,
			"tenant_id":   template.TenantID,
			"channel":     template.Channel,
			"status":      template.Status,
			"created_by":  createdBy,
			"created_at":  now.Unix(),
		}
		_ = h.kafkaProducer.PublishJSON(ctx, "template.created", event)
	}

	// Log activity
	activity := &models.Activity{
		ID:            uuid.MustNewUUID(),
		ActivityType:  "note",
		Title:         "Template Created",
		Description:   "Template created: " + template.Name,
		Owner:         createdBy,
		RelatedToType: "template",
		RelatedToID:   template.ID,
		Status:        "completed",
		Priority:      "medium",
		CreatedBy:     createdBy,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	_ = h.activityRepo.CreateActivityCompat(activity)

	// Return template directly (MongoTemplate has proper JSON tags)
	respondWithJSON(w, http.StatusCreated, template)
}