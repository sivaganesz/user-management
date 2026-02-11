package handlers
import (
	"github.com/white/user-management/pkg/uuid"
	"github.com/white/user-management/internal/middleware"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"fmt"

	"github.com/white/user-management/internal/models"
	"github.com/white/user-management/internal/repositories"
	"github.com/gorilla/mux"
)

// SchedulerHandler handles scheduled message and schedule definition endpoints
type SchedulerHandler struct {
	scheduleDefinitionRepo *repositories.ScheduleDefinitionRepository
	activityRepo    *repositories.ActivityRepository
}

func NewSchedulerHandler(
	scheduleDefinitionRepo *repositories.ScheduleDefinitionRepository,
	activityRepo    *repositories.ActivityRepository,
) *SchedulerHandler {
	return &SchedulerHandler{
		scheduleDefinitionRepo: scheduleDefinitionRepo,
		activityRepo:    activityRepo,
	}
}

// GetScheduleDefinitions godoc
// @Summary Get campaign schedule definitions
// @Description Retrieve all campaign schedule definitions for dropdown selection
// @Tags Communications - Scheduler
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.ScheduleDefinition
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/campaigns/schedule-definitions [get]
func (h *SchedulerHandler) GetScheduleDefinitions(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (stored as string UUID by middleware)
	userID := middleware.GetUserID(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	// Get all active schedule definitions
	schedules, err := h.scheduleDefinitionRepo.GetActiveScheduleDefinitions()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch schedule definitions")
		return
	}

	respondWithJSON(w, http.StatusOK, schedules)
}


// CreateScheduleDefinitionRequest represents the request body for creating a schedule definition
type CreateScheduleDefinitionRequest struct {
	Name               string                 `json:"name"`
	Description        string                 `json:"description,omitempty"`
	Frequency          string                 `json:"frequency"`
	DayOfWeek          int                    `json:"dayOfWeek,omitempty"`
	DayOfMonth         int                    `json:"dayOfMonth,omitempty"`
	Time               string                 `json:"time"`
	TimeZone           string                 `json:"timeZone"`
	UseContactTimeZone bool                   `json:"useContactTimeZone"`
	ExcludedHolidays   []string               `json:"excludedHolidays"`
	SendingWindows     []models.SendingWindow `json:"sendingWindows"`
	IsActive           bool                   `json:"isActive"`
}

// CreateScheduleDefinition godoc
// @Summary Create a campaign schedule definition
// @Description Create a new campaign schedule definition
// @Tags Communications - Scheduler
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param schedule body CreateScheduleDefinitionRequest true "Schedule definition data"
// @Success 201 {object} models.ScheduleDefinition
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/campaigns/schedule-definitions [post]
func (h *SchedulerHandler) CreateScheduleDefinition(w http.ResponseWriter, r *http.Request) {
	//Extract user ID from context
	userID := middleware.GetUserID(r);
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "user ID not found in context");
		return
	}

	var req CreateScheduleDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body");
		return
	}

		// Validate required fields
	if req.Name == "" {
		respondWithError(w, http.StatusBadRequest, "Name is required")
		return
	}
	if req.TimeZone == "" {
		req.TimeZone = "UTC"
	}

	// Create schedule definition
	schedule := &models.ScheduleDefinition{
		ID:                 uuid.MustNewUUID(),
		Name:               req.Name,
		Description:        req.Description,
		Frequency:          req.Frequency,
		DayOfWeek:          req.DayOfWeek,
		DayOfMonth:         req.DayOfMonth,
		Time:               req.Time,
		TimeZone:           req.TimeZone,
		UseContactTimeZone: req.UseContactTimeZone,
		ExcludedHolidays:   req.ExcludedHolidays,
		SendingWindows:     req.SendingWindows,
		IsActive:           req.IsActive,
		CreatedBy:          userID,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Save to database 
	if err := h.scheduleDefinitionRepo.CreateScheduleDefinition(schedule); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create schedule definition")
		return
	}

	// Log activity
	if h.activityRepo != nil {
		activity := &models.Activity{
			ID:           uuid.MustNewUUID(),
			ActivityType: "schedule_definition_created",
			Description:  fmt.Sprintf("Created schedule definition: %s", schedule.Name),
			Owner:        userID,
			Status:       "completed",
			Priority:     "normal",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		h.activityRepo.CreateActivityCompat(activity)
	}

	respondWithJSON(w, http.StatusCreated, schedule)
}

