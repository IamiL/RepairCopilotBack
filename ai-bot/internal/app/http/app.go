package httpApp

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"repairCopilotBot/ai-bot/internal/JWTsecret"
	"repairCopilotBot/ai-bot/internal/handler/htttp"
	"repairCopilotBot/ai-bot/internal/pkg/logger/sl"
	"strconv"
	"time"
)

type Config struct {
	Port    int           `yaml:"port" default:"8080"`
	Timeout time.Duration `yaml:"timeout"`
	Addr    string        `yaml:"address" default:"0.0.0.0"`
}

type App struct {
	log        *slog.Logger
	httpServer *http.Server
	port       int
}

func New(
	log *slog.Logger,
	config *Config,
) *App {
	router := http.NewServeMux()

	Storage := httpHandler.MessagesStorage{
		Storage: make(map[string][]httpHandler.Message),
	}

	secretStorage := JWTsecret.NewJWTSecret([]byte("secret"))

	//secret := []byte(`j12sdJASLHDgfvsd`)

	router.HandleFunc("POST /api/chat", httpHandler.StartChatHandler(log, &Storage, secretStorage))
	router.HandleFunc("DELETE /api/chat", httpHandler.EndChatHandler(log, &Storage, secretStorage))
	router.HandleFunc("POST /api/message", httpHandler.NewMessageHandler(log, &Storage, secretStorage, `/api/message`))
	router.HandleFunc("GET /api/message", httpHandler.GetMessangesHandler(log, &Storage, secretStorage))

	router.HandleFunc("POST /chat", httpHandler.StartChatHandler(log, &Storage, secretStorage))
	router.HandleFunc("DELETE /chat", httpHandler.EndChatHandler(log, &Storage, secretStorage))
	router.HandleFunc("POST /message", httpHandler.NewMessageHandler(log, &Storage, secretStorage, `/message`))
	router.HandleFunc("GET /message", httpHandler.GetMessangesHandler(log, &Storage, secretStorage))
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

	if err := a.httpServer.ListenAndServe(); err != nil {
		a.log.Error("failed to start http server", sl.Err(err))
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
