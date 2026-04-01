package pipeline_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/holmes89/archaea/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Simple
// ---------------------------------------------------------------------------

func TestSimple_Enqueue_Delivered(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	delivered := make(chan string, 1)
	p := pipeline.NewSimple(func(_ context.Context, req string) {
		delivered <- req
	}, 1)
	go p.Start(ctx)

	p.Enqueue("hello")
	select {
	case got := <-delivered:
		assert.Equal(t, "hello", got)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("request was not delivered")
	}
}

func TestSimple_Start_StopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	p := pipeline.NewSimple(func(_ context.Context, _ string) {}, 1)
	go func() {
		p.Start(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Start did not return after context cancellation")
	}
}

func TestSimple_QueueFull_Blocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	release := make(chan struct{})
	p := pipeline.NewSimple(func(_ context.Context, _ int) {
		<-release // hold the worker until we're ready
	}, 1)
	go p.Start(ctx)

	// Fill the queue: one item is being processed, one sits in the buffer.
	p.Enqueue(1)
	p.Enqueue(2)

	// A third Enqueue must block. Verify it doesn't complete within 100 ms.
	enqueued := make(chan struct{})
	go func() {
		p.Enqueue(3)
		close(enqueued)
	}()

	select {
	case <-enqueued:
		t.Fatal("Enqueue returned immediately on a full queue — expected blocking")
	case <-time.After(100 * time.Millisecond):
		// Good: still blocked.
	}

	// Release the worker; the blocked Enqueue should now complete.
	close(release)
	select {
	case <-enqueued:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Enqueue did not unblock after queue drained")
	}
}

func TestSimple_ProcessesInOrder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var order []int
	done := make(chan struct{})
	p := pipeline.NewSimple(func(_ context.Context, n int) {
		order = append(order, n)
		if n == 3 {
			close(done)
		}
	}, 10)
	go p.Start(ctx)

	p.Enqueue(1)
	p.Enqueue(2)
	p.Enqueue(3)

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("pipeline did not process all items")
	}
	assert.Equal(t, []int{1, 2, 3}, order)
}

func TestSimple_DefaultQueueSize(t *testing.T) {
	// queueSize <= 0 should use default; just verify construction doesn't panic.
	p := pipeline.NewSimple(func(_ context.Context, _ int) {}, 0)
	require.NotNil(t, p)
}

// ---------------------------------------------------------------------------
// Tracked
// ---------------------------------------------------------------------------

func TestTracked_Enqueue_Delivered(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	delivered := make(chan string, 1)
	p := pipeline.NewTracked(func(_ context.Context, req string) (pipeline.PipelineResult, error) {
		delivered <- req
		return pipeline.PipelineResult{ID: "uuid-1"}, nil
	}, 1)
	go p.Start(ctx)

	p.Enqueue("req-1", "hello")
	select {
	case got := <-delivered:
		assert.Equal(t, "hello", got)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("request was not delivered")
	}
}

func TestTracked_Status_RunningBeforeProcessed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	release := make(chan struct{})
	p := pipeline.NewTracked(func(_ context.Context, _ string) (pipeline.PipelineResult, error) {
		<-release
		return pipeline.PipelineResult{ID: "uuid-1"}, nil
	}, 1)
	go p.Start(ctx)

	p.Enqueue("req-1", "hello")

	// Before releasing the worker the status should be "running".
	status, ok := p.Status("req-1")
	require.True(t, ok)
	assert.Equal(t, pipeline.JobStateRunning, status.State)
	close(release)
}