// UpdateScheduleDefinition godoc
// @Summary Update a campaign schedule definition


// @Description Update an existing campaign schedule definition
// @Tags Communications - Scheduler
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Schedule definition ID"
// @Param schedule body CreateScheduleDefinitionRequest true "Updated schedule definition data"
// @Success 200 {object} models.ScheduleDefinition
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/campaigns/schedule-definitions/{id} [put]
func (h *SchedulerHandler) UpdateScheduleDefinition(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID := middleware.GetUserID(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	// Get schedule ID from URL
	vars := mux.Vars(r)
	scheduleID := vars["id"]

	// Get existing schedule
	existing, err := h.scheduleDefinitionRepo.GetScheduleDefinitionByID(scheduleID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondWithError(w, http.StatusNotFound, "Schedule definition not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch schedule definition")
		return
	}

	// Parse request body
	var req CreateScheduleDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update fields
	existing.Name = req.Name
	existing.Description = req.Description
	existing.Frequency = req.Frequency
	existing.DayOfWeek = req.DayOfWeek
	existing.DayOfMonth = req.DayOfMonth
	existing.Time = req.Time
	existing.TimeZone = req.TimeZone
	existing.UseContactTimeZone = req.UseContactTimeZone
	existing.ExcludedHolidays = req.ExcludedHolidays
	existing.SendingWindows = req.SendingWindows
	existing.IsActive = req.IsActive
	existing.UpdatedAt = time.Now()

	// Save to database
	if err := h.scheduleDefinitionRepo.UpdateScheduleDefinition(existing); err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondWithError(w, http.StatusNotFound, "Schedule definition not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to update schedule definition")
		return
	}

	// Log activity
	if h.activityRepo != nil {
		activity := &models.Activity{
			ID:           uuid.MustNewUUID(),
			ActivityType: "schedule_definition_updated",
			Description:  fmt.Sprintf("Updated schedule definition: %s", existing.Name),
			Owner:        userID,
			Status:       "completed",
			Priority:     "normal",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		h.activityRepo.CreateActivityCompat(activity)
	}

	respondWithJSON(w, http.StatusOK, existing)
}

// DeleteScheduleDefinition godoc
// @Summary Delete a campaign schedule definition


// @Description Delete a campaign schedule definition
// @Tags Communications - Scheduler
// @Produce json
// @Security BearerAuth
// @Param id path string true "Schedule definition ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/campaigns/schedule-definitions/{id} [delete]
func (h *SchedulerHandler) DeleteScheduleDefinition(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context
	userID := middleware.GetUserID(r)
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	// Get schedule ID from URL
	vars := mux.Vars(r)
	scheduleID := vars["id"]

	// Delete from database
	if err := h.scheduleDefinitionRepo.DeleteScheduleDefinition(scheduleID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondWithError(w, http.StatusNotFound, "Schedule definition not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Failed to delete schedule definition")
		return
	}

	// Log activity
	if h.activityRepo != nil {
		activity := &models.Activity{
			ID:           uuid.MustNewUUID(),
			ActivityType: "schedule_definition_deleted",
			Description:  fmt.Sprintf("Deleted schedule definition: %s", scheduleID),
			Owner:        userID,
			Status:       "completed",
			Priority:     "normal",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		h.activityRepo.CreateActivityCompat(activity)
	}

	w.WriteHeader(http.StatusNoContent)
}