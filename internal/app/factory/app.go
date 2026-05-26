package factory

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ivanSaichkin/wb-search-top/internal/config"
	"github.com/ivanSaichkin/wb-search-top/internal/interfaces/rabbitmq"
	"github.com/redis/go-redis/v9"
)

type App struct {
	cfg           *config.Config
	redisClient   *redis.Client
	services      *Services
	consumer      *rabbitmq.Consumer
	apiServer     *http.Server
	metricsServer *http.Server
}

func NewApp(cfg *config.Config) (*App, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		redisClient.Close()
		return nil, err
	}
	slog.Info("Connected to Redis", "addr", cfg.Redis.Addr)

	srvFactory := NewServiceFactory(redisClient)
	services := srvFactory.Build()

	consumer, err := rabbitmq.NewConsumer(cfg.RabbitMQ.URL, cfg.RabbitMQ.Queue, services.Search)
	if err != nil {
		redisClient.Close()
		return nil, err
	}

	apiServer := initAPIServer(cfg, services)
	metricsServer := initMetricsServer()

	return &App{
		cfg:           cfg,
		redisClient:   redisClient,
		services:      services,
		consumer:      consumer,
		apiServer:     apiServer,
		metricsServer: metricsServer,
	}, nil
}

// запускает горутины рантайма и блокирует поток до системного прерывания OS
func (a *App) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Воркеры и консьюмеры
	go a.services.Search.RunAggregatorWorker(ctx, a.cfg.App.TopInterval)
	go a.consumer.Start(ctx)

	// Старт серверов
	go func() {
		slog.Info("Starting HTTP server", "port", a.cfg.App.HTTPPort)
		if err := a.apiServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server runtime error", "error", err)
		}
	}()

	go func() {
		slog.Info("Starting Prometheus metrics server", "port", ":2112")
		if err := a.metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Prometheus metrics server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down application gracefully...")
	cancel()

	a.consumer.Close()
	a.redisClient.Close()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	var errs error
	if err := a.apiServer.Shutdown(ctxShutdown); err != nil {
		slog.Error("API Server forced to shutdown", "error", err)
		errs = errors.Join(errs, err)
	}

	if err := a.metricsServer.Shutdown(ctxShutdown); err != nil {
		slog.Error("Prometheus metrics server forced to shutdown", "error", err)
		errs = errors.Join(errs, err)
	}

	slog.Info("Application exited cleanly")
	return errs
}
