package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"repairCopilotBot/tz-bot/client"

	"github.com/google/uuid"
)

func GetVersionHandler(
	log *slog.Logger,
	tzBotClient *client.Client,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.GetVersionHandler"

		log := log.With(slog.String("op", op))
		log.Info("processing GetVersion request")

		// Получаем version_id из URL path
		versionIDStr := r.PathValue("version_id")
		if versionIDStr == "" {
			log.Error("version_id not provided in URL path")
			http.Error(w, "version_id is required", http.StatusBadRequest)
			return
		}

		versionID, err := uuid.Parse(versionIDStr)
		if err != nil {
			log.Error("invalid version_id format", slog.String("version_id", versionIDStr), slog.String("error", err.Error()))
			http.Error(w, "invalid version_id format", http.StatusBadRequest)
			return
		}

		// Вызываем tz-bot для получения версии
		result, err := tzBotClient.GetVersion(r.Context(), versionID)
		if err != nil {
			log.Error("failed to get version from tz-bot", slog.String("error", err.Error()))
			http.Error(w, "failed to get version", http.StatusInternalServerError)
			return
		}

		// Конвертация OutInvalidError в HTTP response структуры (ошибки уже отсортированы в tz-bot сервисе)
		invalidErrorsResp := make([]NewTzInvalidErrorResponse, len(result.InvalidErrors))
		for i, e := range result.InvalidErrors {
			invalidErrorsResp[i] = NewTzInvalidErrorResponse{
				Id:                    e.Id,
				IdStr:                 e.IdStr,
				GroupID:               e.GroupID,
				ErrorCode:             e.ErrorCode,
				Quote:                 e.Quote,
				Analysis:              e.Analysis,
				Critique:              e.Critique,
				Verification:          e.Verification,
				SuggestedFix:          e.SuggestedFix,
				Rationale:             e.Rationale,
				OriginalQuote:         e.OriginalQuote,
				QuoteLines:            e.QuoteLines,
				UntilTheEndOfSentence: e.UntilTheEndOfSentence,
				StartLineNumber:       e.StartLineNumber,
				EndLineNumber:         e.EndLineNumber,
			}
		}

		// Конвертация OutMissingError в HTTP response структуры
		missingErrorsResp := make([]NewTzMissingErrorResponse, len(result.MissingErrors))
		for i, e := range result.MissingErrors {
			missingErrorsResp[i] = NewTzMissingErrorResponse{
				Id:           e.Id,
				IdStr:        e.IdStr,
				GroupID:      e.GroupID,
				ErrorCode:    e.ErrorCode,
				Analysis:     e.Analysis,
				Critique:     e.Critique,
				Verification: e.Verification,
				SuggestedFix: e.SuggestedFix,
				Rationale:    e.Rationale,
			}
		}

		response := NewTzResponse{
			Text:          result.HtmlText,
			InvalidErrors: invalidErrorsResp,
			MissingErrors: missingErrorsResp,
			Css:           result.Css,
		}

		// Возвращаем результат
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
			return
		}

		log.Info("GetVersion request processed successfully",
			slog.String("version_id", versionID.String()),
			slog.Int("invalid_errors_count", len(result.InvalidErrors)),
			slog.Int("missing_errors_count", len(result.MissingErrors)))
	}
}
