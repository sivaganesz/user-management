package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/white/user-management/internal/cache"
	"github.com/white/user-management/internal/middleware"
	"github.com/white/user-management/internal/models"
	"github.com/white/user-management/internal/repositories"
	"github.com/white/user-management/internal/services"
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

// ListTemplates godoc
// @Summary List templates with filters
// @Description Retrieves templates with optional filtering by channel, status, created_by, tag, search term, and pagination support. Tag filtering uses templates_by_tag lookup table for efficient retrieval.
// @Tags Templates
// @Accept json
// @Produce json
// @Param channel query string false "Filter by channel (email, sms, whatsapp, linkedin)"
// @Param status query string false "Filter by status (draft, published)"
// @Param created_by query string false "Filter by creator user ID (UUID)"
// @Param tag query string false "Filter by tag (single tag name or comma-separated for multiple tags, uses AND logic)"
// @Param search query string false "Search in template name and description"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 50, max: 100)"
// @Param sort_by query string false "Sort by field (name, created_at, updated_at)"
// @Param sort_order query string false "Sort order (asc, desc)"
// @Success 200 {object} models.TemplateListResponse
// @Failure 400 {object} map[string]string "Invalid query parameters"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/templates [get]
// @Security BearerAuth
func (h *TemplateHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	// Get tenant ID from context (with fallback to user ID)
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid tenant ID")
		return
	}

	dataScope, claims, err := h.getCampaignScope(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if _, denyAll := services.BuildScopeFilter("campaigns", dataScope, claims); denyAll {
		respondWithError(w, http.StatusForbidden, "Permission denied")
		return
	}

	// Parse tag filter (single or comma-separated)
	tagFilter := r.URL.Query().Get("tag")
	var templates []*models.MongoTemplate

	if tagFilter != "" {
		// Use tag-based lookup for efficient filtering
		tags := strings.Split(tagFilter, ",")

		if len(tags) == 1 {
			// Single tag - use templates_by_tag lookup
			tagName := strings.TrimSpace(tags[0])
			templates, err = h.templateRepo.GetTemplatesByTag(tagName, 1000)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to retrieve templates by tag: "+err.Error())
				return
			}
		} else {
			// Multiple tags - use AND logic (template must have ALL tags)
			// Get templates for first tag
			firstTag := strings.TrimSpace(tags[0])
			templates, err = h.templateRepo.GetTemplatesByTag(firstTag, 1000)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to retrieve templates by tag: "+err.Error())
				return
			}

			// Filter to only templates that have ALL tags
			var filteredTemplates []*models.MongoTemplate
			for _, template := range templates {
				hasAllTags := true
				for _, tag := range tags {
					tagName := strings.TrimSpace(tag)
					if !template.HasTag(tagName) {
						hasAllTags = false
						break
					}
				}
				if hasAllTags {
					filteredTemplates = append(filteredTemplates, template)
				}
			}
			templates = filteredTemplates
		}

		// Apply additional filters (channel, status, search, etc.) to tag-filtered results
		// Accept both camelCase (frontend) and snake_case parameter names
		tagSortBy := r.URL.Query().Get("sortBy")
		if tagSortBy == "" {
			tagSortBy = r.URL.Query().Get("sort_by")
		}
		tagSortOrder := r.URL.Query().Get("sortOrder")
		if tagSortOrder == "" {
			tagSortOrder = r.URL.Query().Get("sort_order")
		}

		filters := repositories.TemplateFilters{
			TenantID:  tenantID,
			Channel:   r.URL.Query().Get("channel"),
			Status:    r.URL.Query().Get("status"),
			Search:    r.URL.Query().Get("search"),
			SortBy:    tagSortBy,
			SortOrder: tagSortOrder,
		}

		// Apply filters manually
		var finalTemplates []*models.MongoTemplate
		for _, template := range templates {
			// Channel filter
			if filters.Channel != "" && template.Channel != filters.Channel {
				continue
			}

			// Status filter
			if filters.Status != "" && template.Status != filters.Status {
				continue
			}

			// Search filter
			if filters.Search != "" {
				searchLower := strings.ToLower(filters.Search)
				nameLower := strings.ToLower(template.Name)
				descLower := strings.ToLower(template.Description)
				if !strings.Contains(nameLower, searchLower) && !strings.Contains(descLower, searchLower) {
					continue
				}
			}

			finalTemplates = append(finalTemplates, template)
		}
		templates = finalTemplates
	} else {
		// No tag filter - use standard List method
		// Accept both camelCase (frontend) and snake_case parameter names
		sortBy := r.URL.Query().Get("sortBy")
		if sortBy == "" {
			sortBy = r.URL.Query().Get("sort_by")
		}
		sortOrder := r.URL.Query().Get("sortOrder")
		if sortOrder == "" {
			sortOrder = r.URL.Query().Get("sort_order")
		}

		filters := repositories.TemplateFilters{
			TenantID:     tenantID,
			Channel:      r.URL.Query().Get("channel"),
			Status:       r.URL.Query().Get("status"),
			Search:       r.URL.Query().Get("search"),
			SortBy:       sortBy,
			SortOrder:    sortOrder,
			ApprovalFlag: r.URL.Query().Get("approvalFlag"),
			Performance:  r.URL.Query().Get("performance"),
		}

		// Parse serviceId filter (UUID reference)
		if serviceIDStr := r.URL.Query().Get("serviceId"); serviceIDStr != "" {
			if err := uuid.ValidateUUID(serviceIDStr); err != nil {
				respondWithError(w, http.StatusBadRequest, "Invalid serviceId format")
				return
			}
			filters.ServiceID = serviceIDStr
		}

		// Parse forStage filter - supports both:
		// 1. Multiple params: forStage=prospect&forStage=mql
		// 2. Comma-separated: forStage=prospect,mql
		if forStageValues := r.URL.Query()["forStage"]; len(forStageValues) > 0 {
			for _, val := range forStageValues {
				// Split each value by comma in case of comma-separated format
				parts := strings.Split(val, ",")
				for _, part := range parts {
					trimmed := strings.TrimSpace(part)
					if trimmed != "" {
						filters.ForStage = append(filters.ForStage, trimmed)
					}
				}
			}
		}

		// Parse industries filter - supports both formats
		if industriesValues := r.URL.Query()["industries"]; len(industriesValues) > 0 {
			for _, val := range industriesValues {
				parts := strings.Split(val, ",")
				for _, part := range parts {
					trimmed := strings.TrimSpace(part)
					if trimmed != "" {
						filters.Industries = append(filters.Industries, trimmed)
					}
				}
			}
		}

		// Parse created_by filter
		if createdByStr := r.URL.Query().Get("created_by"); createdByStr != "" {
			if err := uuid.ValidateUUID(createdByStr); err != nil {
				respondWithError(w, http.StatusBadRequest, "Invalid created_by UUID format")
				return
			}
			filters.CreatedBy = createdByStr
		}

		// Parse pagination
		filters.Page = 1
		if pageStr := r.URL.Query().Get("page"); pageStr != "" {
			page, err := strconv.Atoi(pageStr)
			if err != nil || page < 1 {
				respondWithError(w, http.StatusBadRequest, "Invalid page number")
				return
			}
			filters.Page = page
		}

		filters.Limit = 50
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil || limit < 1 {
				respondWithError(w, http.StatusBadRequest, "Invalid limit value")
				return
			}
			if limit > 100 {
				limit = 100
			}
			filters.Limit = limit
		}

		// Validate channel if provided
		if filters.Channel != "" && !models.IsValidChannel(filters.Channel) {
			respondWithError(w, http.StatusBadRequest, "Invalid channel: must be email, sms, whatsapp, or linkedin")
			return
		}

		// Validate status if provided
		if filters.Status != "" && !models.IsValidTemplateStatus(filters.Status) {
			respondWithError(w, http.StatusBadRequest, "Invalid status: must be draft or published")
			return
		}

		// If scope is not "all", fetch a larger window and apply scope filtering + pagination in-memory.
		requestedPage := filters.Page
		requestedLimit := filters.Limit
		scopeValue := strings.ToLower(strings.TrimSpace(services.ScopeValueForResource(dataScope, "campaigns")))
		if scopeValue == "all" || scopeValue == "" {
			// Admin / full scope (or empty = all): do not filter by tenant_id so all org templates are visible
			filters.TenantID = ""
		}
		if scopeValue != "" && scopeValue != "all" {
			filters.Page = 1
			filters.Limit = 1000
		}

		// Call repository (pagination is already applied via filters.Page and filters.Limit)
		templates, err = h.templateRepo.ListTemplates(filters)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to retrieve templates: "+err.Error())
			return
		}

		// Enforce RBAC Data Scope (campaigns scope applies to templates)
		filtered := make([]*models.MongoTemplate, 0, len(templates))
		for _, t := range templates {
			if services.IsInScope("campaigns", dataScope, claims, t) {
				filtered = append(filtered, t)
			}
		}
		templates = filtered
		totalCount := int64(len(templates))

		// Apply pagination if we fetched a larger window
		page := requestedPage
		limit := requestedLimit
		if scopeValue != "" && scopeValue != "all" {
			start := (page - 1) * limit
			if start < 0 {
				start = 0
			}
			end := start + limit
			if start >= len(templates) {
				templates = []*models.MongoTemplate{}
			} else {
				if end > len(templates) {
					end = len(templates)
				}
				templates = templates[start:end]
			}
		}

		// Calculate total pages
		totalPages := (int(totalCount) + requestedLimit - 1) / requestedLimit

		// Build response
		response := map[string]interface{}{
			"templates":  templates,
			"total":      totalCount,
			"page":       requestedPage,
			"limit":      requestedLimit,
			"totalPages": totalPages,
		}

		respondWithJSON(w, http.StatusOK, response)
		return
	}

	// For tag-filtered results, pagination is applied manually
	// Get total count from filtered results
	totalCount := len(templates)

	// Enforce RBAC Data Scope (campaigns scope applies to templates)
	filtered := make([]*models.MongoTemplate, 0, len(templates))
	for _, t := range templates {
		if services.IsInScope("campaigns", dataScope, claims, t) {
			filtered = append(filtered, t)
		}
	}
	templates = filtered
	totalCount = len(templates)

	// Parse pagination params
	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		parsedPage, err := strconv.Atoi(pageStr)
		if err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			if parsedLimit > 100 {
				parsedLimit = 100
			}
			limit = parsedLimit
		}
	}

	// Calculate pagination
	totalPages := (totalCount + limit - 1) / limit
	startIdx := (page - 1) * limit
	endIdx := startIdx + limit

	if startIdx >= totalCount {
		templates = []*models.MongoTemplate{} // Empty results for out-of-range page
	} else {
		if endIdx > totalCount {
			endIdx = totalCount
		}
		templates = templates[startIdx:endIdx]
	}

	// Build response - MongoTemplate has proper JSON tags
	response := map[string]interface{}{
		"templates":  templates,
		"total":      totalCount,
		"page":       page,
		"limit":      limit,
		"totalPages": totalPages,
	}

	respondWithJSON(w, http.StatusOK, response)
}

