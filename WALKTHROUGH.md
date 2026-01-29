# ThreatLog Project Walkthrough

A comprehensive guide to understanding and explaining your distributed log ingestion system.

---

## 🎯 What is ThreatLog?

**ThreatLog** is a **distributed log ingestion and query service** built in Go for security event monitoring. Think of it as a simplified version of systems like **Splunk**, **Elasticsearch**, or **AWS CloudWatch Logs**.

### Real-World Use Case

Security teams at companies receive thousands of log events per second from:
- Firewalls blocking suspicious traffic
- Intrusion detection systems
- Web servers logging failed login attempts
- Network appliances reporting anomalies

ThreatLog **ingests** these events at high speed, **stores** them efficiently, and allows **fast queries** to investigate security incidents.

---

## 📂 Project Structure Explained

```
threatLog/
├── cmd/                    # Application entry points
│   └── server/
│       └── main.go         # THE starting point
├── internal/               # Private application code
│   ├── api/               # HTTP layer
│   │   ├── router.go      # URL routing
│   │   ├── handler/       # Request handlers
│   │   └── middleware/    # Logging, recovery
│   ├── config/            # Configuration loading
│   ├── model/             # Data structures
│   ├── repository/        # Database access
│   ├── service/           # Business logic
│   └── worker/            # Concurrent processing
├── pkg/                   # Reusable packages
│   └── validator/         # Input validation
├── scripts/               # Utility scripts
├── docker/                # Docker setup
├── migrations/            # Database schema
├── test/                  # Integration tests
├── config.yaml            # Configuration file
└── go.mod                 # Go module definition
```

---

## 🏗️ Architecture Deep Dive

```
┌──────────────┐
│  HTTP Client │
└──────┬───────┘
       │
       ▼
┌─────────────────────────────────────┐
│     HTTP API Server (Chi Router)    │
│  ┌────────────────────────────────┐ │
│  │  Ingestion Handler             │ │
│  │  Query Handler                 │ │
│  │  Metrics Handler               │ │
│  └────────────┬───────────────────┘ │
└───────────────┼─────────────────────┘
                │
    ┌───────────┴───────────┐
    │                       │
    ▼                       ▼
┌─────────────┐       ┌────────────┐
│ Worker Pool │       │   Query    |
│             |       |  Service   |   
│ (10 workers)│       │  + Cache   │
└──────┬──────┘       └─────┬──────┘
       │                    │
       │                    │
       ▼                    ▼
┌─────────────┐       ┌────────────┐
│ PostgreSQL  │◄──────│   Redis    │
│ (Cold Store)│       │ (Hot Cache)│
└─────────────┘       └────────────┘
```

### Key Architectural Decisions

| Decision | Why |
|----------|-----|
| **Worker Pool Pattern** | Decouples HTTP response from DB write - client doesn't wait |
| **Batch Processing** | Writes 100 logs at once instead of 1 - massive performance gain |
| **Hot/Cold Storage** | Redis for recent queries, PostgreSQL for all data |
| **Graceful Shutdown** | No data loss when stopping the server |

---

## 📁 File-by-File Breakdown

### 1. Entry Point: `cmd/server/main.go`

**Purpose**: The application starting point that wires everything together.

**Key Concepts**:
- **Dependency Injection**: Creates all components and passes them to each other
- **Graceful Shutdown**: Catches `Ctrl+C` and cleanly stops all workers
- **Connection Pooling**: Reuses database connections for efficiency

```go
// Dependency injection chain:
pgRepo → worker.Pool → IngestionService → IngestHandler → Router
```

**Interview talking point**:
> "I used manual dependency injection to wire up the application. Each component receives its dependencies through constructors, making the code testable and loosely coupled."

---

### 2. Data Models: `internal/model/log.go`

**Purpose**: Defines all data structures used throughout the application.

**Key Structures**:

| Struct | Purpose |
|--------|---------|
| `LogEvent` | Core log entry with ID, timestamp, severity, source, message |
| `IngestRequest` | What the API receives from clients |
| `IngestResponse` | What the API returns after ingestion |
| `QueryRequest` | Query parameters (time range, filters) |
| `QueryResponse` | Query results with pagination |

**Interview talking point**:
> "I used struct tags for JSON serialization and database mapping. The `json:"omitempty"` tag prevents sending empty fields in API responses."

---

### 3. Configuration: `internal/config/config.go`

**Purpose**: Loads settings from `config.yaml` and environment variables.

