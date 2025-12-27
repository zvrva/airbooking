package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Domenick1991/airbooking/config"
	"github.com/Domenick1991/airbooking/internal/domain"
	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client     *redis.Client
	flightsTTL time.Duration
}

func NewRedisCache(cfg config.RedisConfig, flightsTTL time.Duration) *RedisCache {
	return &RedisCache{
		client: redis.NewClient(&redis.Options{Addr: cfg.Addr, Password: cfg.Password, DB: cfg.DB}),
		flightsTTL: flightsTTL,
	}
}

func (c *RedisCache) GetFlights(ctx context.Context) ([]domain.Flight, error) {
	data, err := c.client.Get(ctx, flightsKey()).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var flights []domain.Flight
	if err := json.Unmarshal(data, &flights); err != nil {
		return nil, err
	}
	return flights, nil
}

func (c *RedisCache) SetFlights(ctx context.Context, flights []domain.Flight) error {
	payload, err := json.Marshal(flights)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, flightsKey(), payload, c.flightsTTL).Err()
}

func (c *RedisCache) AcquireSeatLock(ctx context.Context, flightID int64, seat int, ttl time.Duration) (bool, error) {
	key := seatLockKey(flightID, seat)
	return c.client.SetNX(ctx, key, "locked", ttl).Result()
}

func (c *RedisCache) ReleaseSeatLock(ctx context.Context, flightID int64, seat int) error {
	return c.client.Del(ctx, seatLockKey(flightID, seat)).Err()
}

func flightsKey() string {
	return "cache:flights"
}

func seatLockKey(flightID int64, seat int) string {
	return fmt.Sprintf("lock:flight:%d:seat:%d", flightID, seat)
}
