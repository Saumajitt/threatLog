package repository

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Saumajitt/threatLog/internal/model"
	"github.com/redis/go-redis/v9"
)

type RedisRepository struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisRepository(client *redis.Client, ttl time.Duration) *RedisRepository {
	return &RedisRepository{
		client: client,
		ttl:    ttl,
	}
}

// CacheQueryResult caches query results
func (r *RedisRepository) CacheQueryResult(ctx context.Context, req model.QueryRequest, result model.QueryResponse) error {
	key := r.generateCacheKey(req)

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	return r.client.Set(ctx, key, data, r.ttl).Err()
}

// GetCachedQueryResult retrieves cached query results
func (r *RedisRepository) GetCachedQueryResult(ctx context.Context, req model.QueryRequest) (*model.QueryResponse, error) {
	key := r.generateCacheKey(req)

	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cached result: %w", err)
	}

	var result model.QueryResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// InvalidateCache invalidates all cache entries
func (r *RedisRepository) InvalidateCache(ctx context.Context) error {
	iter := r.client.Scan(ctx, 0, "logs:query:*", 0).Iterator()
	for iter.Next(ctx) {
		if err := r.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

// HealthCheck checks if Redis is reachable
func (r *RedisRepository) HealthCheck(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// generateCacheKey generates a unique cache key for query parameters
func (r *RedisRepository) generateCacheKey(req model.QueryRequest) string {
	data := fmt.Sprintf("%v-%v-%v-%v-%d-%d",
		req.StartTime.Unix(),
		req.EndTime.Unix(),
		req.Severity,
		req.Source,
		req.Limit,
		req.Offset,
	)

	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("logs:query:%x", hash)
}
