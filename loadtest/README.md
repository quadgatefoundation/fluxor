# Fluxor Load Testing

Load testing suite for Fluxor using [k6](https://k6.io/).

## Prerequisites

Install k6:

```bash
# macOS
brew install k6

# Ubuntu/Debian
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Docker
docker run --rm -i grafana/k6 run - <load_test.js
```

## Test Types

### Smoke Test (CI/CD)
Quick validation test for pipelines:
```bash
k6 run loadtest/smoke_test.js
```
- 10 VUs for 30 seconds
- P95 < 100ms threshold
- Error rate < 1%

### Load Test (Standard)
Standard load test ramping to 10k concurrent users:
```bash
k6 run loadtest/load_test.js
```
- Ramps to 10,000 VUs over 5 minutes
- P95 < 200ms threshold
- Error rate < 0.1%

### Stress Test
Find the breaking point:
```bash
k6 run loadtest/stress_test.js
```
- Ramps to 20,000 VUs
- Measures backpressure activation
- Identifies system limits

## Configuration

Set the target URL:
```bash
k6 run -e BASE_URL=http://your-server:8080 loadtest/load_test.js
```

Customize VUs and duration:
```bash
k6 run --vus 100 --duration 1m loadtest/load_test.js
```

## Metrics Collected

| Metric | Description |
|--------|-------------|
| `http_req_duration` | Total request duration |
| `health_latency` | /health endpoint latency |
| `api_latency` | /api/* endpoints latency |
| `heavy_json_latency` | Heavy JSON payload latency |
| `errors` | Error rate |
| `backpressure_503` | 503 responses (stress test) |

## Thresholds

| Threshold | Target | Description |
|-----------|--------|-------------|
| P95 Response Time | < 200ms | 95th percentile latency |
| P99 Response Time | < 500ms | 99th percentile latency |
| Error Rate | < 0.1% | Failed requests |
| Backpressure Rate | varies | 503 responses under load |

## Results

Results are saved to:
- `loadtest/results.json` - Load test summary
- `loadtest/stress_results.json` - Stress test summary

## CI Integration

Load tests run automatically on push to main branch.
See `.github/workflows/loadtest.yml`.

## Performance Tuning

Based on load test results, tune these parameters in `pkg/web/fasthttp_server.go`:

| Parameter | Default | Description |
|-----------|---------|-------------|
| Workers | 100 | HTTP worker goroutines |
| MaxQueue | 10000 | Request queue size |
| Utilization | 67% | Target capacity utilization |

See [PERFORMANCE.md](../PERFORMANCE.md) for detailed tuning guide.
