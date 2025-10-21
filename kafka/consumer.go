package kafka

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
	"google.golang.org/protobuf/proto"
)

type Consumer[T any] struct {
	client *kgo.Client
	ch     chan *kgo.Record
	out    chan T
	topic  string
}

func NewConsumer[T proto.Message](brokers []string, groupID *string, convertor func([]byte) (T, error)) *Consumer[T] {
	if groupID == nil {
		uid := uuid.New().String()
		groupID = &uid
	}

	var t T
	topic := strings.Replace(fmt.Sprintf("%T", t), "*", "", 1)
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(*groupID),
		kgo.ConsumeTopics(topic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	if err != nil {
		fmt.Println("failed to create kafka consumer client:", err)
		panic(err)
	}
	ch := make(chan *kgo.Record)
	out := make(chan T)

	go func() {
		for message := range ch {
			msg, err := convertor(message.Value)
			if err != nil {
				log.Printf("unable to process message: %s\n", err)
				continue
			}
			out <- msg
		}
		close(out)
	}()

	c := &Consumer[T]{
		client: client,
		topic:  topic,
		ch:     ch,
		out:    out,
	}
	go c.read()
	return c
}
func (c *Consumer[T]) ReadMessages() <-chan *kgo.Record {
	return c.ch
}

func (c *Consumer[T]) Read() <-chan T {
	return c.out
}

func (c *Consumer[T]) read() {
	ctx := context.Background()
	for {
		fetches := c.client.PollFetches(ctx)
		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			if record == nil {
				continue
			}
			c.ch <- record
		}
	}
}

type Deserializer[T any] interface {
	Unmarshal([]byte, *T) error
}

func (c *Consumer[T]) Close() {
	c.client.Close()
	close(c.ch)
}
