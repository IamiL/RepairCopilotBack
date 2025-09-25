package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"repairCopilotBot/api-gateway-service/internal/repository"
	chatbotclient "repairCopilotBot/chat-bot/pkg/client"
	userserviceclient "repairCopilotBot/user-service/client"

	"github.com/google/uuid"
)

type ConfirmEmailRequest struct {
	Code string `json:"code"`
}

func ConfirmEmail(
	log *slog.Logger,
	userServiceClient *userserviceclient.UserClient,
	sessionRepo *repository.SessionRepository,
	chatBotClient *chatbotclient.ChatBotClient,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.ConfirmEmailHandler"

		log := log.With(slog.String("op", op))
		log.Info("processing confirm email request")

		var req ConfirmEmailRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.With(slog.String("op", op)).Error("failed to decode request", slog.String("error", err.Error()))
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Code == "" {
			log.With(slog.String("op", op)).Error("missing code")
			http.Error(w, "Code are required", http.StatusBadRequest)
			return
		}

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

		err = userServiceClient.ConfirmEmail(r.Context(), uuid.MustParse(session.UserID), req.Code)
		if err != nil {
			log.Info("failed to confirm email", slog.String("error", err.Error()))
			http.Error(w, "Unauthorized", http.StatusBadRequest)
			return
		}

		err = chatBotClient.User.CreateNewUser(r.Context(), session.UserID)
		if err != nil {
			log.Error("failed to create new user in chat-bot service", slog.String("error", err.Error()))
		}
	}
}
