package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"repairCopilotBot/api-gateway-service/internal/repository"
	"repairCopilotBot/tz-bot/client"
	userserviceclient "repairCopilotBot/user-service/client"

	"github.com/google/uuid"
)

type GetUserInfoResponse struct {
	UserID    string                          `json:"user_id"`
	Login     string                          `json:"login"`
	IsAdmin1  bool                            `json:"is_admin1"`
	IsAdmin2  bool                            `json:"is_admin2"`
	CreatedAt string                          `json:"created_at"`
	Versions  []TechnicalSpecificationVersion `json:"versions"`
}

func GetUserInfoHandler(
	log *slog.Logger,
	sessionRepo *repository.SessionRepository,
	tzBotClient *client.Client,
	userServiceClient *userserviceclient.UserClient,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.GetUserInfoHandler"

		log := log.With(slog.String("op", op))
		log.Info("get user info request started")

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

		// Получаем user_id из URL path параметра
		userIDStr := r.PathValue("user_id")
		if userIDStr == "" {
			log.Error("user_id path parameter is required")
			http.Error(w, "user_id is required", http.StatusBadRequest)
			return
		}

		// Валидируем UUID формат
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			log.Error("invalid user_id format", slog.String("user_id", userIDStr), slog.String("error", err.Error()))
			http.Error(w, "invalid user_id format", http.StatusBadRequest)
			return
		}

		log = log.With(slog.String("target_user_id", userIDStr))

		// Получаем информацию о пользователе из user-service
		userInfo, err := userServiceClient.GetUserInfo(r.Context(), userID)
		if err != nil {
			log.Error("failed to get user info", slog.String("error", err.Error()))
			http.Error(w, "Failed to get user info", http.StatusInternalServerError)
			return
		}

		// Получаем версии технических заданий от tz-bot для указанного пользователя
		var versions []TechnicalSpecificationVersion
		tzVersions, err := tzBotClient.GetTechnicalSpecificationVersions(r.Context(), userID)
		if err != nil {
			log.Error("failed to get technical specification versions", slog.String("error", err.Error()))
			http.Error(w, "Failed to get versions", http.StatusInternalServerError)
			return
		}

		// Конвертируем из client.TechnicalSpecificationVersion в handler.TechnicalSpecificationVersion
		versions = make([]TechnicalSpecificationVersion, len(tzVersions))
		for i, tzVersion := range tzVersions {
			versions[i] = TechnicalSpecificationVersion{
				VersionId:                  tzVersion.VersionId,
				TechnicalSpecificationName: tzVersion.TechnicalSpecificationName,
				VersionNumber:              tzVersion.VersionNumber,
				CreatedAt:                  tzVersion.CreatedAt,
			}
		}

		response := GetUserInfoResponse{
			UserID:    userIDStr,
			Login:     userInfo.Login,
			IsAdmin1:  userInfo.IsAdmin1,
			IsAdmin2:  userInfo.IsAdmin2,
			CreatedAt: userInfo.RegisteredAt.String(),
			Versions:  versions,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
			return
		}

		log.Info("get user info request completed successfully",
			slog.String("target_user_id", userIDStr),
			slog.String("login", userInfo.Login),
			slog.Int("versions_count", len(versions)))
	}
}
