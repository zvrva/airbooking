package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, groupID, topic string) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:           brokers,
			GroupID:           groupID,
			Topic:             topic,
			HeartbeatInterval: 3 * time.Second,
			SessionTimeout:    30 * time.Second,
		}),
	}
}

func (c *Consumer) Close() error {
	if c == nil || c.reader == nil {
		return nil
	}
	return c.reader.Close()
}

func (c *Consumer) Consume(ctx context.Context, handler func(context.Context, kafka.Message) error) error {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return err
		}

		if err := handler(ctx, msg); err != nil {
			return err
		}
	}
}
