package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/Saumajitt/threatLog/internal/api/handler"
	custommw "github.com/Saumajitt/threatLog/internal/api/middleware"
)

type Router struct {
	ingestHandler  *handler.IngestHandler
	queryHandler   *handler.QueryHandler
	metricsHandler *handler.MetricsHandler
	healthHandler  *handler.HealthHandler
}

func NewRouter(
	ingestHandler *handler.IngestHandler,
	queryHandler *handler.QueryHandler,
	metricsHandler *handler.MetricsHandler,
	healthHandler *handler.HealthHandler,
) *Router {
	return &Router{
		ingestHandler:  ingestHandler,
		queryHandler:   queryHandler,
		metricsHandler: metricsHandler,
		healthHandler:  healthHandler,
	}
}

// Setup creates and configures the router
func (rt *Router) Setup() *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(custommw.Logger)
	r.Use(custommw.Recovery)
	r.Use(middleware.Compress(5))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", rt.healthHandler.HandleHealth)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Ingestion endpoints
		r.Post("/logs/ingest", rt.ingestHandler.HandleIngest)
		r.Post("/logs/ingest/batch", rt.ingestHandler.HandleBatchIngest)

		// Query endpoint
		r.Get("/logs/query", rt.queryHandler.HandleQuery)

		// Metrics endpoint
		r.Get("/metrics", rt.metricsHandler.HandleMetrics)
	})

	return r
}