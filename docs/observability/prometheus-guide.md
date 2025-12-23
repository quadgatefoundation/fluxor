# Prometheus Observability Guide

## Overview

Fluxor exposes comprehensive Prometheus metrics for monitoring application performance, resource utilization, and system health.

## Quick Start

### 1. Enable Metrics Endpoint

```go
package main

import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/observability/prometheus"
    "github.com/fluxorio/fluxor/pkg/web"
)

func main() {
    ctx := context.Background()
    vertx := core.NewVertx(ctx)
    defer vertx.Close()

    // Create HTTP server
    config := web.CCUBasedConfigWithUtilization(":8080", 5000, 67)
    server := web.NewFastHTTPServer(vertx, config)

    // Get router
    router := server.FastRouter()

    // Register /metrics endpoint
    prometheus.RegisterMetricsEndpoint(router, "/metrics")

    // Add metrics middleware to routes
    metricsMiddleware := prometheus.FastHTTPMetricsMiddleware()
    router.GETFast("/api/users", metricsMiddleware(handleGetUsers))

    // Start server
    server.Listen()
}
```

### 2. Scrape Metrics with curl

```bash
# Basic scrape
curl http://localhost:8080/metrics

# Save to file
curl http://localhost:8080/metrics > metrics.txt

# Watch metrics update
watch -n 1 'curl -s http://localhost:8080/metrics | grep fluxor_http_requests_total'

# Filter specific metrics
curl -s http://localhost:8080/metrics | grep http_requests_total

# Get specific metric values
curl -s http://localhost:8080/metrics | grep 'fluxor_server_current_ccu '
```

## Available Metrics

### HTTP Request Metrics

#### `fluxor_http_requests_total`
**Type**: Counter  
**Labels**: `method`, `path`, `status`  
**Description**: Total number of HTTP requests processed

```prometheus
# Example output
fluxor_http_requests_total{method="GET",path="/api/users",status="2xx"} 1523
fluxor_http_requests_total{method="POST",path="/api/users",status="2xx"} 234
fluxor_http_requests_total{method="GET",path="/api/users",status="4xx"} 12
```

**Query Examples**:
```promql
# Total requests per second
rate(fluxor_http_requests_total[5m])

# Error rate
rate(fluxor_http_requests_total{status=~"4xx|5xx"}[5m])

# Requests by endpoint
sum(rate(fluxor_http_requests_total[5m])) by (path)
```

#### `fluxor_http_request_duration_seconds`
**Type**: Histogram  
**Labels**: `method`, `path`, `status`  
**Description**: HTTP request duration in seconds

**Buckets**: 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10

```prometheus
# Example output
fluxor_http_request_duration_seconds_bucket{method="GET",path="/api/users",status="2xx",le="0.005"} 1234
fluxor_http_request_duration_seconds_bucket{method="GET",path="/api/users",status="2xx",le="0.01"} 1450
fluxor_http_request_duration_seconds_sum{method="GET",path="/api/users",status="2xx"} 45.2
fluxor_http_request_duration_seconds_count{method="GET",path="/api/users",status="2xx"} 1523
```

**Query Examples**:
```promql
# P95 latency
histogram_quantile(0.95, rate(fluxor_http_request_duration_seconds_bucket[5m]))

# P99 latency by endpoint
histogram_quantile(0.99, sum(rate(fluxor_http_request_duration_seconds_bucket[5m])) by (path, le))

# Average latency
rate(fluxor_http_request_duration_seconds_sum[5m]) / rate(fluxor_http_request_duration_seconds_count[5m])
```

### Server Metrics

#### `fluxor_server_current_ccu`
**Type**: Gauge  
**Description**: Current concurrent users (CCU)

```prometheus
# Example output
fluxor_server_current_ccu 245
```

**Query Examples**:
```promql
# Current CCU
fluxor_server_current_ccu

# Max CCU in last hour
max_over_time(fluxor_server_current_ccu[1h])

# Average CCU
avg_over_time(fluxor_server_current_ccu[5m])
```

#### `fluxor_server_normal_ccu`
**Type**: Gauge  
**Description**: Normal capacity CCU (target utilization, e.g., 67%)

