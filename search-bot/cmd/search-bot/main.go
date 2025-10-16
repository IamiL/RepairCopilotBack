package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"repairCopilotBot/search-bot/internal/app/app"
	"repairCopilotBot/search-bot/internal/app/grpc/server"
	"repairCopilotBot/search-bot/internal/config"
	"repairCopilotBot/search-bot/internal/pkg/logger/handlers/slogpretty"
	"repairCopilotBot/search-bot/internal/repository/postgres"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("спарсили конфиг:")

	fmt.Println(*cfg)

	application := app.New(
		log,
		&server.Config{Port: cfg.GRPC.Port},
		&postgres.Config{
			Host:     cfg.Postgres.Host,
			Port:     cfg.Postgres.Port,
			Username: cfg.Postgres.Username,
			Password: cfg.Postgres.Password,
			DBName:   cfg.Postgres.DBName,
			SSLMode:  cfg.Postgres.SSLMode,
		},
		&cfg.LlmClient,
	)

	go func() {
		application.GRPCServer.MustRun()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	application.GRPCServer.Stop()
	log.Info("gracefully stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case "local":
		log = setupPrettySlog()
	case "dev":
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case "prod":
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
