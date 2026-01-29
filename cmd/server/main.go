package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Saumajitt/threatLog/internal/config"
	"github.com/Saumajitt/threatLog/internal/repository"
	"github.com/Saumajitt/threatLog/internal/service"
	"github.com/Saumajitt/threatLog/internal/worker"
)

func main() {
	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Info().Msg("Starting ThreatLog service...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	log.Info().Interface("config", cfg).Msg("Configuration loaded")

	// Initialize PostgreSQL
	pgPool, err := initPostgres(cfg.Postgres)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize PostgreSQL")
	}
	defer pgPool.Close()

	// Initialize Redis
	redisClient := initRedis(cfg.Redis)
	defer redisClient.Close()

	// Initialize repositories
	pgRepo := repository.NewPostgresRepository(pgPool)
	redisRepo := repository.NewRedisRepository(redisClient, cfg.Cache.TTL)

	// Health check
	ctx := context.Background()
	if err := pgRepo.HealthCheck(ctx); err != nil {
		log.Fatal().Err(err).Msg("PostgreSQL health check failed")
	}
	log.Info().Msg("PostgreSQL connected successfully")

	if err := redisRepo.HealthCheck(ctx); err != nil {
		log.Fatal().Err(err).Msg("Redis health check failed")
	}
	log.Info().Msg("Redis connected successfully")

	// Initialize worker pool
	pool := worker.NewPool(
		cfg.Ingestion.WorkerCount,
		cfg.Ingestion.BufferSize,
		cfg.Ingestion.BatchSize,
		cfg.Ingestion.BatchTimeout,
		pgRepo,
	)
	pool.Start()
	defer pool.Stop()

	// Initialize services
	ingestionService := service.NewIngestionService(pool)
	queryService := service.NewQueryService(pgRepo, redisRepo, cfg.Cache.QueryCacheEnabled)
	metricsService := service.NewMetricsService()

	log.Info().
		Interface("ingestion_service", ingestionService).
		Interface("query_service", queryService).
		Interface("metrics_service", metricsService).
		Msg("Services initialized")

	// TODO: Initialize HTTP server (Phase 6)
	log.Info().Msg("HTTP server initialization coming in Phase 6...")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Info().Msg("ThreatLog is running. Press Ctrl+C to stop.")
	<-sigChan

	log.Info().Msg("Shutting down gracefully...")
}

func initPostgres(cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	ctx := context.Background()

	connStr := cfg.GetDSN()
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

func initRedis(cfg config.RedisConfig) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	return client
}
