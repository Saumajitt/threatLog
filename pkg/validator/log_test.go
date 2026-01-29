package validator

import (
	"strings"
	"testing"
	"time"

	"github.com/Saumajitt/threatLog/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestValidateIngestRequest(t *testing.T) {
	tests := []struct {
		name    string
		request model.IngestRequest
		wantErr error
	}{
		{
			name: "valid request",
			request: model.IngestRequest{
				Timestamp: time.Now(),
				Severity:  model.SeverityHigh,
				Source:    "192.168.1.1",
				Message:   "Test message",
			},
			wantErr: nil,
		},
		{
			name: "invalid timestamp",
			request: model.IngestRequest{
				Timestamp: time.Time{},
				Severity:  model.SeverityHigh,
				Source:    "192.168.1.1",
				Message:   "Test message",
			},
			wantErr: ErrInvalidTimestamp,
		},
		{
			name: "invalid severity",
			request: model.IngestRequest{
				Timestamp: time.Now(),
				Severity:  "INVALID",
				Source:    "192.168.1.1",
				Message:   "Test message",
			},
			wantErr: ErrInvalidSeverity,
		},
		{
			name: "empty source",
			request: model.IngestRequest{
				Timestamp: time.Now(),
				Severity:  model.SeverityHigh,
				Source:    "",
				Message:   "Test message",
			},
			wantErr: ErrEmptySource,
		},
		{
			name: "source too long",
			request: model.IngestRequest{
				Timestamp: time.Now(),
				Severity:  model.SeverityHigh,
				Source:    strings.Repeat("a", 256),
				Message:   "Test message",
			},
			wantErr: ErrSourceTooLong,
		},
		{
			name: "empty message",
			request: model.IngestRequest{
				Timestamp: time.Now(),
				Severity:  model.SeverityHigh,
				Source:    "192.168.1.1",
				Message:   "",
			},
			wantErr: ErrEmptyMessage,
		},
		{
			name: "message too long",
			request: model.IngestRequest{
				Timestamp: time.Now(),
				Severity:  model.SeverityHigh,
				Source:    "192.168.1.1",
				Message:   strings.Repeat("a", 4097),
			},
			wantErr: ErrMessageTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIngestRequest(tt.request)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid RFC3339",
			input:   "2026-01-29T12:00:00Z",
			wantErr: false,
		},
		{
			name:    "valid with timezone",
			input:   "2026-01-29T12:00:00+05:30",
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "2026-01-29",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTimestamp(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
