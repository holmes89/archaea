package pipeline

import (
	"context"
	"sync"
)

// JobState represents the lifecycle state of a tracked pipeline request.
type JobState string

const (
	JobStateRunning   JobState = "running"
	JobStateCompleted JobState = "completed"
	JobStateFailed    JobState = "failed"
	JobStateDuplicate JobState = "duplicate"
)

// JobStatus holds the current state of a tracked pipeline request.
type JobStatus struct {
	State JobState
	ID    string // resource ID set on completion (e.g. UUID of created entity)
	Err   string // non-empty when State == JobStateFailed
}

// PipelineResult is returned by the function passed to NewTracked.
type PipelineResult struct {
	ID        string // resource ID created or found
	Duplicate bool   // true if the resource already existed
}

type trackedJob[T any] struct {
	requestID string
	req       T
}

// Tracked is an async pipeline that records the status of each request so
// callers can poll for completion via Status. Enqueue blocks on a full queue.
type Tracked[T any] struct {
	fn    func(ctx context.Context, req T) (PipelineResult, error)
	queue chan trackedJob[T]
	mu    sync.RWMutex
	jobs  map[string]*JobStatus
}

// NewTracked returns a Tracked pipeline backed by fn. If queueSize <= 0,
// a default of 100 is used.
func NewTracked[T any](fn func(ctx context.Context, req T) (PipelineResult, error), queueSize int) *Tracked[T] {
	if queueSize <= 0 {
		queueSize = defaultQueueSize
	}
	return &Tracked[T]{
		fn:    fn,
		queue: make(chan trackedJob[T], queueSize),
		jobs:  make(map[string]*JobStatus),
	}
}

// Enqueue marks requestID as running (if not already known) and adds req to
// the queue. Blocks until space is available.
func (t *Tracked[T]) Enqueue(requestID string, req T) {
	t.mu.Lock()
	if _, exists := t.jobs[requestID]; !exists {
		t.jobs[requestID] = &JobStatus{State: JobStateRunning}
	}
	t.mu.Unlock()

	t.queue <- trackedJob[T]{requestID: requestID, req: req}
}

// Status returns the current status of a tracked request.
// Returns (nil, false) if requestID is not known.
func (t *Tracked[T]) Status(requestID string) (*JobStatus, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	j, ok := t.jobs[requestID]
	return j, ok
}

// Start drains the queue and processes each request until ctx is cancelled.
// Intended to be called as a goroutine:
//
//	go p.Start(ctx)
func (t *Tracked[T]) Start(ctx context.Context) {
	for {
		select {
		case job := <-t.queue:
			result, err := t.fn(ctx, job.req)
			t.mu.Lock()
			switch {
			case err != nil:
				t.jobs[job.requestID] = &JobStatus{State: JobStateFailed, Err: err.Error()}
			case result.Duplicate:
				t.jobs[job.requestID] = &JobStatus{State: JobStateDuplicate, ID: result.ID}
			default:
				t.jobs[job.requestID] = &JobStatus{State: JobStateCompleted, ID: result.ID}
			}
			t.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}
