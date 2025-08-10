package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"repairCopilotBot/api-gateway-service/internal/repository"
)

type MeResponse struct {
	Login string `json:"login"`
	Level int    `json:"level"`
}

func MeHandler(
	log *slog.Logger,
	sessionRepo *repository.SessionRepository,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.MeHandler"

		log := log.With(slog.String("op", op))
		log.Info("processing /me request")

		// Получаем токен из куки
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			log.Info("no auth token cookie found")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token := cookie.Value
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

		if session.Login == "" {
			log.Info("invalid session data - empty login")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Определяем уровень пользователя из данных сессии
		level := 0
		if session.IsAdmin1 {
			level = 1
		} else if session.IsAdmin2 {
			level = 2
		}

		response := MeResponse{
			Login: session.Login,
			Level: level,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
			return
		}

		log.Info("me request processed successfully",
			slog.String("login", session.Login),
			slog.Int("level", level))
	}
}
