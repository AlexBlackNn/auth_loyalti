package serverhttp

import (
	"fmt"
	"github.com/AlexBlackNn/authloyalty/cmd/sso/router"
	"github.com/AlexBlackNn/authloyalty/internal/config"
	handlers "github.com/AlexBlackNn/authloyalty/internal/handlersapi/v1"
	authservice "github.com/AlexBlackNn/authloyalty/internal/services/auth_service"
	"log/slog"
	"net/http"
	"time"
)

// App service consists all entities needed to work.
type App struct {
	Cfg           *config.Config
	Log           *slog.Logger
	Srv           *http.Server
	authService   *authservice.Auth
	HandlersV1    handlers.AuthHandlers
	HealthChecker handlers.HealthHandlers
}

// New creates App collecting handlers and server
func New(
	cfg *config.Config,
	log *slog.Logger,
	authService *authservice.Auth,
) (*App, error) {

	projectHandlersV1 := handlers.New(log, authService)
	healthHandlersV1 := handlers.NewHealth(log, authService)
	srv := &http.Server{
		Addr: fmt.Sprintf(cfg.Address),
		Handler: router.NewChiRouter(
			cfg,
			log,
			projectHandlersV1,
			healthHandlersV1,
		),
		ReadTimeout:  time.Duration(cfg.ServerTimeout.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.ServerTimeout.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.ServerTimeout.IdleTimeout) * time.Second,
	}
	return &App{
		Cfg: cfg,
		Log: log,
		Srv: srv,
	}, nil
}
