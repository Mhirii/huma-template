package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/mhirii/huma-template/internal/config"
	"github.com/mhirii/huma-template/pkg/tokens"
)

func AuthMiddleware(hc huma.Context, next func(huma.Context)) {
	ctx := hc.Context()
	for _, t := range hc.Operation().Tags {
		if strings.ToLower(t) == "public" {
			next(hc)
		}
	}
	h := hc.Header("Authorization")
	if h == "" {
		hc.SetStatus(http.StatusUnauthorized)
		return
	}
	splits := strings.Split(h, " ")
	if len(splits) != 2 {
		hc.SetStatus(http.StatusBadRequest)
		return
	}
	if splits[0] != "Bearer" {
		hc.SetStatus(http.StatusBadRequest)
		return
	}
	if splits[1] == "" {
		hc.SetStatus(http.StatusBadRequest)
		return
	}

	t, err := tokens.ParseToken(ctx, splits[1], config.GetAPICfg().Auth.Secret, "access")
	if err != nil {
		hc.SetStatus(http.StatusUnauthorized)
		hc.SetHeader("Content-Type", "text/plain")
		hc.BodyWriter().Write(
			[]byte(err.Error()),
		)
		return
	}
	if err != nil {
		hc.SetStatus(http.StatusUnauthorized)
		hc.SetHeader("Content-Type", "text/plain")
		hc.BodyWriter().Write(
			[]byte(err.Error()),
		)
		return
	}
	hc = huma.WithContext(hc, context.WithValue(hc.Context(), "userID", t.Subject))
	next(hc)
}