```prometheus
# Example output
fluxor_server_normal_ccu 3350
```

#### `fluxor_server_ccu_utilization`
**Type**: Gauge  
**Description**: CCU utilization percentage (0-100)

```prometheus
# Example output
fluxor_server_ccu_utilization 7.31
```

**Query Examples**:
```promql
# Current utilization
fluxor_server_ccu_utilization

# Alert if utilization > 80%
fluxor_server_ccu_utilization > 80

# Utilization trend
deriv(fluxor_server_ccu_utilization[5m])
```

#### `fluxor_backpressure_queue_length`
**Type**: Gauge  
**Description**: Current backpressure queue length

```prometheus
# Example output
fluxor_backpressure_queue_length 12
```

**Query Examples**:
```promql
# Current queue length
fluxor_backpressure_queue_length

# Alert if queue > 80% of capacity
fluxor_backpressure_queue_length > 2400  # Example: 80% of 3000

# Queue growth rate
deriv(fluxor_backpressure_queue_length[1m])
```

#### `fluxor_server_rejected_requests_total`
**Type**: Counter  
**Description**: Total number of rejected HTTP requests (503)

```prometheus
# Example output
fluxor_server_rejected_requests_total 45
```

**Query Examples**:
```promql
# Rejection rate
rate(fluxor_server_rejected_requests_total[5m])

# Alert on rejections
rate(fluxor_server_rejected_requests_total[5m]) > 0
```

### Verticle Metrics

#### `fluxor_verticle_count`
**Type**: Gauge  
**Description**: Number of deployed verticles

```prometheus
# Example output
fluxor_verticle_count 12
```

**Query Examples**:
```promql
# Current verticle count
fluxor_verticle_count

# Verticle deployments/undeployments
changes(fluxor_verticle_count[5m])
```

### Database Metrics

#### `fluxor_database_connections_open`
**Type**: Gauge  
**Description**: Number of open database connections

```prometheus
# Example output
fluxor_database_connections_open 50
```

#### `fluxor_database_connections_in_use`
**Type**: Gauge  
**Description**: Number of database connections in use

```prometheus
# Example output
fluxor_database_connections_in_use 12
```

#### `fluxor_database_query_duration_seconds`
**Type**: Histogram  
**Labels**: `operation`  
**Description**: Database query duration in seconds

```prometheus
# Example output
fluxor_database_query_duration_seconds_bucket{operation="SELECT",le="0.001"} 1234
fluxor_database_query_duration_seconds_sum{operation="SELECT"} 12.5
fluxor_database_query_duration_seconds_count{operation="SELECT"} 5000
```

**Query Examples**:
```promql
# P95 query latency
histogram_quantile(0.95, rate(fluxor_database_query_duration_seconds_bucket[5m]))

# Slow queries (> 1s)
rate(fluxor_database_query_duration_seconds_bucket{le="1"}[5m]) - 
rate(fluxor_database_query_duration_seconds_bucket{le="0.5"}[5m])
```

### EventBus Metrics

#### `fluxor_eventbus_messages_total`
**Type**: Counter  
**Labels**: `address`, `type`  
**Description**: Total number of EventBus messages

```prometheus
# Example output
fluxor_eventbus_messages_total{address="user.events",type="publish"} 1234
fluxor_eventbus_messages_total{address="order.events",type="send"} 567
```

**Query Examples**:
```promql
# Message rate by address
rate(fluxor_eventbus_messages_total[5m])

# Messages by type
sum(rate(fluxor_eventbus_messages_total[5m])) by (type)
```

## Prometheus Configuration

### prometheus.yml

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'fluxor'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s
    scrape_timeout: 5s

  - job_name: 'fluxor-cluster'
    static_configs:
      - targets:
        - 'fluxor-1:8080'
        - 'fluxor-2:8080'
        - 'fluxor-3:8080'
    metrics_path: '/metrics'
```

### Docker Compose

```yaml
version: '3.8'

