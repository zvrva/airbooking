// package kafka

// import (
// 	"context"
// 	"encoding/json"
// 	"time"

// 	"github.com/segmentio/kafka-go"
// )

// type BookingEvent struct {
// 	Type       string    `json:"type"`
// 	Token      string    `json:"token"`
// 	FlightID   int64     `json:"flight_id"`
// 	SeatNumber int       `json:"seat_number"`
// 	Email      string    `json:"email"`
// 	Status     string    `json:"status"`
// 	ExpiresAt  time.Time `json:"expires_at"`
// }

// type Producer struct {
// 	brokers []string
// }

// func NewProducer(brokers []string) *Producer {
// 	return &Producer{brokers: brokers}
// }

// func (p *Producer) Publish(ctx context.Context, topic, key string, payload interface{}) error {
// 	data, err := json.Marshal(payload)
// 	if err != nil {
// 		return err
// 	}

// 	writer := &kafka.Writer{Addr: kafka.TCP(p.brokers...), Topic: topic, Balancer: &kafka.LeastBytes{}}
// 	defer writer.Close()

// 	return writer.WriteMessages(ctx, kafka.Message{Key: []byte(key), Value: data})
// }

package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	// УДАЛИТЬ или ЗАКОММЕНТИРОВАТЬ эту строку:
	// "github.com/segmentio/kafka-go/transport"
)

type BookingEvent struct {
	Type       string    `json:"type"`
	Token      string    `json:"token"`
	FlightID   int64     `json:"flight_id"`
	SeatNumber int       `json:"seat_number"`
	Email      string    `json:"email"`
	Status     string    `json:"status"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type Producer struct {
	brokers []string
	writer  *kafka.Writer
}

func NewProducer(brokers []string) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 50 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
		Async:        false,
		// УДАЛИТЬ или ЗАКОММЕНТИРОВАТЬ Transport:
		// Transport: &transport.Transport{
		//     DialTimeout:  10 * time.Second,
		//     ReadTimeout:  30 * time.Second,
		//     WriteTimeout: 30 * time.Second,
		// },
	}

	return &Producer{
		brokers: brokers,
		writer:  writer,
	}
}

func (p *Producer) Publish(ctx context.Context, topic, key string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	log.Printf("Publishing to Kafka - Topic: %s, Key: %s, Payload: %s", topic, key, string(data))

	// Создаем новое сообщение
	message := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
		Time:  time.Now(),
	}

	// Пытаемся отправить сообщение
	err = p.writer.WriteMessages(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to write message to Kafka: %w", err)
	}

	log.Printf("Successfully published to Kafka - Topic: %s, Key: %s", topic, key)
	return nil
}

func (p *Producer) PublishWithRetry(ctx context.Context, topic, key string, payload interface{}, maxRetries int) error {
	var lastErr error
	
	for i := 0; i < maxRetries; i++ {
		err := p.Publish(ctx, topic, key, payload)
		if err == nil {
			return nil
		}
		
		lastErr = err
		log.Printf("Attempt %d failed: %v", i+1, err)
		
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
		}
	}
	
	return fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func (p *Producer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// Метод для проверки подключения к Kafka
func (p *Producer) CheckConnection(ctx context.Context) error {
	conn, err := kafka.DialContext(ctx, "tcp", p.brokers[0])
	if err != nil {
		return fmt.Errorf("failed to connect to Kafka: %w", err)
	}
	defer conn.Close()

	// Получаем список топиков
	partitions, err := conn.ReadPartitions()
	if err != nil {
		return fmt.Errorf("failed to read partitions: %w", err)
	}

	log.Printf("Connected to Kafka. Available topics: %v", partitions)
	return nil
}