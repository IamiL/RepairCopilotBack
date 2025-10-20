package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	userserviceclient "repairCopilotBot/user-service/client"
)

type RecoveryRequest struct {
	Email string `json:"email"`
}

type RecoveryResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func RecoveryHandler(
	log *slog.Logger,
	userServiceClient *userserviceclient.UserClient,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.RecoveryHandler"

		log := log.With(slog.String("op", op))
		log.Info("processing recovery request")

		var req RecoveryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode request body", slog.String("error", err.Error()))
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Валидация входных данных
		if req.Email == "" {
			log.Info("empty email provided")
			http.Error(w, "Email is required", http.StatusBadRequest)
			return
		}

		log.Info("initiating account recovery", slog.String("email", req.Email))

		// Вызываем метод восстановления в user-service
		resp, err := userServiceClient.Recovery(r.Context(), req.Email)
		if err != nil {
			log.Error("failed to recover account", slog.String("error", err.Error()), slog.String("email", req.Email))

			// Определяем статус код в зависимости от ошибки
			statusCode := http.StatusInternalServerError
			errorMessage := "Account recovery failed"

			if err.Error() == "user not found" {
				statusCode = http.StatusNotFound
				errorMessage = "User with this email not found"
			} else if err.Error() == "invalid email: email is required" {
				statusCode = http.StatusBadRequest
				errorMessage = "Invalid email address"
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			json.NewEncoder(w).Encode(RecoveryResponse{
				Success: false,
				Message: errorMessage,
			})
			return
		}

		log.Info("account recovery successful", slog.String("email", req.Email))

		// Формируем успешный ответ
		response := RecoveryResponse{
			Success: resp.Success,
			Message: resp.Message,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
			return
		}

		log.Info("recovery request processed successfully", slog.String("email", req.Email))
	}
}