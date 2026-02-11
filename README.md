# Archaea - Common Base Abstractions

This directory contains common patterns and base implementations that should be moved to the `github.com/holmes89/archaea` repository.

## Structure

### `base/` - Core Interfaces and Base Implementations

#### `model.go`
- **Entity** - Interface for domain entities
- **Repository[T]** - Generic repository interface with CRUD operations
- **Request/Response Interfaces** - ListRequest, GetRequest, CreateRequest, etc.
- **Generic Response Types** - Concrete implementations of response interfaces

**Why move to archaea:**
- Used by every generated service
- Provides type-safe generic operations
- Reduces boilerplate in generated code

#### `service.go`
- **Service[T]** - Generic service interface
- **GenericService[T]** - Base service implementation
- Implements common List/Get/Create patterns using repositories

**Why move to archaea:**
- Eliminates duplicate service logic across projects
- Provides consistent behavior
- Easy to test and maintain

#### `grpc.go`
- **GenericGRPCService[T]** - Base gRPC/Connect RPC service
- Wraps business logic services
- Handles logging and event publishing
- Implements List/Get/Create/Update/Delete operations

**Why move to archaea:**
- Every gRPC service uses this pattern
- Consistent error handling and logging
- Built-in event publishing support

#### `consumer.go`
- **Consumer[T]** - Interface for message consumers
- **GenericConsumer[T]** - Base consumer implementation
- Processes messages through services

**Why move to archaea:**
- Standard pattern for Kafka consumers
- Decouples message handling from business logic

#### `producer.go`
- **Producer[T]** - Interface for message producers
- Simple event publishing abstraction

**Why move to archaea:**
- Used by all services that publish events
- Clean abstraction over messaging infrastructure

### `kafka/` - Kafka-Specific Implementations

#### `conn.go`
- **Conn** - Kafka connection wrapper
- Topic management
- Reader/Writer factory methods

**Why move to archaea:**
- Every project using Kafka needs this
- Handles connection pooling and configuration
- Topic creation logic

#### `producer.go`
- **Producer[T]** - Kafka producer implementation
- Protobuf serialization
- Error handling

**Why move to archaea:**
- Concrete implementation of Producer interface
- Handles proto marshaling automatically
- Standard pattern across all projects

#### `consumer.go`
- **Consumer[T]** - Kafka consumer implementation
- Protobuf deserialization
- Message processing loop
- Auto-topic detection

**Why move to archaea:**
- Concrete implementation of Consumer interface
- Handles proto unmarshaling automatically
- Background processing with proper cleanup

## Usage in Generated Code

### Before (Current Template):
```go
type BookService struct {
    repo base.Repository[*Book]
}

func (s *BookService) List(ctx context.Context, req base.ListRequest) (base.ListResponse[*Book], error) {
    data, err := s.repo.List(ctx, req.GetCursor(), uint(req.GetCount()), nil)
    return &base.ListGenericResponse[*Book]{
        Cursor: "",
        Count:  10,
        Data:   data,
    }, err
}
```

### After (With Archaea):
```go
type BookService struct {
    *base.GenericService[*Book]
}

func NewBookService(repo base.Repository[*Book]) *BookService {
    return &BookService{
        GenericService: base.NewGenericService(repo),
    }
}
// List, Get, Create methods inherited!
```

### gRPC Service Before:
```go
type BookGRPCService struct {
    service BookService
    publisher base.Producer[*Book]
}

func (s *BookGRPCService) ListBooks(ctx context.Context, req *connect.Request[...]) (*connect.Response[...], error) {
    // 30+ lines of boilerplate logging, error handling, response wrapping
}
```

### gRPC Service After:
```go
type BookGRPCService struct {
    *base.GenericGRPCService[*Book]
}

func NewBookGRPCService(svc BookService, pub base.Producer[*Book]) *BookGRPCService {
    return &BookGRPCService{
        GenericGRPCService: base.NewGenericGRPCService(svc, pub),
    }
}

func (s *BookGRPCService) ListBooks(ctx context.Context, req *connect.Request[servicev1.ListBooksRequest]) (*connect.Response[servicev1.ListBooksResponse], error) {
    data, err := s.List(ctx, req, req.Msg) // Delegates to generic implementation
    if err != nil {
        return nil, err
    }
    return connect.NewResponse(&servicev1.ListBooksResponse{
        Data:   data.GetData(),
        Cursor: data.GetCursor(),
        Count:  data.GetCount(),
    }), nil
}
```

## Benefits

1. **Less Code Generation** - Generate only entity-specific logic, not boilerplate
2. **Easier Updates** - Fix bugs once in archaea, all services benefit
3. **Consistent Behavior** - All services handle errors, logging, events the same way
4. **Type Safety** - Generics ensure compile-time correctness
5. **Testability** - Test generic implementations thoroughly once
6. **Documentation** - Central place for patterns and best practices

## Migration Path

1. Copy these files to `github.com/holmes89/archaea`
2. Update beaver templates to use archaea imports
3. Regenerate existing projects to benefit from reduced boilerplate
4. Add tests to archaea for generic implementations
5. Version archaea releases for stability

## Dependencies

- `connectrpc.com/connect` - Connect RPC framework
- `google.golang.org/protobuf` - Protocol Buffers
- `github.com/segmentio/kafka-go` - Kafka client

## Next Steps

1. Move to archaea repository
2. Add comprehensive tests
3. Add examples and documentation
4. Create versioned releases
5. Update beaver templates to use archaea
6. Add more generic patterns as they emerge (validation, caching, etc.)
