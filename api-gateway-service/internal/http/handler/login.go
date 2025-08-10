package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"repairCopilotBot/api-gateway-service/internal/repository"
	userserviceclient "repairCopilotBot/user-service/client"

	"github.com/google/uuid"
)

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Message string `json:"message"`
}

func LoginHandler(
	log *slog.Logger,
	userServiceClient *userserviceclient.UserClient,
	sessionRepo *repository.SessionRepository,
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

		//userID, err := strconv.Atoi(loginResp.UserID)
		//if err != nil {
		//	log.With(slog.String("op", op)).Error("invalid user ID format", slog.String("error", err.Error()))
		//	http.Error(w, "Internal server error", http.StatusInternalServerError)
		//	return
		//}

		uid := uuid.MustParse(loginResp.UserID)

		seseionId := uuid.New()

		err = sessionRepo.CreateSession(seseionId, uid, req.Login, loginResp.IsAdmin1, loginResp.IsAdmin2)
		if err != nil {
			log.With(slog.String("op", op)).Error("failed to create session", slog.String("error", err.Error()))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		cookie := &http.Cookie{
			Name:     "session_id",
			Value:    seseionId.String(),
			Path:     "/",
			MaxAge:   24 * 60 * 60, // 24 hours
			HttpOnly: true,
			Secure:   false, // set to true in production with HTTPS
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, cookie)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(LoginResponse{Message: "Login successful"})

		log.With(slog.String("op", op)).Info("user logged in successfully", slog.String("login", req.Login), slog.String("sessionID", seseionId.String()))
	}
}
