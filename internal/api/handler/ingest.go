package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/Saumajitt/threatLog/internal/model"
	"github.com/Saumajitt/threatLog/internal/service"
	"github.com/Saumajitt/threatLog/pkg/validator"
)

type IngestHandler struct {
	ingestionService *service.IngestionService
	metricsService   *service.MetricsService
}

func NewIngestHandler(
	ingestionService *service.IngestionService,
	metricsService *service.MetricsService,
) *IngestHandler {
	return &IngestHandler{
		ingestionService: ingestionService,
		metricsService:   metricsService,
	}
}

// HandleIngest handles single log ingestion
func (h *IngestHandler) HandleIngest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		h.metricsService.RecordIngestion(time.Since(start))
	}()

	var req model.IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload", nil)
		return
	}

	// Validate request
	if err := validator.ValidateIngestRequest(req); err != nil {
		h.respondError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error(), nil)
		return
	}

	// Ingest log
	response, err := h.ingestionService.IngestLog(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to ingest log")
		h.respondError(w, http.StatusInternalServerError, "ingestion_failed", "Failed to ingest log", nil)
		return
	}

	h.respondJSON(w, http.StatusCreated, response)
}

// HandleBatchIngest handles batch log ingestion
func (h *IngestHandler) HandleBatchIngest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		h.metricsService.RecordIngestion(time.Since(start))
	}()

	var req model.BatchIngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload", nil)
		return
	}

	if len(req.Logs) == 0 {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "No logs provided", nil)
		return
	}

	// Validate all logs
	for i, logReq := range req.Logs {
		if err := validator.ValidateIngestRequest(logReq); err != nil {
			h.respondError(w, http.StatusUnprocessableEntity, "validation_failed", err.Error(), map[string]interface{}{
				"log_index": i,
			})
			return
		}
	}

	// Ingest batch
	response, err := h.ingestionService.IngestBatch(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to ingest batch")
		h.respondError(w, http.StatusInternalServerError, "ingestion_failed", "Failed to ingest batch", nil)
		return
	}

	h.respondJSON(w, http.StatusAccepted, response)
}

func (h *IngestHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *IngestHandler) respondError(w http.ResponseWriter, status int, error, message string, details map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(model.ErrorResponse{
		Error:   error,
		Message: message,
		Details: details,
	})
}