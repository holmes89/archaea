# Data Model

`archaea` is a generic library and does not define any concrete domain entities. Instead, it defines the structural contracts (interfaces and generic types) that all domain entities and their request/response envelopes must satisfy.

## Core Entity Contract

### `base.Entity` (interface)

Every domain object used with this library must implement:

```go
type Entity interface {
    GetUuid() string
}
```

`GetUuid` returns the unique identifier for the entity. In practice this is always a UUID string. Protobuf-generated types satisfy this interface automatically when the `.proto` definition includes a `string uuid` field.

## Repository Layer

### `base.Repository[T Entity]` (interface)

The persistence contract for any entity type `T`:

| Method | Parameters | Returns | Description |
|---|---|---|---|
| `Create` | `ctx`, `entity T` | `error` | Persist a new entity |
| `Update` | `ctx`, `id string`, `entity T` | `error` | Replace entity by UUID |
| `Delete` | `ctx`, `id string` | `error` | Remove entity by UUID |
| `Get` | `ctx`, `id string` | `(T, error)` | Fetch single entity by UUID |
| `List` | `ctx`, `cursor string`, `limit uint`, `filter map[string][]any` | `([]T, error)` | Fetch a page of entities |

`filter` is a multi-valued map allowing callers to pass arbitrary key/value filter criteria; the semantics are defined by the implementing repository.

## Request Interfaces

### `base.ListRequest`

```go
type ListRequest interface {
    GetCursor() string  // opaque pagination cursor (empty = first page)
    GetCount()  int32   // maximum number of results to return
}
```

### `base.GetRequest[T Entity]`

```go
type GetRequest[T Entity] interface {
    GetUuid() string  // UUID of the entity to fetch
}
```

### `base.CreateRequest[T Entity]`

```go
type CreateRequest[T Entity] interface {
    GetData() T  // the entity to persist
}
```

### `base.UpdateRequest[T Entity]`

```go
type UpdateRequest[T Entity] interface {
    GetUuid() string  // UUID of the entity to update
    GetData() T       // replacement entity data
}
```

### `base.DeleteRequest`

```go
type DeleteRequest interface {
    GetUuid() string  // UUID of the entity to delete
}
```

## Response Interfaces

### `base.ListResponse[T Entity]`

```go
type ListResponse[T Entity] interface {
    GetData()   []T    // slice of matching entities
    GetCursor() string // cursor for the next page (empty = no more pages)
    GetCount()  int32  // number of entities returned
}
```

### `base.GetResponse[T Entity]`

```go
type GetResponse[T Entity] interface {
    GetData() T  // the fetched entity
}
```

### `base.CreateResponse[T Entity]`

```go
type CreateResponse[T Entity] interface {
    GetData() T  // the entity as persisted (may include server-set fields)
}
```

### `base.UpdateResponse[T Entity]`

```go
type UpdateResponse[T Entity] interface {
    GetData() T  // the entity after update
}
```

### `base.DeleteResponse`

```go
type DeleteResponse interface {
    GetSuccess() bool  // true if deletion succeeded
}
```

## Concrete Generic Response Structs

| Struct | Fields | Implements |
|---|---|---|
| `ListGenericResponse[T]` | `Data []T`, `Cursor string`, `Count int32` | `ListResponse[T]` |
| `GetGenericResponse[T]` | `Data T` | `GetResponse[T]` |
| `CreateGenericResponse[T]` | `Data T` | `CreateResponse[T]` |
| `UpdateGenericResponse[T]` | `Data T` | `UpdateResponse[T]` |
| `DeleteGenericResponse` | `Success bool` | `DeleteResponse` |

## Messaging Layer

### `base.Consumer[T Entity]` (interface)

```go
type Consumer[T Entity] interface {
    Read()  <-chan T  // receive channel of decoded entities
    Close()           // stop consuming and release resources
}
```

### `base.Producer[T Entity]` (interface)

```go
type Producer[T Entity] interface {
    Publish(ctx context.Context, entity T) error
    Close() error
}
```

### Internal: `base.createReq[T]`

A private adapter struct used inside `GenericConsumer` to wrap a plain entity value in a `CreateRequest`:

```go
type createReq[T Entity] struct{ Data T }
func (r *createReq[T]) GetData() T { return r.Data }
```

## Kafka Wire Format

### Topic naming

Both `kafka.Producer[T]` and `kafka.Consumer[T]` derive the Kafka topic name from the Go type of the generic parameter at construction time:

```
topic = strings.Replace(fmt.Sprintf("%T", zeroValue), "*", "", 1)
```

For example, a `*mypackage.BookEvent` yields the topic name `mypackage.BookEvent`.

### Message encoding

All Kafka messages are encoded with `proto.Marshal` (binary Protobuf). Consumers must provide a `convertor func([]byte) (T, error)` that calls the appropriate `proto.Unmarshal`.

### Kafka record structure

| Field | Value |
|---|---|
| Topic | Derived from Go type name |
| Key | Entity UUID as raw bytes (`[]byte(id)`) |
| Value | `proto.Marshal(entity)` |

### Topic configuration

Topics are created with 1 partition and replication factor 1 via `kadm.Client.CreateTopics`.
