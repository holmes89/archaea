// Package pipeline provides generic async pipeline processors backed by
// buffered channels. Enqueue always blocks on a full queue to provide
// backpressure rather than silently dropping messages.
package pipeline

import "context"

const defaultQueueSize = 100

// Simple is a fire-and-forget async pipeline. Each enqueued request is
// processed in order by fn. No status tracking is performed. Enqueue
// blocks when the queue is full.
type Simple[T any] struct {
	fn    func(ctx context.Context, req T)
	queue chan T
}

// NewSimple returns a Simple pipeline backed by fn. If queueSize <= 0,
// a default of 100 is used.
func NewSimple[T any](fn func(ctx context.Context, req T), queueSize int) *Simple[T] {
	if queueSize <= 0 {
		queueSize = defaultQueueSize
	}
	return &Simple[T]{
		fn:    fn,
		queue: make(chan T, queueSize),
	}
}

// Enqueue adds req to the processing queue. Blocks until space is
// available, providing natural backpressure to the caller.
func (s *Simple[T]) Enqueue(req T) {
	s.queue <- req
}

// Start drains the queue and calls fn for each request until ctx is
// cancelled. Intended to be called as a goroutine:
//
//	go p.Start(ctx)
func (s *Simple[T]) Start(ctx context.Context) {
	for {
		select {
		case req := <-s.queue:
			s.fn(ctx, req)
		case <-ctx.Done():
			return
		}
	}
}
