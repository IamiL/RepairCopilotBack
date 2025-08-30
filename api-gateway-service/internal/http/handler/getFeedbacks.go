package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"repairCopilotBot/api-gateway-service/internal/repository"
	"repairCopilotBot/tz-bot/client"
	userserviceclient "repairCopilotBot/user-service/client"
	"time"

	"github.com/google/uuid"
)

func GetFeedbacks(
	log *slog.Logger,
	tzBotClient *client.Client,
	userServiceClient *userserviceclient.UserClient,
	sessionRepo *repository.SessionRepository,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.GetFeedbacks"

		log := log.With(slog.String("op", op))
		log.Info("inspections request started")

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

		// Парсим userID в UUID
		//userID, err := uuid.Parse(userIDStr)
		//if err != nil {
		//	log.Error("invalid user ID format", slog.String("user_id", userIDStr), slog.String("error", err.Error()))
		//	http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		//	return
		//}

		// Создаем контекст с таймаутом
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		// Вызываем gRPC метод GetAllVersionsAdminDashboard
		feedbacks, err := tzBotClient.GetFeedbacks(ctx, uuid.Nil)
		if err != nil {
			log.Error("failed to get inspections", slog.String("error", err.Error()))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if feedbacks != nil && len(*feedbacks) > 0 {
			// Собираем уникальные ID пользователей из versions
			userIDsMap := make(map[string]struct{})
			for _, feedback := range *feedbacks {
				if feedback.FeedbackUser != "" {
					userIDsMap[feedback.FeedbackUser] = struct{}{}
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

					// Обогащаем versions именами
					for _, feedback := range *feedbacks {
						if fullName, exists := fullNames[feedback.FeedbackUser]; exists {
							feedback.User.FirstName = fullName.FirstName
							feedback.User.LastName = fullName.LastName
						}
					}
				}
			}
		}

		//// Обогащаем версии именами пользователей
		//if inspections != nil && len(inspections) > 0 {
		//	// Собираем уникальные ID пользователей из versions
		//	userIDsMap := make(map[string]struct{})
		//	for _, version := range inspections {
		//		if version.UserId != "" {
		//			userIDsMap[version.UserId] = struct{}{}
		//		}
		//	}
		//
		//	// Преобразуем в slice
		//	userIDs := make([]string, 0, len(userIDsMap))
		//	for id := range userIDsMap {
		//		userIDs = append(userIDs, id)
		//	}
		//
		//	// Получаем имена пользователей если есть ID
		//	if len(userIDs) > 0 {
		//		log.Info("fetching user names", slog.Int("user_ids_count", len(userIDs)))
		//
		//		fullNames, err := userServiceClient.GetFullNamesById(ctx, userIDs)
		//		if err != nil {
		//			log.Error("failed to get user names", slog.String("error", err.Error()))
		//		} else {
		//			log.Info("user names fetched successfully", slog.Int("names_count", len(fullNames)))
		//
		//			// Обогащаем versions именами
		//			for _, version := range inspections {
		//				if fullName, exists := fullNames[version.UserId]; exists {
		//					version.User.FirstName = fullName.FirstName
		//					version.User.LastName = fullName.LastName
		//				}
		//			}
		//		}
		//	}
		//}

		// Отдаем данные в JSON формате
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(feedbacks); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
			return
		}

		log.Info("inspections request completed successfully", slog.Int("inspections_count", len(*feedbacks)))
	}
}
