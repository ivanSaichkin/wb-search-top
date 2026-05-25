package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ivanSaichkin/wb-search-top/internal/app/factory"
	"github.com/ivanSaichkin/wb-search-top/internal/config"
	httpAdapter "github.com/ivanSaichkin/wb-search-top/internal/interfaces/http"
	"github.com/ivanSaichkin/wb-search-top/internal/interfaces/rabbitmq"
	"github.com/ivanSaichkin/wb-search-top/internal/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()
	logger := logger.InitLogger(cfg.Logger)
	slog.SetDefault(logger)
	logger.Info("Logger initialized", "level", cfg.Logger.Level, "format", cfg.Logger.Format)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to Redis", "addr", cfg.Redis.Addr)

	srvFactory := factory.NewServiceFactory(redisClient)
	services := srvFactory.Build()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запуск фонового воркера агрегации
	go services.Search.RunAggregatorWorker(ctx, cfg.App.TopInterval)

	// Запуск RabbitMQ Consumer
	consumer, err := rabbitmq.NewConsumer(cfg.RabbitMQ.URL, cfg.RabbitMQ.Queue, services.Search)
	if err != nil {
		slog.Error("Failed to initialize RabbitMQ consumer", "error", err)
		os.Exit(1)
	}
	go consumer.Start(ctx)
	defer consumer.Close()

	// Инициализация HTTP сервера
	mux := http.NewServeMux()
	handler := httpAdapter.NewHandler(services.Search, services.StopList)
	handler.RegisterRoutes(mux)
	wrappedHandler := httpAdapter.MetricsMiddleware(mux)

	srv := &http.Server{
		Addr:    cfg.App.HTTPPort,
		Handler: wrappedHandler,
	}

	go func() {
		slog.Info("Starting HTTP server", "port", cfg.App.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
		slog.Info("HTTP server started successfully", "port", cfg.App.HTTPPort)
	}()

	// Запуск отдельного HTTP-сервера для сбора метрик Прометеусом
	go func() {
		metricsMux := http.NewServeMux()
		metricsMux.Handle("/metrics", promhttp.Handler())

		slog.Info("Starting Prometheus metrics server", "port", ":2112")
		if err := http.ListenAndServe(":2112", metricsMux); err != nil {
			slog.Error("Metrics server failed", "error", err)
		}
		slog.Info("Prometheus metrics server started successfully", "port", ":2112")
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	cancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server exiting gracefully")
}
