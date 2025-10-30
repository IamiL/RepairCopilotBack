package app

import (
	"fmt"
	"log/slog"
	grpcapp "repairCopilotBot/user-service/internal/app/grpc"
	"repairCopilotBot/user-service/internal/migrator"
	"repairCopilotBot/user-service/internal/repository/postgres"
	postgresUser "repairCopilotBot/user-service/internal/repository/postgres/user"
	userservice "repairCopilotBot/user-service/internal/service/user"

	"github.com/jackc/pgx/v5/stdlib"
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
	mailToken string,
) *App {
	postgresConn, err := postgres.NewConnPool(postgresConfig)
	if err != nil {
		panic(err)
	}

	postgres, err := postgresUser.New(postgresConn)
	if err != nil {
		panic(err)
	}

	migratorRunner := migrator.NewMigrator(stdlib.OpenDB(*postgresConn.Config().ConnConfig.Copy()), postgresConfig.MigrationsDir)

	err = migratorRunner.Up()
	if err != nil {
		log.Error("Ошибка миграции базы данных: %v\n", err)
		panic(fmt.Errorf("cannot run migrator - %w", err).Error())
	}

	usrService := userservice.New(log, postgres, postgres, mailToken)

	grpcApp := grpcapp.NewUserGRPCServer(log, usrService, grpcConfig)

	return &App{
		GRPCServer: grpcApp,
	}
}
