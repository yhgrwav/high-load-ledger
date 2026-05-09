package telemetry

import "github.com/prometheus/client_golang/prometheus"

type Metrics interface {
	ObserveTotalRequests(method, code string)
	ObserveResponseTime(method, status string, duration float64)
}

// кастомные метрики для приложения
var (
	requestsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "ledger",                // prefix
		Subsystem: "grpc",                  // api type
		Name:      "total_requests",        // name
		Help:      "grpc requests counter", // description
	}, []string{"service", "method", "code"}, // labels
	)

	requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "ledger",
		Subsystem: "grpc",
		Name:      "request_duration",
		Help:      "duration of grpc requests",
		Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
	}, []string{"method", "code"})
)

type prometheusMetrics struct {
	requestsTotal *prometheus.CounterVec
	responseTime  *prometheus.HistogramVec
	serviceName   string
}

func NewPrometheusMetrics(serviceName string) Metrics {
	m := &prometheusMetrics{
		requestsTotal: requestsCounter,
		responseTime:  requestDuration,
		serviceName:   serviceName,
	}

	prometheus.MustRegister(m.requestsTotal, m.responseTime)

	return m
}

func (m prometheusMetrics) ObserveTotalRequests(method, code string) {
	m.requestsTotal.WithLabelValues(m.serviceName, method, code).Inc()
}

func (m prometheusMetrics) ObserveResponseTime(method, status string, duration float64) {
	m.responseTime.WithLabelValues(method, status).Observe(duration)
}