// GetTemplate godoc
// @Summary Get template by ID
// @Description Retrieves a specific template by its unique identifier with all content fields
// @Tags Templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID (UUID)"
// @Success 200 {object} models.MongoTemplate
// @Failure 400 {object} map[string]string "Invalid template ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Template not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/templates/{id} [get]
// @Security BearerAuth
func (h *TemplateHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templateID := vars["id"]
	err := uuid.ValidateUUID(templateID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid template ID format")
		return
	}

	// Get tenant ID from context (with fallback to user ID)
	tenantID, err := h.getTenantID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid tenant ID")
		return
	}

	var template *models.MongoTemplate

	// Cache-aside pattern: Try to get from cache first (only if cache is available)
	if h.cache != nil {
		cachedTemplate, err := h.cache.Get(tenantID, templateID)
		if err == nil {
			// Cache hit
			template = cachedTemplate
			goto respondTemplate
		}
		// Cache miss - continue to database query
	}

	// Get template from database
	template, err = h.templateRepo.GetTemplateByIDCompat(tenantID, templateID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Template not found: "+err.Error())
		return
	}

respondTemplate:
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

	// Cache the template if it's published (only cache published templates)
	// Draft templates change frequently and should not be cached
	if h.cache != nil && template.Status == "published" {
		if err := h.cache.Set(template); err != nil {
			// Log error but don't fail the request
			// Caching is a performance optimization, not a requirement
			// We could add logging here: log.Printf("Failed to cache template: %v", err)
		}
	}

	// Return frontend-compatible response
	respondWithJSON(w, http.StatusOK, template)
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

