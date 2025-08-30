package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"repairCopilotBot/api-gateway-service/internal/repository"
	"repairCopilotBot/tz-bot/client"
	tzv1 "repairCopilotBot/tz-bot/pkg/tz/v1"
	userserviceclient "repairCopilotBot/user-service/client"

	"github.com/google/uuid"
)

type UserTechnicalSpecificationVersion struct {
	VersionId                  string  `json:"version_id"`
	TechnicalSpecificationName string  `json:"technical_specification_name"`
	VersionNumber              int32   `json:"version_number"`
	CreatedAt                  string  `json:"created_at"`
	OriginalFileLink           string  `json:"original_file_link"`
	ReportFileLink             *string `json:"report_file_link"`
}

type GetUserByIdResponse struct {
	*userserviceclient.GetUserInfoResponse
	//Name                string                              `json:"firstName"`
	//Surname             string                              `json:"lastName"`
	//Email               string                              `json:"email"`
	//Login               string                              `json:"login"`
	//IsAdmin1            bool                                `json:"isAdmin1"`
	//IsAdmin2            bool                                `json:"isAdmin2"`
	//RegisteredAt        string                              `json:"registeredAt"`
	//LastVisitAt         string                              `json:"lastVisitAt"`
	//InspectionsCount    int                                 `json:"inspectionsCount"`
	//ErrorFeedbackCount  int                                 `json:"errorFeedbackCount"`
	//InspectionsPerDay   int                                 `json:"inspectionsPerDay"`
	//InspectionsForToday int                                 `json:"inspectionsForToday"`
	Versions  []UserTechnicalSpecificationVersion `json:"versions"`
	Feedbacks []*tzv1.FeedbackInstance            `json:"feedbacks"`
}

func GetUserByIdHandler(
	log *slog.Logger,
	userServiceClient *userserviceclient.UserClient,
	sessionRepo *repository.SessionRepository,
	tzBotClient *client.Client,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.GetUserByIdHandler"

		log := log.With(slog.String("op", op))
		log.Info("get user by id request started")

		// Получаем токен из куки для проверки аутентификации
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

		// Получаем user_id из URL параметров
		userIDStr := r.PathValue("user_id")
		if userIDStr == "" {
			log.Info("user_id parameter is missing")
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		// Парсим userID в UUID
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			log.Error("invalid user ID format", slog.String("user_id", userIDStr), slog.String("error", err.Error()))
			http.Error(w, "Invalid user ID format", http.StatusBadRequest)
			return
		}

		log.Info("fetching user info", slog.String("user_id", userIDStr))

		// Получаем информацию о пользователе из user-service
		userInfo, err := userServiceClient.GetUserInfo(r.Context(), userID)
		if err != nil {
			log.Error("failed to get user info", slog.String("error", err.Error()))
			http.Error(w, "Failed to get user info", http.StatusInternalServerError)
			return
		}

		// Получаем версии технических заданий пользователя
		var versions []UserTechnicalSpecificationVersion
		tzVersions, err := tzBotClient.GetVersionsMe(r.Context(), userID)
		if err != nil {
			log.Error("failed to get technical specification versions", slog.String("error", err.Error()))
			// Не возвращаем ошибку, продолжаем с пустым массивом версий
			versions = []UserTechnicalSpecificationVersion{}
		} else {
			// Конвертируем в response структуру
			versions = make([]UserTechnicalSpecificationVersion, len(tzVersions))
			for i, tzVersion := range tzVersions {
				versions[i] = UserTechnicalSpecificationVersion{
					VersionId:                  tzVersion.VersionId,
					TechnicalSpecificationName: tzVersion.TechnicalSpecificationName,
					VersionNumber:              tzVersion.VersionNumber,
					CreatedAt:                  tzVersion.CreatedAt.String(),
					OriginalFileLink:           tzVersion.OriginalFileLink,
					ReportFileLink:             tzVersion.ReportFileLink,
				}
			}
		}

		log.Info("user versions fetched", slog.Int("versions_count", len(versions)))

		feedbacks, err := tzBotClient.GetFeedbacks(r.Context(), userID)
		if err != nil {
			log.Error("failed to get inspections", slog.String("error", err.Error()))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Конвертируем в response структуру с правильными JSON тегами
		response := GetUserByIdResponse{userInfo,
			versions, feedbacks.Feedbacks}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
			return
		}

		log.Info("get user by id request completed successfully",
			slog.String("user_id", userIDStr),
			slog.String("login", userInfo.Login),
			slog.Int("versions_count", len(versions)))
	}
}
