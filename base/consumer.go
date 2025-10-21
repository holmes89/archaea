package base

type Consumer[T any] interface {
	Read() <-chan T
	Close()
}

type ConsumerMessage struct {
	Value []byte
	Key   string
}
