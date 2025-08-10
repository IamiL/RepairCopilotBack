package app

import (
	"log/slog"
	grpcapp "repairCopilotBot/user-service/internal/app/grpc"
	"repairCopilotBot/user-service/internal/repository/postgres"
	postgresUser "repairCopilotBot/user-service/internal/repository/postgres/user"
	userservice "repairCopilotBot/user-service/internal/service/user"
)

//type Config struct {
//	TokenTTL time.Duration `yaml:"token_ttl" env-default:"300h"`
//	GRPCPort string        `yaml:"grpc_port" env-default:":50051"`
//}

type App struct {
	GRPCServer *grpcapp.UserGRPCServer
}

func New(
	log *slog.Logger,
	grpcConfig *grpcapp.Config,
	postgresConfig *postgres.Config,
) *App {
	postgresConn, err := postgres.NewConnPool(postgresConfig)
	if err != nil {
		panic(err)
	}

	postgres, err := postgresUser.New(postgresConn)
	if err != nil {
		panic(err)
	}

	usrService := userservice.New(log, postgres, postgres)

	grpcApp := grpcapp.NewUserGRPCServer(log, usrService, grpcConfig)

	return &App{
		GRPCServer: grpcApp,
	}
}
