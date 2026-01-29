package model

import (
	"time"

	"github.com/google/uuid"
)

// Severity levels for log events
const (
	SeverityCritical = "CRITICAL"
	SeverityHigh     = "HIGH"
	SeverityMedium   = "MEDIUM"
	SeverityLow      = "LOW"
	SeverityInfo     = "INFO"
)

// LogEvent represents a single log entry
type LogEvent struct {
	ID         string    `json:"id" db:"id"`
	Timestamp  time.Time `json:"timestamp" db:"timestamp"`
	Severity   string    `json:"severity" db:"severity"`
	Source     string    `json:"source" db:"source"`
	Message    string    `json:"message" db:"message"`
	IngestedAt time.Time `json:"ingested_at,omitempty" db:"ingested_at"`
}

// NewLogEvent creates a new log event with generated ID
func NewLogEvent(timestamp time.Time, severity, source, message string) *LogEvent {
	return &LogEvent{
		ID:        uuid.New().String(),
		Timestamp: timestamp,
		Severity:  severity,
		Source:    source,
		Message:   message,
	}
}

// IsValidSeverity checks if severity is valid
func IsValidSeverity(severity string) bool {
	switch severity {
	case SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo:
		return true
	default:
		return false
	}
}

// IngestRequest represents the API request for log ingestion
type IngestRequest struct {
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
}

// BatchIngestRequest represents batch ingestion request
type BatchIngestRequest struct {
	Logs []IngestRequest `json:"logs"`
}

// IngestResponse represents the API response for log ingestion
type IngestResponse struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// BatchIngestResponse represents batch ingestion response
type BatchIngestResponse struct {
	Accepted int                    `json:"accepted"`
	Rejected int                    `json:"rejected"`
	Errors   []map[string]string    `json:"errors,omitempty"`
}

// QueryRequest represents query parameters
type QueryRequest struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Severity  []string  `json:"severity,omitempty"`
	Source    string    `json:"source,omitempty"`
	Limit     int       `json:"limit"`
	Offset    int       `json:"offset"`
}

// QueryResponse represents query results
type QueryResponse struct {
	Total int        `json:"total"`
	Count int        `json:"count"`
	Logs  []LogEvent `json:"logs"`
}

// ErrorResponse represents error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}