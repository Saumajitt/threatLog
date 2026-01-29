package worker

import (
	"context"
	"sync"
	"time"

	
	"github.com/Saumajitt/threatLog/internal/model"
	"github.com/Saumajitt/threatLog/internal/repository"
	"github.com/rs/zerolog/log"
)

// Pool represents a worker pool for processing log events
type Pool struct {
	workers      int
	logChannel   chan model.LogEvent
	batchSize    int
	batchTimeout time.Duration
	repo         *repository.PostgresRepository
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewPool creates a new worker pool
func NewPool(
	workers int,
	bufferSize int,
	batchSize int,
	batchTimeout time.Duration,
	repo *repository.PostgresRepository,
) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Pool{
		workers:      workers,
		logChannel:   make(chan model.LogEvent, bufferSize),
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		repo:         repo,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start starts all workers
func (p *Pool) Start() {
	log.Info().
		Int("workers", p.workers).
		Int("batch_size", p.batchSize).
		Dur("batch_timeout", p.batchTimeout).
		Msg("Starting worker pool")

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Submit submits a log event to the worker pool
func (p *Pool) Submit(log model.LogEvent) error {
	select {
	case p.logChannel <- log:
		return nil
	case <-p.ctx.Done():
		return p.ctx.Err()
	default:
		return ErrChannelFull
	}
}

// Stop gracefully stops all workers
func (p *Pool) Stop() {
	log.Info().Msg("Stopping worker pool")
	p.cancel()
	close(p.logChannel)
	p.wg.Wait()
	log.Info().Msg("Worker pool stopped")
}

// worker processes log events in batches
func (p *Pool) worker(id int) {
	defer p.wg.Done()

	batch := make([]model.LogEvent, 0, p.batchSize)
	ticker := time.NewTicker(p.batchTimeout)
	defer ticker.Stop()

	log.Info().Int("worker_id", id).Msg("Worker started")

	for {
		select {
		case logEvent, ok := <-p.logChannel:
			if !ok {
				// Channel closed, flush remaining batch
				if len(batch) > 0 {
					p.flushBatch(id, batch)
				}
				log.Info().Int("worker_id", id).Msg("Worker stopped")
				return
			}

			batch = append(batch, logEvent)

			if len(batch) >= p.batchSize {
				p.flushBatch(id, batch)
				batch = batch[:0] // Reset batch
				ticker.Reset(p.batchTimeout)
			}

		case <-ticker.C:
			if len(batch) > 0 {
				p.flushBatch(id, batch)
				batch = batch[:0] // Reset batch
			}

		case <-p.ctx.Done():
			// Graceful shutdown, flush remaining batch
			if len(batch) > 0 {
				p.flushBatch(id, batch)
			}
			log.Info().Int("worker_id", id).Msg("Worker stopped")
			return
		}
	}
}

// flushBatch writes a batch of logs to the database
func (p *Pool) flushBatch(workerID int, batch []model.LogEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	err := p.repo.BatchInsertLogs(ctx, batch)
	duration := time.Since(start)

	if err != nil {
		log.Error().
			Err(err).
			Int("worker_id", workerID).
			Int("batch_size", len(batch)).
			Dur("duration", duration).
			Msg("Failed to insert batch")
		return
	}

	log.Debug().
		Int("worker_id", workerID).
		Int("batch_size", len(batch)).
		Dur("duration", duration).
		Msg("Batch inserted successfully")
}

// Errors
var (
	ErrChannelFull = &PoolError{"worker pool channel is full"}
)

type PoolError struct {
	message string
}

func (e *PoolError) Error() string {
	return e.message
}