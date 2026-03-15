package middleware

import (
	"context"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/mhirii/huma-template/pkg/metrics"
)

func MetricsMiddleware(hc huma.Context, next func(huma.Context)) {
	ctx := hc.Context()
	start := hc.Context().Value("requestStart").(time.Time)

	ctx = context.WithValue(ctx, "operationID", hc.Operation().OperationID)
	ctx = context.WithValue(ctx, "requestStart", start)

	next(hc)

	status := strconv.Itoa(hc.Status())
	labels := map[string]string{
		"status":    status,
		"operation": hc.Operation().OperationID,
		"method":    hc.Method(),
		"path":      hc.URL().Path,
	}

	metrics.Metrics.HttpRequestsSeconds.With(labels).Observe(time.Since(start).Seconds())
	metrics.Metrics.HttpRequestsTotal.With(labels).Inc()
}
