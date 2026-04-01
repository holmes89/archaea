// Package worker provides a graceful shutdown runner for background workers.
// It blocks until SIGINT, SIGTERM, or the context is cancelled, then closes
// all registered closables and calls the cancel function.
package worker

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// Closable is any resource that can be shut down (e.g. a Kafka consumer).
type Closable interface {
	Close()
}

// Run blocks until SIGINT, SIGTERM, or ctx is cancelled, then:
//  1. Closes every closable in order.
//  2. Calls cancel to stop any goroutines started with the same ctx.
//
// Typical usage in cmd/worker/main.go:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//	go enricher.Start(ctx)
//	consumer := kafka.NewConsumer(...)
//	worker.Run(ctx, cancel, consumer)
func Run(ctx context.Context, cancel context.CancelFunc, closables ...Closable) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigs)

	select {
	case s := <-sigs:
		log.Printf("worker: signal received: %s", s)
	case <-ctx.Done():
		log.Printf("worker: context cancelled")
	}

	for _, c := range closables {
		c.Close()
	}
	cancel()
}
