package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/status"
)

const (
	StreamValid           = "valid"
	StreamInvalidBalance  = "invalid_balance"
	StreamInvalidCurrency = "invalid_currency"

	outcomeOK    = "ok"
	outcomeError = "error"
)

type Metrics struct {
	targetRPS  *prometheus.GaugeVec
	dispatched *prometheus.CounterVec
	completed  *prometheus.CounterVec
	queueDepth prometheus.Gauge
	server     *http.Server
}

func NewMetrics(port string) *Metrics {
	m := &Metrics{
		targetRPS: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "loadgen",
			Name:      "target_rps",
			Help:      "Configured target dispatch RPS per stream.",
		}, []string{"stream"}),
		dispatched: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "loadgen",
			Name:      "dispatched_total",
			Help:      "Transfer jobs queued for workers.",
		}, []string{"stream"}),
		completed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "loadgen",
			Name:      "completed_total",
			Help:      "Finished gRPC Transfer calls from loadgen.",
		}, []string{"stream", "outcome", "grpc_code"}),
		queueDepth: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "loadgen",
			Name:      "queue_depth",
			Help:      "Transfer jobs waiting in the channel.",
		}),
	}

	prometheus.MustRegister(m.targetRPS, m.dispatched, m.completed, m.queueDepth)

	m.server = &http.Server{
		Addr:              ":" + port,
		Handler:           promhttp.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	return m
}

func (m *Metrics) SetTarget(stream string, rps float64) {
	m.targetRPS.WithLabelValues(stream).Set(rps)
}

func (m *Metrics) RecordDispatched(stream string) {
	m.dispatched.WithLabelValues(stream).Inc()
}

func (m *Metrics) RecordCompleted(stream string, err error) {
	outcome := outcomeOK
	grpcCode := "OK"
	if err != nil {
		outcome = outcomeError
		grpcCode = "Unknown"
		if st, ok := status.FromError(err); ok {
			grpcCode = st.Code().String()
		}
	}
	m.completed.WithLabelValues(stream, outcome, grpcCode).Inc()
}

func (m *Metrics) SetQueueDepth(depth int) {
	m.queueDepth.Set(float64(depth))
}

func (m *Metrics) Start() {
	go func() {
		log.Printf("loadgen metrics: listening on %s/metrics", m.server.Addr)
		if err := m.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("loadgen metrics server: %v", err)
		}
	}()
}

func (m *Metrics) Stop(ctx context.Context) {
	if err := m.server.Shutdown(ctx); err != nil {
		log.Printf("loadgen metrics shutdown: %v", err)
	}
}

func ValidateAchieved(stream string, target, achieved float64, tolerance float64) error {
	if target <= 0 {
		return nil
	}
	if achieved < target*(1-tolerance) {
		return fmt.Errorf("stream %s: achieved %.1f rps < target %.1f rps (tolerance %.0f%%)",
			stream, achieved, target, tolerance*100)
	}
	return nil
}
