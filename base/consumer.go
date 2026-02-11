package base

import (
	"context"
	"log"
)

// Consumer defines the interface for message consumers
type Consumer[T Entity] interface {
	Read() <-chan T
	Close()
}

// GenericConsumer consumes messages from a Consumer and invokes a Service.Create
// Uses a background goroutine to process messages; Close stops consumption.
type GenericConsumer[T Entity] struct {
	Consumer Consumer[T]
	Service  Service[T]
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewGenericConsumer wires a consumer to a service and starts processing.
func NewGenericConsumer[T Entity](consumer Consumer[T], service Service[T]) *GenericConsumer[T] {
	ctx, cancel := context.WithCancel(context.Background())
	gc := &GenericConsumer[T]{
		Consumer: consumer,
		Service:  service,
		ctx:      ctx,
		cancel:   cancel,
	}
	go gc.start()
	return gc
}

// start reads messages and forwards them to the service.
func (gc *GenericConsumer[T]) start() {
	for {
		select {
		case <-gc.ctx.Done():
			return
		case msg, ok := <-gc.Consumer.Read():
			if !ok {
				return
			}
			req := &createReq[T]{Data: msg}
			if _, err := gc.Service.Create(context.Background(), req); err != nil {
				log.Printf("error processing message: %v", err)
				continue
			}
		}
	}
}

// Close stops consumption and closes the underlying consumer.
func (gc *GenericConsumer[T]) Close() {
	gc.cancel()
	gc.Consumer.Close()
}

// createReq is a lightweight adapter implementing CreateRequest for a plain entity.
type createReq[T Entity] struct{ Data T }

func (r *createReq[T]) GetData() T { return r.Data }