package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Conn struct {
	client      *kgo.Client
	adminClient *kadm.Client
	brokers     []string
}

func NewConn(brokers []string) *Conn {
	fmt.Println("connecting to kafka...")

	// Create the base kgo client
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.RequestTimeoutOverhead(30*time.Second), // 10 second timeout
	)
	if err != nil {
		fmt.Println("failed to create kafka client:", err)
		panic(err)
	}

	// Create admin client from the same base client
	adminClient := kadm.NewClient(client)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 30 second timeout
	defer cancel()

	res, err := adminClient.ListTopics(ctx)
	if err != nil {
		fmt.Println("failed to list kafka topics:", err)
		client.Close()
		panic(err)
	}
	fmt.Printf("found %d topics\n", len(res))
	fmt.Println("connected.")

	return &Conn{
		client:      client,
		adminClient: adminClient,
		brokers:     brokers,
	}
}

func (c *Conn) TopicExists(topic string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 10 second timeout
	defer cancel()

	topicsMetadata, err := c.adminClient.ListTopics(ctx)
	if err != nil {
		fmt.Println("failed to verify topics:", err)
		return false // Return false instead of panic to handle gracefully
	}

	for _, metadata := range topicsMetadata {
		if metadata.Topic == topic {
			return true
		}
	}
	return false
}

func (c *Conn) CreateTopicIfNotExists(topic string) {
	fmt.Println("verifying topic:", topic)
	if !c.TopicExists(topic) {
		c.CreateTopic(topic)
	}
}

func (c *Conn) CreateTopic(topic string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 30 second timeout
	defer cancel()

	fmt.Println("creating topic on brokers:", c.brokers)

	// Create topic with proper configuration
	resp, err := c.adminClient.CreateTopics(ctx, 1, 1, nil, topic)
	if err != nil {
		fmt.Println("failed to create kafka topic:", err)
		panic(err)
	}

	for _, ctr := range resp {
		if ctr.Err != nil {
			fmt.Printf("Unable to create topic '%s': %s\n", ctr.Topic, ctr.Err)
		} else {
			fmt.Printf("Created topic '%s'\n", ctr.Topic)
		}
	}
}

// GetClient returns the underlying kgo.Client for producer/consumer operations
func (c *Conn) GetClient() *kgo.Client {
	return c.client
}

// GetAdminClient returns the admin client for administrative operations
func (c *Conn) GetAdminClient() *kadm.Client {
	return c.adminClient
}

func (c *Conn) Close() {
	if c.client != nil {
		c.client.Close()
	}
}
