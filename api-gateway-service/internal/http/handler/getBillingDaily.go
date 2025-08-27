package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"repairCopilotBot/api-gateway-service/internal/pkg/logger/sl"
	"repairCopilotBot/tz-bot/client"
)

// DailyAnalyticsResponse представляет ответ API для ежедневной аналитики
type DailyAnalyticsResponse struct {
	Series []*DailyAnalyticsPoint `json:"series"`
}

// DailyAnalyticsPoint представляет одну точку в ежедневной аналитике
type DailyAnalyticsPoint struct {
	Date        string   `json:"date"`
	Consumption *int64   `json:"consumption,omitempty"`
	ToPay       *float64 `json:"toPay,omitempty"`
	Tz          *int32   `json:"tz,omitempty"`
}

func GetBillingDaily(log *slog.Logger, tzBotClient *client.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.GetBillingDaily"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", r.Header.Get("X-Request-ID")),
		)

		log.Info("handling get billing daily request")

		// Валидация и извлечение параметров запроса
		query := r.URL.Query()

		fromDate := query.Get("from")
		if fromDate == "" {
			log.Error("missing required parameter: from")
			http.Error(w, "Missing required parameter: from", http.StatusBadRequest)
			return
		}

		toDate := query.Get("to")
		if toDate == "" {
			log.Error("missing required parameter: to")
			http.Error(w, "Missing required parameter: to", http.StatusBadRequest)
			return
		}

		timezone := query.Get("tz") // Опциональный параметр
		groupBy := query.Get("groupBy")
		if groupBy == "" {
			groupBy = "day" // По умолчанию - день
		}

		// Пока поддерживаем только группировку по дням
		if groupBy != "day" {
			log.Error("unsupported groupBy parameter", slog.String("groupBy", groupBy))
			http.Error(w, "Currently only 'day' groupBy is supported", http.StatusBadRequest)
			return
		}

		// Парсим параметр metrics
		var metrics []string
		metricsParam := query.Get("metrics")
		if metricsParam != "" {
			metrics = strings.Split(metricsParam, ",")
			// Очищаем от пробелов
			for i, metric := range metrics {
				metrics[i] = strings.TrimSpace(metric)
			}
			
			// Валидация метрик
			for _, metric := range metrics {
				if metric != "consumption" && metric != "toPay" && metric != "tz" {
					log.Error("invalid metric", slog.String("metric", metric))
					http.Error(w, "Invalid metric: "+metric+". Supported: consumption,toPay,tz", http.StatusBadRequest)
					return
				}
			}
		}

		log.Info("processing request parameters",
			slog.String("from_date", fromDate),
			slog.String("to_date", toDate),
			slog.String("timezone", timezone),
			slog.String("group_by", groupBy),
			slog.Any("metrics", metrics))

		// Получаем данные от tz-bot
		analytics, err := tzBotClient.GetDailyAnalytics(r.Context(), fromDate, toDate, timezone, metrics)
		if err != nil {
			log.Error("failed to get daily analytics", sl.Err(err))
			http.Error(w, "Failed to get daily analytics", http.StatusInternalServerError)
			return
		}

		// Преобразуем ответ tz-bot в формат API
		response := &DailyAnalyticsResponse{
			Series: make([]*DailyAnalyticsPoint, len(analytics.Series)),
		}

		for i, point := range analytics.Series {
			response.Series[i] = &DailyAnalyticsPoint{
				Date:        point.Date,
				Consumption: point.Consumption,
				ToPay:       point.ToPay,
				Tz:          point.Tz,
			}
		}

		// Устанавливаем заголовки ответа
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Отправляем JSON ответ
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode response", sl.Err(err))
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}

		log.Info("billing daily request processed successfully",
			slog.Int("points_count", len(response.Series)))
	}
}