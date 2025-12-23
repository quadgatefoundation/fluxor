// Fluxor Load Testing Script
// Uses k6 (https://k6.io/) for load testing
//
// Run with: k6 run loadtest/load_test.js
// Or with options: k6 run --vus 100 --duration 30s loadtest/load_test.js

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const healthLatency = new Trend('health_latency', true);
const apiLatency = new Trend('api_latency', true);
const heavyJsonLatency = new Trend('heavy_json_latency', true);
const requestsTotal = new Counter('requests_total');

// Test configuration
export const options = {
  // Ramp up to 10k concurrent requests over 5 minutes
  stages: [
    { duration: '30s', target: 100 },    // Warm up
    { duration: '1m', target: 500 },     // Ramp up
    { duration: '1m', target: 2000 },    // Scale up
    { duration: '1m', target: 5000 },    // Heavy load
    { duration: '1m', target: 10000 },   // Peak load (10k VUs)
    { duration: '30s', target: 10000 },  // Sustain peak
    { duration: '30s', target: 0 },      // Ramp down
  ],
  
  // Performance thresholds
  thresholds: {
    // 95th percentile response time < 200ms
    'http_req_duration{endpoint:health}': ['p(95)<200'],
    'http_req_duration{endpoint:api}': ['p(95)<200'],
    'http_req_duration{endpoint:heavy}': ['p(95)<500'],  // Allow 500ms for heavy JSON
    
    // Overall 95th percentile < 200ms
    http_req_duration: ['p(95)<200', 'p(99)<500'],
    
    // Error rate < 0.1%
    errors: ['rate<0.001'],
    
    // At least 99.9% success rate
    http_req_failed: ['rate<0.001'],
  },
  
  // Don't abort on threshold failures during ramp-up
  thresholdAbortOnFail: false,
  
  // Summary output
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
};

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Heavy JSON payload for stress testing
const heavyPayload = JSON.stringify({
  users: Array.from({ length: 100 }, (_, i) => ({
    id: i + 1,
    name: `User ${i + 1}`,
    email: `user${i + 1}@example.com`,
    metadata: {
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      preferences: {
        theme: 'dark',
        language: 'en',
        notifications: true,
        features: ['feature1', 'feature2', 'feature3'],
      },
      tags: ['tag1', 'tag2', 'tag3', 'tag4', 'tag5'],
    },
  })),
  metadata: {
    total: 100,
    page: 1,
    per_page: 100,
    timestamp: Date.now(),
  },
});

// Test scenarios
export default function () {
  // Randomly select endpoint with weighted distribution
  const rand = Math.random();
  
  if (rand < 0.4) {
    // 40% - Health check (lightweight)
    healthCheck();
  } else if (rand < 0.8) {
    // 40% - API endpoint
    apiEndpoint();
  } else {
    // 20% - Heavy JSON endpoint
    heavyJsonEndpoint();
  }
  
  // Small sleep to simulate realistic user behavior
  sleep(Math.random() * 0.1);
}

function healthCheck() {
  const start = Date.now();
  const res = http.get(`${BASE_URL}/health`, {
    tags: { endpoint: 'health' },
  });
  
  healthLatency.add(Date.now() - start);
  requestsTotal.add(1);
  
  const success = check(res, {
    'health: status is 200': (r) => r.status === 200,
    'health: response has status': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.status !== undefined;
      } catch {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
}

function apiEndpoint() {
  const start = Date.now();
  const res = http.get(`${BASE_URL}/api/status`, {
    tags: { endpoint: 'api' },
  });
  
  apiLatency.add(Date.now() - start);
  requestsTotal.add(1);
  
  const success = check(res, {
    'api: status is 200': (r) => r.status === 200,
    'api: response is JSON': (r) => {
      try {
        JSON.parse(r.body);
        return true;
      } catch {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
}

function heavyJsonEndpoint() {
  const start = Date.now();
  const res = http.post(`${BASE_URL}/api/echo`, heavyPayload, {
    headers: {
      'Content-Type': 'application/json',
    },
    tags: { endpoint: 'heavy' },
  });
  
  heavyJsonLatency.add(Date.now() - start);
  requestsTotal.add(1);
  
  const success = check(res, {
    'heavy: status is 200': (r) => r.status === 200,
    'heavy: response echoes data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.echo !== undefined;
      } catch {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
}

// Lifecycle hooks
export function setup() {
  console.log(`Starting load test against ${BASE_URL}`);
  console.log('Warming up server...');
  
  // Warm up the server
  for (let i = 0; i < 10; i++) {
    http.get(`${BASE_URL}/health`);
  }
  
  return { startTime: Date.now() };
}

export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log(`Load test completed in ${duration.toFixed(2)} seconds`);
}

// Custom summary handler
export function handleSummary(data) {
  const summary = {
    timestamp: new Date().toISOString(),
    duration_seconds: data.state.testRunDurationMs / 1000,
    vus_max: data.metrics.vus_max ? data.metrics.vus_max.values.max : 0,
    requests_total: data.metrics.http_reqs ? data.metrics.http_reqs.values.count : 0,
    requests_per_second: data.metrics.http_reqs ? 
      data.metrics.http_reqs.values.rate.toFixed(2) : 0,
    response_time: {
      avg: data.metrics.http_req_duration ? 
        data.metrics.http_req_duration.values.avg.toFixed(2) : 0,
      p95: data.metrics.http_req_duration ? 
        data.metrics.http_req_duration.values['p(95)'].toFixed(2) : 0,
      p99: data.metrics.http_req_duration ? 
        data.metrics.http_req_duration.values['p(99)'].toFixed(2) : 0,
    },
    error_rate: data.metrics.errors ? 
      (data.metrics.errors.values.rate * 100).toFixed(4) + '%' : '0%',
    thresholds_passed: Object.values(data.thresholds || {})
      .every(t => t.ok),
  };
  
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'loadtest/results.json': JSON.stringify(summary, null, 2),
  };
}

// Text summary helper
function textSummary(data, options) {
  const lines = [];
  lines.push('\n' + '='.repeat(60));
  lines.push('FLUXOR LOAD TEST SUMMARY');
  lines.push('='.repeat(60));
  
  if (data.metrics.http_reqs) {
    lines.push(`Total Requests: ${data.metrics.http_reqs.values.count}`);
    lines.push(`Requests/sec: ${data.metrics.http_reqs.values.rate.toFixed(2)}`);
  }
  
  if (data.metrics.http_req_duration) {
    lines.push('\nResponse Times:');
    lines.push(`  Average: ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms`);
    lines.push(`  P95: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms`);
    lines.push(`  P99: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms`);
    lines.push(`  Max: ${data.metrics.http_req_duration.values.max.toFixed(2)}ms`);
  }
  
  if (data.metrics.errors) {
    lines.push(`\nError Rate: ${(data.metrics.errors.values.rate * 100).toFixed(4)}%`);
  }
  
  lines.push('\nThresholds:');
  for (const [name, threshold] of Object.entries(data.thresholds || {})) {
    const status = threshold.ok ? '✓ PASS' : '✗ FAIL';
    lines.push(`  ${status}: ${name}`);
  }
  
  lines.push('='.repeat(60) + '\n');
  
  return lines.join('\n');
}
