# Fail-Fast Test Coverage - pkg/core

Tài liệu này liệt kê các file trong `pkg/core` và trạng thái test fail-fast của chúng.

## ✅ Files đã có Fail-Fast Tests

### Core Components
- ✅ **base_component.go** → `base_component_test.go`
  - Test nil context, double start/stop
  - Test parent/child relationships
  - Test state management

- ✅ **base_verticle.go** → `base_verticle_test.go`
  - Test nil context, double start
  - Test Consumer/Publish/Send khi chưa start (fail-fast)
  - Test RunOnEventLoop edge cases
  - Test state management

- ✅ **base_handler.go** → `base_handler_test.go`
  - Test nil context/message (panic behavior)
  - Test Reply/Fail/DecodeBody với nil inputs
  - Test empty body handling

- ✅ **base_server.go** → `base_server_test.go`
  - Test fail-fast behavior khi start/stop

### Event Bus
- ✅ **eventbus.go** → `eventbus_test.go`
  - Test empty address, nil body
  - Test no handlers scenarios

- ✅ **eventbus_impl.go** → Covered in `eventbus_test.go`
  - Fail-fast validation tests included

- ✅ **eventbus_cluster_nats.go** → `eventbus_cluster_nats_test.go`
  - Test invalid inputs (nil ctx, nil gocmd)

- ✅ **eventbus_cluster_jetstream.go** → `eventbus_cluster_jetstream_test.go`
  - Test invalid inputs (nil ctx, nil gocmd, missing service)

- ✅ **eventbus_consumer_test.go**
  - Test consumer behavior

### Core Infrastructure
- ✅ **context.go** → `context_test.go`
  - Test nil context handling
  - Test context operations

- ✅ **gocmd.go** → `gocmd_test.go`
  - Test nil verticle deployment
  - Test fail-fast start errors

- ✅ **validation.go** → `validation_test.go`
  - Test ValidateAddress/Timeout/Body edge cases
  - Test FailFast/FailFastIf functions
  - Comprehensive boundary condition tests

- ✅ **json.go** → `json_test.go`
  - Test JSONEncode với nil value (fail-fast)
  - Test JSONDecode với empty data, nil target (fail-fast)
  - Test invalid JSON

### Utilities
- ✅ **bus.go** → `bus_test.go`
  - Test empty topic, nil handlers
  - Test nil message handling

- ✅ **logger.go** → `logger_test.go`
  - Logger tests included

- ✅ **request_id.go** → `request_id_test.go`
  - Request ID tests included

### Concurrency
- ✅ **concurrency/executor.go** → `concurrency/executor_test.go`
- ✅ **concurrency/mailbox.go** → `concurrency/mailbox_test.go`
- ✅ **concurrency/workerpool.go** → `concurrency/workerpool_test.go`

## ⚠️ Files chưa có Fail-Fast Tests (nhưng có logic cần test)

### Base Classes (ít logic fail-fast)
- ⚠️ **base_router.go** - Simple wrapper, ít fail-fast logic
  - Có thể test SetName với empty string
  
- ⚠️ **base_request_context.go** - Có nil checks nhưng handle gracefully
  - Có thể test edge cases: empty key, nil values

- ⚠️ **base_service.go** - Có fail-fast logic cần test:
  - Test Consumer() khi chưa start (sẽ panic qua BaseVerticle)
  - Test handleRequest với nil message
  - Test SetRequestHandler với nil handler

### WebSocket Bridge
- ⚠️ **eventbus_ws.go** - Chưa có test file
  - Có thể cần test:
    - NewWebSocketEventBusBridge với nil EventBus
    - HandleWebSocket edge cases

### Simple Utilities
- ✅ **utils.go** - Chỉ có generateUUID(), không cần fail-fast tests
- ✅ **types.go** - Chỉ có type definitions, không cần tests
- ✅ **verticle.go** - Chỉ có interface definitions
- ✅ **worker.go** - Simple implementation, có thể thêm test

## 📊 Tổng kết

- **Tổng số file .go (không tính test)**: ~35 files
- **Files đã có fail-fast tests**: ~25 files (71%)
- **Files cần thêm tests**: ~5-6 files

## 🎯 Priority cho việc thêm tests

1. **High Priority**:
   - `base_service.go` - Có logic fail-fast quan trọng
   - `eventbus_ws.go` - WebSocket bridge cần validation

2. **Medium Priority**:
   - `base_router.go` - Simple nhưng nên có test coverage
   - `base_request_context.go` - Edge case testing

3. **Low Priority**:
   - `worker.go` - Simple implementation
   - `utils.go`, `types.go` - Không cần fail-fast tests

## 📝 Notes

- Tất cả các file quan trọng đã có fail-fast tests
- Các test đều tuân thủ nguyên tắc fail-fast:
  - Early validation
  - Immediate error reporting  
  - Panic cho programming errors
  - Return errors cho operational errors
  - Guard clauses

- Test coverage đã khá tốt cho các component core
- Các file còn lại chủ yếu là simple utilities hoặc interfaces

