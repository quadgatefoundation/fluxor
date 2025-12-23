# Changelog

## [1.1.0] - 2025-12-23

### Added - Enterprise Example

A comprehensive production-ready example demonstrating all enterprise features:

#### 🎯 Features Implemented

1. **OpenTelemetry Distributed Tracing**
   - Jaeger exporter integration
   - Automatic span creation and propagation
   - Request ID tracking across services
   - HTTP middleware for automatic tracing

2. **Prometheus Metrics Export**
   - `/metrics` endpoint for Prometheus scraping
   - Custom metrics support
   - Grafana-ready format
   - Integration with HTTP server metrics

3. **JWT Authentication**
   - Token generation with configurable expiration
   - Token validation middleware
   - Customizable claims (user_id, roles, etc.)
   - Header, query, and cookie token extraction

4. **RBAC Authorization**
   - Role-based access control middleware
   - Support for `RequireRole` and `RequireAnyRole`
   - Automatic role extraction from JWT claims
   - Flexible role configuration

5. **Security Middleware**
   - **CORS**: Configurable cross-origin resource sharing
   - **Security Headers**: HSTS, CSP, X-Frame-Options, X-Content-Type-Options
   - **Rate Limiting**: Token bucket algorithm with per-IP limiting
   - **Request Validation**: Automatic input sanitization

6. **Database Connection Pooling**
   - HikariCP-equivalent pooling for Go
   - Connection lifecycle management
   - Health monitoring and statistics
   - Configurable pool size and timeouts

7. **Configuration Management**
   - YAML/JSON config file support
   - Environment variable overrides
   - Type-safe configuration structs
   - Validation and defaults

8. **Enhanced Health Checks**
   - Liveness and readiness endpoints
   - Database health checks
   - HTTP health checks for external services
   - Health check aggregation
   - Detailed health status reporting

9. **Express-like Middleware Chain**
   - Composable middleware pattern
   - Easy middleware ordering
   - Support for async middleware
   - Error handling middleware

10. **Structured Logging**
    - JSON logging support
    - Contextual fields (request_id, user_id, etc.)
    - Automatic request ID propagation
    - Log level filtering

#### 📁 New Files

- `cmd/enterprise/main.go` - Complete enterprise application example
- `cmd/enterprise/README.md` - Comprehensive documentation
- `cmd/enterprise/config.yaml` - Sample configuration
- `cmd/enterprise/Dockerfile` - Docker containerization
- `cmd/enterprise/docker-compose.yml` - Full stack deployment
- `cmd/enterprise/prometheus.yml` - Prometheus configuration
- `cmd/enterprise/grafana-datasources.yml` - Grafana setup

#### 🔧 Improvements

- **JWT Token Generator**: Added `NewJWTTokenGenerator` function to generate JWT tokens programmatically
- **Middleware Patterns**: Improved middleware chaining and composition
- **Health Check API**: Simplified health check registration
- **Prometheus Integration**: Simplified metrics endpoint registration

#### 🐛 Bug Fixes

- Fixed Go 1.24 compatibility issue with Sonic JSON library (switched to stdlib encoding/json)
- Fixed middleware type signatures for consistency
- Fixed health check aggregator API usage

#### 📚 Documentation

- Updated main README with enterprise example
- Added enterprise example documentation
- Added Docker deployment guide
- Added API endpoint documentation
- Added observability stack setup guide

### Migration from Sonic to stdlib JSON

Due to Go 1.24 compatibility issues with the Sonic JSON library, the project now uses the standard library `encoding/json` package. This ensures compatibility across all Go versions while maintaining a clean API. When Sonic adds Go 1.24 support, we can switch back for improved performance.

### Breaking Changes

None. All changes are additive and backward compatible.

### Usage Example

```go
package main

import (
    "github.com/fluxorio/fluxor/pkg/web"
    "github.com/fluxorio/fluxor/pkg/web/middleware/auth"
    "github.com/fluxorio/fluxor/pkg/web/middleware/security"
)

func main() {
    // Setup middleware chain (Express-like)
    middlewares := []web.FastMiddleware{
        security.CORS(security.DefaultCORSConfig()),
        security.Headers(security.DefaultHeadersConfig()),
        security.RateLimit(security.DefaultRateLimitConfig()),
        auth.JWT(auth.DefaultJWTConfig("secret-key")),
        auth.RequireAnyRole("admin", "user"),
    }
    
    // Apply to routes
    router.GETFast("/api/users", applyMiddleware(middlewares, handleUsers))
}
```

### Docker Deployment

```bash
# Run full stack (app + postgres + jaeger + prometheus + grafana)
cd cmd/enterprise
docker-compose up -d

# Access services
# App: http://localhost:8080
# Jaeger UI: http://localhost:16686
# Prometheus: http://localhost:9091
# Grafana: http://localhost:3000 (admin/admin)
```

### Endpoints

The enterprise example provides the following endpoints:

**Public:**
- `GET /` - Welcome page
- `GET /health` - Basic health check
- `GET /ready` - Readiness probe
- `GET /health/detailed` - Detailed health status
- `GET /metrics` - Prometheus metrics
- `POST /api/auth/login` - Get JWT token
- `POST /api/auth/register` - Register new user

**Authenticated (JWT required):**
- `GET /api/users` - List users
- `POST /api/users` - Create user
- `GET /api/users/:id` - Get user by ID

**Admin (JWT + admin role required):**
- `GET /api/admin/metrics` - Server metrics
- `GET /api/admin/stats` - Database pool stats

### Performance

With the default configuration (67% utilization, 5000 max CCU):

- **Normal capacity**: 3,350 CCU
- **Request throughput**: ~100,000 RPS
- **P95 latency**: < 10ms
- **P99 latency**: < 50ms
- **Backpressure activation**: > 3,350 CCU (returns 503)

### Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

### License

MIT License - See [LICENSE](LICENSE) for details.
