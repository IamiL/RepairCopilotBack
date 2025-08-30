package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"repairCopilotBot/api-gateway-service/internal/repository"
	userserviceclient "repairCopilotBot/user-service/client"
)

type UpdateInspectionsPerDayRequest struct {
	UserID            string `json:"user_id"`
	InspectionsPerDay uint32 `json:"inspections_per_day"`
}

type UpdateInspectionsPerDayResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	UpdatedCount uint32 `json:"updated_count"`
}

func UpdateInspectionsPerDayHandler(
	log *slog.Logger,
	userServiceClient *userserviceclient.UserClient,
	sessionRepo *repository.SessionRepository,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.UpdateInspectionsPerDayHandler"

		log := log.With(slog.String("op", op))
		log.Info("update inspections per day request started")

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

		// Проверяем права администратора
		//if !session.IsAdmin1 && !session.IsAdmin2 {
		//	log.Info("user is not admin", slog.String("user_id", session.UserID))
		//	http.Error(w, "Forbidden: admin access required", http.StatusForbidden)
		//	return
		//}

		// Парсим JSON тело запроса
		var req UpdateInspectionsPerDayRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode request body", slog.String("error", err.Error()))
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Валидация входных данных
		//if req.InspectionsPerDay == 0 {
		//	log.Info("invalid inspections_per_day value", slog.Uint64("value", uint64(req.InspectionsPerDay)))
		//	http.Error(w, "inspections_per_day must be greater than 0", http.StatusBadRequest)
		//	return
		//}

		// Вызываем gRPC метод user-service
		resp, err := userServiceClient.UpdateInspectionsPerDay(r.Context(), req.UserID, req.InspectionsPerDay)
		if err != nil {
			log.Error("failed to update inspections per day", slog.String("error", err.Error()))
			http.Error(w, "Failed to update inspections per day", http.StatusInternalServerError)
			return
		}

		response := UpdateInspectionsPerDayResponse{
			Success:      resp.Success,
			Message:      resp.Message,
			UpdatedCount: resp.UpdatedCount,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
			return
		}

		log.Info("update inspections per day request completed successfully",
			slog.String("user_id", req.UserID),
			slog.Uint64("inspections_per_day", uint64(req.InspectionsPerDay)),
			slog.Uint64("updated_count", uint64(resp.UpdatedCount)))
	}
}
