package router

import (
	"compress/gzip"
	"log/slog"
	"time"

	_ "github.com/AlexBlackNn/authloyalty/sso/cmd/sso/docs"
	"github.com/AlexBlackNn/authloyalty/sso/internal/config"
	v1 "github.com/AlexBlackNn/authloyalty/sso/internal/handlershttp/http/v1"
	customMiddleware "github.com/AlexBlackNn/authloyalty/sso/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func NewChiRouter(
	cfg *config.Config,
	log *slog.Logger,
	authHandlerV1 v1.AuthHandlers,
	healthHandlerV1 v1.HealthHandlers,
) *chi.Mux {

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	//	Rate limit by IP and URL path (aka endpoint)
	router.Use(httprate.Limit(
		cfg.RateLimit, // requests
		time.Second,   // per duration
		httprate.WithKeyFuncs(httprate.KeyByIP, httprate.KeyByEndpoint),
	))
	router.Use(customMiddleware.Logger(log))

	router.Use(middleware.Recoverer)

	router.Route("/auth", func(r chi.Router) {
		r.Use(customMiddleware.GzipDecompressor(log))
		r.Use(customMiddleware.GzipCompressor(log, gzip.BestCompression))
		r.Get("/ready", healthHandlerV1.ReadinessProbe)
		r.Get("/healthz", healthHandlerV1.LivenessProbe)
		r.Post("/login", authHandlerV1.Login)
		r.Post("/logout", authHandlerV1.Logout)
		r.Post("/registration", authHandlerV1.Register)
		r.Post("/refresh", authHandlerV1.Refresh)
		r.Get("/info", authHandlerV1.Info)
	})
	router.Route("/", func(r chi.Router) {
		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("http://localhost:8000/swagger/doc.json"),
		))
		r.Get("/metrics", promhttp.Handler().ServeHTTP)
	})

	return router
}
