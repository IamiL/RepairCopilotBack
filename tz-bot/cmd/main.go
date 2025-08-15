package main

import (
	"log/slog"
	"os"
	"os/signal"
	"repairCopilotBot/tz-bot/internal/app"
	"repairCopilotBot/tz-bot/internal/config"
	"repairCopilotBot/tz-bot/internal/pkg/logger/handlers/slogpretty"
	"syscall"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	application := app.New(
		log,
		&cfg.GRPC,
		&cfg.Llm,
		&cfg.WordParser,
		&cfg.MarkdownService,
		&cfg.PromtBuilder,
		&cfg.Tg,
		&cfg.S3minio,
		&cfg.Postgres,
	)

	application.GRPCServer.MustRun()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop
	log.Info("stopping server")

	application.GRPCServer.Stop()

	log.Info("server stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(
				os.Stdout,
				&slog.HandlerOptions{Level: slog.LevelDebug},
			),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(
				os.Stdout,
				&slog.HandlerOptions{Level: slog.LevelInfo},
			),
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
