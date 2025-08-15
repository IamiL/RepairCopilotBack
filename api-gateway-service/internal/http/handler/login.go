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

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginResponse struct {
	UserID   string                          `json:"user_id"`
	Login    string                          `json:"login"`
	Level    int                             `json:"level"`
	Versions []TechnicalSpecificationVersion `json:"versions"`
}

func LoginHandler(
	log *slog.Logger,
	userServiceClient *userserviceclient.UserClient,
	sessionRepo *repository.SessionRepository,
	tzBotClient *client.Client,
	actionLogRepo repository.ActionLogRepository,
) func(
	w http.ResponseWriter, r *http.Request,
) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.LoginHandler"

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.With(slog.String("op", op)).Error("failed to decode request", slog.String("error", err.Error()))
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Login == "" || req.Password == "" {
			log.With(slog.String("op", op)).Error("missing login or password")
			http.Error(w, "Login and password are required", http.StatusBadRequest)
			return
		}

		loginResp, err := userServiceClient.Login(r.Context(), req.Login, req.Password)
		if err != nil {
			log.With(slog.String("op", op)).Error("authentication failed", slog.String("error", err.Error()))
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}

		uid := uuid.MustParse(loginResp.UserID)

		// Получаем версии технических заданий от tz-bot
		var versions []TechnicalSpecificationVersion
		tzVersions, err := tzBotClient.GetTechnicalSpecificationVersions(r.Context(), uid)
		if err != nil {
			log.With(slog.String("op", op)).Error("failed to get technical specification versions", slog.String("error", err.Error()))
			// Не возвращаем ошибку, продолжаем с пустым массивом версий
		} else {
			// Конвертируем из client.TechnicalSpecificationVersion в handler.TechnicalSpecificationVersion
			versions = make([]TechnicalSpecificationVersion, len(tzVersions))
			for i, tzVersion := range tzVersions {
				versions[i] = TechnicalSpecificationVersion{
					VersionId:                 tzVersion.VersionId,
					TechnicalSpecificationName: tzVersion.TechnicalSpecificationName,
					VersionNumber:             tzVersion.VersionNumber,
					CreatedAt:                 tzVersion.CreatedAt,
				}
			}
		}

		// Создаем сессию
		sessionId := uuid.New()
		err = sessionRepo.CreateSession(sessionId, uid, req.Login, loginResp.IsAdmin1, loginResp.IsAdmin2)
		if err != nil {
			log.With(slog.String("op", op)).Error("failed to create session", slog.String("error", err.Error()))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Устанавливаем cookie с правильным именем (auth_token, как используется в /me)
		cookie := &http.Cookie{
			Name:     "auth_token",
			Value:    sessionId.String(),
			Path:     "/",
			MaxAge:   24 * 60 * 60, // 24 hours
			HttpOnly: true,
			Secure:   false, // set to true in production with HTTPS
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, cookie)

		// Определяем уровень пользователя
		level := 0
		if loginResp.IsAdmin1 {
			level = 1
		} else if loginResp.IsAdmin2 {
			level = 2
		}

		// Формируем ответ
		response := LoginResponse{
			UserID:   loginResp.UserID,
			Login:    req.Login,
			Level:    level,
			Versions: versions,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.With(slog.String("op", op)).Error("failed to encode response", slog.String("error", err.Error()))
			return
		}

		// Логируем действие входа в систему
		err = actionLogRepo.CreateActionLog(r.Context(), "User login: "+req.Login, uid)
		if err != nil {
			log.With(slog.String("op", op)).Error("failed to create action log", slog.String("error", err.Error()))
		}

		log.With(slog.String("op", op)).Info("user logged in successfully", 
			slog.String("login", req.Login), 
			slog.String("sessionID", sessionId.String()),
			slog.String("userID", loginResp.UserID),
			slog.Int("versions_count", len(versions)))
	}
}
