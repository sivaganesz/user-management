package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/white/user-management/internal/events"
	"github.com/white/user-management/internal/middleware"
	"github.com/white/user-management/internal/models"
	"github.com/white/user-management/internal/repositories"
	// "github.com/gorilla/mux"
)

// SettingsHandler handles settings-related HTTP requests
// NOTE: Profile is read-only (managed by O365), so no userRepo needed for password changes
type SettingsHandler struct {
	repo *repositories.SettingsRepository
	// approvalRuleRepo *repositories.ApprovalRuleRepository
	auditPublisher *events.AuditPublisher
}

// NewSettingsHandler creates a new SettingsHandler
// func NewSettingsHandler(repo *repositories.SettingsRepository, approvalRuleRepo *repositories.ApprovalRuleRepository) *SettingsHandler {
func NewSettingsHandler(repo *repositories.SettingsRepository, auditPublisher *events.AuditPublisher) *SettingsHandler {
	return &SettingsHandler{
		repo: repo,
		// approvalRuleRepo: approvalRuleRepo,
		auditPublisher: auditPublisher,
	}
}

// Helper to get userID from context
func (h *SettingsHandler) getUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	return userID, ok
}

// ==================== User Profile ====================

// GetProfile godoc
// @Summary Get user profile
// @Description Get current user's profile information
// @Tags Settings
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /settings/profile [get]
func (h *SettingsHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	profile, err := h.repo.GetUserProfile(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get profile: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    profile,
	})
}

// ==================== Company Info ====================

// GetCompanyInfo godoc
// @Summary Get company information
// @Description Get company information
// @Tags Settings
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /system/company [get]
func (h *SettingsHandler) GetCompanyInfo(w http.ResponseWriter, r *http.Request) {
	_, ok := h.getUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	info, err := h.repo.GetCompanyInfo(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get company info: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    info,
	})
}

// UpdateCompanyInfo godoc
// @Summary Update company information
// @Description Update company information (admin only)
// @Tags Settings
// @Accept json
// @Produce json
// @Param request body models.SettingsUpdateCompanyInfoRequest true "Company info update data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /system/company [put]
func (h *SettingsHandler) UpdateCompanyInfo(w http.ResponseWriter, r *http.Request) {
	_, ok := h.getUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req models.SettingsUpdateCompanyInfoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	info, err := h.repo.UpdateCompanyInfo(r.Context(), &req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update company info: "+err.Error())
		return
	}

	// Publish audit log event (fire-and-forget)
	if h.auditPublisher != nil {
		userID, _ := h.getUserID(r)
		userName, _ := r.Context().Value(middleware.NameKey).(string)
		h.auditPublisher.PublishSettingsEvent(
			r,
			userID,
			userName,
			events.ActionSettingsUpdated,
			"Company information updated",
		)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    info,
		"message": "Company information updated successfully",
	})
}

// ==================== Notification Settings ====================

// GetNotificationSettings godoc
// @Summary Get notification settings
// @Description Get current user's notification settings
// @Tags Settings
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /system/notifications [get]
func (h *SettingsHandler) GetNotificationSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	settings, err := h.repo.GetNotificationSettings(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get notification settings: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    settings,
	})
}

// UpdateNotificationSettings godoc
// @Summary Update notification settings
// @Description Update current user's notification settings
// @Tags Settings
// @Accept json
// @Produce json
// @Param request body models.SettingsUpdateNotificationSettingsRequest true "Notification settings update data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /system/notifications [put]
func (h *SettingsHandler) UpdateNotificationSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req models.SettingsUpdateNotificationSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	settings, err := h.repo.UpdateNotificationSettings(r.Context(), userID, &req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update notification settings: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    settings,
		"message": "Notification settings updated successfully",
	})
}

// ==================== Audit Logs ====================

// GetAuditLogs godoc
// @Summary Get audit logs
// @Description Get system audit logs with pagination
// @Tags Settings
// @Accept json
// @Produce json
// @Param limit query int false "Number of logs to return (default 10, max 100)"
// @Param offset query int false "Number of logs to skip (default 0)"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /system/audit-logs [get]
func (h *SettingsHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	_, ok := h.getUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 10
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	logs, total, err := h.repo.GetAuditLogs(r.Context(), limit, offset)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get audit logs: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"logs":   logs,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// ==================== System Default Settings ====================

// GetSystemDefaultSettings godoc
// @Summary Get system default settings
// @Description Get system-wide default settings (timezone, currency, language, etc.)
// @Tags Settings
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /system/defaults [get]
func (h *SettingsHandler) GetSystemDefaultSettings(w http.ResponseWriter, r *http.Request) {
	_, ok := h.getUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	settings, err := h.repo.GetSystemDefaultSettings(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get system default settings: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    settings,
	})
}


// UpdateSystemDefaultSettings godoc
// @Summary Update system default settings
// @Description Update system-wide default settings (admin only)
// @Tags Settings
// @Accept json
// @Produce json
// @Param request body models.UpdateSystemDefaultSettingsRequest true "Settings update data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /system/defaults [put]
func (h *SettingsHandler) UpdateSystemDefaultSettings(w http.ResponseWriter, r *http.Request) {
	_, ok := h.getUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req models.UpdateSystemDefaultSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	settings, err := h.repo.UpdateSystemDefaultSettings(r.Context(), &req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update system default settings: "+err.Error())
		return
	}

	// Publish audit log event (fire-and-forget)
	if h.auditPublisher != nil {
		userID, _ := h.getUserID(r)
		userName, _ := r.Context().Value(middleware.NameKey).(string)
		h.auditPublisher.PublishSettingsEvent(
			r,
			userID,
			userName,
			events.ActionSettingsUpdated,
			"System default settings updated",
		)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    settings,
		"message": "System default settings updated successfully",
	})
}


// ==================== System Email & Notification Settings ====================

// GetSystemEmailNotificationSettings godoc
// @Summary Get system email notification settings
// @Description Get system-wide email and notification settings
// @Tags Settings
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /system/email-notifications [get]
func (h *SettingsHandler) GetSystemEmailNotificationSettings(w http.ResponseWriter, r *http.Request) {
	_, ok := h.getUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	settings, err := h.repo.GetSystemEmailNotificationSettings(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get system email notification settings: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    settings,
	})
}

// UpdateSystemEmailNotificationSettings godoc
// @Summary Update system email notification settings
// @Description Update system-wide email and notification settings (admin only)
// @Tags Settings
// @Accept json
// @Produce json
// @Param request body models.UpdateSystemEmailNotificationSettingsRequest true "Email notification settings update data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /system/email-notifications [put]
func (h *SettingsHandler) UpdateSystemEmailNotificationSettings(w http.ResponseWriter, r *http.Request) {
	_, ok := h.getUserID(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req models.UpdateSystemEmailNotificationSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	settings, err := h.repo.UpdateSystemEmailNotificationSettings(r.Context(), &req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update system email notification settings: "+err.Error())
		return
	}

	// Publish audit log event (fire-and-forget)
	if h.auditPublisher != nil {
		userID, _ := h.getUserID(r)
		userName, _ := r.Context().Value(middleware.NameKey).(string)
		h.auditPublisher.PublishSettingsEvent(
			r,
			userID,
			userName,
			events.ActionSettingsUpdated,
			"System email notification settings updated",
		)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    settings,
		"message": "System email notification settings updated successfully",
	})
}
