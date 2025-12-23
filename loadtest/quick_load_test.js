// Fluxor Quick Load Test
// Scaled-down version for quick performance validation
//
// Run with: k6 run loadtest/quick_load_test.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

const errorRate = new Rate('errors');
const healthLatency = new Trend('health_latency', true);
const apiLatency = new Trend('api_latency', true);
const heavyJsonLatency = new Trend('heavy_json_latency', true);

export const options = {
  // Quick load test: 2 minutes ramp to 1000 VUs
  stages: [
    { duration: '15s', target: 100 },   // Warm up
    { duration: '30s', target: 500 },   // Ramp up
    { duration: '30s', target: 1000 },  // Peak load
    { duration: '15s', target: 0 },     // Ramp down
  ],
  
  thresholds: {
    http_req_duration: ['p(95)<100', 'p(99)<200'],
    errors: ['rate<0.01'],
    http_req_failed: ['rate<0.001'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Heavy JSON payload
const heavyPayload = JSON.stringify({
  users: Array.from({ length: 50 }, (_, i) => ({
    id: i + 1,
    name: `User ${i + 1}`,
    email: `user${i + 1}@example.com`,
    metadata: { created: new Date().toISOString() },
  })),
});

export default function () {
  const rand = Math.random();
  
  if (rand < 0.4) {
    healthCheck();
  } else if (rand < 0.8) {
    apiEndpoint();
  } else {
    heavyJsonEndpoint();
  }
  
  sleep(Math.random() * 0.05);
}

function healthCheck() {
  const start = Date.now();
  const res = http.get(`${BASE_URL}/health`);
  healthLatency.add(Date.now() - start);
  
  const success = check(res, {
    'health: status is 200': (r) => r.status === 200,
  });
  errorRate.add(!success);
}

function apiEndpoint() {
  const start = Date.now();
  const res = http.get(`${BASE_URL}/api/status`);
  apiLatency.add(Date.now() - start);
  
  const success = check(res, {
    'api: status is 200': (r) => r.status === 200,
  });
  errorRate.add(!success);
}

function heavyJsonEndpoint() {
  const start = Date.now();
  const res = http.post(`${BASE_URL}/api/echo`, heavyPayload, {
    headers: { 'Content-Type': 'application/json' },
  });
  heavyJsonLatency.add(Date.now() - start);
  
  const success = check(res, {
    'heavy: status is 200': (r) => r.status === 200,
  });
  errorRate.add(!success);
}

export function handleSummary(data) {
  const metrics = data.metrics;
  
  const summary = {
    timestamp: new Date().toISOString(),
    duration_seconds: data.state.testRunDurationMs / 1000,
    vus_max: metrics.vus_max ? metrics.vus_max.values.max : 0,
    requests_total: metrics.http_reqs ? metrics.http_reqs.values.count : 0,
    rps: metrics.http_reqs ? metrics.http_reqs.values.rate.toFixed(2) : 0,
    latency: {
      avg_ms: metrics.http_req_duration ? metrics.http_req_duration.values.avg.toFixed(2) : 0,
      p50_ms: metrics.http_req_duration ? metrics.http_req_duration.values.med.toFixed(2) : 0,
      p90_ms: metrics.http_req_duration ? metrics.http_req_duration.values['p(90)'].toFixed(2) : 0,
      p95_ms: metrics.http_req_duration ? metrics.http_req_duration.values['p(95)'].toFixed(2) : 0,
      p99_ms: metrics.http_req_duration ? metrics.http_req_duration.values['p(99)'].toFixed(2) : 0,
      max_ms: metrics.http_req_duration ? metrics.http_req_duration.values.max.toFixed(2) : 0,
    },
    error_rate_percent: metrics.errors ? (metrics.errors.values.rate * 100).toFixed(4) : 0,
    thresholds_passed: Object.values(data.thresholds || {}).every(t => t.ok),
  };
  
  let output = '\n' + '='.repeat(70) + '\n';
  output += 'FLUXOR PERFORMANCE TEST RESULTS\n';
  output += '='.repeat(70) + '\n';
  output += `Duration: ${summary.duration_seconds}s\n`;
  output += `Max VUs: ${summary.vus_max}\n`;
  output += `Total Requests: ${summary.requests_total}\n`;
  output += `Throughput: ${summary.rps} req/s\n`;
  output += `\nLatency Distribution:\n`;
  output += `  Average: ${summary.latency.avg_ms}ms\n`;
  output += `  Median (P50): ${summary.latency.p50_ms}ms\n`;
  output += `  P90: ${summary.latency.p90_ms}ms\n`;
  output += `  P95: ${summary.latency.p95_ms}ms\n`;
  output += `  P99: ${summary.latency.p99_ms}ms\n`;
  output += `  Max: ${summary.latency.max_ms}ms\n`;
  output += `\nError Rate: ${summary.error_rate_percent}%\n`;
  output += `Thresholds Passed: ${summary.thresholds_passed ? 'YES ✓' : 'NO ✗'}\n`;
  output += '='.repeat(70) + '\n';
  
  return {
    'stdout': output,
    'loadtest/quick_results.json': JSON.stringify(summary, null, 2),
  };
}
