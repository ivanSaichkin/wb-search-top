package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	App      AppConfig
	Redis    RedisConfig
	RabbitMQ RabbitMQConfig
	Logger   LoggerConfig
}

type AppConfig struct {
	HTTPPort    string
	TopInterval time.Duration // Как часто воркер пересчитывает топ
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type RabbitMQConfig struct {
	URL   string
	Queue string
}

type LoggerConfig struct {
	Level  string
	Format string
}

func Load() *Config {
	return &Config{
		App: AppConfig{
			HTTPPort:    getEnvString("HTTP_PORT", ":8080"),
			TopInterval: getEnvDuration("TOP_COMPUTE_INTERVAL", 2*time.Second),
		},
		Redis: RedisConfig{
			Addr:     getEnvString("REDIS_ADDR", "localhost:6379"),
			Password: getEnvString("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		RabbitMQ: RabbitMQConfig{
			URL:   getEnvString("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
			Queue: getEnvString("RABBITMQ_QUEUE", "search_events"),
		},
		Logger: LoggerConfig{
			Level:  getEnvString("LOG_LEVEL", "debug"),
			Format: getEnvString("LOG_FORMAT", "text"),
		},
	}
}

func getEnvString(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}

	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if durationVal, err := time.ParseDuration(value); err == nil {
			return durationVal
		}
	}

	return defaultVal
}
