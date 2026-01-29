package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/Saumajitt/threatLog/internal/model"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// InsertLog inserts a single log event
func (r *PostgresRepository) InsertLog(ctx context.Context, log *model.LogEvent) error {
	query := `
		INSERT INTO logs (id, timestamp, severity, source, message, ingested_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	
	_, err := r.pool.Exec(ctx, query,
		log.ID,
		log.Timestamp,
		log.Severity,
		log.Source,
		log.Message,
		time.Now(),
	)
	
	return err
}

// BatchInsertLogs inserts multiple logs in a single transaction
func (r *PostgresRepository) BatchInsertLogs(ctx context.Context, logs []model.LogEvent) error {
	if len(logs) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO logs (id, timestamp, severity, source, message, ingested_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, log := range logs {
		_, err := tx.Exec(ctx, query,
			log.ID,
			log.Timestamp,
			log.Severity,
			log.Source,
			log.Message,
			time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to insert log: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// QueryLogs queries logs with filters
func (r *PostgresRepository) QueryLogs(ctx context.Context, req model.QueryRequest) ([]model.LogEvent, int, error) {
	// Build WHERE clause
	var conditions []string
	var args []interface{}
	argPos := 1

	conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argPos))
	args = append(args, req.StartTime)
	argPos++

	conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argPos))
	args = append(args, req.EndTime)
	argPos++

	if len(req.Severity) > 0 {
		placeholders := make([]string, len(req.Severity))
		for i, sev := range req.Severity {
			placeholders[i] = fmt.Sprintf("$%d", argPos)
			args = append(args, sev)
			argPos++
		}
		conditions = append(conditions, fmt.Sprintf("severity IN (%s)", strings.Join(placeholders, ",")))
	}

	if req.Source != "" {
		conditions = append(conditions, fmt.Sprintf("source = $%d", argPos))
		args = append(args, req.Source)
		argPos++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM logs WHERE %s", whereClause)
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count logs: %w", err)
	}

	// Query logs
	query := fmt.Sprintf(`
		SELECT id, timestamp, severity, source, message, ingested_at
		FROM logs
		WHERE %s
		ORDER BY timestamp DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	args = append(args, req.Limit, req.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []model.LogEvent
	for rows.Next() {
		var log model.LogEvent
		err := rows.Scan(
			&log.ID,
			&log.Timestamp,
			&log.Severity,
			&log.Source,
			&log.Message,
			&log.IngestedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, total, nil
}

// GetLogByID retrieves a log by ID
func (r *PostgresRepository) GetLogByID(ctx context.Context, id string) (*model.LogEvent, error) {
	query := `
		SELECT id, timestamp, severity, source, message, ingested_at
		FROM logs
		WHERE id = $1
	`

	var log model.LogEvent
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&log.ID,
		&log.Timestamp,
		&log.Severity,
		&log.Source,
		&log.Message,
		&log.IngestedAt,
	)
	if err != nil {
		return nil, err
	}

	return &log, nil
}

// HealthCheck checks if database is reachable
func (r *PostgresRepository) HealthCheck(ctx context.Context) error {
	return r.pool.Ping(ctx)
}