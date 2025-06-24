package httpapp

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"repairCopilotBot/tz-bot/internal/http/handler"
	"repairCopilotBot/tz-bot/internal/pkg/logger/sl"
	tz_llm_client "repairCopilotBot/tz-bot/package/llm"
	tg_client "repairCopilotBot/tz-bot/package/tg"
	word_parser_client "repairCopilotBot/tz-bot/package/word-parser"
	"strconv"

	"time"
)

type Config struct {
	Port    int           `yaml:"port" default:"8080"`
	Timeout time.Duration `yaml:"timeout"`
}

type App struct {
	log        *slog.Logger
	httpServer *http.Server
	port       int
	Tls        bool
}

func New(
	log *slog.Logger,
	config *Config,
	wordConverterClient *word_parser_client.Client,
	llmClient *tz_llm_client.Client,
	tgClient *tg_client.Client,
) *App {
	router := http.NewServeMux()

	router.HandleFunc(
		"POST /tz",
		handler.NewTzHandler(log, wordConverterClient, llmClient, tgClient),
	)

	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(config.Port),
		Handler: router,
	}

	return &App{log: log, httpServer: srv, port: config.Port}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *App) Run() error {
	const op = "httpapp.Run"

	a.log.With(slog.String("op", op)).
		Info("server started", slog.Int("port", a.port))

	if a.Tls {
		if err := a.httpServer.ListenAndServeTLS(
			"server.crt",
			"server.key",
		); err != nil {
			a.log.Error("failed to start https server", sl.Err(err))
		}
	} else {
		if err := a.httpServer.ListenAndServe(); err != nil {
			a.log.Error("failed to start http server", sl.Err(err))
		}
	}

	return nil
}

func (a *App) Stop() {
	const op = "httpapp.Stop"

	a.log.With(slog.String("op", op)).
		Info("stopping HTTP server", slog.Int("port", a.port))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.httpServer.Shutdown(ctx); err != nil {
		a.log.Error("server closed with err: %+v", err)
		os.Exit(1)
	}

	a.log.Info("Gracefully stopped")
}
