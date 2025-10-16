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

type ChatWithUser struct {
	Id            string    `json:"id"`
	UserId        string    `json:"user_id"`
	CreatedAt     time.Time `json:"created_at"`
	MessagesCount uint32    `json:"messages_count"`
	IsFinished    bool      `json:"is_finished"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
}

type GetAllChatsResponse struct {
	Chats []ChatWithUser `json:"chats"`
}

func GetAllChatsHandler(
	log *slog.Logger,
	sessionRepo *repository.SessionRepository,
	chatBotClient *chatbotclient.ChatBotClient,
	userServiceClient *userserviceclient.UserClient,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.GetAllChatsHandler"

		log := log.With(slog.String("op", op))
		log.Info("get all chats request started")

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

		// Создаем контекст с таймаутом
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		// Вызываем метод клиента search-bot с nil userId для получения всех чатов
		chats, err := chatBotClient.Chat.GetChats(ctx, nil)
		if err != nil {
			log.Error("failed to get all chats", slog.String("error", err.Error()))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Преобразуем чаты в нашу структуру с дополнительными полями
		chatsWithUser := make([]ChatWithUser, 0, len(chats))
		for _, chat := range chats {
			chatsWithUser = append(chatsWithUser, ChatWithUser{
				Id:            chat.Id,
				UserId:        chat.UserId,
				CreatedAt:     chat.CreatedAt,
				MessagesCount: chat.MessagesCount,
				IsFinished:    chat.IsFinished,
				FirstName:     "",
				LastName:      "",
			})
		}

		// Обогащаем чаты именами пользователей
		if len(chatsWithUser) > 0 {
			// Собираем уникальные ID пользователей из чатов
			userIDsMap := make(map[string]struct{})
			for _, chat := range chatsWithUser {
				if chat.UserId != "" {
					userIDsMap[chat.UserId] = struct{}{}
				}
			}

			// Преобразуем в slice
			userIDs := make([]string, 0, len(userIDsMap))
			for id := range userIDsMap {
				userIDs = append(userIDs, id)
			}

			// Получаем имена пользователей если есть ID
			if len(userIDs) > 0 {
				log.Info("fetching user names", slog.Int("user_ids_count", len(userIDs)))

				fullNames, err := userServiceClient.GetFullNamesById(ctx, userIDs)
				if err != nil {
					log.Error("failed to get user names", slog.String("error", err.Error()))
				} else {
					log.Info("user names fetched successfully", slog.Int("names_count", len(fullNames)))

					// Обогащаем чаты именами
					for i := range chatsWithUser {
						if fullName, exists := fullNames[chatsWithUser[i].UserId]; exists {
							chatsWithUser[i].FirstName = fullName.FirstName
							chatsWithUser[i].LastName = fullName.LastName
						}
					}
				}
			}
		}

		// Формируем ответ
		response := GetAllChatsResponse{
			Chats: chatsWithUser,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
			return
		}

		log.Info("get all chats request completed successfully",
			slog.Int("chats_count", len(chats)))
	}
}
