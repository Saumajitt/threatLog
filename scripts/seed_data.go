//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

var (
	url   = flag.String("url", "http://localhost:8080", "Base URL")
	count = flag.Int("count", 1000, "Number of logs to generate")
)

type LogRequest struct {
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
}

var (
	severities = []string{"CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO"}
	sources    = []string{
		"192.168.1.100", "192.168.1.101", "192.168.1.102",
		"10.0.0.50", "10.0.0.51", "firewall-01", "web-server-01",
		"db-server-01", "api-gateway", "load-balancer",
	}
	messages = []string{
		"Failed login attempt detected",
		"Successful authentication",
		"Database connection established",
		"API rate limit exceeded",
		"Suspicious network activity detected",
		"System health check passed",
		"Disk usage above threshold",
		"Memory usage critical",
		"Service started successfully",
		"Configuration updated",
		"Backup completed",
		"SSL certificate expiring soon",
		"Unauthorized access attempt",
		"Request timeout",
		"Cache invalidated",
	}
)

func main() {
	flag.Parse()

	fmt.Printf("Generating %d sample logs...\n", *count)

	client := &http.Client{Timeout: 10 * time.Second}
	successCount := 0
	failCount := 0

	startTime := time.Now().Add(-24 * time.Hour) // Start from 24 hours ago

	for i := 0; i < *count; i++ {
		log := generateRandomLog(startTime, i, *count)

		if sendLog(client, log) {
			successCount++
		} else {
			failCount++
		}

		if (i+1)%100 == 0 {
			fmt.Printf("Progress: %d/%d logs sent (Success: %d, Failed: %d)\n",
				i+1, *count, successCount, failCount)
		}
	}

	fmt.Printf("\nCompleted!\n")
	fmt.Printf("Total: %d | Success: %d | Failed: %d\n", *count, successCount, failCount)
}

func generateRandomLog(startTime time.Time, index, total int) LogRequest {
	// Distribute logs evenly across the time range
	timeDelta := time.Duration(index) * (24 * time.Hour) / time.Duration(total)
	timestamp := startTime.Add(timeDelta)

	return LogRequest{
		Timestamp: timestamp,
		Severity:  severities[rand.Intn(len(severities))],
		Source:    sources[rand.Intn(len(sources))],
		Message:   messages[rand.Intn(len(messages))],
	}
}

func sendLog(client *http.Client, log LogRequest) bool {
	payload, err := json.Marshal(log)
	if err != nil {
		return false
	}

	resp, err := client.Post(
		*url+"/api/v1/logs/ingest",
		"application/json",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusCreated
}