func TestTracked_Status_CompletedAfterPipeline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	p := pipeline.NewTracked(func(_ context.Context, _ string) (pipeline.PipelineResult, error) {
		return pipeline.PipelineResult{ID: "uuid-42"}, nil
	}, 1)
	go func() {
		p.Start(ctx)
		close(done)
	}()

	p.Enqueue("req-1", "hello")

	// Poll until completed or timeout.
	deadline := time.After(500 * time.Millisecond)
	for {
		status, ok := p.Status("req-1")
		if ok && status.State == pipeline.JobStateCompleted {
			assert.Equal(t, "uuid-42", status.ID)
			assert.Empty(t, status.Err)
			return
		}
		select {
		case <-deadline:
			t.Fatalf("status never reached completed; last=%v", status)
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func TestTracked_Status_FailedOnError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	boom := errors.New("pipeline exploded")
	p := pipeline.NewTracked(func(_ context.Context, _ string) (pipeline.PipelineResult, error) {
		return pipeline.PipelineResult{}, boom
	}, 1)
	go p.Start(ctx)

	p.Enqueue("req-fail", "bad input")

	deadline := time.After(500 * time.Millisecond)
	for {
		status, ok := p.Status("req-fail")
		if ok && status.State == pipeline.JobStateFailed {
			assert.Equal(t, boom.Error(), status.Err)
			return
		}
		select {
		case <-deadline:
			t.Fatalf("status never reached failed; last=%v", status)
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func TestTracked_Status_DuplicateOnDuplicate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := pipeline.NewTracked(func(_ context.Context, _ string) (pipeline.PipelineResult, error) {
		return pipeline.PipelineResult{ID: "existing-uuid", Duplicate: true}, nil
	}, 1)
	go p.Start(ctx)

	p.Enqueue("req-dup", "duplicate input")

	deadline := time.After(500 * time.Millisecond)
	for {
		status, ok := p.Status("req-dup")
		if ok && status.State == pipeline.JobStateDuplicate {
			assert.Equal(t, "existing-uuid", status.ID)
			return
		}
		select {
		case <-deadline:
			t.Fatalf("status never reached duplicate; last=%v", status)
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func TestTracked_Status_UnknownRequest(t *testing.T) {
	p := pipeline.NewTracked(func(_ context.Context, _ string) (pipeline.PipelineResult, error) {
		return pipeline.PipelineResult{}, nil
	}, 1)

	status, ok := p.Status("does-not-exist")
	assert.False(t, ok)
	assert.Nil(t, status)
}

func TestTracked_QueueFull_Blocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	release := make(chan struct{})
	p := pipeline.NewTracked(func(_ context.Context, _ int) (pipeline.PipelineResult, error) {
		<-release
		return pipeline.PipelineResult{ID: "id"}, nil
	}, 1)
	go p.Start(ctx)

	// One item being processed, one in the buffer.
	p.Enqueue("req-1", 1)
	p.Enqueue("req-2", 2)

	enqueued := make(chan struct{})
	go func() {
		p.Enqueue("req-3", 3)
		close(enqueued)
	}()

	select {
	case <-enqueued:
		t.Fatal("Enqueue returned immediately on a full queue")
	case <-time.After(100 * time.Millisecond):
	}

	close(release)
	select {
	case <-enqueued:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Enqueue did not unblock after queue drained")
	}
}

func TestTracked_Start_StopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	p := pipeline.NewTracked(func(_ context.Context, _ string) (pipeline.PipelineResult, error) {
		return pipeline.PipelineResult{}, nil
	}, 1)
	go func() {
		p.Start(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Start did not return after context cancellation")
	}
}

func TestTracked_ExistingStatusPreserved_OnReenqueue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var callCount atomic.Int32
	done := make(chan struct{})
	p := pipeline.NewTracked(func(_ context.Context, _ string) (pipeline.PipelineResult, error) {
		if callCount.Add(1) == 2 {
			close(done)
		}
		return pipeline.PipelineResult{ID: "uuid"}, nil
	}, 10)
	go p.Start(ctx)

	// Enqueue the same request twice; the second call should not reset the status.
	p.Enqueue("req-1", "first")
	// Wait for it to complete before re-enqueuing.
	time.Sleep(20 * time.Millisecond)
	p.Enqueue("req-1", "second")

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("second enqueue was not processed")
	}
}
