package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	App   AppConfig
	Redis RedisConfig
	Kafka KafkaConfig
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

type KafkaConfig struct {
	Brokers []string
	Topic   string
	GroupID string
}

func Load() *Config {
	return &Config{
		App: AppConfig{
			HTTPPort:    getEnvString("HTTP_PORT", ":8080"),
			TopInterval: getEnvDuration("TOP_COMPUT_INTERVAL", 2*time.Second),
		},
		Redis: RedisConfig{
			Addr:     getEnvString("REDIS_ADDR", "localhost:6379"),
			Password: getEnvString("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnvString("KAFKA_BROKERS", "localhost:9092")},
			Topic:   getEnvString("KAFKA_TOPIC", "search_events"),
			GroupID: getEnvString("KAFKA_GROUP_ID", "top_processor"),
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
