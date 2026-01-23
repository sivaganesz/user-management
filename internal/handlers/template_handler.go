package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
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

// UpdateTemplate godoc
// @Summary Update a template
// @Description Updates an existing template. Only draft templates should be updated directly. For published templates, consider creating a new version.
// @Tags Templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID (UUID)"
// @Param template body models.UpdateTemplateRequest true "Template update request"
// @Success 200 {object} models.MongoTemplate
// @Failure 400 {object} map[string]string "Invalid request or attempting to update published template"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Template not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/templates/{id} [put]
// @Security BearerAuth
func (h *TemplateHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ctx := context.Background()
	templateID := vars["id"]
	err := uuid.ValidateUUID(templateID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid template ID format")
		return
	}

	var req models.UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Get user ID from context
	var updatedBy string
	if userID, ok := r.Context().Value("user_id").(string); ok {
		updatedBy = userID
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

	// Fetch existing template
	template, err := h.templateRepo.GetTemplateByIDCompat(tenantID, templateID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Template not found: "+err.Error())
		return
	}

	// Enforce RBAC Data Scope (campaigns scope applies to templates)
	dataScope, claims, err := h.getCampaignScope(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if _, denyAll := services.BuildScopeFilter("campaigns", dataScope, claims); denyAll || !services.IsInScope("campaigns", dataScope, claims, template) {
		respondWithError(w, http.StatusForbidden, "Permission denied")
		return
	}

	// Check if template can be edited
	if !template.CanEdit() {
		respondWithError(w, http.StatusBadRequest, "System templates cannot be edited")
		return
	}

	// Warn if updating published template
	if template.Status == "published" {
		// Allow updates but user should be aware - consider adding a warning header
		w.Header().Set("X-Warning", "Updating published template - consider creating a new version")
	}

	// Apply updates
	if req.Name != "" {
		template.Name = req.Name
	}
	if req.Description != "" {
		template.Description = req.Description
	}
	if req.Content != nil && len(req.Content) > 0 {
		template.Content = req.Content
		// Re-extract merge tags after content update
		template.Variables = template.ExtractMergeTags()
	}
	if req.CustomFields != nil {
		template.CustomFields = req.CustomFields
	}
	if req.Tags != nil {
		template.Tags = req.Tags
	}

	// Apply frontend-compatible fields
	if req.ForStage != nil {
		template.ForStage = req.ForStage
	}
	if req.Industries != nil {
		template.Industries = req.Industries
	}
	if req.ApprovalFlag != "" {
		template.ApprovalFlag = req.ApprovalFlag
	}
	if req.AiEnhanced != nil {
		template.AiEnhanced = *req.AiEnhanced
	}

	// Apply channel-specific fields (email: subject, message)
	if req.Subject != "" {
		if template.Content == nil {
			template.Content = make(map[string]string)
		}
		template.Content["subject"] = req.Subject
	}
	if req.Message != "" {
		if template.Content == nil {
			template.Content = make(map[string]string)
		}
		template.Content["body"] = req.Message
		// Re-extract merge tags from message
		template.Variables = template.ExtractMergeTags()
	}

	// Apply status update
	if req.Status != "" {
		template.Status = req.Status
	}

	// Apply LinkedIn-specific fields
	if req.TemplateType != "" {
		template.TemplateType = req.TemplateType
	}

	// Apply WhatsApp-specific fields
	if req.MetaTemplateName != "" {
		template.MetaTemplateName = req.MetaTemplateName
	}
	if req.Category != "" {
		if template.Content == nil {
			template.Content = make(map[string]string)
		}
		template.Content["category"] = req.Category
	}
	if req.Language != "" {
		if template.Content == nil {
			template.Content = make(map[string]string)
		}
		template.Content["language"] = req.Language
	}

	// Apply service ID (validate UUID)
	if req.ServiceID != "" {
		if err := uuid.ValidateUUID(req.ServiceID); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid service ID format")
			return
		}
		template.ServiceID = req.ServiceID
	}

	// Update timestamp
	template.UpdatedAt = time.Now()

	// Validate merge tags (returns warnings for undefined tags)
	mergeTagWarnings := template.ValidateMergeTags()
	if len(mergeTagWarnings) > 0 {
		// Add warnings to response header (non-fatal)
		w.Header().Set("X-Merge-Tag-Warnings", strings.Join(mergeTagWarnings, "; "))
	}

	// Validate template after updates
	if err := template.Validate(); err != nil {
		respondWithError(w, http.StatusBadRequest, "Template validation failed: "+err.Error())
		return
	}

	// Update in database
	if err := h.templateRepo.UpdateTemplateCompat(template); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update template: "+err.Error())
		return
	}

	// Invalidate cache (template content changed)
	if h.cache != nil {
		_ = h.cache.Delete(tenantID, templateID)
	}

	// Publish Kafka event (fire-and-forget)
	if h.kafkaProducer != nil {
		event := map[string]interface{}{
			"event_type":  "template.updated",
			"template_id": template.ID,
			"tenant_id":   template.TenantID,
			"updated_by":  updatedBy,
			"updated_at":  template.UpdatedAt.Unix(),
		}
		_ = h.kafkaProducer.PublishJSON(ctx, "template.updated", event)
	}

	// Log activity
	now := time.Now()
	activity := &models.Activity{
		ID:            uuid.MustNewUUID(),
		ActivityType:  "note",
		Title:         "Template Updated",
		Description:   "Template updated: " + template.Name,
		Owner:         updatedBy,
		RelatedToType: "template",
		RelatedToID:   template.ID,
		Status:        "completed",
		Priority:      "medium",
		CreatedBy:     updatedBy,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_ = h.activityRepo.CreateActivityCompat(activity)

	// Return frontend-compatible response
	respondWithJSON(w, http.StatusOK, template)
}