// DeleteTemplate godoc
// @Summary Delete a template
// @Description Deletes a template (hard delete). System templates cannot be deleted. Removes tag associations and updates counters.
// @Tags Templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID (UUID)"
// @Success 204 "No Content - Template deleted successfully"
// @Failure 400 {object} map[string]string "Invalid template ID or cannot delete system template"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Template not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/templates/{id} [delete]
// @Security BearerAuth
func (h *TemplateHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	vars := mux.Vars(r)
	templateID := vars["id"]
	err := uuid.ValidateUUID(templateID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid template ID format")
		return
	}

	// Get user ID from context
	var deletedBy string
	if userID, ok := r.Context().Value("user_id").(string); ok {
		deletedBy = userID
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

	// Check if template can be deleted
	if !template.CanDelete() {
		respondWithError(w, http.StatusBadRequest, "System templates (library) cannot be deleted")
		return
	}

	// Remove tag associations and update counters
	for _, tagName := range template.Tags {
		_ = h.templateRepo.RemoveTagFromTemplateCompat(tenantID, tagName)
	}

	// Delete template
	if err := h.templateRepo.DeleteTemplateCompat(tenantID, templateID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete template: "+err.Error())
		return
	}

	// Invalidate cache (template deleted)
	if h.cache != nil {
		_ = h.cache.Delete(tenantID, templateID)
	}

	// Publish Kafka event (fire-and-forget)
	if h.kafkaProducer != nil {
		event := map[string]interface{}{
			"event_type":  "template.deleted",
			"template_id": templateID,
			"tenant_id":   tenantID,
			"deleted_by":  deletedBy,
			"deleted_at":  time.Now().Unix(),
		}
		_ = h.kafkaProducer.PublishJSON(ctx, "template.deleted", event)
	}

	// Log activity
	now := time.Now()
	activity := &models.Activity{
		ID:            uuid.MustNewUUID(),
		ActivityType:  "note",
		Title:         "Template Deleted",
		Description:   "Template deleted: " + template.Name,
		Owner:         deletedBy,
		RelatedToType: "template",
		RelatedToID:   templateID,
		Status:        "completed",
		Priority:      "high",
		CreatedBy:     deletedBy,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_ = h.activityRepo.CreateActivityCompat(activity)

	w.WriteHeader(http.StatusNoContent)
}

func (h *TemplateHandler) getCampaignScope(r *http.Request) (models.DataScope, services.ScopeClaims, error) {
	ctx := r.Context()
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return models.DataScope{}, services.ScopeClaims{}, fmt.Errorf("user ID not found")
	}
	team, _ := ctx.Value(middleware.TeamKey).(string)
	// region, _ := ctx.Value(middleware.RegionKey).(string)
	region := ""

	dataScope := models.DataScope{Customers: "all", Campaigns: "all"}
	if ds, ok := ctx.Value(middleware.DataScopeKey).(models.DataScope); ok {
		dataScope = ds
	}

	teamUserIDs, err := services.GetTeamUserIDs(ctx, h.userRepo, team)
	if err != nil {
		return models.DataScope{}, services.ScopeClaims{}, err
	}
	return dataScope, services.ScopeClaims{UserID: userID, Team: team, Region: region, TeamUserIDs: teamUserIDs}, nil
}