services:
  fluxor:
    image: fluxor-app:latest
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - CCU_MAX=5000
      - CCU_UTILIZATION=67

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana-dashboards:/etc/grafana/provisioning/dashboards
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    depends_on:
      - prometheus

volumes:
  prometheus_data:
  grafana_data:
```

## Alert Rules

### alerts.yml

```yaml
groups:
  - name: fluxor_alerts
    interval: 30s
    rules:
      # High error rate
      - alert: HighErrorRate
        expr: |
          rate(fluxor_http_requests_total{status=~"5xx"}[5m]) > 0.01
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} errors/sec"

      # High latency
      - alert: HighLatency
        expr: |
          histogram_quantile(0.95, 
            rate(fluxor_http_request_duration_seconds_bucket[5m])
          ) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High P95 latency detected"
          description: "P95 latency is {{ $value }}s"

      # High CCU utilization
      - alert: HighCCUUtilization
        expr: fluxor_server_ccu_utilization > 80
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "CCU utilization is high"
          description: "CCU utilization is {{ $value }}%"

      # Backpressure active
      - alert: BackpressureActive
        expr: rate(fluxor_server_rejected_requests_total[5m]) > 0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Backpressure is rejecting requests"
          description: "{{ $value }} requests/sec are being rejected"

      # Database connection pool exhaustion
      - alert: DatabasePoolExhaustion
        expr: |
          fluxor_database_connections_in_use / 
          fluxor_database_connections_open > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Database connection pool nearly exhausted"
          description: "{{ $value | humanizePercentage }} of connections in use"
```

## Best Practices

### 1. Metric Cardinality

**DO**:
- Use low-cardinality labels (status class: 2xx, 3xx, 4xx, 5xx)
- Aggregate paths: `/api/users/:id` → `/api/users/:id`
- Limit label values

**DON'T**:
- Use high-cardinality labels (user IDs, request IDs)
- Use unbounded label values
- Create metrics per user/session

### 2. Recording Rules

```yaml
groups:
  - name: fluxor_recording_rules
    interval: 30s
    rules:
      # Pre-calculate P95 latency
      - record: fluxor:http_request_duration_seconds:p95
        expr: |
          histogram_quantile(0.95, 
            sum(rate(fluxor_http_request_duration_seconds_bucket[5m])) by (le)
          )

      # Pre-calculate error rate
      - record: fluxor:http_requests:error_rate
        expr: |
          rate(fluxor_http_requests_total{status=~"5xx"}[5m]) /
          rate(fluxor_http_requests_total[5m])

      # Pre-calculate RPS
      - record: fluxor:http_requests:rps
        expr: rate(fluxor_http_requests_total[5m])
```

### 3. Dashboard Queries

**P95 Latency**:
```promql
histogram_quantile(0.95, 
  sum(rate(fluxor_http_request_duration_seconds_bucket[5m])) by (le)
)
```

**Request Rate**:
```promql
sum(rate(fluxor_http_requests_total[5m]))
```

**Error Rate**:
```promql
sum(rate(fluxor_http_requests_total{status=~"5xx"}[5m])) /
sum(rate(fluxor_http_requests_total[5m]))
```

**CCU Utilization**:
```promql
fluxor_server_ccu_utilization
```

## Troubleshooting

### Metrics Not Appearing

1. **Check endpoint is accessible**:
   ```bash
   curl http://localhost:8080/metrics
   ```

2. **Verify Prometheus is scraping**:
   - Go to http://localhost:9090/targets
   - Check if target is UP

3. **Check labels**:
   ```bash
   curl -s http://localhost:8080/metrics | grep fluxor_
   ```

### High Cardinality

```bash
# Count unique metric combinations
curl -s http://localhost:8080/metrics | grep -c "fluxor_http_requests_total"

# If > 1000, you have high cardinality issues
```

**Solution**: Aggregate paths, reduce label values

### Missing Metrics

```bash
# Check what metrics are available
curl -s http://localhost:8080/metrics | grep "# TYPE" | sort
```

## References

- [Prometheus Documentation](https://prometheus.io/docs/)
- [PromQL Guide](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Best Practices](https://prometheus.io/docs/practices/naming/)
