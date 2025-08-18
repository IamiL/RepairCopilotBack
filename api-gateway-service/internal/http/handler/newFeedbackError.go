package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"repairCopilotBot/api-gateway-service/internal/repository"
	"repairCopilotBot/tz-bot/client"

	"github.com/google/uuid"
)

type NewFeedbackErrorRequest struct {
	VersionID    string `json:"version_id"`
	ErrorID      string `json:"error_id"`
	ErrorType    string `json:"error_type"`
	FeedbackType uint32 `json:"feedback_type"`
	Comment      string `json:"comment"`
}

type NewFeedbackErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func NewFeedbackErrorHandler(
	log *slog.Logger,
	tzBotClient *client.Client,
	sessionRepo *repository.SessionRepository,
	actionLogRepo repository.ActionLogRepository,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.NewFeedbackErrorHandler"

		log := log.With(slog.String("op", op))
		log.Info("processing NewFeedbackError request")

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

		userID, err := uuid.Parse(session.UserID)
		if err != nil {
			log.Error("invalid user ID in session", slog.String("user_id", session.UserID), slog.String("error", err.Error()))
			http.Error(w, "Invalid session", http.StatusInternalServerError)
			return
		}

		// Парсим JSON запрос
		var req NewFeedbackErrorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode request body", slog.String("error", err.Error()))
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Валидация входных данных
		if req.VersionID == "" {
			log.Error("version_id is required")
			http.Error(w, "version_id is required", http.StatusBadRequest)
			return
		}

		if req.ErrorID == "" {
			log.Error("error_id is required")
			http.Error(w, "error_id is required", http.StatusBadRequest)
			return
		}

		if req.ErrorType != "invalid" && req.ErrorType != "missing" {
			log.Error("invalid error_type", slog.String("error_type", req.ErrorType))
			http.Error(w, "error_type must be 'invalid' or 'missing'", http.StatusBadRequest)
			return
		}

		// Валидация UUID
		if _, err := uuid.Parse(req.VersionID); err != nil {
			log.Error("invalid version_id format", slog.String("version_id", req.VersionID), slog.String("error", err.Error()))
			http.Error(w, "invalid version_id format", http.StatusBadRequest)
			return
		}

		// Вызываем tz-bot для создания feedback
		err = tzBotClient.NewFeedbackError(r.Context(), req.VersionID, req.ErrorID, req.ErrorType, req.FeedbackType, req.Comment, userID.String())
		if err != nil {
			log.Error("failed to create feedback error in tz-bot", slog.String("error", err.Error()))
			http.Error(w, "failed to create feedback error", http.StatusInternalServerError)
			return
		}

		// Логируем действие пользователя
		//err = actionLogRepo.LogAction(userID, "feedback_error", map[string]interface{}{
		//	"version_id":    req.VersionID,
		//	"error_id":      req.ErrorID,
		//	"error_type":    req.ErrorType,
		//	"feedback_type": req.FeedbackType,
		//	"comment":       req.Comment,
		//})
		//if err != nil {
		//	log.Error("failed to log action", slog.String("error", err.Error()))
		//	// Не прерываем выполнение, если логирование не удалось
		//}

		response := NewFeedbackErrorResponse{
			Success: true,
			Message: "Feedback created successfully",
		}

		// Возвращаем успешный ответ
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
			return
		}

		log.Info("NewFeedbackError request processed successfully",
			slog.String("user_id", userID.String()),
			slog.String("version_id", req.VersionID),
			slog.String("error_id", req.ErrorID),
			slog.String("error_type", req.ErrorType))
	}
}
