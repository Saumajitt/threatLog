package service

import (
	"sync"
	"sync/atomic"
	"time"
)

// MetricsService tracks system metrics
type MetricsService struct {
	mu                sync.RWMutex
	startTime         time.Time
	totalIngested     atomic.Int64
	totalQueries      atomic.Int64
	ingestionLatency  []time.Duration
	queryLatency      []time.Duration
	cacheHits         atomic.Int64
	cacheMisses       atomic.Int64
	maxLatencyEntries int
}

// NewMetricsService creates a new metrics service
func NewMetricsService() *MetricsService {
	return &MetricsService{
		startTime:         time.Now(),
		maxLatencyEntries: 1000,
	}
}

// RecordIngestion records an ingestion event
func (m *MetricsService) RecordIngestion(latency time.Duration) {
	m.totalIngested.Add(1)
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.ingestionLatency = append(m.ingestionLatency, latency)
	if len(m.ingestionLatency) > m.maxLatencyEntries {
		m.ingestionLatency = m.ingestionLatency[1:]
	}
}

// RecordQuery records a query event
func (m *MetricsService) RecordQuery(latency time.Duration, cacheHit bool) {
	m.totalQueries.Add(1)
	
	if cacheHit {
		m.cacheHits.Add(1)
	} else {
		m.cacheMisses.Add(1)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.queryLatency = append(m.queryLatency, latency)
	if len(m.queryLatency) > m.maxLatencyEntries {
		m.queryLatency = m.queryLatency[1:]
	}
}

// GetMetrics returns current metrics
func (m *MetricsService) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uptime := time.Since(m.startTime).Seconds()
	ingestionRate := float64(m.totalIngested.Load()) / uptime

	totalCache := m.cacheHits.Load() + m.cacheMisses.Load()
	cacheHitRatio := 0.0
	if totalCache > 0 {
		cacheHitRatio = float64(m.cacheHits.Load()) / float64(totalCache)
	}

	return map[string]interface{}{
		"ingestion_rate":          ingestionRate,
		"total_logs_ingested":     m.totalIngested.Load(),
		"total_queries":           m.totalQueries.Load(),
		"avg_ingestion_latency_ms": m.calculateAvg(m.ingestionLatency),
		"p95_ingestion_latency_ms": m.calculatePercentile(m.ingestionLatency, 0.95),
		"p99_ingestion_latency_ms": m.calculatePercentile(m.ingestionLatency, 0.99),
		"avg_query_latency_ms":     m.calculateAvg(m.queryLatency),
		"p95_query_latency_ms":     m.calculatePercentile(m.queryLatency, 0.95),
		"cache_hit_ratio":          cacheHitRatio,
		"cache_hits":               m.cacheHits.Load(),
		"cache_misses":             m.cacheMisses.Load(),
		"uptime_seconds":           uptime,
	}
}

func (m *MetricsService) calculateAvg(latencies []time.Duration) float64 {
	if len(latencies) == 0 {
		return 0
	}
	
	var sum time.Duration
	for _, l := range latencies {
		sum += l
	}
	
	return float64(sum.Milliseconds()) / float64(len(latencies))
}

func (m *MetricsService) calculatePercentile(latencies []time.Duration, p float64) float64 {
	if len(latencies) == 0 {
		return 0
	}
	
	// Simple percentile calculation (for production, use a proper implementation)
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	
	// Basic bubble sort (OK for small datasets)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	index := int(float64(len(sorted)) * p)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	
	return float64(sorted[index].Milliseconds())
}