**Uses [Viper](https://github.com/spf13/viper)** - the standard Go library for configuration.

**Key Features**:
- Hierarchical config with defaults
- Environment variable override (12-factor app compatible)
- Type-safe config structs with `mapstructure` tags

**Interview talking point**:
> "The config system follows the 12-factor app methodology - it can be configured via files for development and environment variables for production/containers."

---

### 4. HTTP Router: `internal/api/router.go`

**Purpose**: Maps URLs to handlers using [Chi router](https://github.com/go-chi/chi).

**Routes**:
```
GET  /health              → Health check
POST /api/v1/logs/ingest  → Single log ingestion
POST /api/v1/logs/ingest/batch → Batch ingestion
GET  /api/v1/logs/query   → Query logs
GET  /api/v1/metrics      → Performance metrics
```

**Middleware Stack**:
1. `RequestID` - Adds unique ID to each request for tracing
2. `RealIP` - Gets client IP behind proxies
3. `Logger` - Logs all requests
4. `Recovery` - Catches panics, returns 500 instead of crashing
5. `Compress` - Gzip responses for efficiency
6. `CORS` - Allows cross-origin requests

**Interview talking point**:
> "I chose Chi over Gin because it's more idiomatic Go and has zero dependencies. The middleware chain provides observability and resilience."

---

### 5. Request Handlers: `internal/api/handler/`

**Purpose**: Handle HTTP requests, validate input, call services, return responses.

#### `ingest.go` - Ingestion Handler

```go
func (h *IngestHandler) HandleIngest(w http.ResponseWriter, r *http.Request) {
    // 1. Record latency for metrics
    start := time.Now()
    defer h.metricsService.RecordIngestion(time.Since(start))
    
    // 2. Parse JSON body
    json.NewDecoder(r.Body).Decode(&req)
    
    // 3. Validate input
    validator.ValidateIngestRequest(req)
    
    // 4. Submit to service
    h.ingestionService.IngestLog(req)
    
    // 5. Return 201 Created
    h.respondJSON(w, http.StatusCreated, response)
}
```

**Interview talking point**:
> "The handler follows the single responsibility principle - it only handles HTTP concerns. Business logic is delegated to the service layer."

---

### 6. Worker Pool: `internal/worker/pool.go`

**Purpose**: Processes log events concurrently using goroutines.

This is the **most important file for high throughput**.

```go
type Pool struct {
    workers      int              // Number of goroutines (default: 10)
    logChannel   chan LogEvent    // Buffered channel (default: 10,000)
    batchSize    int              // Logs per DB write (default: 100)
    batchTimeout time.Duration    // Max wait before flush (default: 1s)
}
```

**Key Code - Batch Processing**:
```go
for {
    select {
    case logEvent := <-p.logChannel:
        batch = append(batch, logEvent)
        if len(batch) >= p.batchSize {
            p.flushBatch(batch)  // Write to DB
            batch = batch[:0]    // Reset
        }
    case <-ticker.C:
        if len(batch) > 0 {
            p.flushBatch(batch)  // Timeout flush
        }
    case <-p.ctx.Done():
        p.flushBatch(batch)  // Graceful shutdown
        return
    }
}
```

**Interview talking points**:
> "The worker pool pattern decouples the HTTP response from the database write. The client gets a response immediately while the actual persistence happens asynchronously."

> "I implemented backpressure handling - if the channel is full, we return an error instead of blocking. This prevents the server from running out of memory."

> "Batch processing dramatically improves throughput. Writing 100 logs in one transaction is much faster than 100 individual writes."

---

### 7. PostgreSQL Repository: `internal/repository/postgres.go`

**Purpose**: All PostgreSQL database operations.

**Key Methods**:

| Method | Purpose |
|--------|---------|
| `InsertLog` | Single insert |
| `BatchInsertLogs` | Bulk insert with transaction |
| `QueryLogs` | Dynamic query builder |
| `HealthCheck` | Connection test |

**Batch Insert with Transaction**:
```go
func (r *PostgresRepository) BatchInsertLogs(ctx context.Context, logs []LogEvent) error {
    tx, _ := r.pool.Begin(ctx)        // Start transaction
    defer tx.Rollback(ctx)            // Rollback on error
    
    for _, log := range logs {
        tx.Exec(ctx, insertQuery, ...)  // Insert each log
    }
    
    return tx.Commit(ctx)             // Commit all at once
}
```

**Dynamic Query Builder**:
```go
// Builds: WHERE timestamp >= $1 AND timestamp <= $2 AND severity IN ($3,$4)
conditions = append(conditions, "timestamp >= $1")
if len(req.Severity) > 0 {
    conditions = append(conditions, "severity IN ($3,$4)")
}
whereClause := strings.Join(conditions, " AND ")
```

**Interview talking point**:
> "I use `pgx` instead of the standard `database/sql` because it's faster and has better PostgreSQL-specific features like connection pooling."

---

### 8. Redis Repository: `internal/repository/redis.go`

**Purpose**: Caches query results to avoid repeated database hits.

**Cache Strategy**:
1. Generate SHA256 hash of query parameters → cache key
2. On query: check Redis first (cache hit = instant response)
3. On miss: query PostgreSQL, then cache result
4. TTL: 5 minutes (configurable)

```go
func (r *RedisRepository) generateCacheKey(req QueryRequest) string {
    data := fmt.Sprintf("%v-%v-%v-%v-%d-%d",
        req.StartTime.Unix(), req.EndTime.Unix(),
        req.Severity, req.Source, req.Limit, req.Offset)
    hash := sha256.Sum256([]byte(data))
    return fmt.Sprintf("logs:query:%x", hash)
}
```

**Interview talking point**:
> "I implemented a cache-aside pattern. The query service checks Redis first, and on a miss, it queries PostgreSQL and populates the cache. This achieved a 5x improvement in query latency for repeated queries."

---

### 9. Services: `internal/service/`

**Purpose**: Business logic layer between handlers and repositories.

#### `ingestion.go`
- Generates UUID for each log
- Submits to worker pool
- Returns response immediately

#### `query.go`
- Implements cache-aside pattern
- Tries Redis → falls back to PostgreSQL → caches result

#### `metrics.go`
- Tracks ingestion rate, latencies, cache hit ratio
- Thread-safe using atomic operations

**Interview talking point**:
> "The service layer implements the business logic independent of HTTP or database concerns. This makes unit testing easier and allows swapping out implementations."

---

### 10. Validator: `pkg/validator/log.go`

**Purpose**: Input validation with clear error messages.

**Validates**:
- Timestamp not in future
- Severity is valid enum
- Source not empty, max 255 chars
- Message not empty, max 10,000 chars

**Interview talking point**:
> "I put the validator in `pkg/` instead of `internal/` because it could be reused by other projects or CLI tools."

---

## 🔧 Key Go Concepts Used

| Concept | Where Used | Why Important |
|---------|------------|---------------|
| **Goroutines** | Worker pool | Concurrent processing |
| **Channels** | Worker pool buffer | Thread-safe communication |
| **Context** | All layers | Cancellation & timeouts |
| **Interfaces** | Services | Testability |
| **Defer** | Resource cleanup | Guarantee cleanup runs |
| **select** | Worker loop | Multi-channel listening |
| **sync.WaitGroup** | Shutdown | Wait for goroutines |
| **atomic** | Metrics | Thread-safe counters |

---

## 📊 Performance Achieved

| Metric | Value |
|--------|-------|
| **Throughput** | 900+ requests/second |
| **Avg Latency** | 5-6ms |
| **Max Latency** | <200ms |
| **Success Rate** | 100% at target load |

---

## 🎤 Interview Questions & Answers

### "Why did you choose Go for this project?"

> "Go is ideal for high-performance network services. Its goroutines make concurrent programming simple, and the compiled binary has no runtime dependencies - perfect for containers. Companies like Uber, Dropbox, and Cloudflare use Go for similar log processing systems."

### "How does your system handle high load?"

> "Three mechanisms: (1) Buffered channel absorbs traffic spikes, (2) Worker pool processes logs asynchronously so HTTP responses are instant, (3) Batch processing reduces database round trips by 100x."

### "What happens if the server crashes?"

> "In-flight logs in the buffer would be lost. For production, I'd add a message queue like Kafka or RabbitMQ for durability. The current design prioritizes throughput for this resume project."

### "How would you scale this horizontally?"

> "The stateless design allows horizontal scaling. Run multiple instances behind a load balancer. The only shared state is in PostgreSQL and Redis, which both support clustering."

### "What would you improve?"

> "Three things: (1) Add Kafka for durability, (2) Implement sharding for time-series data, (3) Add Prometheus metrics for observability. The architecture already supports these additions."

---

## 🏆 Resume Line

> Built a **distributed log ingestion system** in Go processing **900+ events/second** with **<6ms latency**, featuring a **worker pool pattern**, **hot/cold storage architecture** (Redis + PostgreSQL), and **batch processing** for optimal throughput.
