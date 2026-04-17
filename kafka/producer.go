package kafka

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Producer[T stringer] struct {
	client *kgo.Client
	topic  string
}

type stringer interface {
	ProtoReflect() protoreflect.Message
}

func NewProducer[T stringer](conn *Conn) *Producer[T] {
	var t T
	topic := strings.Replace(fmt.Sprintf("%T", t), "*", "", 1)
	conn.CreateTopic(topic)
	return &Producer[T]{
		client: conn.client,
		topic:  topic,
	}
}
func (p *Producer[T]) Publish(ctx context.Context, message T, id string, _ time.Time) error {
	i := []byte(id)
	b, err := proto.Marshal(message)
	if err != nil {
		fmt.Printf("unable to send message: %s", err)
		return errors.New("unable to publish message")
	}
	fmt.Println("producing message to topic:", p.topic)
	results := p.client.ProduceSync(ctx, &kgo.Record{Topic: p.topic, Value: b, Key: i})
	if err := results.FirstErr(); err != nil {
		fmt.Printf("failed to produce message to %s: %s\n", p.topic, err)
		return fmt.Errorf("publish to %s: %w", p.topic, err)
	}
	fmt.Printf("message %s produced successfully on topic %s\n", id, p.topic)
	return nil
}
func (p *Producer[T]) Close() error {
	p.client.Close()
	return nil
}
