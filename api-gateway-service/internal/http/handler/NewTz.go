package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"repairCopilotBot/api-gateway-service/internal/repository"
	"repairCopilotBot/tz-bot/client"
	userserviceclient "repairCopilotBot/user-service/client"
	"strings"

	"github.com/google/uuid"
)

type NewTzResponse struct {
	Text          string                      `json:"text"`
	Css           string                      `json:"css"`
	DocId         *string                     `json:"doc_id"`
	InvalidErrors []NewTzInvalidErrorResponse `json:"invalid_errors"`
	MissingErrors []NewTzMissingErrorResponse `json:"missing_errors"`
}

type NewTzInvalidErrorResponse struct {
	Id                    uint32    `json:"numeric_id"`
	IdStr                 string    `json:"id"`
	GroupID               string    `json:"group_id"`
	ErrorCode             string    `json:"error_code"`
	Quote                 string    `json:"quote"`
	Analysis              string    `json:"analysis"`
	Critique              string    `json:"critique"`
	Verification          string    `json:"verification"`
	SuggestedFix          string    `json:"suggested_fix"`
	Rationale             string    `json:"rationale"`
	OriginalQuote         string    `json:"original_quote"`
	QuoteLines            *[]string `json:"quote_lines"`
	UntilTheEndOfSentence bool      `json:"until_the_end_of_sentence"`
	StartLineNumber       *int      `json:"start_line_number"`
	EndLineNumber         *int      `json:"end_line_number"`
}

type NewTzMissingErrorResponse struct {
	Id           uint32 `json:"id"`
	IdStr        string `json:"id_str"`
	GroupID      string `json:"group_id"`
	ErrorCode    string `json:"error_code"`
	Analysis     string `json:"analysis"`
	Critique     string `json:"critique"`
	Verification string `json:"verification"`
	SuggestedFix string `json:"suggested_fix"`
	Rationale    string `json:"rationale"`
}

func NewTzHandler(
	log *slog.Logger,
	tzBotClient *client.Client,
	sessionRepo *repository.SessionRepository,
	userServiceClient *userserviceclient.UserClient,
	actionLogRepo repository.ActionLogRepository,
) func(
	w http.ResponseWriter, r *http.Request,
) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.NewTzHandler"

		log := log.With(slog.String("op", op))
		log.Info("TZ processing request started")

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

		uid, err := uuid.Parse(session.UserID)
		if err != nil {
			log.Info("failed to parse session uid")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}

		// Парсим multipart form (максимум 10MB)
		err = r.ParseMultipartForm(10 << 20)
		if err != nil {
			log.Error("failed to parse multipart form", slog.String("error", err.Error()))
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		log.Debug("multipart form parsed successfully")

		// Получаем файл из формы
		file, header, err := r.FormFile("file")
		if err != nil {
			log.Error("failed to get file from form", slog.String("error", err.Error()))
			http.Error(w, "File not found in request", http.StatusBadRequest)
			return
		}
		defer file.Close()

		log.Info("file received",
			slog.String("filename", header.Filename),
			slog.Int64("size", header.Size))

		// Читаем файл в байты
		fileBytes, err := io.ReadAll(file)
		if err != nil {
			log.Error("failed to read file content", slog.String("error", err.Error()))
			http.Error(w, "Failed to read file", http.StatusInternalServerError)
			return
		}

		filename := header.Filename

		//requestID, err := uuid.NewUUID()
		//if err != nil {
		//	log.Error("failed to generate request ID", slog.String("error", err.Error()))
		//	http.Error(w, "Internal server error", http.StatusInternalServerError)
		//	return
		//}

		log = log.With(slog.String("requestID", session.UserID))
		log.Info("processing TZ file", slog.String("filename", filename))

		checkTzResult, err := tzBotClient.CheckTz(r.Context(), fileBytes, filename, uid)
		if err != nil {
			log.Error("TZ processing failed", slog.String("error", err.Error()))
			http.Error(w, "TZ processing failed", http.StatusInternalServerError)
			return
		}

		log.Info("TZ processing completed successfully",
			slog.Int("invalid_errors_count", len(checkTzResult.InvalidErrors)),
			slog.Int("missing_errors_count", len(checkTzResult.MissingErrors)),
			slog.String("doc_id", checkTzResult.DocId))

		// Логируем событие отправки документа
		userInfo, userInfoErr := userServiceClient.GetUserInfo(r.Context(), session.UserID)
		if userInfoErr == nil {
			actionText := "Пользователь " + userInfo.Login + " отправил документ " + filename + " на проверку"
			if err := actionLogRepo.CreateActionLog(r.Context(), actionText, uid); err != nil {
				log.Error("failed to create action log for TZ submission", slog.String("error", err.Error()))
			}
		}

		// Конвертация OutInvalidError в HTTP response структуры (ошибки уже отсортированы в tz-bot сервисе)
		invalidErrorsResp := make([]NewTzInvalidErrorResponse, len(checkTzResult.InvalidErrors))
		for i, e := range checkTzResult.InvalidErrors {
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
		missingErrorsResp := make([]NewTzMissingErrorResponse, len(checkTzResult.MissingErrors))
		for i, e := range checkTzResult.MissingErrors {
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
			Text:          checkTzResult.HtmlText,
			InvalidErrors: invalidErrorsResp,
			MissingErrors: missingErrorsResp,
			Css:           checkTzResult.Css,
			DocId:         &checkTzResult.DocId,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
			return
		}

		log.Info("TZ processing request completed successfully")
	}
}

func writeStringToFile(content string, filename string) error {
	// Заменяем символы новой строки на литеральные \n
	escapedContent := strings.ReplaceAll(content, "\n", "\\n")

	// Создаем файл в корне проекта
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("не удалось создать файл: %v", err)
	}
	defer file.Close()

	// Записываем содержимое в файл
	_, err = file.WriteString(escapedContent)
	if err != nil {
		return fmt.Errorf("не удалось записать в файл: %v", err)
	}

	return nil
}
