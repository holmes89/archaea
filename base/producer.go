package base

import "context"

// Producer defines the interface for message producers
type Producer[T Entity] interface {
	Publish(ctx context.Context, entity T) error
	Close() error
}
