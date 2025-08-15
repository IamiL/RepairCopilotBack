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

type TechnicalSpecificationVersion struct {
	VersionId                  string `json:"version_id"`
	TechnicalSpecificationName string `json:"technical_specification_name"`
	VersionNumber              int32  `json:"version_number"`
	CreatedAt                  string `json:"created_at"`
}

type MeResponse struct {
	Login    string                          `json:"login"`
	Level    int                             `json:"level"`
	Versions []TechnicalSpecificationVersion `json:"versions"`
}

func MeHandler(
	log *slog.Logger,
	sessionRepo *repository.SessionRepository,
	tzBotClient *client.Client,
	userServiceClient *userserviceclient.UserClient,
	actionLogRepo repository.ActionLogRepository,
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

		// Получаем версии технических заданий от tz-bot
		var versions []TechnicalSpecificationVersion
		if session.UserID != "" {
			userID, err := uuid.Parse(session.UserID)
			if err != nil {
				log.Error("invalid user ID format in session", slog.String("user_id", session.UserID), slog.String("error", err.Error()))
			} else {
				// Получаем информацию о пользователе для логирования
				userInfo, userInfoErr := userServiceClient.GetUserInfo(r.Context(), session.UserID)
				if userInfoErr == nil {
					// Логируем событие входа на сайт
					actionText := "Пользователь " + userInfo.Login + " зашёл на сайт"
					if err := actionLogRepo.CreateActionLog(r.Context(), actionText, userID); err != nil {
						log.Error("failed to create action log for site access", slog.String("error", err.Error()))
					}
				}

				tzVersions, err := tzBotClient.GetTechnicalSpecificationVersions(r.Context(), userID)
				if err != nil {
					log.Error("failed to get technical specification versions", slog.String("error", err.Error()))
					// Не возвращаем ошибку, продолжаем с пустым массивом версий
				} else {
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
				}
			}
		}

		response := MeResponse{
			Login:    session.Login,
			Level:    level,
			Versions: versions,
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
