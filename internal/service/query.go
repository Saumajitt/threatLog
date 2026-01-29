package service

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/Saumajitt/threatLog/internal/model"
	"github.com/Saumajitt/threatLog/internal/repository"
)

// QueryService handles log queries
type QueryService struct {
	pgRepo    *repository.PostgresRepository
	redisRepo *repository.RedisRepository
	cacheEnabled bool
}

// NewQueryService creates a new query service
func NewQueryService(
	pgRepo *repository.PostgresRepository,
	redisRepo *repository.RedisRepository,
	cacheEnabled bool,
) *QueryService {
	return &QueryService{
		pgRepo:    pgRepo,
		redisRepo: redisRepo,
		cacheEnabled: cacheEnabled,
	}
}

// QueryLogs queries logs with optional caching
func (s *QueryService) QueryLogs(ctx context.Context, req model.QueryRequest) (*model.QueryResponse, error) {
	// Try cache first if enabled
	if s.cacheEnabled {
		cached, err := s.redisRepo.GetCachedQueryResult(ctx, req)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to get cached result")
		} else if cached != nil {
			log.Debug().Msg("Cache hit")
			return cached, nil
		}
	}

	// Cache miss, query database
	log.Debug().Msg("Cache miss, querying database")
	logs, total, err := s.pgRepo.QueryLogs(ctx, req)
	if err != nil {
		return nil, err
	}

	response := &model.QueryResponse{
		Total: total,
		Count: len(logs),
		Logs:  logs,
	}

	// Cache the result if enabled
	if s.cacheEnabled {
		if err := s.redisRepo.CacheQueryResult(ctx, req, *response); err != nil {
			log.Warn().Err(err).Msg("Failed to cache query result")
		}
	}

	return response, nil
}