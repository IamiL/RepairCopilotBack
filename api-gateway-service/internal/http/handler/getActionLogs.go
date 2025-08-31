package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"repairCopilotBot/api-gateway-service/internal/pkg/logger/sl"
	"repairCopilotBot/api-gateway-service/internal/repository"
)

func GetActionLogsHandler(
	log *slog.Logger,
	actionLogRepo repository.ActionLogRepository,
	sessionRepo *repository.SessionRepository,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.GetActionLogsHandler"
		log := log.With(slog.String("op", op))

		sessionCookie, err := r.Cookie("auth_token")
		if err != nil {
			log.Warn("no session cookie")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token := sessionCookie.Value
		if token == "" {
			log.Info("empty auth token")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Проверяем сессию в Redis
		session, err := sessionRepo.GetSession(token)
		if err != nil {
			log.Info("failed to get session from Redis", slog.String("error", err.Error()))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if session == nil {
			log.Info("session not found")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		logs, err := actionLogRepo.GetAllActionLogs(r.Context())
		if err != nil {
			log.Error("failed to get action logs", sl.Err(err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(logs); err != nil {
			log.Error("failed to encode response", sl.Err(err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}
}
