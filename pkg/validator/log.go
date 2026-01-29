package validator

import (
	"errors"
	"time"

	"github.com/Saumajitt/threatLog/internal/model"
)

var (
	ErrInvalidTimestamp = errors.New("invalid timestamp")
	ErrInvalidSeverity  = errors.New("invalid severity level")
	ErrEmptySource      = errors.New("source cannot be empty")
	ErrEmptyMessage     = errors.New("message cannot be empty")
	ErrSourceTooLong    = errors.New("source exceeds 255 characters")
	ErrMessageTooLong   = errors.New("message exceeds 4096 characters")
)

// ValidateIngestRequest validates a single log ingest request
func ValidateIngestRequest(req model.IngestRequest) error {
	// Validate timestamp
	if req.Timestamp.IsZero() {
		return ErrInvalidTimestamp
	}

	// Validate severity
	if !model.IsValidSeverity(req.Severity) {
		return ErrInvalidSeverity
	}

	// Validate source
	if req.Source == "" {
		return ErrEmptySource
	}
	if len(req.Source) > 255 {
		return ErrSourceTooLong
	}

	// Validate message
	if req.Message == "" {
		return ErrEmptyMessage
	}
	if len(req.Message) > 4096 {
		return ErrMessageTooLong
	}

	return nil
}

// ValidateQueryRequest validates a query request
func ValidateQueryRequest(req model.QueryRequest) error {
	// Validate time range
	if req.StartTime.IsZero() || req.EndTime.IsZero() {
		return errors.New("start_time and end_time are required")
	}

	if req.EndTime.Before(req.StartTime) {
		return errors.New("end_time must be after start_time")
	}

	// Validate limit
	if req.Limit <= 0 {
		req.Limit = 100
	}
	if req.Limit > 1000 {
		return errors.New("limit cannot exceed 1000")
	}

	// Validate offset
	if req.Offset < 0 {
		return errors.New("offset cannot be negative")
	}

	// Validate severity values
	for _, sev := range req.Severity {
		if !model.IsValidSeverity(sev) {
			return ErrInvalidSeverity
		}
	}

	return nil
}

// ParseTimestamp parses RFC3339 timestamp string
func ParseTimestamp(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, ErrInvalidTimestamp
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, ErrInvalidTimestamp
	}

	return t, nil
}