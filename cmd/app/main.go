package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ivanSaichkin/wb-search-top/internal/app/factory"
	"github.com/ivanSaichkin/wb-search-top/internal/config"
	httpAdapter "github.com/ivanSaichkin/wb-search-top/internal/interfaces/http"
	"github.com/ivanSaichkin/wb-search-top/internal/interfaces/kafka"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	srvFactory := factory.NewServiceFactory(redisClient)
	services := srvFactory.Build()

	// Запуск фонового воркера агрегации
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go services.Search.RunAggregatorWorker(ctx, cfg.App.TopInterval)

	// Запуск Kafka Consumer
	consumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.GroupID, services.Search)
	go consumer.Start(ctx)
	defer consumer.Close()

	// Инициализация HTTP сервера
	mux := http.NewServeMux()
	handler := httpAdapter.NewHandler(services.Search, services.StopList)
	handler.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:    cfg.App.HTTPPort,
		Handler: mux,
	}

	go func() {
		log.Printf("Starting HTTP server on %s", cfg.App.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	cancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
