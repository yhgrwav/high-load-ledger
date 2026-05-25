package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

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

type PrometheusMetrics struct {
	requestsTotal            *prometheus.CounterVec
	responseTime             *prometheus.HistogramVec
	TransactionResultCounter *prometheus.CounterVec
	PostingBalancesCorrected prometheus.Counter
	serviceName              string
}

func NewPrometheusMetrics(serviceName string) *PrometheusMetrics {
	m := &PrometheusMetrics{
		requestsTotal: requestsCounter,
		responseTime:  requestDuration,
		serviceName:   serviceName,
		TransactionResultCounter: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "ledger_transaction_execution_total",
			Help: "Total number of processed transaction by status",
		}, []string{"status"},
		),
		PostingBalancesCorrected: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "posting_worker",
			Name:      "balances_corrected_total",
			Help:      "Number of account balances corrected by posting worker",
		}),
	}

	prometheus.MustRegister(m.requestsTotal, m.responseTime)

	return m
}

func (m PrometheusMetrics) ObserveTotalRequests(method, code string) {
	m.requestsTotal.WithLabelValues(m.serviceName, method, code).Inc()
}

func (m PrometheusMetrics) ObserveResponseTime(method, status string, duration float64) {
	m.responseTime.WithLabelValues(method, status).Observe(duration)
}
