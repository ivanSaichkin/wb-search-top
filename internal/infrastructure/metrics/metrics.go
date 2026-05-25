package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Счётчик обработанных сообщений из RabbitMQ
	RabbitMQMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "search_top_rabbitmq_messages_total",
			Help: "Total number of processed RabbitMQ messages",
		},
		[]string{"status"},
	)

	// Гистограмма времени работы воркера агрегации в Redis
	AggregationDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "search_top_aggregation_duration_seconds",
			Help:    "Duration of top queries aggregation loop in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	// Счётчик HTTP запросов к API
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "search_top_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"path", "status"},
	)
)
