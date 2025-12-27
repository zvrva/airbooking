package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Domenick1991/airbooking/config"
	"github.com/Domenick1991/airbooking/internal/cache"
	"github.com/Domenick1991/airbooking/internal/email"
	"github.com/Domenick1991/airbooking/internal/kafka"
	"github.com/Domenick1991/airbooking/internal/repository"
	"github.com/Domenick1991/airbooking/internal/service/booking"
	"github.com/jackc/pgx/v5/pgxpool"
	kafkaGo "github.com/segmentio/kafka-go"
)

func main() {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	producer := kafka.NewProducer(cfg.Kafka.Brokers)
	redisCache := cache.NewRedisCache(cfg.Redis, time.Duration(cfg.Booking.FlightsCacheTTL)*time.Second)

	flightRepo := repository.NewFlightRepository(pool)
	bookingRepo := repository.NewBookingRepository(pool)
	bookingService := booking.NewBookingService(
		bookingRepo,
		flightRepo,
		redisCache,
		producer,
		cfg.Kafka.BookingEventsTopic,
		time.Duration(cfg.Booking.HoldTTLMinutes)*time.Minute,
		time.Duration(cfg.Booking.ConfirmationTTL)*time.Minute,
		booking.WithNotificationsTopic(cfg.Kafka.NotificationsTopic),
	)

	consumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.GroupID, cfg.Kafka.NotificationsTopic)
	defer consumer.Close()

	emailSender := email.NewSender()

	go func() {
		if err := consumer.Consume(ctx, func(ctx context.Context, msg kafkaGo.Message) error {
			var event kafka.BookingEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("decode event error: %v", err)
				return nil
			}
			return emailSender.Send(ctx, event)
		}); err != nil {
			log.Printf("consumer stopped: %v", err)
		}
	}()

	expireTicker := time.NewTicker(time.Duration(cfg.Worker.ExpirationSweepMinutes) * time.Minute)
	defer expireTicker.Stop()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-expireTicker.C:
			expired, err := bookingService.ExpirePendingBookings(ctx)
			if err != nil {
				log.Printf("expire bookings error: %v", err)
				continue
			}
			if len(expired) > 0 {
				log.Printf("expired %d bookings", len(expired))
			}
		case s := <-sig:
			log.Printf("received signal %v, shutting down", s)
			return
		}
	}
}
