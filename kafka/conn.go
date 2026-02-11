package kafka

import (
	"log"

	"github.com/segmentio/kafka-go"
)

// Conn wraps a Kafka connection with common configuration.
type Conn struct {
	brokers []string
}

// NewConn creates a new Kafka connection helper.
func NewConn(brokers []string) *Conn {
	return &Conn{brokers: brokers}
}

// GetBrokers returns the configured broker list.
func (c *Conn) GetBrokers() []string { return c.brokers }

// Close is a no-op placeholder (writers/readers manage their own lifecycle).
func (c *Conn) Close() error { return nil }

// EnsureTopic creates a topic if it does not exist.
func (c *Conn) EnsureTopic(topic string, numPartitions int, replicationFactor int) error {
	conn, err := kafka.Dial("tcp", c.brokers[0])
	if err != nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}

	controllerConn, err := kafka.Dial("tcp", controller.Host)
	if err != nil {
		return err
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{{
		Topic:             topic,
		NumPartitions:     numPartitions,
		ReplicationFactor: replicationFactor,
	}}

	if err := controllerConn.CreateTopics(topicConfigs...); err != nil {
		log.Printf("Warning: could not create topic %s: %v", topic, err)
	}
	return nil
}

// CreateWriter creates a Kafka writer for a specific topic.
func (c *Conn) CreateWriter(topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:     kafka.TCP(c.brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
}

// CreateReader creates a Kafka reader for a specific topic and consumer group.
func (c *Conn) CreateReader(topic string, groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:  c.brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
}
