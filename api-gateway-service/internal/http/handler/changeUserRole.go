package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"repairCopilotBot/api-gateway-service/internal/repository"
	userserviceclient "repairCopilotBot/user-service/client"

	"github.com/google/uuid"
)

type ChangeUserRoleRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"` // "user" или "admin"
}

type ChangeUserRoleResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func ChangeUserRoleHandler(
	log *slog.Logger,
	userServiceClient *userserviceclient.UserClient,
	sessionRepo *repository.SessionRepository,
	actionLogRepo repository.ActionLogRepository,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.ChangeUserRoleHandler"

		log := log.With(slog.String("op", op))
		log.Info("change user role request started")

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
		// TODO: раскомментировать когда будет готова проверка прав
		// if !session.IsAdmin1 && !session.IsAdmin2 {
		// 	log.Info("user is not admin", slog.String("user_id", session.UserID))
		// 	http.Error(w, "Forbidden: admin access required", http.StatusForbidden)
		// 	return
		// }

		// Парсим JSON тело запроса
		var req ChangeUserRoleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode request body", slog.String("error", err.Error()))
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Валидация входных данных
		if req.UserID == "" {
			log.Info("empty user_id provided")
			http.Error(w, "user_id is required", http.StatusBadRequest)
			return
		}

		// Валидация UUID
		if _, err := uuid.Parse(req.UserID); err != nil {
			log.Info("invalid user_id format", slog.String("user_id", req.UserID))
			http.Error(w, "user_id must be a valid UUID", http.StatusBadRequest)
			return
		}

		// Валидация роли
		if req.Role != "user" && req.Role != "admin" {
			log.Info("invalid role value", slog.String("role", req.Role))
			http.Error(w, "role must be 'user' or 'admin'", http.StatusBadRequest)
			return
		}

		// Конвертируем строку роли в булево значение для isAdmin
		isAdmin := req.Role == "admin"

		// Вызываем метод userServiceClient для смены роли
		resp, err := userServiceClient.ChangeUserRole(r.Context(), req.UserID, isAdmin)
		if err != nil {
			log.Error("failed to change user role", slog.String("error", err.Error()))
			http.Error(w, "Failed to change user role", http.StatusInternalServerError)
			return
		}

		response := ChangeUserRoleResponse{
			Success: resp.Success,
			Message: resp.Message,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
			return
		}

		// Логируем событие смены роли
		userInfo, userInfoErr := userServiceClient.GetUserInfo(r.Context(), uuid.MustParse(session.UserID))
		if userInfoErr == nil {
			targetUserInfo, targetUserInfoErr := userServiceClient.GetUserInfo(r.Context(), uuid.MustParse(req.UserID))
			if targetUserInfoErr == nil {
				actionText := "Администратор " + userInfo.FirstName + " " + userInfo.LastName +
					" изменил роль пользователя " + targetUserInfo.FirstName + " " + targetUserInfo.LastName +
					" на '" + req.Role + "'"
				if err := actionLogRepo.CreateActionLog(r.Context(), actionText, uuid.MustParse(session.UserID), 5); err != nil {
					log.Error("failed to create action log for role change", slog.String("error", err.Error()))
				}
			}
		}

		log.Info("change user role request completed",
			slog.String("target_user_id", req.UserID),
			slog.String("new_role", req.Role),
			slog.String("admin_user_id", session.UserID))
	}
}