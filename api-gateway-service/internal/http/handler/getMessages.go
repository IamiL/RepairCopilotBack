package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"repairCopilotBot/api-gateway-service/internal/repository"
	chatbotclient "repairCopilotBot/chat-bot/pkg/client"
	chatbotclientChat "repairCopilotBot/chat-bot/pkg/client/chat"
	"time"
)

type GetMessagesResponse struct {
	Messages []chatbotclientChat.Message `json:"messages"`
}

func GetMessagesHandler(
	log *slog.Logger,
	sessionRepo *repository.SessionRepository,
	chatBotClient *chatbotclient.ChatBotClient,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.GetMessagesHandler"

		log := log.With(slog.String("op", op))
		log.Info("get messages request started")

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

		// Получаем chat_id из URL параметра
		chatID := r.PathValue("chat_id")
		if chatID == "" {
			log.Info("chat_id parameter is required")
			http.Error(w, "chat_id is required", http.StatusBadRequest)
			return
		}

		// Создаем контекст с таймаутом
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		// Вызываем метод клиента chat-bot
		messages, err := chatBotClient.Chat.GetMessages(ctx, chatID)
		if err != nil {
			log.Error("failed to get messages", slog.String("error", err.Error()))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Формируем ответ
		response := GetMessagesResponse{
			Messages: messages,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
			return
		}

		log.Info("get messages request completed successfully",
			slog.String("chat_id", chatID),
			slog.Int("messages_count", len(messages)))
	}
}
