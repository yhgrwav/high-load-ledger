package telemetry

import (
	"context"
	"errors"
	"fmt"
	"high-load-ledger/internal/config"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Observability struct {
	logger  *slog.Logger
	Metrics *PrometheusMetrics
	server  *http.Server
}

func New(cfg *config.Config, logger slog.Logger) *Observability {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	return &Observability{
		logger:  &logger,
		Metrics: NewPrometheusMetrics(cfg.ServiceName),
		server: &http.Server{
			Addr:         ":" + cfg.MetricsPort,
			Handler:      mux,
			ReadTimeout:  5 * time.Second,  // можно было читать из конфига, но :/
			WriteTimeout: 10 * time.Second, //
		},
	}
}

func (o *Observability) Start(errCh chan<- error) {
	go func() {
		o.logger.Info("Prometheus server is starting on port " + o.server.Addr)

		if err := o.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("telemetry server error: %w", err)
		}
	}()
}

func (o *Observability) Stop(ctx context.Context) error {
	o.logger.Info("Stopping Prometheus server")

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Second)

	defer shutdownCancel()

	if err := o.server.Shutdown(shutdownCtx); err != nil {
		o.logger.Error("Failed to shutdown Prometheus server", "error", err)
		return err
	}

	return nil
}
