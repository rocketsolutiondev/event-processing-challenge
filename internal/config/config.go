package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	NATSURL    string
	EventDelayMS int

	// Exchange rate settings
	ExchangeRateMemoryCacheDuration string
	ExchangeRateDBCacheDuration    string
	ExchangeRateRefreshInterval    string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "casino"),
		DBPassword: getEnv("DB_PASSWORD", "casino"),
		DBName:     getEnv("DB_NAME", "casino"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),
		NATSURL:    getEnv("NATS_URL", "nats://localhost:4222"),
		EventDelayMS: getIntEnv("EVENT_DELAY_MS", 1000),

		// Exchange rate settings
		ExchangeRateMemoryCacheDuration: getEnv("EXCHANGE_RATE_MEMORY_CACHE_DURATION", "1m"),
		ExchangeRateDBCacheDuration:    getEnv("EXCHANGE_RATE_DB_CACHE_DURATION", "24h"),
		ExchangeRateRefreshInterval:    getEnv("EXCHANGE_RATE_REFRESH_INTERVAL", "1h"),
	}, nil
}

func (c *Config) GetDBURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser,
		c.DBPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
		c.DBSSLMode,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return value
	}
	return defaultValue
} 