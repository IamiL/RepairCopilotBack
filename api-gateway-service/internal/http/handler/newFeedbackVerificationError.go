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

type NewFeedbackVerificationErrorRequest struct {
	InstanceID      string  `json:"instance_id"`
	InstanceType    string  `json:"instance_type"`
	FeedbackMark    *bool   `json:"feedback_mark"`
	FeedbackComment *string `json:"feedback_comment"`
}

type NewFeedbackVerificationErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func NewFeedbackVerificationErrorHandler(
	log *slog.Logger,
	tzBotClient *client.Client,
	userServiceClient *userserviceclient.UserClient,
	sessionRepo *repository.SessionRepository,
	actionLogRepo repository.ActionLogRepository,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.NewFeedbackError"

		log := log.With(slog.String("op", op))

		// Получение токена из заголовков
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

		// Проверка сессии
		session, err := sessionRepo.GetSession(token)
		if err != nil {
			log.Error("invalid session", slog.String("error", err.Error()))
			http.Error(w, "invalid session", http.StatusUnauthorized)
			return
		}

		// Парсинг тела запроса
		var req NewFeedbackErrorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode request", slog.String("error", err.Error()))
			http.Error(w, "invalid request format", http.StatusBadRequest)
			return
		}

		// Валидация параметров
		if req.InstanceID == "" {
			log.Error("instance_id is required")
			http.Error(w, "instance_id is required", http.StatusBadRequest)
			return
		}

		if req.InstanceType != "invalid" && req.InstanceType != "missing" {
			log.Error("invalid instance_type", slog.String("instance_type", req.InstanceType))
			http.Error(w, "instance_type must be 'invalid' or 'missing'", http.StatusBadRequest)
			return
		}

		// Проверяем, что предоставлен хотя бы один тип фидбека
		hasValidComment := req.FeedbackComment != nil && *req.FeedbackComment != ""
		if req.FeedbackMark == nil && !hasValidComment {
			log.Error("at least one of feedback_mark or feedback_comment is required")
			http.Error(w, "at least one of feedback_mark or feedback_comment is required", http.StatusBadRequest)
			return
		}

		// Парсинг UUID
		instanceID, err := uuid.Parse(req.InstanceID)
		if err != nil {
			log.Error("invalid instance_id format", slog.String("error", err.Error()))
			http.Error(w, "invalid instance_id format", http.StatusBadRequest)
			return
		}

		// Парсинг userID из сессии
		userID, err := uuid.Parse(session.UserID)
		if err != nil {
			log.Error("invalid session user ID", slog.String("error", err.Error()))
			http.Error(w, "invalid session", http.StatusInternalServerError)
			return
		}

		// Вызов gRPC метода
		err = tzBotClient.NewFeedbackError(r.Context(), instanceID, req.InstanceType, req.FeedbackMark, req.FeedbackComment, userID, true)
		if err != nil {
			log.Error("failed to create feedback", slog.String("error", err.Error()))
			http.Error(w, "failed to create feedback", http.StatusInternalServerError)
			return
		}

		userInfo, userInfoErr := userServiceClient.GetUserInfo(r.Context(), userID)
		if userInfoErr == nil {
			// Логирование действия
			if actionLogRepo != nil {
				actionText := "Пользователь " + userInfo.FirstName + " " + userInfo.LastName + " оставил обратную связь"
				err = actionLogRepo.CreateActionLog(r.Context(), actionText, userID, 2)
				if err != nil {
					log.Error("failed to log action", slog.String("error", err.Error()))
					// Не прерываем выполнение, просто логируем ошибку
				}
			}
		}

		// Успешный ответ
		response := NewFeedbackErrorResponse{
			Success: true,
			Message: "Feedback created successfully",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
		}

		log.Info("feedback created successfully",
			slog.String("instance_id", req.InstanceID),
			slog.String("instance_type", req.InstanceType),
			slog.String("user_id", userID.String()))
	}
}
