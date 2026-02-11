package base

import "context"

// Entity represents a domain entity with basic CRUD operations
type Entity interface {
	GetUuid() string
}

// Repository defines the generic repository interface for data access
type Repository[T Entity] interface {
	Create(ctx context.Context, entity T) error
	Update(ctx context.Context, id string, entity T) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (T, error)
	List(ctx context.Context, cursor string, limit uint, filter map[string][]any) ([]T, error)
}

// ListRequest defines the interface for list operations
type ListRequest interface {
	GetCursor() string
	GetCount() int32
}

// ListResponse defines the interface for list operation responses
type ListResponse[T Entity] interface {
	GetData() []T
	GetCursor() string
	GetCount() int32
}

// GetRequest defines the interface for get operations
type GetRequest[T Entity] interface {
	GetUuid() string
}

// GetResponse defines the interface for get operation responses
type GetResponse[T Entity] interface {
	GetData() T
}

// CreateRequest defines the interface for create operations
type CreateRequest[T Entity] interface {
	GetData() T
}

// CreateResponse defines the interface for create operation responses
type CreateResponse[T Entity] interface {
	GetData() T
}

// UpdateRequest defines the interface for update operations
type UpdateRequest[T Entity] interface {
	GetUuid() string
	GetData() T
}

// UpdateResponse defines the interface for update operation responses
type UpdateResponse[T Entity] interface {
	GetData() T
}

// DeleteRequest defines the interface for delete operations
type DeleteRequest interface {
	GetUuid() string
}

// DeleteResponse defines the interface for delete operation responses
type DeleteResponse interface {
	GetSuccess() bool
}

// Generic response implementations

// ListGenericResponse provides a generic implementation of ListResponse
type ListGenericResponse[T Entity] struct {
	Data   []T
	Cursor string
	Count  int32
}

func (r *ListGenericResponse[T]) GetData() []T      { return r.Data }
func (r *ListGenericResponse[T]) GetCursor() string { return r.Cursor }
func (r *ListGenericResponse[T]) GetCount() int32   { return r.Count }

// GetGenericResponse provides a generic implementation of GetResponse
type GetGenericResponse[T Entity] struct {
	Data T
}

func (r *GetGenericResponse[T]) GetData() T { return r.Data }

// CreateGenericResponse provides a generic implementation of CreateResponse
type CreateGenericResponse[T Entity] struct {
	Data T
}

func (r *CreateGenericResponse[T]) GetData() T { return r.Data }

// UpdateGenericResponse provides a generic implementation of UpdateResponse
type UpdateGenericResponse[T Entity] struct {
	Data T
}

func (r *UpdateGenericResponse[T]) GetData() T { return r.Data }

// DeleteGenericResponse provides a generic implementation of DeleteResponse
type DeleteGenericResponse struct {
	Success bool
}

func (r *DeleteGenericResponse) GetSuccess() bool { return r.Success }
