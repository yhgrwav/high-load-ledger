package interceptors

import "github.com/prometheus/client_golang/prometheus"

var (
	requestsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "leger",                 // prefix
		Subsystem: "grpc",                  // api type
		Name:      "total_requests",        // name
		Help:      "grpc requests counter", // description
	}, []string{"service", "method,", "code"}, // labels
	)

	requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "ledger",
		Subsystem: "grpc",
		Name:      "request_duration",
		Help:      "duration of grpc requests",
		Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
	}, []string{"method", "code"})
)
