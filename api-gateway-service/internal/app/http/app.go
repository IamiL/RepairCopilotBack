package httpapp

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"repairCopilotBot/api-gateway-service/internal/http/handler"
	"repairCopilotBot/api-gateway-service/internal/pkg/logger/sl"
	"repairCopilotBot/api-gateway-service/internal/repository"
	chatbotclient "repairCopilotBot/chat-bot/pkg/client"
	searchbotclient "repairCopilotBot/search-bot/pkg/client"
	"repairCopilotBot/tz-bot/client"
	userserviceclient "repairCopilotBot/user-service/client"
	"strconv"

	"time"
)

type Config struct {
	Port    int           `env:"PORT" env-default:"8080"`
	Timeout time.Duration `env:"TIMEOUT" env-default:"1000s"`
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
	tzBotClient *client.Client,
	userServiceClient *userserviceclient.UserClient,
	chatBotClient *chatbotclient.ChatBotClient,
	searchBotClient *searchbotclient.SearchBotClient,
	sessionRepo *repository.SessionRepository,
	actionLogRepo repository.ActionLogRepository,
) *App {
	router := http.NewServeMux()

	router.HandleFunc(
		"POST /api/tz",
		handler.NewTzHandler(log, tzBotClient, sessionRepo, userServiceClient, actionLogRepo),
	)

	router.HandleFunc(
		"POST /api/users/login",
		handler.LoginHandler(log, userServiceClient, sessionRepo, tzBotClient, actionLogRepo),
	)

	router.HandleFunc(
		"POST /api/register",
		handler.RegisterHandler(log, userServiceClient, sessionRepo, actionLogRepo),
	)

	router.HandleFunc("POST /api/confirm-email",
		handler.ConfirmEmail(log, userServiceClient, sessionRepo, chatBotClient))

	router.HandleFunc("POST /api/confirm",
		handler.ConfirmEmail(log, userServiceClient, sessionRepo, chatBotClient))

	router.HandleFunc("POST /api/users/recovery",
		handler.RecoveryHandler(log, userServiceClient))

	router.HandleFunc(
		"GET /api/logout",
		handler.LogoutHandler(log))

	router.HandleFunc(
		"GET /api/me",
		handler.MeHandler(log, sessionRepo, tzBotClient, userServiceClient, chatBotClient, searchBotClient),
	)

	router.HandleFunc(
		"GET /api/tz/{version_id}",
		handler.GetVersionHandler(log, tzBotClient),
	)

	router.HandleFunc(
		"GET /api/users",
		handler.GetUsersHandler(log, userServiceClient, sessionRepo),
	)

	router.HandleFunc(
		"GET /api/users/{user_id}/info",
		handler.GetUserInfoHandler(log, sessionRepo, tzBotClient, userServiceClient),
	)

	router.HandleFunc(
		"GET /api/users/{user_id}",
		handler.GetUserByIdHandler(log, userServiceClient, sessionRepo, tzBotClient),
	)

	router.HandleFunc(
		"GET /api/action-logs",
		handler.GetActionLogsHandler(log, actionLogRepo, sessionRepo),
	)

	router.HandleFunc(
		"GET /api/admin/dashboard",
		handler.GetAdminDashboardHandler(log, userServiceClient, tzBotClient, sessionRepo, actionLogRepo),
	)

	router.HandleFunc(
		"GET /api/admin/inspections",
		handler.GetInspectionsHandler(log, tzBotClient, userServiceClient, sessionRepo),
	)

	router.HandleFunc(
		"POST /api/feedback",
		handler.NewFeedbackErrorHandler(log, tzBotClient, userServiceClient, sessionRepo, actionLogRepo),
	)

	router.HandleFunc(
		"POST /api/feedback/verification",
		handler.NewFeedbackVerificationErrorHandler(log, tzBotClient, userServiceClient, sessionRepo, actionLogRepo),
	)

	router.HandleFunc(
		"GET /api/analytics/billing/limits",
		handler.GetBillingLimits(log, tzBotClient),
	)

	router.HandleFunc(
		"GET /api/analytics/billing/daily",
		handler.GetBillingDaily(log, tzBotClient),
	)

	router.HandleFunc(
		"GET /api/feedbacks",
		handler.GetFeedbacks(log, tzBotClient, userServiceClient, sessionRepo),
	)

	router.HandleFunc(
		"POST /api/admin/users/update-inspections-per-day",
		handler.UpdateInspectionsPerDayHandler(log, userServiceClient, sessionRepo, actionLogRepo),
	)

	router.HandleFunc(
		"POST /api/admin/users/change-role",
		handler.ChangeUserRoleHandler(log, userServiceClient, sessionRepo, actionLogRepo),
	)

	router.HandleFunc(
		"GET /api/users/inspection-limit",
		handler.CheckInspectionLimitHandler(log, sessionRepo, userServiceClient),
	)

	// Chat Bot routes
	router.HandleFunc(
		"POST /api/searchchat/message",
		handler.CreateNewSearchMessageHandler(log, sessionRepo, searchBotClient, userServiceClient, actionLogRepo),
	)

	router.HandleFunc(
		"GET /api/searchchat/{chat_id}/messages",
		handler.GetSearchMessagesHandler(log, sessionRepo, searchBotClient),
	)

	// Chat Bot routes
	router.HandleFunc(
		"POST /api/chat/message",
		handler.CreateNewMessageHandler(log, sessionRepo, chatBotClient, userServiceClient, actionLogRepo),
	)

	router.HandleFunc(
		"GET /api/chat/{chat_id}/messages",
		handler.GetMessagesHandler(log, sessionRepo, chatBotClient),
	)

	router.HandleFunc(
		"POST /api/chat/finish",
		handler.FinishChatHandler(log, sessionRepo, chatBotClient, actionLogRepo),
	)

	router.HandleFunc(
		"GET /api/admin/chats/all",
		handler.GetAllChatsHandler(log, sessionRepo, chatBotClient, userServiceClient),
	)

	routerWithCorsHandler := corsMiddleware(log, router)

	srv := &http.Server{
		Addr:         ":" + strconv.Itoa(config.Port),
		Handler:      routerWithCorsHandler,
		IdleTimeout:  30 * time.Minute,
		ReadTimeout:  30 * time.Minute,
		WriteTimeout: 30 * time.Minute,
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
				"http://localhost:3006":     true,
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
