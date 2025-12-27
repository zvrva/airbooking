package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTP    HTTPConfig    `yaml:"http"`
	GRPC    GRPCConfig    `yaml:"grpc"`
	Database DatabaseConfig `yaml:"database"`
	Redis   RedisConfig   `yaml:"redis"`
	Kafka   KafkaConfig   `yaml:"kafka"`
	Booking BookingConfig `yaml:"booking"`
	Worker  WorkerConfig  `yaml:"worker"`
}

type HTTPConfig struct {
	Address    string `yaml:"address"`
	SwaggerDir string `yaml:"swagger_dir"`
}

type GRPCConfig struct {
	Address string `yaml:"address"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	SSLMode  string `yaml:"ssl_mode"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode)
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type KafkaConfig struct {
	Brokers            []string `yaml:"brokers"`
	BookingTopic       string   `yaml:"booking_topic"`
	BookingEventsTopic string   `yaml:"booking_events_topic"`
	NotificationsTopic string   `yaml:"notifications_topic"`
	GroupID            string   `yaml:"group_id"`
}

type BookingConfig struct {
	HoldTTLMinutes    int `yaml:"hold_ttl_minutes"`
	FlightsCacheTTL   int `yaml:"flights_cache_ttl_seconds"`
	ConfirmationTTL   int `yaml:"confirmation_ttl_minutes"`
}

type WorkerConfig struct {
	ExpirationSweepMinutes int `yaml:"expiration_sweep_minutes"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}
