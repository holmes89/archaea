# Architecture

## Overview

`archaea` is a library, not a deployed service. It provides two packages of generic building blocks that downstream microservices import and embed. The architecture is layered:

```
┌──────────────────────────────────────────────────────┐
│                  Downstream Service                  │
│  ┌─────────────────┐      ┌────────────────────────┐ │
│  │  gRPC Handler   │      │   Kafka Consumer loop  │ │
│  │  (ConnectRPC)   │      │   (inbound events)     │ │
│  └────────┬────────┘      └───────────┬────────────┘ │
│           │                           │               │
│  ┌────────▼───────────────────────────▼────────────┐ │
│  │               Service[T]                        │ │
│  │          (GenericService[T])                    │ │
│  └────────────────────┬────────────────────────────┘ │
│                       │                               │
│  ┌────────────────────▼────────────────────────────┐ │
│  │              Repository[T]                      │ │
│  │       (implemented by downstream service)       │ │
│  └─────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────┘
         │ publish events          │ consume events
         ▼                         ▼
┌─────────────────────────────────────────────────────┐
│                      Apache Kafka                   │
│           (one topic per Protobuf message type)     │
└─────────────────────────────────────────────────────┘
```

## Components

### `base.GenericGRPCService[T]`

The top of the request-handling stack. Accepts ConnectRPC requests, logs each operation, delegates to `Service[T]`, and on successful Create or Update calls, fires an event to `Producer[T]`. A nil Publisher is a valid configuration (no events published). Publishing errors are non-fatal.

### `base.GenericService[T]`

The business logic layer. Holds a reference to `Repository[T]` and implements `Service[T]`:

- `List` — calls `Repository.List` with cursor and count from the request; returns `ListGenericResponse[T]` (cursor always empty in current implementation).
- `Get` — calls `Repository.Get` by UUID.
- `Create` — calls `Repository.Create` with the entity from the request.
- `Update` — calls `Repository.Update` by UUID with the entity from the request.
- `Delete` — calls `Repository.Delete` by UUID.

### `base.GenericConsumer[T]`

Bridges the messaging layer to the service layer. On construction it starts a background goroutine that selects on either a cancellation context or the consumer's output channel. Each received message is wrapped in a `createReq[T]` adapter and forwarded to `Service.Create`. Errors are logged and the loop continues. `Close()` cancels the goroutine and delegates to the underlying consumer's `Close()`.

### `kafka.Conn`

Singleton connection object created once at service startup. Holds a `kgo.Client` (shared by producers and consumers that are built from it) and a `kadm.Client` for admin operations. `NewConn` panics if the broker is unreachable or if the initial `ListTopics` call fails.

### `kafka.Producer[T]`

Takes a `*Conn` at construction time, derives the topic name from the Go type (`fmt.Sprintf("%T", t)` with the leading `*` stripped), creates the topic on the broker, and reuses the shared `kgo.Client` for all produce calls. Messages are serialised with `proto.Marshal`. The Kafka record key is the entity ID bytes. Delivery is asynchronous; errors are only surfaced in a callback log line.

### `kafka.Consumer[T]`

Creates its own independent `kgo.Client` (not shared with the producer). Topic name derivation follows the same convention as the producer. If no consumer group ID is supplied, a fresh UUID is generated, which means the consumer always starts from the beginning of the topic (`kgo.NewOffset().AtStart()`). Two goroutines are started: a poll loop (`read()`) that pushes raw `*kgo.Record` values onto an internal channel, and a conversion loop that runs the caller-supplied `convertor` function and pushes typed values onto the `out` channel.

## Data Flow

### Inbound (Kafka → Service → Repository)

```
Kafka broker → kafka.Consumer.read() → raw record channel
             → convertor goroutine   → typed entity channel
             → base.GenericConsumer  → Service.Create
                                     → Repository.Create
```

### Outbound (gRPC → Service → Repository + Kafka)

```
ConnectRPC client → GenericGRPCService.Create/Update
                  → Service.Create/Update
                  → Repository.Create/Update
                  → kafka.Producer.Publish (async)
                  → Kafka broker
```

## External Dependencies

| System | How Used |
|---|---|
| Apache Kafka | Topic-per-type event streaming; topics are auto-created by the producer at startup |
| Protobuf | Wire format for all Kafka messages; type system used for topic name derivation |
| ConnectRPC | Transport protocol for gRPC handlers |

## Deployment

`archaea` is a library and has no standalone deployment artifact. Downstream services that import it are responsible for:

1. Providing reachable Kafka broker addresses (passed as `[]string` to `kafka.NewConn` or `kafka.NewConsumer`).
2. Implementing `base.Repository[T]` against their chosen data store.
3. Registering ConnectRPC handlers with an HTTP server.
4. Managing the lifecycle of `kafka.Conn` (call `Close()` on shutdown).
5. Managing the lifecycle of `base.GenericConsumer` (call `Close()` on shutdown).

Topic creation is handled automatically by `kafka.Producer` and `kafka.Conn.CreateTopicIfNotExists`. Topics are created with 1 partition and replication factor 1.
