package http

import (
	"net/http"
	"strconv"

	"github.com/ivanSaichkin/wb-search-top/internal/infrastructure/metrics"
)

// Обертка над стандартным ResponseWriter для перехвата HTTP-статуса
type responseWriterDelegator struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriterDelegator) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriterDelegator) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}

// оборачивает хендлер и логирует метрики в Prometheus
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		delegate := &responseWriterDelegator{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(delegate, r)

		path := r.URL.Path
		statusStr := strconv.Itoa(delegate.status)

		metrics.HttpRequestsTotal.WithLabelValues(path, statusStr).Inc()
	})
}
