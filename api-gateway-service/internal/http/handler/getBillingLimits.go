package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"repairCopilotBot/api-gateway-service/internal/pkg/logger/sl"
	"repairCopilotBot/tz-bot/client"
)

type BillingLimitsResponse struct {
	MinDate string `json:"min_date"`
	MaxDate string `json:"max_date"`
}

func GetBillingLimits(log *slog.Logger, tzBotClient *client.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.GetBillingLimits"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", r.Header.Get("X-Request-ID")),
		)

		log.Info("handling get billing limits request")

		// Получаем диапазон дат из tz-service
		dateRange, err := tzBotClient.GetVersionsDateRange(r.Context())
		if err != nil {
			log.Error("failed to get versions date range", sl.Err(err))
			http.Error(w, "Failed to get billing limits", http.StatusInternalServerError)
			return
		}

		response := BillingLimitsResponse{
			MinDate: dateRange.MinDate,
			MaxDate: dateRange.MaxDate,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode response", sl.Err(err))
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}

		log.Info("billing limits request processed successfully",
			slog.String("min_date", response.MinDate),
			slog.String("max_date", response.MaxDate))
	}
}