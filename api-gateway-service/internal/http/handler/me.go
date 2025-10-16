package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"repairCopilotBot/api-gateway-service/internal/repository"
	chatbotclient "repairCopilotBot/chat-bot/pkg/client"
	chatbotclientChat "repairCopilotBot/chat-bot/pkg/client/chat"
	searchbotclient "repairCopilotBot/search-bot/pkg/client"
	searchbotclientChat "repairCopilotBot/search-bot/pkg/client/chat"
	"repairCopilotBot/tz-bot/client"
	userserviceclient "repairCopilotBot/user-service/client"
	"time"

	"github.com/google/uuid"
)

//type TechnicalSpecificationVersion struct {
//	VersionId                  string `json:"version_id"`
//	TechnicalSpecificationName string `json:"technical_specification_name"`
//	VersionNumber              int32  `json:"version_number"`
//	CreatedAt                  string `json:"created_at"`
//}

type MeResponse struct {
	//Login       string                         `json:"login"`
	Level int `json:"level"`
	//FirstName   string                         `json:"firstName"`
	//LastName    string                         `json:"lastName"`
	Versions    []*client.GetVersionMeResponse `json:"versions"`
	Chats       []chatbotclientChat.Chat       `json:"chats"`
	SearchChats []searchbotclientChat.Chat     `json:"searchChats"`
	//Email       string                         `json:"email"`
	//IsConfirmed bool                           `json:"is_confirmed"`
	*userserviceclient.GetUserInfoResponse
}

func MeHandler(
	log *slog.Logger,
	sessionRepo *repository.SessionRepository,
	tzBotClient *client.Client,
	userServiceClient *userserviceclient.UserClient,
	chatBotClient *chatbotclient.ChatBotClient,
	searchbotclient *searchbotclient.SearchBotClient,
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

		var userInfo *userserviceclient.GetUserInfoResponse
		// Определяем уровень пользователя из данных сессии
		level := 0
		var tzVersions []*client.GetVersionMeResponse
		var chats []chatbotclientChat.Chat
		var searchChats []searchbotclientChat.Chat
		// Получаем версии технических заданий от tz-bot
		if session.UserID != "" {
			userID, err := uuid.Parse(session.UserID)
			if err != nil {
				log.Error("invalid user ID format in session", slog.String("user_id", session.UserID), slog.String("error", err.Error()))
			} else {
				// Получаем информацию о пользователе для логирования
				userInfoResp, userInfoErr := userServiceClient.GetUserInfo(r.Context(), userID)
				if userInfoErr == nil {
					// Логируем событие входа на сайт
					//actionText := "Пользователь " + userInfo.FirstName + " " + userInfo.LastName + " зашёл на сайт"
					//if err := actionLogRepo.CreateActionLog(r.Context(), actionText, userID); err != nil {
					//	log.Error("failed to create action log for site access", slog.String("error", err.Error()))
					//}

					userInfo = userInfoResp

					// Асинхронно регистрируем посещение пользователя
					go func() {
						ctx, cancelFunc := context.WithTimeout(context.Background(), time.Minute)
						defer cancelFunc()

						if err := userServiceClient.RegisterVisit(ctx, userID.String()); err != nil {
							log.Error("failed to register user visit", slog.String("user_id", userID.String()), slog.String("error", err.Error()))
						}
					}()
					if userInfo.IsAdmin1 {
						level = 1
					} else if userInfo.IsAdmin2 {
						level = 2
					}
				}

				if userInfo != nil && userInfo.IsConfirmed {
					tzVersions, err = tzBotClient.GetVersionsMe(r.Context(), userID)
					if err != nil || tzVersions == nil {
						log.Error("failed to get technical specification versions", slog.String("error", err.Error()))
						// Не возвращаем ошибку, продолжаем с пустым массивом версий
					}

					// Получаем чаты пользователя
					userIDString := userID.String()
					chats, err = chatBotClient.Chat.GetChats(r.Context(), &userIDString)
					if err != nil {
						log.Error("failed to get user chats", slog.String("error", err.Error()))
						// Не возвращаем ошибку, продолжаем с пустым массивом чатов
						chats = []chatbotclientChat.Chat{}
					}

					searchChats, err = searchbotclient.Chat.GetChats(r.Context(), &userIDString)
					if err != nil {
						log.Error("failed to get user chats", slog.String("error", err.Error()))
						// Не возвращаем ошибку, продолжаем с пустым массивом чатов
						chats = []chatbotclientChat.Chat{}
					}
				}
			}
		}

		response := MeResponse{
			Level:               level,
			Versions:            tzVersions,
			Chats:               chats,
			SearchChats:         searchChats,
			GetUserInfoResponse: userInfo,
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
