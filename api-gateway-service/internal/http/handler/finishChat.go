package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"repairCopilotBot/api-gateway-service/internal/repository"
	chatbotclient "repairCopilotBot/chat-bot/pkg/client"
	"time"
)

type FinishChatRequest struct {
	ChatID string `json:"chat_id"`
}

type FinishChatResponse struct {
	Message string `json:"message"`
}

func FinishChatHandler(
	log *slog.Logger,
	sessionRepo *repository.SessionRepository,
	chatBotClient *chatbotclient.ChatBotClient,
	actionLogRepo repository.ActionLogRepository,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.FinishChatHandler"

		log := log.With(slog.String("op", op))
		log.Info("finish chat request started")

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

		// Парсим тело запроса
		var req FinishChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode request body", slog.String("error", err.Error()))
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if req.ChatID == "" {
			log.Info("empty chat_id in request")
			http.Error(w, "chat_id is required", http.StatusBadRequest)
			return
		}

		// Создаем контекст с таймаутом
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		// Вызываем метод клиента chat-bot
		message, err := chatBotClient.Chat.FinishChat(ctx, req.ChatID, session.UserID)
		if err != nil {
			log.Error("failed to finish chat", slog.String("error", err.Error()))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Логируем действие пользователя
		//userID, err := uuid.Parse(session.UserID)
		//if err == nil {
		//	actionText := "Пользователь завершил чат"
		//	if err := actionLogRepo.CreateActionLog(ctx, actionText, userID); err != nil {
		//		log.Error("failed to create action log", slog.String("error", err.Error()))
		//	}
		//}

		// Формируем ответ
		response := FinishChatResponse{
			Message: message,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
			return
		}

		log.Info("finish chat request completed successfully",
			slog.String("user_id", session.UserID),
			slog.String("chat_id", req.ChatID))
	}
}
