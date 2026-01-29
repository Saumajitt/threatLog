//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var (
	baseURL     = flag.String("url", "http://localhost:8080", "Base URL")
	concurrency = flag.Int("concurrency", 100, "Number of concurrent workers")
	duration    = flag.Int("duration", 60, "Test duration in seconds")
	rps         = flag.Int("rps", 1000, "Target requests per second")
)

type Stats struct {
	totalRequests   atomic.Int64
	successRequests atomic.Int64
	failedRequests  atomic.Int64
	totalLatency    atomic.Int64
	minLatency      atomic.Int64
	maxLatency      atomic.Int64
}

type LogRequest struct {
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
}

func main() {
	flag.Parse()

	fmt.Printf("Load Test Configuration:\n")
	fmt.Printf("  URL: %s\n", *baseURL)
	fmt.Printf("  Concurrency: %d\n", *concurrency)
	fmt.Printf("  Duration: %d seconds\n", *duration)
	fmt.Printf("  Target RPS: %d\n", *rps)
	fmt.Println()

	stats := &Stats{}
	stats.minLatency.Store(int64(time.Hour))

	startTime := time.Now()
	endTime := startTime.Add(time.Duration(*duration) * time.Second)

	var wg sync.WaitGroup
	requestInterval := time.Second / time.Duration(*rps)

	// Start workers
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go worker(i, endTime, requestInterval, stats, &wg)
	}

	// Print stats periodically
	go printStats(stats, startTime)

	wg.Wait()

	// Final stats
	elapsed := time.Since(startTime)
	printFinalStats(stats, elapsed)
}

func worker(id int, endTime time.Time, interval time.Duration, stats *Stats, wg *sync.WaitGroup) {
	defer wg.Done()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	severities := []string{"CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO"}

	for {
		select {
		case <-ticker.C:
			if time.Now().After(endTime) {
				return
			}

			req := LogRequest{
				Timestamp: time.Now(),
				Severity:  severities[stats.totalRequests.Load()%int64(len(severities))],
				Source:    fmt.Sprintf("worker-%d", id),
				Message:   fmt.Sprintf("Load test message from worker %d", id),
			}

			sendRequest(client, req, stats)
		}
	}
}

func sendRequest(client *http.Client, logReq LogRequest, stats *Stats) {
	stats.totalRequests.Add(1)

	payload, err := json.Marshal(logReq)
	if err != nil {
		stats.failedRequests.Add(1)
		return
	}

	start := time.Now()
	resp, err := client.Post(
		*baseURL+"/api/v1/logs/ingest",
		"application/json",
		bytes.NewBuffer(payload),
	)
	latency := time.Since(start)

	if err != nil {
		stats.failedRequests.Add(1)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		stats.successRequests.Add(1)
	} else {
		stats.failedRequests.Add(1)
	}

	// Update latency stats
	latencyMs := latency.Milliseconds()
	stats.totalLatency.Add(latencyMs)

	// Update min/max
	for {
		current := stats.minLatency.Load()
		if latencyMs >= current {
			break
		}
		if stats.minLatency.CompareAndSwap(current, latencyMs) {
			break
		}
	}

	for {
		current := stats.maxLatency.Load()
		if latencyMs <= current {
			break
		}
		if stats.maxLatency.CompareAndSwap(current, latencyMs) {
			break
		}
	}
}

func printStats(stats *Stats, startTime time.Time) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		elapsed := time.Since(startTime).Seconds()
		total := stats.totalRequests.Load()
		success := stats.successRequests.Load()
		failed := stats.failedRequests.Load()

		rps := float64(total) / elapsed
		avgLatency := float64(0)
		if total > 0 {
			avgLatency = float64(stats.totalLatency.Load()) / float64(total)
		}

		fmt.Printf("[%.0fs] Requests: %d | Success: %d | Failed: %d | RPS: %.2f | Avg Latency: %.2fms\n",
			elapsed, total, success, failed, rps, avgLatency)
	}
}

func printFinalStats(stats *Stats, elapsed time.Duration) {
	total := stats.totalRequests.Load()
	success := stats.successRequests.Load()
	failed := stats.failedRequests.Load()

	successRate := float64(success) / float64(total) * 100
	rps := float64(total) / elapsed.Seconds()

	avgLatency := float64(0)
	if total > 0 {
		avgLatency = float64(stats.totalLatency.Load()) / float64(total)
	}

	fmt.Println("\n" + string(bytes.Repeat([]byte("="), 60)))
	fmt.Println("LOAD TEST RESULTS")
	fmt.Println(string(bytes.Repeat([]byte("="), 60)))
	fmt.Printf("Total Duration:     %.2f seconds\n", elapsed.Seconds())
	fmt.Printf("Total Requests:     %d\n", total)
	fmt.Printf("Successful:         %d (%.2f%%)\n", success, successRate)
	fmt.Printf("Failed:             %d\n", failed)
	fmt.Printf("Requests/Second:    %.2f\n", rps)
	fmt.Printf("Avg Latency:        %.2f ms\n", avgLatency)
	fmt.Printf("Min Latency:        %d ms\n", stats.minLatency.Load())
	fmt.Printf("Max Latency:        %d ms\n", stats.maxLatency.Load())
	fmt.Println(string(bytes.Repeat([]byte("="), 60)))
}
