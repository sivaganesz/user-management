package handlers

import (
	"encoding/json"
	"net/http"
)

type HealthResponse struct {
	Status  string                 `json:"status"`
	Service string                 `json:"service"`
	Version string                 `json:"version"`
	Checks  map[string]HealthCheck `json:"checks,omitempty"`
}
type HealthCheck struct {
	Status  string      `json:"status"`
	Latency string      `json:"latency,omitempty"`
	Details interface{} `json:"details,omitempty"`
	Error   string      `json:"error,omitempty"`
}
func GetOverallHealth(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Service: "csa-backend-api",
		Version: "1.0.0",
		Checks:  make(map[string]HealthCheck),
	}

	allHealthy := true

	if allHealthy {
		response.Status = "healthy"
		w.WriteHeader(http.StatusOK)
	}else{
		response.Status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

