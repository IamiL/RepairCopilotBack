package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"repairCopilotBot/api-gateway-service/internal/repository"
	chatbotclient "repairCopilotBot/chat-bot/pkg/client"
	userserviceclient "repairCopilotBot/user-service/client"
	"time"
)

type CreateNewMessageRequest struct {
	ChatID  *string `json:"chat_id,omitempty"`
	Message string  `json:"message"`
}

type CreateNewMessageResponse struct {
	ChatID  string `json:"chat_id"`
	Message string `json:"message"`
}

func CreateNewMessageHandler(
	log *slog.Logger,
	sessionRepo *repository.SessionRepository,
	chatBotClient *chatbotclient.ChatBotClient,
	userServiceClient *userserviceclient.UserClient,
	actionLogRepo repository.ActionLogRepository,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.CreateNewMessageHandler"

		log := log.With(slog.String("op", op))
		log.Info("create new message request started")

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

		if session.UserID == "" {
			log.Info("invalid session data - empty user_id")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		log.Debug("userID: "+session.UserID, slog.String("userID", session.UserID))

		// Парсим тело запроса
		var req CreateNewMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode request body", slog.String("error", err.Error()))
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if req.Message == "" {
			log.Info("empty message in request")
			http.Error(w, "Message is required", http.StatusBadRequest)
			return
		}

		// Создаем контекст с таймаутом
		_, cancel := context.WithTimeout(r.Context(), 1000*time.Second)
		defer cancel()

		chatID, responseMessage, err := chatBotClient.Chat.CreateNewMessage(r.Context(), req.ChatID, session.UserID, req.Message)
		if err != nil {
			log.Error("failed to create new message", slog.String("error", err.Error()))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Логируем действие пользователя
		//userID, err := uuid.Parse(session.UserID)
		//if err == nil {
		//	actionText := "Пользователь отправил сообщение в чат"
		//	if err := actionLogRepo.CreateActionLog(ctx, actionText, userID); err != nil {
		//		log.Error("failed to create action log", slog.String("error", err.Error()))
		//	}
		//}

		// Формируем ответ
		response := CreateNewMessageResponse{
			ChatID:  chatID,
			Message: responseMessage,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
			return
		}

		log.Info("create new message request completed successfully",
			slog.String("user_id", session.UserID),
			slog.String("chat_id", chatID))
	}
}
