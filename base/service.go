package base

import "context"

// Service defines the generic service interface
type Service[T Entity] interface {
	List(ctx context.Context, req ListRequest) (ListResponse[T], error)
	Get(ctx context.Context, req GetRequest[T]) (GetResponse[T], error)
	Create(ctx context.Context, req CreateRequest[T]) (CreateResponse[T], error)
	Update(ctx context.Context, req UpdateRequest[T]) (UpdateResponse[T], error)
	Delete(ctx context.Context, id string) error
}

// GenericService provides a base implementation of common service operations
type GenericService[T Entity] struct {
	Repository Repository[T]
}

// NewGenericService creates a new generic service
func NewGenericService[T Entity](repo Repository[T]) *GenericService[T] {
	return &GenericService[T]{
		Repository: repo,
	}
}

// List implements the List operation using the repository
func (s *GenericService[T]) List(ctx context.Context, req ListRequest) (ListResponse[T], error) {
	data, err := s.Repository.List(ctx, req.GetCursor(), uint(req.GetCount()), nil)
	if err != nil {
		return nil, err
	}

	return &ListGenericResponse[T]{
		Cursor: "",
		Count:  int32(len(data)),
		Data:   data,
	}, nil
}

// Get implements the Get operation using the repository
func (s *GenericService[T]) Get(ctx context.Context, req GetRequest[T]) (GetResponse[T], error) {
	data, err := s.Repository.Get(ctx, req.GetUuid())
	if err != nil {
		var zero T
		return &GetGenericResponse[T]{Data: zero}, err
	}

	return &GetGenericResponse[T]{
		Data: data,
	}, nil
}

// Create implements the Create operation using the repository
func (s *GenericService[T]) Create(ctx context.Context, req CreateRequest[T]) (CreateResponse[T], error) {
	data := req.GetData()
	err := s.Repository.Create(ctx, data)
	if err != nil {
		var zero T
		return &CreateGenericResponse[T]{Data: zero}, err
	}

	return &CreateGenericResponse[T]{
		Data: data,
	}, nil
}

// Update implements the Update operation using the repository
func (s *GenericService[T]) Update(ctx context.Context, req UpdateRequest[T]) (UpdateResponse[T], error) {
	data := req.GetData()
	err := s.Repository.Update(ctx, req.GetUuid(), data)
	if err != nil {
		var zero T
		return &UpdateGenericResponse[T]{Data: zero}, err
	}

	return &UpdateGenericResponse[T]{
		Data: data,
	}, nil
}
