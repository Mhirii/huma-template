package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/mhirii/huma-template/internal/config"
	"github.com/mhirii/huma-template/pkg/logging"
	"github.com/mhirii/huma-template/pkg/tokens"
)

func AuthMiddleware(hc huma.Context, next func(huma.Context)) {
	ctx := hc.Context()
	l := logging.L().With().Str("path", hc.Operation().Path).Str("operation", hc.Operation().OperationID).Logger()
	for _, t := range hc.Operation().Tags {
		if strings.ToLower(t) == "public" {
			l.Info().Msg("public access")
			next(hc)
			return
		}
	}
	h := hc.Header("Authorization")
	if h == "" {
		hc.SetStatus(http.StatusUnauthorized)
		l.Warn().Msg("empty Authorization header")
		return
	}
	splits := strings.Split(h, " ")
	if len(splits) != 2 {
		hc.SetStatus(http.StatusBadRequest)
		l.Warn().Msg("Authorization header is not in the format 'Bearer <token>', got " + h)
		return
	}
	if splits[0] != "Bearer" {
		hc.SetStatus(http.StatusBadRequest)
		l.Warn().Msg("Authorization header is not in the format 'Bearer <token>', got " + splits[0])
		return
	}
	if splits[1] == "" {
		hc.SetStatus(http.StatusBadRequest)
		l.Warn().Msg("Authorization header does not contain a token")
		return
	}

	t, err := tokens.ParseToken(ctx, splits[1], config.GetAPICfg().Auth.Secret, "access")
	if err != nil {
		hc.SetStatus(http.StatusUnauthorized)
		hc.SetHeader("Content-Type", "text/plain")
		hc.BodyWriter().Write(
			[]byte(err.Error()),
		)
		l.Warn().Err(err).Msg("failed to parse token")
		return
	}
	l.Debug().Str("subject", t.Subject).Msg("successfully parsed token")
	hc = huma.WithContext(hc, context.WithValue(hc.Context(), "userID", t.Subject))
	next(hc)
}
