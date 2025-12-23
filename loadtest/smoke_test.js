// Fluxor Smoke Test
// Quick validation test for CI/CD pipelines
//
// Run with: k6 run loadtest/smoke_test.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export const options = {
  // Quick smoke test: 10 VUs for 30 seconds
  vus: 10,
  duration: '30s',
  
  thresholds: {
    http_req_duration: ['p(95)<100', 'p(99)<200'],
    errors: ['rate<0.01'],
    http_req_failed: ['rate<0.01'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  // Health check
  let res = http.get(`${BASE_URL}/health`);
  check(res, {
    'health: status is 200': (r) => r.status === 200,
  }) || errorRate.add(1);
  
  // API status
  res = http.get(`${BASE_URL}/api/status`);
  check(res, {
    'api: status is 200': (r) => r.status === 200,
  }) || errorRate.add(1);
  
  // Ready check
  res = http.get(`${BASE_URL}/ready`);
  check(res, {
    'ready: status is 200 or 503': (r) => r.status === 200 || r.status === 503,
  }) || errorRate.add(1);
  
  sleep(0.1);
}

export function handleSummary(data) {
  return {
    'stdout': JSON.stringify({
      passed: Object.values(data.thresholds || {}).every(t => t.ok),
      requests: data.metrics.http_reqs ? data.metrics.http_reqs.values.count : 0,
      errors: data.metrics.errors ? data.metrics.errors.values.rate : 0,
      p95: data.metrics.http_req_duration ? 
        data.metrics.http_req_duration.values['p(95)'] : 0,
    }, null, 2) + '\n',
  };
}
