package worker_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/holmes89/archaea/worker"
	"github.com/stretchr/testify/assert"
)

type mockClosable struct {
	closed atomic.Bool
}

func (m *mockClosable) Close() { m.closed.Store(true) }

func TestRun_ClosesAllClosablesOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	mc1, mc2 := &mockClosable{}, &mockClosable{}

	done := make(chan struct{})
	go func() {
		worker.Run(ctx, cancel, mc1, mc2)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Run did not return after context cancellation")
	}

	assert.True(t, mc1.closed.Load(), "first closable should be closed")
	assert.True(t, mc2.closed.Load(), "second closable should be closed")
}

func TestRun_CancelsContextOnReturn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		worker.Run(ctx, cancel)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Run did not return")
	}

	assert.Equal(t, context.Canceled, ctx.Err(), "context should be cancelled after Run returns")
}

func TestRun_NoClosables_DoesNotPanic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Run panicked: %v", r)
			}
			close(done)
		}()
		worker.Run(ctx, cancel)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Run did not return")
	}
}

func TestRun_ClosesInOrder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var order []int

	type orderedClosable struct {
		n int
	}
	// Use a slice of Closable values.
	closables := make([]worker.Closable, 3)
	for i := 0; i < 3; i++ {
		i := i
		closables[i] = closableFunc(func() { order = append(order, i) })
	}

	done := make(chan struct{})
	go func() {
		worker.Run(ctx, cancel, closables...)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Run did not return")
	}

	assert.Equal(t, []int{0, 1, 2}, order, "closables should be closed in registration order")
}

// closableFunc adapts a plain function to the Closable interface.
type closableFunc func()

func (f closableFunc) Close() { f() }
