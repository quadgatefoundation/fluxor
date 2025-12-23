# Load Testing Quick Start Guide

## Prerequisites

Install k6 load testing tool:

```bash
# macOS
brew install k6

# Linux (Debian/Ubuntu)
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Docker
docker pull grafana/k6
```

## Run Load Tests

### 1. Start the Server

```bash
# Terminal 1: Start enterprise example
cd /workspace
go run cmd/enterprise/main.go
```

Wait for server to start (should see "Server started" message).

### 2. Run Tests

```bash
# Terminal 2: Run load tests

# Load test (10k concurrent users)
k6 run loadtest/load-test.js

# Spike test (sudden burst)
k6 run loadtest/spike-test.js

# Stress test (find breaking point)
k6 run loadtest/stress-test.js
```

## Expected Results

### Load Test (10k users)

**Phase 1: Normal Load (< 3,350 users)**
```
✓ P95 latency < 50ms
✓ Error rate < 0.1%
✓ 50,000+ RPS
✓ All checks passing
```

**Phase 2: Peak Load (3,350-5,000 users)**
```
⚠ P95 latency 50-200ms
⚠ Some 503 responses (backpressure)
✓ System stable
✓ Graceful degradation
```

**Phase 3: Overload (> 5,000 users)**
```
✗ ~50% error rate (503 responses)
✓ System remains stable
✓ Accepted requests perform well
✓ No crashes or timeouts
```

### Spike Test

```
✓ Baseline: < 10ms P95
✓ Spike: < 500ms P95
✓ Error rate < 1%
✓ Quick recovery
```

### Stress Test

```
Breaking point: ~5,000-10,000 users
System behavior: Graceful degradation
Status: System remains stable
```

## Interpreting Results

### Good Performance ✅

```
✓ http_req_duration: avg=45ms  p(95)=120ms  max=500ms
✓ http_req_failed: 0.03%
✓ http_reqs: 125,000 (2,083 RPS)
✓ errors: 0.05%
```

### Performance Issues ❌

```
✗ http_req_duration: avg=500ms  p(95)=2000ms  max=10s
✗ http_req_failed: 5%
✗ http_reqs: 50,000 (833 RPS)
✗ errors: 10%
```

## Tuning Based on Results

### If P95 > 200ms

1. Increase worker pool size
2. Reduce queue size (fail faster)
3. Check database performance
4. Add caching

### If Error Rate > 0.1%

1. Increase maxCCU
2. Scale horizontally (add instances)
3. Check rate limiting
4. Review logs for errors

### If Many 503 Responses

1. System is at capacity (expected behavior)
2. Options:
   - Increase maxCCU (vertical scaling)
   - Add more instances (horizontal scaling)
   - Optimize application code
   - Add caching layer

## Performance Targets

**Current Configuration (Single Instance):**
- Normal capacity: 3,350 CCU
- Peak capacity: 5,000 CCU
- Target RPS: 50,000+
- Target P95: < 50ms

**With Horizontal Scaling (3 Instances):**
- Normal capacity: 10,050 CCU
- Peak capacity: 15,000 CCU
- Target RPS: 150,000+
- Target P95: < 50ms

## CI/CD Integration

Load tests run automatically on:
- Push to `main` branch
- Pull requests
- Nightly builds
- Manual workflow dispatch

View results in GitHub Actions artifacts.

## Troubleshooting

### Server won't start

```bash
# Check if port is in use
lsof -i :8080

# Kill existing process
pkill -f "cmd/enterprise"

# Restart server
go run cmd/enterprise/main.go
```

### Connection refused

```bash
# Ensure server is running
curl http://localhost:8080/health

# Check server logs
# Look for "Server started" message
```

### k6 not found

```bash
# Verify installation
k6 version

# Reinstall if needed
brew reinstall k6  # macOS
```

## Next Steps

1. ✅ Run actual k6 tests
2. ✅ Compare results with simulated analysis
3. ✅ Tune configuration based on results
4. ✅ Set up monitoring (Prometheus + Grafana)
5. ✅ Plan scaling strategy
6. ✅ Deploy to production

## Documentation

- [Load Testing README](loadtest/README.md) - Detailed guide
- [PERFORMANCE.md](PERFORMANCE.md) - Complete performance guide
- [PERFORMANCE_ENGINEERING_REPORT.md](PERFORMANCE_ENGINEERING_REPORT.md) - Engineering report
- [ARCHITECTURE.md](ARCHITECTURE.md) - Architecture with benchmarks

---

**Quick Commands:**

```bash
# Terminal 1: Start server
go run cmd/enterprise/main.go

# Terminal 2: Run load test
k6 run loadtest/load-test.js

# View results
cat load-test-results.json | jq
```

**Status**: ✅ Ready to run load tests
