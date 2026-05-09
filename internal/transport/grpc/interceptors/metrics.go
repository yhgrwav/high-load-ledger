package interceptors

import (
	"context"
	"high-load-ledger/internal/infra/telemetry"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func UnaryMetricsInterceptor(m telemetry.Metrics) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)
		if err != nil {
			return nil, err
		}

		result := float64(time.Since(start).Seconds())

		st, _ := status.FromError(err)

		code := st.Code().String()

		m.ObserveResponseTime(info.FullMethod, code, result)
		m.ObserveTotalRequests(info.FullMethod, code)

		return resp, err
	}
}
