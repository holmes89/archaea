# archaea

`archaea` (`github.com/holmes89/archaea`) is a Go library of shared base abstractions for building event-driven microservices. It provides generic, reusable implementations of the most common service patterns — CRUD operations over a repository, gRPC/ConnectRPC handler scaffolding, and Kafka-backed messaging — so that downstream services only need to implement domain-specific logic.

## Packages

### `base` — Core interfaces and generic implementations

- **`Entity`** — interface every domain object must satisfy (`GetUuid() string`).
- **`Repository[T]`** — generic data-access interface (Create, Update, Delete, Get, List).
- **Request/Response interfaces** — typed contracts for every CRUD operation.
- **`GenericService[T]`** — concrete `Service[T]` that delegates to a `Repository[T]`. Embedding this eliminates List/Get/Create/Update boilerplate.
- **`GenericGRPCService[T]`** — ConnectRPC handler that wraps a `Service[T]`, adds structured logging, and optionally publishes events via a `Producer[T]` on Create and Update. Publishing failures are logged but do not fail the RPC.
- **`Consumer[T]` / `GenericConsumer[T]`** — bridges a message consumer to a `Service[T]`. Runs a background goroutine that reads from the consumer channel and calls `Service.Create` for each message.
- **`Producer[T]`** — single-method abstraction (`Publish`) over any message transport.

### `kafka` — Kafka-specific implementations

Built on `github.com/twmb/franz-go`:

- **`Conn`** — establishes a Kafka connection at startup (panics on failure). Wraps both a `kgo.Client` and a `kadm.Client`. Exposes `TopicExists`, `CreateTopicIfNotExists`, and `CreateTopic`.
- **`Consumer[T]`** — generic Kafka consumer. Topic name is derived from the Go type name. Accepts an optional consumer group ID (random UUID if not provided). Implements `base.Consumer[T]`.
- **`Producer[T]`** — generic Kafka producer. Topic derived from Go type name, created automatically at construction. Serialises with `proto.Marshal` and produces asynchronously.

## Usage

```go
// 1. Implement base.Entity on your Protobuf-generated type (GetUuid already generated).

// 2. Implement base.Repository[*YourEntity] against your database.

// 3. Service layer — no List/Get/Create/Update code needed:
type YourService struct {
    *base.GenericService[*YourEntity]
}
func NewYourService(repo base.Repository[*YourEntity]) *YourService {
    return &YourService{GenericService: base.NewGenericService(repo)}
}

// 4. RPC handler — logging and event publishing are handled generically:
type YourGRPCService struct {
    *base.GenericGRPCService[*YourEntity]
}
func NewYourGRPCService(svc *YourService, pub base.Producer[*YourEntity]) *YourGRPCService {
    return &YourGRPCService{GenericGRPCService: base.NewGenericGRPCService(svc, pub)}
}

// 5. Wire a Kafka consumer to persist inbound events automatically:
consumer := kafka.NewConsumer[*YourEntity](brokers, nil, unmarshalFunc)
base.NewGenericConsumer[*YourEntity](consumer, svc)
```

## Dependencies

| Dependency | Purpose |
|---|---|
| `connectrpc.com/connect` | ConnectRPC framework used in `GenericGRPCService` |
| `github.com/twmb/franz-go` | Kafka client (produce/consume) |
| `github.com/twmb/franz-go/pkg/kadm` | Kafka admin client (topic management) |
| `google.golang.org/protobuf` | Protobuf serialisation/deserialisation in Kafka layer |
| `github.com/google/uuid` | Random consumer group ID generation |

## Known Limitations

- `kafka.Producer.Publish` signature does not match the `base.Producer[T]` interface — callers need a thin adapter.
- Kafka produce calls are fire-and-forget; callers should not rely on synchronous delivery guarantees.
- `Conn.CreateTopic` and `NewConn` panic on failure rather than returning errors.
- `GenericService.List` always returns an empty cursor string — cursor-based pagination is not yet functional.
