package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Saumajitt/threatLog/internal/service"
)

type MetricsHandler struct {
	metricsService *service.MetricsService
}

func NewMetricsHandler(metricsService *service.MetricsService) *MetricsHandler {
	return &MetricsHandler{
		metricsService: metricsService,
	}
}

// HandleMetrics returns system metrics
func (h *MetricsHandler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := h.metricsService.GetMetrics()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}