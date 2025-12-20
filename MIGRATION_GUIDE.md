# Migration Guide: Java/Node.js to Go/Fluxor

Hướng dẫn chuyển đổi từ Java/Node.js sang Go với Fluxor framework.

## Mục lục

1. [Tổng quan](#tổng-quan)
2. [Java Developer Migration](#java-developer-migration)
3. [Node.js Developer Migration](#nodejs-developer-migration)
4. [Go Concepts for Beginners](#go-concepts-for-beginners)
5. [Pattern Mapping](#pattern-mapping)
6. [Step-by-Step Migration](#step-by-step-migration)
7. [Common Pitfalls](#common-pitfalls)
8. [Resources](#resources)

---

## Tổng quan

Fluxor được thiết kế để giúp developers từ Java (Vert.x) và Node.js dễ dàng chuyển sang Go. Guide này sẽ giúp bạn:

- ✅ Hiểu các khái niệm Go cơ bản
- ✅ Map patterns từ Java/Node.js sang Go/Fluxor
- ✅ Tránh các lỗi thường gặp
- ✅ Migration từng bước một cách an toàn

---

## Java Developer Migration

### 1. Class → Struct

**Java:**
```java
public class UserService {
    private EventBus eventBus;
    private String name;
    
    public UserService(String name) {
        this.name = name;
    }
}
```

**Go/Fluxor:**
```go
type UserService struct {
    *core.BaseService  // Embed base class (like extends)
    name string
}

func NewUserService(name string) *UserService {
    return &UserService{
        BaseService: core.NewBaseService("user-service", "user.service"),
        name:        name,
    }
}
```

**Key Differences:**
- Go uses `struct` instead of `class`
- No `private/public` keywords (uppercase = public, lowercase = private)
- Constructor is a function (convention: `NewXxx`)
- Embedding (`*core.BaseService`) is like inheritance

### 2. Interface Implementation

**Java:**
```java
public class MyVerticle implements Verticle {
    @Override
    public void start(Vertx vertx) {
        // implementation
    }
}
```

**Go/Fluxor:**
```go
// Option 1: Using Premium Pattern (Recommended)
type MyVerticle struct {
    *core.BaseVerticle
}

func (v *MyVerticle) doStart(ctx core.FluxorContext) error {
    // implementation
    return nil
}

// Option 2: Direct interface implementation
type MyVerticle struct{}

func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    // implementation
    return nil
}
```

**Key Differences:**
- Go: implement interface implicitly (no `implements` keyword)
- Methods are functions with receiver: `func (v *MyVerticle) Start(...)`
- Premium Pattern provides base implementation (like abstract class)

### 3. Inheritance → Embedding

**Java:**
```java
public class UserService extends BaseService {
    // Inherits all BaseService methods
}
```

**Go/Fluxor:**
```go
type UserService struct {
    *core.BaseService  // Embed (composition)
    // Can access all BaseService methods
}

// Usage
service := &UserService{
    BaseService: core.NewBaseService("user", "user.service"),
}
service.Publish("event", data) // Can call BaseService methods
```

**Key Differences:**
- Go uses composition (embedding) not inheritance
- Embedding gives you all methods from embedded struct
- More flexible than inheritance

### 4. Abstract Class → Base Class (Premium Pattern)

**Java:**
```java
public abstract class BaseVerticle {
    protected EventBus eventBus;
    
    public final void start(Vertx vertx) {
        this.eventBus = vertx.eventBus();
        doStart();
    }
    
    protected abstract void doStart();
}
```

**Go/Fluxor:**
```go
// BaseVerticle provides default implementation
type MyVerticle struct {
    *core.BaseVerticle
}

// Override hook method (like abstract method)
func (v *MyVerticle) doStart(ctx core.FluxorContext) error {
    // Custom implementation
    return nil
}
```

**Key Differences:**
- Go doesn't have abstract classes
- Premium Pattern provides base classes with hook methods
- Hook methods (`doStart`, `doStop`) are like abstract methods

### 5. Exception Handling

**Java:**
```java
try {
    result = process();
} catch (Exception e) {
    logger.error("Error", e);
    throw new ServiceException("Failed", e);
}
```

**Go/Fluxor:**
```go
result, err := process()
if err != nil {
    logger.Errorf("Error: %v", err)
    return fmt.Errorf("failed: %w", err) // Wrap error
}
// Use result
```

**Key Differences:**
- Go uses explicit error returns, not exceptions
- Always check `err != nil`
- Use `fmt.Errorf` with `%w` to wrap errors
- No try-catch, use if statements

### 6. Async/Await → Futures/Promises

**Java (Vert.x):**
```java
Future<String> future = eventBus.request("address", data);
future.onSuccess(result -> {
    System.out.println(result);
}).onFailure(err -> {
    System.err.println(err);
});
```

**Go/Fluxor:**
```go
// Option 1: Vert.x style (callbacks)
future := fluxor.NewFuture()
future.OnSuccess(func(result interface{}) {
    fmt.Println(result)
}).OnFailure(func(err error) {
    fmt.Println(err)
})

// Option 2: Async/await style (Premium Pattern)
promise := fluxor.NewPromiseT[string]()
go func() {
    promise.Complete("result")
}()
result, err := promise.Await(ctx) // Like await in Java
```

**Key Differences:**
- Go supports both callback and await-style patterns
- Use `Await(ctx)` for async/await-like syntax
- Context (`ctx`) is required for cancellation

---

## Node.js Developer Migration

### 1. Module System

**Node.js:**
```javascript
// Export
module.exports = {
    UserService: class UserService { ... }
};

// Import
const { UserService } = require('./user-service');
```

**Go/Fluxor:**
```go
// Export (automatic - uppercase = public)
package user

type UserService struct {
    *core.BaseService
}

// Import
import "github.com/yourproject/user"

service := user.NewUserService()
```

**Key Differences:**
- Go packages are directories
- Uppercase = exported (public), lowercase = private
- Import by package path, not file path

### 2. Callbacks → Error Returns

**Node.js:**
```javascript
function processData(data, callback) {
    if (error) {
        callback(error, null);
    } else {
        callback(null, result);
    }
}

processData(data, (err, result) => {
    if (err) {
        console.error(err);
        return;
    }
    console.log(result);
});
```

**Go/Fluxor:**
```go
func processData(data interface{}) (interface{}, error) {
    if error {
        return nil, fmt.Errorf("error: %v", error)
    }
    return result, nil
}

result, err := processData(data)
if err != nil {
    logger.Errorf("Error: %v", err)
    return
}
logger.Infof("Result: %v", result)
```

**Key Differences:**
- Go uses explicit error returns: `(result, error)`
- Always check `err != nil` first
- No callback hell, linear code flow

### 3. Promises → Futures

**Node.js:**
```javascript
const promise = new Promise((resolve, reject) => {
    setTimeout(() => resolve("result"), 100);
});

promise
    .then(result => {
        return process(result);
    })
    .then(processed => {
        console.log(processed);
    })
    .catch(err => {
        console.error(err);
    });
```

**Go/Fluxor:**
```go
// Option 1: Promise.then() style
promise := fluxor.NewPromiseT[string]()
go func() {
    time.Sleep(100 * time.Millisecond)
    promise.Complete("result")
}()

fluxor.Then(promise, func(s string) (string, error) {
    return process(s), nil
}).OnSuccess(func(result string) {
    fmt.Println(result)
}).OnFailure(func(err error) {
    fmt.Println(err)
})

// Option 2: Async/await style (easier!)
result, err := promise.Await(ctx)
if err != nil {
    fmt.Println(err)
    return
}
processed, err := process(result)
fmt.Println(processed)
```

**Key Differences:**
- Go supports both Promise.then() and async/await patterns
- `Await(ctx)` is like `await` in JavaScript
- Context required for cancellation/timeout

### 4. Express Middleware → Fluxor Handlers

**Node.js/Express:**
```javascript
app.use((req, res, next) => {
    req.requestId = generateId();
    next();
});

app.get('/api/users', (req, res) => {
    res.json({ users: [] });
});
```

**Go/Fluxor:**
```go
// Request ID is automatic, no middleware needed!
router.GETFast("/api/users", func(ctx *web.FastRequestContext) error {
    requestID := ctx.RequestID() // Already set
    return ctx.JSON(200, map[string]interface{}{
        "users": []interface{}{},
    })
})
```

**Key Differences:**
- Fluxor handles request ID automatically
- Handlers return `error`, not callbacks
- JSON is default format

### 5. Event Emitter → EventBus

**Node.js:**
```javascript
const EventEmitter = require('events');
const emitter = new EventEmitter();

emitter.on('user.created', (user) => {
    console.log('User created:', user);
});

emitter.emit('user.created', userData);
```

**Go/Fluxor:**
```go
// Register consumer
consumer := eventBus.Consumer("user.created")
consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    var user map[string]interface{}
    core.JSONDecode(msg.Body().([]byte), &user)
    logger.Infof("User created: %v", user)
    return nil
})

// Publish event
eventBus.Publish("user.created", userData)
```

**Key Differences:**
- EventBus is like EventEmitter but type-safe
- Messages are automatically JSON encoded
- Handlers receive context and message

### 6. Async/Await

**Node.js:**
```javascript
async function getUser(id) {
    const user = await db.getUser(id);
    const profile = await db.getProfile(user.id);
    return { user, profile };
}
```

**Go/Fluxor:**
```go
func getUser(ctx context.Context, id string) (map[string]interface{}, error) {
    user, err := db.GetUser(ctx, id)
    if err != nil {
        return nil, err
    }
    
    profile, err := db.GetProfile(ctx, user.ID)
    if err != nil {
        return nil, err
    }
    
    return map[string]interface{}{
        "user":    user,
        "profile": profile,
    }, nil
}

// Or with Futures (async/await style)
func getUserAsync(ctx context.Context, id string) *fluxor.FutureT[map[string]interface{}] {
    userFuture := db.GetUserAsync(ctx, id)
    return fluxor.Then(userFuture, func(user User) (map[string]interface{}, error) {
        profileFuture := db.GetProfileAsync(ctx, user.ID)
        profile, err := profileFuture.Await(ctx)
        if err != nil {
            return nil, err
        }
        return map[string]interface{}{
            "user":    user,
            "profile": profile,
        }, nil
    })
}
```

**Key Differences:**
- Go: explicit error handling, no try-catch
- Futures provide async/await-like syntax
- Context required for cancellation

---

## Go Concepts for Beginners

### 1. Pointers (`*` and `&`)

```go
// * = pointer type, & = address of
var x int = 10
var p *int = &x  // p points to x
*p = 20          // Change value through pointer
fmt.Println(x)    // 20
```

**When to use:**
- Struct methods: `func (v *MyVerticle) Start(...)` - receiver is pointer
- Passing by reference: `&MyStruct{}` - pass address, not copy

### 2. Interfaces

```go
// Interface defines contract
type Verticle interface {
    Start(ctx FluxorContext) error
    Stop(ctx FluxorContext) error
}

// Implement implicitly (no "implements" keyword)
type MyVerticle struct{}
func (v *MyVerticle) Start(ctx FluxorContext) error { return nil }
func (v *MyVerticle) Stop(ctx FluxorContext) error { return nil }
```

**Key Points:**
- No explicit implementation declaration
- If struct has all methods, it implements interface
- Very flexible and powerful

### 3. Error Handling

```go
// Functions return (result, error)
result, err := doSomething()
if err != nil {
    // Handle error
    return err
}
// Use result
```

**Best Practices:**
- Always check `err != nil` first
- Return errors, don't ignore them
- Use `fmt.Errorf` with `%w` to wrap errors

### 4. Struct Embedding (Composition)

```go
type Base struct {
    name string
}

func (b *Base) GetName() string {
    return b.name
}

type Derived struct {
    *Base  // Embed Base
    age int
}

d := &Derived{Base: &Base{name: "John"}, age: 30}
fmt.Println(d.GetName()) // Can call Base methods
```

**Key Points:**
- Like inheritance but more flexible
- Can embed multiple structs
- Access embedded methods directly

### 5. Goroutines (Concurrency)

```go
// Start goroutine (like thread)
go func() {
    // This runs concurrently
    doWork()
}()

// Wait for completion
done := make(chan bool)
go func() {
    doWork()
    done <- true
}()
<-done // Wait
```

**Key Points:**
- `go` keyword starts goroutine
- Use channels for communication
- Much lighter than threads

### 6. Context (Cancellation/Timeout)

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Pass context to operations
result, err := doWork(ctx)
if err != nil {
    // Could be timeout or cancellation
}
```

**Key Points:**
- Context carries cancellation/timeout
- Always pass context to async operations
- Use `defer cancel()` to cleanup

---

## Pattern Mapping

### Java Vert.x → Go Fluxor

| Java/Vert.x | Go/Fluxor | Notes |
|------------|-----------|-------|
| `AbstractVerticle` | `BaseVerticle` | Premium Pattern |
| `Future<T>` | `FutureT[T]` | Type-safe futures |
| `Promise<T>` | `PromiseT[T]` | Type-safe promises |
| `eventBus.send()` | `eventBus.Send()` | Same API |
| `eventBus.publish()` | `eventBus.Publish()` | Same API |
| `eventBus.request()` | `eventBus.Request()` | Same API |
| `vertx.deployVerticle()` | `vertx.DeployVerticle()` | Same API |
| `@Override` | Override method | No annotation needed |

### Node.js → Go Fluxor

| Node.js | Go/Fluxor | Notes |
|---------|-----------|-------|
| `Promise` | `PromiseT[T]` | Type-safe |
| `async/await` | `future.Await(ctx)` | Same pattern |
| `EventEmitter` | `EventBus` | Similar API |
| `express.Router()` | `router.GETFast()` | Similar patterns |
| `req.body` | `msg.Body()` | Similar access |
| `res.json()` | `ctx.JSON()` | Similar API |

---

## Step-by-Step Migration

### Phase 1: Learn Go Basics (1-2 weeks)

1. **Install Go**: https://golang.org/dl/
2. **Learn basics**:
   - Variables, types, functions
   - Structs and methods
   - Interfaces
   - Error handling
   - Pointers
3. **Practice**: Write simple programs

### Phase 2: Understand Fluxor (1 week)

1. **Read documentation**:
   - `README.md` - Overview
   - `ARCHITECTURE.md` - Architecture
   - `BASE_CLASSES.md` - Premium Pattern
2. **Run examples**:
   - `cmd/main.go` - Full example
   - Test files - Unit tests
3. **Understand patterns**:
   - Verticles
   - EventBus
   - Futures/Promises

### Phase 3: Small Migration (2-3 weeks)

1. **Start with simple service**:
   ```go
   type HelloService struct {
       *core.BaseService
   }
   
   func (s *HelloService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
       return s.Reply(msg, "Hello, World!")
   }
   ```

2. **Add one feature at a time**:
   - Add database component
   - Add error handling
   - Add logging

3. **Test thoroughly**:
   - Unit tests
   - Integration tests

### Phase 4: Full Migration (1-2 months)

1. **Migrate core services**
2. **Migrate handlers**
3. **Migrate components**
4. **Performance testing**
5. **Production deployment**

---

## Common Pitfalls

### 1. Forgetting Error Checks

**❌ Wrong:**
```go
result, err := doSomething()
// Forgot to check err!
useResult(result)
```

**✅ Correct:**
```go
result, err := doSomething()
if err != nil {
    return err
}
useResult(result)
```

### 2. Not Using Pointers for Receivers

**❌ Wrong:**
```go
func (v MyVerticle) Start(ctx FluxorContext) error {
    // Changes won't persist (copy)
}
```

**✅ Correct:**
```go
func (v *MyVerticle) Start(ctx FluxorContext) error {
    // Changes persist (pointer)
}
```

### 3. Ignoring Context

**❌ Wrong:**
```go
func doWork() {
    // No context, can't cancel
}
```

**✅ Correct:**
```go
func doWork(ctx context.Context) error {
    // Can be cancelled
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // Do work
    }
}
```

### 4. Not Using Premium Pattern

**❌ Wrong:**
```go
type MyVerticle struct{}

func (v *MyVerticle) Start(ctx FluxorContext) error {
    // Write all boilerplate yourself
    consumer := ctx.EventBus().Consumer("address")
    // ... lots of code
}
```

**✅ Correct:**
```go
type MyVerticle struct {
    *core.BaseVerticle
}

func (v *MyVerticle) doStart(ctx core.FluxorContext) error {
    // BaseVerticle handles boilerplate
    consumer := v.Consumer("address")
    // Just your logic
}
```

### 5. Panic Instead of Error

**❌ Wrong:**
```go
if err != nil {
    panic(err) // Don't do this!
}
```

**✅ Correct:**
```go
if err != nil {
    return err // Return error
}
```

---

## Resources

### Go Learning

- **Official Tutorial**: https://go.dev/tour/
- **Effective Go**: https://go.dev/doc/effective_go
- **Go by Example**: https://gobyexample.com/
- **Go Blog**: https://go.dev/blog/

### Fluxor Documentation

- **README.md**: Quick start
- **ARCHITECTURE.md**: Architecture details
- **BASE_CLASSES.md**: Premium Pattern guide
- **DATABASE_POOLING.md**: Connection pooling guide (HikariCP equivalent)
- **Examples**: `cmd/main.go`, test files

### Community

- **Go Forum**: https://forum.golangbridge.org/
- **Stack Overflow**: Tag `go` and `fluxor`
- **GitHub Issues**: Report bugs, ask questions

---

## Quick Reference

### Common Patterns

```go
// 1. Create Service (Premium Pattern)
type MyService struct {
    *core.BaseService
}

func NewMyService() *MyService {
    return &MyService{
        BaseService: core.NewBaseService("my-service", "my.service"),
    }
}

func (s *MyService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    // Handle request
    return s.Reply(msg, result)
}

// 2. Create Verticle (Premium Pattern)
type MyVerticle struct {
    *core.BaseVerticle
}

func (v *MyVerticle) doStart(ctx core.FluxorContext) error {
    consumer := v.Consumer("address")
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        return msg.Reply("processed")
    })
    return nil
}

// 3. Async/Await Pattern
promise := fluxor.NewPromiseT[string]()
go func() {
    promise.Complete("result")
}()
result, err := promise.Await(ctx)

// 4. Error Handling
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed: %w", err)
}
```

---

## Summary

**Key Takeaways:**

1. ✅ **Go is simpler** than Java/Node.js in many ways
2. ✅ **Fluxor provides familiar patterns** from Vert.x/Node.js
3. ✅ **Premium Pattern** makes migration easier
4. ✅ **Error handling** is explicit (better than exceptions)
5. ✅ **Type safety** with generics (Go 1.18+)
6. ✅ **Performance** is excellent (compiled language)

**Migration Path:**
1. Learn Go basics (1-2 weeks)
2. Understand Fluxor (1 week)
3. Small migration (2-3 weeks)
4. Full migration (1-2 months)

**Remember:**
- Go is different, not worse
- Fluxor bridges the gap
- Premium Pattern helps
- Practice makes perfect!

---

**Happy Migrating! 🚀**

