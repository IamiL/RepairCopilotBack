package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"repairCopilotBot/api-gateway-service/internal/repository"
	userserviceclient "repairCopilotBot/user-service/client"
	"time"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Login     string `json:"login"`
	Password  string `json:"password"`
}

type RegisterResponse struct {
	Message string `json:"message"`
	Login   string `json:"login"`
	UserID  string `json:"user_id"`
}

func RegisterHandler(
	log *slog.Logger,
	userServiceClient *userserviceclient.UserClient,
	sessionRepo *repository.SessionRepository,
	actionLogRepo repository.ActionLogRepository,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.RegisterHandler"

		log := log.With(slog.String("op", op))
		log.Info("processing register request")

		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode request body", slog.String("error", err.Error()))
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Валидация входных данных
		if req.Login == "" {
			log.Info("empty login provided")
			http.Error(w, "Login is required", http.StatusBadRequest)
			return
		}

		if req.Password == "" {
			log.Info("empty password provided")
			http.Error(w, "Password is required", http.StatusBadRequest)
			return
		}

		log.Info("registering new user", slog.String("login", req.Login))

		// Регистрируем пользователя в user-service
		userID, err := userServiceClient.RegisterUser(r.Context(), req.Email, req.FirstName, req.LastName, req.Login, req.Password)
		if err != nil {
			log.Error("failed to register user", slog.String("error", err.Error()), slog.String("login", req.Login))
			http.Error(w, "Registration failed: "+err.Error(), http.StatusConflict)
			return
		}

		log.Info("user registered successfully", slog.String("user_id", userID), slog.String("login", req.Login))

		// Парсим userID из строки в UUID для логирования
		userUUID, parseErr := uuid.Parse(userID)
		if parseErr == nil {
			// Логируем событие регистрации
			actionText := "зарегистрирован пользователь - " + req.FirstName + " " + req.LastName + " ; логин - " + req.Login
			if err := actionLogRepo.CreateActionLog(r.Context(), actionText, userUUID); err != nil {
				log.Error("failed to create action log", slog.String("error", err.Error()))
			}
		}

		// Генерируем новый UUID для сессии
		sessionID := uuid.New()

		// Парсим userID из строки в UUID
		userUUID, err = uuid.Parse(userID)
		if err != nil {
			log.Error("failed to parse user ID as UUID", slog.String("error", err.Error()), slog.String("user_id", userID))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Создаем сессию в Redis (новый пользователь не админ)
		err = sessionRepo.CreateSession(sessionID, userUUID, req.Login, false, false)
		if err != nil {
			log.Error("failed to create session", slog.String("error", err.Error()))
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}

		log.Info("session created successfully",
			slog.String("session_id", sessionID.String()),
			slog.String("user_id", userID))

		// Устанавливаем cookie с токеном сессии
		cookie := &http.Cookie{
			Name:     "auth_token",
			Value:    sessionID.String(),
			Path:     "/",
			HttpOnly: true,
			Secure:   false, // В продакшене должно быть true для HTTPS
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int((time.Hour * 24 * 30).Seconds()), // 30 дней
		}
		http.SetCookie(w, cookie)

		// Формируем успешный ответ
		response := RegisterResponse{
			Message: "User registered successfully",
			Login:   req.Login,
			UserID:  userID,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
			return
		}

		log.Info("register request processed successfully",
			slog.String("login", req.Login),
			slog.String("user_id", userID),
			slog.String("session_id", sessionID.String()))
	}
}
