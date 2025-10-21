package base

import "context"

type Entity interface {
	GetUuid() string
}

type Repository[T Entity] interface {
	Create(context.Context, T) error
	Update(context.Context, string, T) error
	Delete(context.Context, string) error
	Get(context.Context, string) (T, error)
	List(context.Context, string, uint, map[string][]any) ([]T, error)
}
