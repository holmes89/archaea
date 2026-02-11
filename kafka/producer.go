package kafka

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

// Producer implements a simple Kafka producer.
type Producer[T proto.Message] struct {
	writer *kafka.Writer
	topic  string
}

// NewProducer creates a new Kafka producer for a given topic.
func NewProducer[T proto.Message](conn *Conn, topic string) *Producer[T] {
	_ = conn.EnsureTopic(topic, 1, 1)
	return &Producer[T]{
		writer: conn.CreateWriter(topic),
		topic:  topic,
	}
}

// Publish marshals and sends a message to Kafka.
func (p *Producer[T]) Publish(ctx context.Context, entity T) error {
	data, err := proto.Marshal(entity)
	if err != nil {
		return fmt.Errorf("failed to marshal proto message: %w", err)
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{Value: data})
	if err != nil {
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}
	return nil
}

// Close closes the producer.
func (p *Producer[T]) Close() error {
	return p.writer.Close()
}
