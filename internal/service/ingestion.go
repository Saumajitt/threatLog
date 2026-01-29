package service

import (
	"github.com/Saumajitt/threatLog/internal/model"
	"github.com/Saumajitt/threatLog/internal/worker"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// IngestionService handles log ingestion
type IngestionService struct {
	pool *worker.Pool
}

// NewIngestionService creates a new ingestion service
func NewIngestionService(pool *worker.Pool) *IngestionService {
	return &IngestionService{
		pool: pool,
	}
}

// IngestLog ingests a single log event
func (s *IngestionService) IngestLog(req model.IngestRequest) (*model.IngestResponse, error) {
	// Generate ID if not provided
	logEvent := model.LogEvent{
		ID:        uuid.New().String(),
		Timestamp: req.Timestamp,
		Severity:  req.Severity,
		Source:    req.Source,
		Message:   req.Message,
	}

	// Submit to worker pool
	if err := s.pool.Submit(logEvent); err != nil {
		log.Error().Err(err).Msg("Failed to submit log to worker pool")
		return nil, err
	}

	return &model.IngestResponse{
		ID:        logEvent.ID,
		Status:    "ingested",
		Timestamp: logEvent.Timestamp,
	}, nil
}

// IngestBatch ingests multiple log events
func (s *IngestionService) IngestBatch(req model.BatchIngestRequest) (*model.BatchIngestResponse, error) {
	response := &model.BatchIngestResponse{
		Accepted: 0,
		Rejected: 0,
		Errors:   make([]map[string]string, 0),
	}

	for i, logReq := range req.Logs {
		logEvent := model.LogEvent{
			ID:        uuid.New().String(),
			Timestamp: logReq.Timestamp,
			Severity:  logReq.Severity,
			Source:    logReq.Source,
			Message:   logReq.Message,
		}

		if err := s.pool.Submit(logEvent); err != nil {
			response.Rejected++
			response.Errors = append(response.Errors, map[string]string{
				"index":  string(rune(i)),
				"error":  err.Error(),
				"log_id": logEvent.ID,
			})
			continue
		}

		response.Accepted++
	}

	return response, nil
}
