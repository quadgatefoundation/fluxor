# Fluxor Performance Guide

This document provides performance benchmarks, tuning guidelines, and optimization recommendations for Fluxor applications.

## Performance Benchmarks

### Load Test Results (Single Node)

Test Environment:
- **CPU**: Intel Xeon (4 cores)
- **Memory**: 8GB RAM
- **OS**: Linux (Ubuntu)
- **Go Version**: 1.24
- **Test Tool**: k6

#### Quick Load Test (90 seconds, ramping to 1000 VUs)

| Metric | Value |
|--------|-------|
| **Total Requests** | 1,374,767 |
| **Throughput** | 15,271 req/s |
| **Error Rate** | 0.00% |
| **P50 Latency** | 0.45ms |
| **P90 Latency** | 5.29ms |
| **P95 Latency** | 11.7ms |
| **P99 Latency** | ~30ms |
| **Max Latency** | 93.73ms |

#### Endpoint-Specific Latency

| Endpoint | Avg Latency | P90 | P95 |
|----------|-------------|-----|-----|
| `/health` | 2.59ms | 5ms | 12ms |
| `/api/status` | 2.6ms | 5ms | 12ms |
| `/api/echo` (heavy JSON) | 2.81ms | 6ms | 13ms |

### Smoke Test Results (10 VUs, 30 seconds)

| Metric | Value |
|--------|-------|
| **Throughput** | ~300 req/s |
| **P95 Latency** | 0.46ms |
| **Error Rate** | 0.00% |

## Performance Thresholds

Fluxor is configured to meet these performance thresholds:

| Threshold | Target | Measured |
|-----------|--------|----------|
| P95 Response Time | < 200ms | ✅ 11.7ms |
| P99 Response Time | < 500ms | ✅ ~30ms |
| Error Rate | < 0.1% | ✅ 0.00% |
| Throughput | > 10k RPS | ✅ 15,271 RPS |

## Server Configuration

### Default Configuration

```go
config := web.DefaultFastHTTPServerConfig(":8080")
// Workers:         100
// MaxQueue:        10,000
// ReadTimeout:     10s
// WriteTimeout:    10s
// MaxConns:        100,000
```

### CCU-Based Configuration (Recommended)

For production workloads, use CCU-based configuration:

```go
// Target: 5000 max CCU with 67% utilization
maxCCU := 5000
utilizationPercent := 67
config := web.CCUBasedConfigWithUtilization(":8080", maxCCU, utilizationPercent)

// Resulting configuration:
// - Normal capacity: 3,350 CCU (67% of 5000)
// - Workers: 335
// - Queue size: 3,015
// - Headroom: 33% for traffic spikes
```

## Tuning Parameters

### 1. Worker Pool Size

The worker pool size determines how many concurrent requests can be processed simultaneously.

| Use Case | Workers | Rationale |
|----------|---------|-----------|
| Low latency | 50-100 | Minimize context switching |
| High throughput | 200-500 | Maximize parallelism |
| CPU-intensive | cores * 2 | Balance CPU utilization |
| I/O-intensive | cores * 10 | Handle blocking I/O |

**Recommendation**: Start with `maxCCU / 10` (10% of max concurrent users).

### 2. Queue Size

The queue size determines how many requests can be buffered before backpressure kicks in.

| Queue Size | Effect |
|------------|--------|
| Small (100-1000) | Fast fail, low memory |
| Medium (1000-5000) | Balanced |
| Large (5000-20000) | High burst absorption |

**Recommendation**: Set to `maxCCU - workers` for smooth backpressure.

### 3. Backpressure Target (Utilization)

The utilization percentage determines the normal operating capacity.

| Utilization | Effect |
|-------------|--------|
| 50% | Conservative, large headroom |
| 67% | **Recommended** for production |
| 80% | Aggressive, small headroom |
| 90%+ | Risk of overload |

**Recommendation**: Use 67% utilization for production workloads.

### 4. Timeouts

Configure appropriate timeouts based on your use case:

```go
config := &web.FastHTTPServerConfig{
    ReadTimeout:  10 * time.Second,  // Time to read full request
    WriteTimeout: 10 * time.Second,  // Time to write full response
    // For API servers with quick responses:
    // ReadTimeout:  5 * time.Second,
    // WriteTimeout: 5 * time.Second,
}
```

## Memory Optimization

### Buffer Sizes

Adjust buffer sizes based on typical request/response sizes:

```go
config := &web.FastHTTPServerConfig{
    ReadBufferSize:  8192,   // 8KB for typical requests
    WriteBufferSize: 8192,   // 8KB for typical responses
    // For large payloads:
    // ReadBufferSize:  65536,  // 64KB
    // WriteBufferSize: 65536,  // 64KB
}
```

### Reduce Memory Usage

Enable memory reduction for high-connection scenarios:

```go
server := &fasthttp.Server{
    ReduceMemoryUsage: true,  // Reduces per-connection memory
}
```

## JSON Performance

Fluxor uses the standard library's `encoding/json` for JSON encoding/decoding.

### Benchmark Results

| Operation | Time | Allocations |
|-----------|------|-------------|
| JSON Encode | 617 ns/op | 11 allocs |
| JSON Decode | 1.1 µs/op | 19 allocs |
| Parallel Encode | 99 ns/op | 6 allocs |

### Optimization Tips

1. **Use structs instead of maps** for better performance
2. **Pre-allocate slices** when size is known
3. **Avoid unnecessary encoding** - return pre-encoded responses for static data

## Concurrency Model

### Reactor Pattern

Fluxor uses a reactor pattern with bounded queues:

```
Request → Backpressure Check → Queue → Worker Pool → Handler → Response
              ↓                  ↓
           503 if full      Bounded capacity
```

### Best Practices

1. **Keep handlers non-blocking** - Use async patterns for I/O
2. **Use context for cancellation** - Propagate cancellation properly
3. **Avoid goroutine leaks** - Always handle context cancellation
4. **Monitor metrics** - Watch queue utilization and CCU

## Monitoring

### Key Metrics to Monitor

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| `queue_utilization` | Percentage of queue used | > 80% |
| `ccu_utilization` | Concurrent user utilization | > 90% |
| `rejected_requests` | 503 responses | Increasing trend |
| `error_rate` | 5xx responses | > 0.1% |
| `p95_latency` | 95th percentile latency | > 200ms |

### Health Endpoints

```bash
# Basic health check
curl http://localhost:8080/health

# Readiness check (includes metrics)
curl http://localhost:8080/ready

# Detailed metrics
curl http://localhost:8080/api/metrics
```

## Load Testing

### Running Load Tests

```bash
# Smoke test (quick validation)
k6 run loadtest/smoke_test.js

# Quick load test (90 seconds, 1000 VUs)
k6 run loadtest/quick_load_test.js

# Full load test (5 minutes, 10k VUs)
k6 run loadtest/load_test.js

# Stress test (find breaking point)
k6 run loadtest/stress_test.js
```

### Custom Load Test Parameters

```bash
# Custom VUs and duration
k6 run --vus 500 --duration 2m loadtest/smoke_test.js

# Target different server
k6 run -e BASE_URL=http://production:8080 loadtest/load_test.js
```

## Production Checklist

- [ ] Configure CCU-based backpressure
- [ ] Set appropriate timeouts
- [ ] Enable health and readiness endpoints
- [ ] Configure Prometheus metrics export
- [ ] Set up alerting on key metrics
- [ ] Run load tests in staging environment
- [ ] Document expected throughput and latency
- [ ] Plan for horizontal scaling
