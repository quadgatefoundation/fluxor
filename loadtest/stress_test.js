// Fluxor Stress Test
// Find the breaking point of the system
//
// Run with: k6 run loadtest/stress_test.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

const errorRate = new Rate('errors');
const backpressureRate = new Rate('backpressure_503');
const latency = new Trend('request_latency', true);

export const options = {
  // Stress test: Gradually increase load until system breaks
  stages: [
    { duration: '1m', target: 1000 },   // Warm up
    { duration: '2m', target: 5000 },   // Ramp up
    { duration: '2m', target: 10000 },  // Heavy load
    { duration: '2m', target: 15000 },  // Stress
    { duration: '2m', target: 20000 },  // Breaking point?
    { duration: '1m', target: 0 },      // Recovery
  ],
  
  thresholds: {
    // Track but don't fail on thresholds (we want to find limits)
    http_req_duration: ['p(95)<1000'],
    errors: ['rate<0.5'],  // Allow up to 50% errors at peak
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const start = Date.now();
  const res = http.get(`${BASE_URL}/health`, {
    timeout: '10s',
  });
  
  latency.add(Date.now() - start);
  
  const is503 = res.status === 503;
  backpressureRate.add(is503);
  
  const success = check(res, {
    'status is 200 or 503': (r) => r.status === 200 || r.status === 503,
  });
  
  errorRate.add(!success && !is503);
  
  // No sleep - maximum pressure
}

export function handleSummary(data) {
  const summary = {
    timestamp: new Date().toISOString(),
    max_vus: data.metrics.vus_max ? data.metrics.vus_max.values.max : 0,
    total_requests: data.metrics.http_reqs ? data.metrics.http_reqs.values.count : 0,
    rps: data.metrics.http_reqs ? data.metrics.http_reqs.values.rate.toFixed(2) : 0,
    latency: {
      avg: data.metrics.http_req_duration ? 
        data.metrics.http_req_duration.values.avg.toFixed(2) : 0,
      p95: data.metrics.http_req_duration ? 
        data.metrics.http_req_duration.values['p(95)'].toFixed(2) : 0,
      p99: data.metrics.http_req_duration ? 
        data.metrics.http_req_duration.values['p(99)'].toFixed(2) : 0,
      max: data.metrics.http_req_duration ? 
        data.metrics.http_req_duration.values.max.toFixed(2) : 0,
    },
    error_rate: data.metrics.errors ? 
      (data.metrics.errors.values.rate * 100).toFixed(4) + '%' : '0%',
    backpressure_rate: data.metrics.backpressure_503 ?
      (data.metrics.backpressure_503.values.rate * 100).toFixed(2) + '%' : '0%',
  };
  
  let output = '\n' + '='.repeat(60) + '\n';
  output += 'FLUXOR STRESS TEST RESULTS\n';
  output += '='.repeat(60) + '\n';
  output += `Max VUs: ${summary.max_vus}\n`;
  output += `Total Requests: ${summary.total_requests}\n`;
  output += `Requests/sec: ${summary.rps}\n`;
  output += `\nLatency:\n`;
  output += `  Average: ${summary.latency.avg}ms\n`;
  output += `  P95: ${summary.latency.p95}ms\n`;
  output += `  P99: ${summary.latency.p99}ms\n`;
  output += `  Max: ${summary.latency.max}ms\n`;
  output += `\nError Rate: ${summary.error_rate}\n`;
  output += `Backpressure (503s): ${summary.backpressure_rate}\n`;
  output += '='.repeat(60) + '\n';
  
  return {
    'stdout': output,
    'loadtest/stress_results.json': JSON.stringify(summary, null, 2),
  };
}
