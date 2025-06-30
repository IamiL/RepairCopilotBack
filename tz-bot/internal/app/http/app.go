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

	routerWithCorsHandler := corsMiddleware(log, router)

	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(config.Port),
		Handler: routerWithCorsHandler,
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

func corsMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			log.Info("origin: ", r.Header.Get("origin"))

			// Определяем Origin запроса
			origin := r.Header.Get("Origin")
			allowedOrigins := map[string]bool{
				"http://tauri.localhost":    true,
				"https://localhost:3000":    true, // React dev server
				"http://195.19.39.177:3000": true,
				"https://iamil.github.io":   true,
				"http://localhost:8002":     true, // Swagger UI
				"http://localhost:5173":     true,
				"http://localhost:4173":     true,
				"http://timuroid.ru":        true,
				"www.timuroid.ru":           true,
				"http://www.timuroid.ru":    true,
				"http://timuroid.ru/":       true,
			}

			// Устанавливаем CORS заголовки только для разрешенных origins
			if allowedOrigins[origin] {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set(
					"Access-Control-Allow-Methods",
					"GET, POST, OPTIONS, PUT, DELETE, PATCH, HEAD",
				)
				w.Header().Set(
					"Access-Control-Allow-Headers",
					"Origin, Content-Type, Authorization, Accept, X-Requested-With, X-Access-Token",
				)
				w.Header().Set(
					"Access-Control-Expose-Headers",
					"Content-Length",
				)
				w.Header().Set("Access-Control-Max-Age", "43200") // 12 hours
			}

			// Кэширование и другие заголовки для React приложения
			w.Header().Set(
				"Cache-Control",
				"no-store, no-cache, must-revalidate, private",
			)
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")

			// Если это OPTIONS запрос, возвращаем пустой ответ
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			// Передаем управление следующему обработчику
			next.ServeHTTP(w, r)
		},
	)
}
