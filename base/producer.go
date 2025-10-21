package base

import (
	"context"
	"time"
)

type Producer[T any] interface {
	Publish(context.Context, T, string, time.Time) error
	Close() error
}
