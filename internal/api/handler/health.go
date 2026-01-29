package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Saumajitt/threatLog/internal/repository"
)

type HealthHandler struct {
	pgRepo    *repository.PostgresRepository
	redisRepo *repository.RedisRepository
}

func NewHealthHandler(
	pgRepo *repository.PostgresRepository,
	redisRepo *repository.RedisRepository,
) *HealthHandler {
	return &HealthHandler{
		pgRepo:    pgRepo,
		redisRepo: redisRepo,
	}
}

// HandleHealth returns health status
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	pgStatus := "connected"
	if err := h.pgRepo.HealthCheck(ctx); err != nil {
		pgStatus = "disconnected"
	}

	redisStatus := "connected"
	if err := h.redisRepo.HealthCheck(ctx); err != nil {
		redisStatus = "disconnected"
	}

	status := "healthy"
	statusCode := http.StatusOK
	if pgStatus != "connected" || redisStatus != "connected" {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	response := map[string]string{
		"status":   status,
		"postgres": pgStatus,
		"redis":    redisStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}