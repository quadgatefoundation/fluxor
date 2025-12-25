# TCP/HTTP Load Balancing Example

This example demonstrates a **Master-Worker load balancing pattern** with both TCP and HTTP interfaces using the Fluxor framework.

## Architecture

```
                    ┌─────────────────────────────────────┐
                    │           Master Node               │
                    │        (Load Balancer)              │
                    │                                     │
    HTTP ────────►  │  ┌─────────────┐ ┌─────────────┐   │
    :8080           │  │ HTTP Server │ │ TCP Server  │   │
                    │  └──────┬──────┘ └──────┬──────┘   │
    TCP ─────────►  │         │               │          │
    :9090           │         └───────┬───────┘          │
                    │                 │                  │
                    │         Round-Robin LB             │
                    │                 │                  │
                    └─────────────────┼──────────────────┘
                                      │
                          ┌───────────┴───────────┐
                          │      EventBus         │
                          └───────────┬───────────┘
                                      │
              ┌───────────────────────┼───────────────────────┐
              │                       │                       │
    ┌─────────▼─────────┐   ┌─────────▼─────────┐   ┌─────────▼─────────┐
    │    Worker A       │   │    Worker B       │   │    Worker N       │
    │                   │   │                   │   │                   │
    │ worker.process.A  │   │ worker.process.B  │   │ worker.process.N  │
    └───────────────────┘   └───────────────────┘   └───────────────────┘
```

## Components

### Master Node (1 instance)
- **HTTP Server** on port `:8080` - REST API endpoints
- **TCP Server** on port `:9090` - Line-based text protocol
- **Load Balancer** - Round-robin distribution to workers

### Worker Nodes (2+ instances)
- Process work requests received via EventBus
- Each worker has a unique address: `worker.process.<ID>`
- Simulates variable processing time based on priority

## Running

### Basic (2 workers)
```bash
go run examples/tcp-http-load-balancing/main.go
```

### Custom configuration
```bash
# 4 workers with custom ports
go run examples/tcp-http-load-balancing/main.go -workers=4 -http=:8081 -tcp=:9091
```

### Command line options
| Flag | Default | Description |
|------|---------|-------------|
| `-http` | `:8080` | HTTP server address |
| `-tcp` | `:9090` | TCP server address |
| `-workers` | `2` | Number of worker nodes |

## HTTP API

### Health Check
```bash
curl http://localhost:8080/health
```
Response:
```json
{
  "status": "healthy",
  "workers": 2,
  "http": ":8080",
  "tcp": ":9090",
  "requests": 0
}
```

### Detailed Status
```bash
curl http://localhost:8080/status
```
Response:
```json
{
  "worker_count": 2,
  "active_workers": ["A", "B"],
  "total_processed": 0,
  "http_addr": ":8080",
  "tcp_addr": ":9090"
}
```

### List Workers
```bash
curl http://localhost:8080/workers
```
Response:
```json
{
  "count": 2,
  "workers": [
    {"id": "A", "address": "worker.process.A"},
    {"id": "B", "address": "worker.process.B"}
  ]
}
```

### Process Request (GET)
```bash
curl "http://localhost:8080/process?data=hello&priority=2"
```
Response:
```json
{
  "id": "http-1234567890",
  "result": "Processed 'hello' by worker-A",
  "worker": "A",
  "duration_ms": 70
}
```

### Process Request (POST)
```bash
curl -X POST http://localhost:8080/process \
  -H "Content-Type: application/json" \
  -d '{"payload": "important task", "priority": 5}'
```
Response:
```json
{
  "id": "http-1234567890",
  "result": "Processed 'important task' by worker-B",
  "worker": "B",
  "duration_ms": 100
}
```

## TCP Protocol

### Line-based protocol
Send a line of text, receive a response line.

### Process Request
```bash
echo "hello world" | nc localhost 9090
```
Response:
```
OK: worker=A, result=Processed 'hello world' by worker-A, duration=60ms
```

### PING Command
```bash
echo "PING" | nc localhost 9090
```
Response:
```
PONG
```

### STATUS Command
```bash
echo "STATUS" | nc localhost 9090
```
Response:
```
MASTER: workers=2, processed=5
```

## Load Testing

### HTTP Load Test (using hey)
```bash
# Install hey: go install github.com/rakyll/hey@latest
hey -n 1000 -c 10 "http://localhost:8080/process?data=test"
```

### TCP Load Test (using netcat in a loop)
```bash
for i in {1..100}; do
  echo "test-$i" | nc localhost 9090 &
done
wait
```

## Primary Pattern

This example follows the Fluxor **Primary Pattern**:

```go
func main() {
    // 1. Create app
    app, err := fluxor.NewMainVerticle("")
    
    // 2. Deploy workers first (dependencies first)
    for _, id := range workerIDs {
        app.DeployVerticle(verticles.NewWorkerVerticle(id))
    }
    
    // 3. Deploy master (depends on workers)
    app.DeployVerticle(verticles.NewMasterVerticle(workerIDs, httpAddr, tcpAddr))
    
    // 4. Start and block
    app.Start()
}
```

## Key Concepts

### Load Balancing
- **Round-robin** distribution ensures even load across workers
- Atomic counter ensures thread-safe worker selection

### EventBus Communication
- Master sends requests to workers via EventBus
- Workers reply with results via request-reply pattern
- Decouples master from worker implementation

### Fail-Fast
- TCP server uses backpressure to reject excess connections
- HTTP server returns 503 when workers unavailable
- Workers validate request bodies before processing

### Graceful Shutdown
- `Ctrl+C` triggers graceful shutdown
- Workers complete in-flight requests
- Master stops accepting new connections

## Files

```
examples/tcp-http-load-balancing/
├── main.go                 # Entry point, CLI flags
├── config.json            # Configuration file
├── README.md              # This file
├── contracts/
│   └── contracts.go       # Message types and addresses
└── verticles/
    ├── master.go          # Master verticle (HTTP + TCP + LB)
    └── worker.go          # Worker verticle
```
