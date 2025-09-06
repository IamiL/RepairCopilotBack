package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"repairCopilotBot/api-gateway-service/internal/repository"
	userserviceclient "repairCopilotBot/user-service/client"
	"time"

	"github.com/google/uuid"
)

type CheckInspectionLimitResponse struct {
	Status         string `json:"status"`                    // "success" или "limit_exhausted"
	InspectionsLeft *uint32 `json:"inspections_left,omitempty"` // Количество оставшихся проверок (если лимит не исчерпан)
	TimeUntilReset *string `json:"time_until_reset,omitempty"`  // Время до полуночи в формате "HH:MM:SS" (если лимит исчерпан)
}

// calculateTimeUntilMidnight вычисляет время до полуночи
func calculateTimeUntilMidnight() string {
	now := time.Now()
	
	// Находим следующую полночь
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	
	// Вычисляем разность
	duration := nextMidnight.Sub(now)
	
	// Преобразуем в часы, минуты, секунды
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60
	
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func CheckInspectionLimitHandler(
	log *slog.Logger,
	sessionRepo *repository.SessionRepository,
	userServiceClient *userserviceclient.UserClient,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.CheckInspectionLimitHandler"

		log := log.With(slog.String("op", op))
		log.Info("processing check inspection limit request")

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
			log.Info("invalid session data - empty user ID")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Парсим UserID
		userID, err := uuid.Parse(session.UserID)
		if err != nil {
			log.Error("invalid user ID format in session", slog.String("user_id", session.UserID), slog.String("error", err.Error()))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Проверяем лимит проверок
		inspectionsLeft, err := userServiceClient.CheckInspectionLimit(r.Context(), userID.String())
		if err != nil {
			// Проверяем, является ли это ошибкой исчерпанного лимита
			var limitErr userserviceclient.CheckInspectionLimitError
			if errors.As(err, &limitErr) {
				// Лимит исчерпан - возвращаем время до полуночи
				timeUntilReset := calculateTimeUntilMidnight()
				response := CheckInspectionLimitResponse{
					Status:          "limit_exhausted",
					TimeUntilReset:  &timeUntilReset,
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				if err := json.NewEncoder(w).Encode(response); err != nil {
					log.Error("failed to encode limit exhausted response", slog.String("error", err.Error()))
					return
				}

				log.Info("inspection limit check completed - limit exhausted",
					slog.String("user_id", userID.String()),
					slog.String("time_until_reset", timeUntilReset))
				return
			}

			// Другие ошибки - internal server error
			log.Error("failed to check inspection limit", slog.String("error", err.Error()))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Лимит не исчерпан - возвращаем количество оставшихся проверок
		response := CheckInspectionLimitResponse{
			Status:          "success",
			InspectionsLeft: &inspectionsLeft,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode success response", slog.String("error", err.Error()))
			return
		}

		log.Info("inspection limit check completed successfully",
			slog.String("user_id", userID.String()),
			slog.Uint64("inspections_left", uint64(inspectionsLeft)))
	}
}