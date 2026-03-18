package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/mhirii/huma-template/cmd"
	"github.com/mhirii/huma-template/internal/config"
	"github.com/mhirii/huma-template/internal/handlers"
	"github.com/mhirii/huma-template/internal/middleware"
	"github.com/mhirii/huma-template/internal/svc"
	"github.com/mhirii/huma-template/pkg/db"
	"github.com/mhirii/huma-template/pkg/logging"
	"github.com/mhirii/huma-template/pkg/metrics"
	"github.com/mhirii/huma-template/pkg/tokens"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Options struct{}

func main() {

	logging.InitLogger(logging.LoggerConfig{
		LogLevel: "info", LogFormat: "text",
	})

	env := "api"
	config.Load(&env)

	l := logging.L()

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		action := "help"
		if len(os.Args) > 2 {
			switch os.Args[2] {
			case "up", "-up", "--up", "-u":
				action = "up"
			case "down", "-down", "--down", "-d":
				action = "down"
			case "status", "-status", "--status", "-s":
				action = "status"
			case "help", "-help", "--help", "-h":
				action = "help"
			default:
				action = "help"
			}
		}
		cmd.Migrate(action)
		os.Exit(0)
	}

	cli := humacli.New(func(hooks humacli.Hooks, options *Options) {
		cfg := config.GetAPICfg()
		router := chi.NewMux()
		dsn := db.GetDSN(&cfg.DB)
		dbconn, err := db.New(dsn)
		if err != nil {
			panic(err)
		}

		h := cors.Handler(cors.Options{
			// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
			AllowedOrigins: []string{"https://*", "http://*"},
			// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: false,
			MaxAge:           300, // Maximum value not ignored by any of major browsers
		})
		router.Use(h)

		api := humachi.New(router, huma.DefaultConfig("Huma Template API", "1.0.0"))

		tokenProvider, err := tokens.NewTokenProvider(tokens.TokenProviderArgs{
			Secret:          cfg.Auth.Secret,
			AccessTokenTTL:  cfg.Auth.AccessTokenTTL,
			RefreshTokenTTL: cfg.Auth.RefreshTokenTTL,
		})
		if err != nil {
			l.Err(err).Msg("Failed to create token provider, this is a critical module, exiting")
			os.Exit(1)
		}
		_ = tokenProvider
		_ = dbconn

		tokensSvc := svc.NewTokensService(tokenProvider, dbconn)
		userSvc, err := svc.NewUsersService(dbconn)
		if err != nil {
			l.Err(err).Msg("Failed to create user service, this is a critical module, exiting")
			os.Exit(1)
		}
		authsvc, err := svc.NewAuthService(userSvc, tokensSvc, dbconn)
		if err != nil {
			l.Err(err).Msg("Failed to create auth service, this is a critical module, exiting")
			os.Exit(1)
		}

		api.UseMiddleware(middleware.MetricsMiddleware)
		api.UseMiddleware(middleware.GeneralMiddleware)

		handlers.RegisterAuthRoutes(api, authsvc)
		api.UseMiddleware(middleware.AuthMiddleware)

		handlers.RegisterUserRoutes(api, userSvc)

		reg := metrics.InitMetrics()
		reg.MustRegister(
			collectors.NewGoCollector(),
			collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		)

		l.Info().Msg("Starting backend")
		l.Info().Interface("cfg", cfg).Msg("config")

		hooks.OnStart(func() {
			l.Info().Int("port", cfg.Server.Port).Msg("API server listening")
			mux := http.NewServeMux()
			wrapped := otelhttp.NewHandler(router, "http.server")
			prom := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
			mux.Handle("/", wrapped)
			mux.Handle("/metrics", prom)
			wrappedWithMetrics := mux
			wrappedWithLogger := middleware.OnStartMiddleware(wrappedWithMetrics)
			if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Server.Port), wrappedWithLogger); err != nil {
				// if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Server.Port), wrapped); err != nil {
				panic(fmt.Sprintf("failed to start server: %v", err))
			}

		})

		hooks.OnStop(func() {
			l.Info().Msg("Shutting down API server...")
		})
	})
	cli.Run()
}
