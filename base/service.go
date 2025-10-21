package base

import (
	"context"
)

type GetRequest[T Entity] interface {
	GetUuid() string
}

type GetResponse[T Entity] interface {
	GetData() T
}

type CreateRequest[T Entity] interface {
	GetData() T
}

type CreateResponse[T Entity] interface {
	GetData() T
}

type ListRequest interface {
	GetCount() int32
	GetCursor() string
}

type ListResponse[T Entity] interface {
	GetData() []T
	GetCount() int32
	GetCursor() string
}

type Service[T Entity] interface {
	List(ctx context.Context, req ListRequest) (ListResponse[T], error)
	Get(ctx context.Context, req GetRequest[T]) (GetResponse[T], error)
	Create(ctx context.Context, req CreateRequest[T]) (CreateResponse[T], error)
}

func NewService[T Entity](repo Repository[T]) Service[T] {
	return &service[T]{
		repo: repo,
	}
}

type service[T Entity] struct {
	repo Repository[T]
}

type ListGenericResponse[T Entity] struct {
	Data   []T
	Cursor string
	Count  int32
}

func (r *ListGenericResponse[T]) GetData() []T {
	return r.Data
}

func (r *ListGenericResponse[T]) GetCount() int32 {
	return r.Count
}

func (r *ListGenericResponse[T]) GetCursor() string {
	return r.Cursor
}

type GetGenericResponse[T Entity] struct {
	Data T
}

func (r *GetGenericResponse[T]) GetData() T {
	return r.Data
}

type CreateGenericResponse[T Entity] struct {
	Data T
}

func (r *CreateGenericResponse[T]) GetData() T {
	return r.Data
}

func (s *service[T]) List(ctx context.Context, req ListRequest) (ListResponse[T], error) {
	data, err := s.repo.List(ctx, req.GetCursor(), uint(req.GetCount()), nil)
	return &ListGenericResponse[T]{
		Data:   data,
		Cursor: "",
		Count:  int32(0),
	}, err
}
func (s *service[T]) Get(ctx context.Context, req GetRequest[T]) (GetResponse[T], error) {
	data, err := s.repo.Get(ctx, req.GetUuid())
	return &GetGenericResponse[T]{
		Data: data,
	}, err
}

func (s *service[T]) Create(ctx context.Context, req CreateRequest[T]) (CreateResponse[T], error) {
	err := s.repo.Create(ctx, req.GetData())
	if err != nil {
		return nil, err
	}
	return &GetGenericResponse[T]{
		Data: req.GetData(),
	}, err
}
