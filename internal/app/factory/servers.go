package factory

import (
	"net/http"

	"github.com/ivanSaichkin/wb-search-top/internal/config"
	httpAdapter "github.com/ivanSaichkin/wb-search-top/internal/interfaces/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// конфигурирует основной HTTP-сервер приложения
func initAPIServer(cfg *config.Config, services *Services) *http.Server {
	mux := http.NewServeMux()
	handler := httpAdapter.NewHandler(services.Search, services.StopList)
	handler.RegisterRoutes(mux)
	wrappedHandler := httpAdapter.MetricsMiddleware(mux)

	return &http.Server{
		Addr:    cfg.App.HTTPPort,
		Handler: wrappedHandler,
	}
}

// конфигурирует изолированный сервер для сбора метрик Прометеусом
func initMetricsServer() *http.Server {
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	return &http.Server{
		Addr:    ":2112",
		Handler: metricsMux,
	}
}
