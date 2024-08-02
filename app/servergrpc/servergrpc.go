package servergrpc

import (
	"context"
	"github.com/AlexBlackNn/authloyalty/internal/config"
	"github.com/AlexBlackNn/authloyalty/internal/domain/models"
	authtransport "github.com/AlexBlackNn/authloyalty/internal/grpc_transport/auth"
	authservice "github.com/AlexBlackNn/authloyalty/internal/services/auth_service"
	"github.com/AlexBlackNn/authloyalty/pkg/broker"
	patroni "github.com/AlexBlackNn/authloyalty/pkg/storage/patroni"
	redis "github.com/AlexBlackNn/authloyalty/pkg/storage/redis-sentinel"
	authgen "github.com/AlexBlackNn/authloyalty/protos/proto/sso/gen"
	rkboot "github.com/rookie-ninja/rk-boot"
	rkgrpc "github.com/rookie-ninja/rk-grpc/boot"
	"google.golang.org/grpc"
	"log/slog"
	"time"
)

type UserStorage interface {
	SaveUser(
		ctx context.Context,
		email string,
		passHash []byte,
	) (context.Context, int64, error)
	GetUser(
		ctx context.Context,
		value any,
	) (context.Context, models.User, error)
	Stop() error
}

// TODO: add close as in UserStorage!!!

// TokenStorage describe interface for storages saving revoked tokens
type TokenStorage interface {
	SaveToken(ctx context.Context, token string, ttl time.Duration) (context.Context, error)
	GetToken(ctx context.Context, token string) (context.Context, string, error)
	CheckTokenExists(ctx context.Context, token string) (context.Context, int64, error)
}

type App struct {
	Cfg          *config.Config
	Log          *slog.Logger
	Srv          *rkboot.Boot
	UserStorage  UserStorage
	TokenStorage TokenStorage
	authService  *authservice.Auth
}

func New(cfg *config.Config, log *slog.Logger) (*App, error) {

	userStorage, err := patroni.New(cfg) // Use cfg from the closure
	if err != nil {
		return nil, err
	}
	tokenStorage := redis.New(cfg)

	producer, err := broker.NewProducer(cfg.Kafka.KafkaURL, cfg.Kafka.SchemaRegistryURL)
	if err != nil {
		return nil, err
	}
	authService := authservice.New(cfg, log, userStorage, tokenStorage, producer)

	boot := rkboot.NewBoot()
	// Get grpc entry with name
	grpcEntry := boot.GetEntry("sso").(*rkgrpc.GrpcEntry)
	// Register grpc registration function
	registerAuth := registerAuthFunc(authService)
	grpcEntry.AddRegFuncGrpc(registerAuth)
	// Register grpc-gateway registration function
	grpcEntry.AddRegFuncGw(authgen.RegisterAuthHandlerFromEndpoint)
	// Bootstrap
	return &App{Srv: boot}, nil
}

func registerAuthFunc(authService *authservice.Auth) func(server *grpc.Server) {
	return func(server *grpc.Server) { // Use the provided server
		authtransport.Register(server, authService)
	}
}

func (a *App) Stop() error {
	err := a.UserStorage.Stop()
	if err != nil {
		return err
	}
	// TODO: Close TokenStorage
	return nil
}