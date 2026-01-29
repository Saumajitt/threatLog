package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/Saumajitt/threatLog/internal/model"
	"github.com/Saumajitt/threatLog/internal/service"
	"github.com/Saumajitt/threatLog/pkg/validator"
)

type QueryHandler struct {
	queryService   *service.QueryService
	metricsService *service.MetricsService
}

func NewQueryHandler(
	queryService *service.QueryService,
	metricsService *service.MetricsService,
) *QueryHandler {
	return &QueryHandler{
		queryService:   queryService,
		metricsService: metricsService,
	}
}

// HandleQuery handles log queries
func (h *QueryHandler) HandleQuery(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	cacheHit := false
	defer func() {
		h.metricsService.RecordQuery(time.Since(start), cacheHit)
	}()

	// Parse query parameters
	queryParams := r.URL.Query()

	// Parse timestamps
	startTime, err := validator.ParseTimestamp(queryParams.Get("start_time"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_parameter", "Invalid start_time", nil)
		return
	}

	endTime, err := validator.ParseTimestamp(queryParams.Get("end_time"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_parameter", "Invalid end_time", nil)
		return
	}

	// Parse severity (comma-separated)
	var severities []string
	if sev := queryParams.Get("severity"); sev != "" {
		severities = strings.Split(sev, ",")
		for i := range severities {
			severities[i] = strings.TrimSpace(severities[i])
		}
	}

	// Parse source
	source := queryParams.Get("source")

	// Parse limit
	limit := 100
	if l := queryParams.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	// Parse offset
	offset := 0
	if o := queryParams.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	// Build query request
	req := model.QueryRequest{
		StartTime: startTime,
		EndTime:   endTime,
		Severity:  severities,
		Source:    source,
		Limit:     limit,
		Offset:    offset,
	}

	// Validate request
	if err := validator.ValidateQueryRequest(req); err != nil {
		h.respondError(w, http.StatusBadRequest, "validation_failed", err.Error(), nil)
		return
	}

	// Execute query
	response, err := h.queryService.QueryLogs(r.Context(), req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query logs")
		h.respondError(w, http.StatusInternalServerError, "query_failed", "Failed to query logs", nil)
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

func (h *QueryHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *QueryHandler) respondError(w http.ResponseWriter, status int, error, message string, details map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(model.ErrorResponse{
		Error:   error,
		Message: message,
		Details: details,
	})
}