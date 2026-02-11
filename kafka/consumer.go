package kafka

import (
	"context"
	"fmt"
	"log"
	"reflect"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

// Consumer implements a simple Kafka consumer.
type Consumer[T proto.Message] struct {
	reader    *kafka.Reader
	messages  chan T
	unmarshal func([]byte) (T, error)
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewConsumer creates a new Kafka consumer for a topic inferred from message type.
func NewConsumer[T proto.Message](brokers []string, groupID *string, unmarshal func([]byte) (T, error)) *Consumer[T] {
	group := "default-group"
	if groupID != nil {
		group = *groupID
	}

	var msg T
	topic := fmt.Sprintf("%T", msg)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  group,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	ctx, cancel := context.WithCancel(context.Background())
	c := &Consumer[T]{
		reader:    reader,
		messages:  make(chan T, 100),
		unmarshal: unmarshal,
		ctx:       ctx,
		cancel:    cancel,
	}
	go c.start()
	return c
}

// Read returns a channel for reading messages.
func (c *Consumer[T]) Read() <-chan T {
	return c.messages
}

// Close stops the consumer and closes the reader.
func (c *Consumer[T]) Close() {
	c.cancel()
	if err := c.reader.Close(); err != nil {
		log.Printf("error closing Kafka reader: %v", err)
	}
}

func (c *Consumer[T]) start() {
	defer close(c.messages)
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			msg, err := c.reader.ReadMessage(c.ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				log.Printf("error reading message: %v", err)
				continue
			}

			entity, err := c.unmarshal(msg.Value)
			if err != nil {
				log.Printf("error unmarshaling message: %v", err)
				continue
			}
			c.messages <- entity
		}
	}
}

// ProtoUnmarshal is a helper to unmarshal protobuf messages generically.
func ProtoUnmarshal[T proto.Message](data []byte) (T, error) {
	var zero T
	msg := newMessage[T]()
	if err := proto.Unmarshal(data, msg); err != nil {
		return zero, err
	}
	return msg, nil
}

func newMessage[T proto.Message]() T {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		v := reflect.New(t.Elem())
		return v.Interface().(T)
	}
	v := reflect.New(t).Elem()
	return v.Addr().Interface().(T)
}
