package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLogEvent(t *testing.T) {
	timestamp := time.Now()
	log := NewLogEvent(timestamp, SeverityHigh, "192.168.1.1", "Test message")

	assert.NotEmpty(t, log.ID)
	assert.Equal(t, timestamp, log.Timestamp)
	assert.Equal(t, SeverityHigh, log.Severity)
	assert.Equal(t, "192.168.1.1", log.Source)
	assert.Equal(t, "Test message", log.Message)
}

func TestIsValidSeverity(t *testing.T) {
	tests := []struct {
		severity string
		valid    bool
	}{
		{SeverityCritical, true},
		{SeverityHigh, true},
		{SeverityMedium, true},
		{SeverityLow, true},
		{SeverityInfo, true},
		{"INVALID", false},
		{"", false},
		{"high", false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			result := IsValidSeverity(tt.severity)
			assert.Equal(t, tt.valid, result)
		})
	}
}