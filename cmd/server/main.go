package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Saumajitt/threatLog/internal/api"
	"github.com/Saumajitt/threatLog/internal/api/handler"
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

	log.Info().Msg("Configuration loaded")

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

	// Initialize handlers
	ingestHandler := handler.NewIngestHandler(ingestionService, metricsService)
	queryHandler := handler.NewQueryHandler(queryService, metricsService)
	metricsHandler := handler.NewMetricsHandler(metricsService)
	healthHandler := handler.NewHealthHandler(pgRepo, redisRepo)

	// Setup router
	router := api.NewRouter(ingestHandler, queryHandler, metricsHandler, healthHandler)
	r := router.Setup()

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Info().Int("port", cfg.Server.Port).Msg("HTTP server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	log.Info().Msg("ThreatLog is running. Press Ctrl+C to stop.")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Info().Msg("Shutting down gracefully...")

	// Shutdown HTTP server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown error")
	}

	log.Info().Msg("Shutdown complete")
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
