package telemetry

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics interface {
	ObserveGRPCRequest(rpc, code string)
	ObserveGRPCDuration(rpc, code string, seconds float64)
	ObserveTotalRequests(fullMethod, code string)
	ObserveResponseTime(fullMethod, code string, seconds float64)
	RecordTransfer(result string)
	RecordBalanceCorrection()
}

type PrometheusMetrics struct {
	serviceName string

	grpcRequests       *prometheus.CounterVec
	grpcDuration       *prometheus.HistogramVec
	transferTotal      *prometheus.CounterVec
	balanceCorrections prometheus.Counter
}

func NewPrometheusMetrics(serviceName string) *PrometheusMetrics {
	return &PrometheusMetrics{
		serviceName: serviceName,
		grpcRequests: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "ledger",
			Subsystem: "grpc",
			Name:      "requests_total",
			Help:      "gRPC requests handled, by RPC and status code.",
		}, []string{"service", "rpc", "code"}),
		grpcDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "ledger",
			Subsystem: "grpc",
			Name:      "request_duration_seconds",
			Help:      "gRPC request latency in seconds.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		}, []string{"service", "rpc", "code"}),
		transferTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "ledger",
			Name:      "transfer_total",
			Help:      "Transfer use case outcomes (business and system).",
		}, []string{"service", "result"}),
		balanceCorrections: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "ledger",
			Subsystem: "posting_worker",
			Name:      "balance_corrections_total",
			Help:      "Account balances corrected by posting worker.",
		}),
	}
}

func (m *PrometheusMetrics) ObserveGRPCRequest(rpc, code string) {
	m.grpcRequests.WithLabelValues(m.serviceName, rpc, code).Inc()
}

func (m *PrometheusMetrics) ObserveGRPCDuration(rpc, code string, seconds float64) {
	m.grpcDuration.WithLabelValues(m.serviceName, rpc, code).Observe(seconds)
}

func (m *PrometheusMetrics) RecordTransfer(result string) {
	m.transferTotal.WithLabelValues(m.serviceName, result).Inc()
}

func (m *PrometheusMetrics) RecordBalanceCorrection() {
	m.balanceCorrections.Inc()
}

func (m *PrometheusMetrics) ObserveTotalRequests(fullMethod, code string) {
	m.ObserveGRPCRequest(grpcRPC(fullMethod), code)
}

func (m *PrometheusMetrics) ObserveResponseTime(fullMethod, code string, duration float64) {
	m.ObserveGRPCDuration(grpcRPC(fullMethod), code, duration)
}

func grpcRPC(fullMethod string) string {
	if i := strings.LastIndex(fullMethod, "/"); i >= 0 && i < len(fullMethod)-1 {
		return fullMethod[i+1:]
	}
	if fullMethod != "" {
		return fullMethod
	}
	return "unknown"
}
