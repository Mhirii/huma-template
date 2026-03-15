package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/mhirii/huma-template/pkg/logging"
	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func GeneralMiddleware(hc huma.Context, next func(huma.Context)) {
	log := logging.L().With().Logger()
	ctx := hc.Context()

	fromctx := ctx.Value("logger")
	if fromctx != nil {
		log = fromctx.(zerolog.Logger)
	}

	userIDValue := ctx.Value("userID")
	if userIDValue != nil {
		userID := userIDValue.(string)
		log = log.With().Str("userID", userID).Logger()
	}

	log.With().Str("operationID", hc.Operation().OperationID)
	ctx = context.WithValue(ctx, "operationID", hc.Operation().OperationID)

	ctx = context.WithValue(ctx, "logger", log)
	hc = huma.WithContext(hc, ctx)

	next(hc)

	logger := hc.Context().Value("logger").(zerolog.Logger)
	start := hc.Context().Value("requestStart").(time.Time)
	logger.Info().
		Int("status", hc.Status()).
		Str("method", hc.Method()).
		Str("path", hc.URL().Path).
		Dur("duration", time.Since(start)).
		Str("operationID", hc.Operation().OperationID).
		Msg(fmt.Sprintf("%d", hc.Status()))
}

func OnStartMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logging.L()

		// excluded := []string{"/openapi.yaml", "/docs"}

		start := time.Now()
		ctx = context.WithValue(ctx, "requestStart", start)

		ww := &responseWriter{ResponseWriter: w, status: 200}

		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			fromctx := ctx.Value("requestID")
			if fromctx != nil {
				requestID = fromctx.(string)
			} else {
				requestID = ulid.Make().String()
			}
		}
		ctx = context.WithValue(ctx, "requestID", requestID)
		logctx := l.With().Str("requestID", requestID)
		ctx = context.WithValue(ctx, "logger", logctx.Logger())
		ctx = logctx.Logger().WithContext(ctx)

		next.ServeHTTP(ww, r.WithContext(ctx))
	})
}
