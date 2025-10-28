package tzservice

import (
	"context"
	"fmt"
	"log/slog"
	modelrepo "repairCopilotBot/tz-bot/internal/repository/models"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"

	"repairCopilotBot/tz-bot/internal/pkg/llm"
	"repairCopilotBot/tz-bot/internal/pkg/logger/sl"
)

// ErrorInstance описывает одну ошибку из LLM API
type ErrorInstance struct {
	GroupID      string  `json:"group_id"`
	Code         string  `json:"code"`
	ErrType      string  `json:"err_type"`
	Snippet      string  `json:"snippet"`
	LineStart    *int    `json:"line_start"`
	LineEnd      *int    `json:"line_end"`
	SuggestedFix *string `json:"suggested_fix"`
	Rationale    string  `json:"rationale"`
}

// HtmlBlock связывает HTML-блок с Markdown-диапазоном
type HtmlBlock struct {
	ElementID     string `json:"html_element_id"`
	HtmlContent   string `json:"html_content"`
	MarkdownStart int    `json:"markdown_line_start"`
	MarkdownEnd   int    `json:"markdown_line_end"`
}

type OutError struct {
	ID           string  `json:"id"`
	GroupID      string  `json:"group_id"`
	Code         string  `json:"code"`
	SuggestedFix *string `json:"suggested_fix"`
	Rationale    string  `json:"rationale"`
}

// llmRequestResult represents the result of a single LLM request
type llmRequestResult struct {
	groupReport *tz_llm_client.GroupReport
	ResultRaw   string
	cost        *float64
	tokens      *int64
	duration    int64
	err         error
}

// saveTechnicalSpecificationData saves technical specification data to database
// func (tz *Tz) saveTechnicalSpecificationData(
//
//	ctx context.Context,
//	filename string,
//	userID uuid.UUID,
//	outHTML string,
//	css string,
//	originalFileID string,
//	invalidErrors *[]OutInvalidError,
//	missingErrors *[]OutMissingError,
//	errors *[]Error,
//	allRubs float64,
//	allTokens int64,
//	inspectionTime time.Duration,
//	log *slog.Logger,
//
//	) error {
//		now := time.Now()
//
//		// Создаем техническое задание
//		tsID := uuid.New()
//		tsReq := &modelrepo.CreateTechnicalSpecificationRequest{
//			ID:        tsID,
//			Name:      filename,
//			UserID:    userID,
//			CreatedAt: now,
//			UpdatedAt: now,
//		}
//
//		ts, err := tz.repo.CreateTechnicalSpecification(ctx, tsReq)
//		if err != nil {
//			return fmt.Errorf("failed to create technical specification: %w", err)
//		}
//
//		log.Info("technical specification created", slog.String("ts_id", ts.ID.String()))
//
//		// Создаем версию
//		versionID := uuid.New()
//		versionReq := &repository.CreateVersionRequest{
//			ID:                       versionID,
//			TechnicalSpecificationID: tsID,
//			VersionNumber:            1, // Первая версия
//			CreatedAt:                now,
//			UpdatedAt:                now,
//			OriginalFileID:           originalFileID,
//			OutHTML:                  outHTML,
//			CSS:                      css,
//			CheckedFileID:            "", // Пока пустое
//			AllRubs:                  allRubs,
//			AllTokens:                allTokens,
//			InspectionTime:           inspectionTime,
//		}
//
//		version, err := tz.repo.CreateVersion(ctx, versionReq)
//		if err != nil {
//			return fmt.Errorf("failed to create version: %w", err)
//		}
//
//		log.Info("version created", slog.String("version_id", version.ID.String()))
//
//		// Сохраняем InvalidErrors
//		if invalidErrors != nil && len(*invalidErrors) > 0 {
//			invalidErrorData := make([]repository.InvalidErrorData, 0, len(*invalidErrors))
//			for i, err := range *invalidErrors {
//				invalidErrorData = append(invalidErrorData, repository.InvalidErrorData{
//					ID:           uuid.New(),
//					ErrorID:      int(err.Id),
//					ErrorIDStr:   err.IdStr,
//					GroupID:      err.GroupID,
//					ErrorCode:    err.ErrorCode,
//					Quote:        err.Quote,
//					Analysis:     err.Analysis,
//					Critique:     err.Critique,
//					Verification: err.Verification,
//					SuggestedFix: err.SuggestedFix,
//					Rationale:    err.Rationale,
//					OrderNumber:  i, // Порядковый номер (индекс в массиве)
//					CreatedAt:    now,
//				})
//			}
//
//			invalidReq := &repository.CreateInvalidErrorsRequest{
//				VersionID: versionID,
//				Errors:    invalidErrorData,
//			}
//
//			err = tz.repo.CreateInvalidErrors(ctx, invalidReq)
//			if err != nil {
//				return fmt.Errorf("failed to create invalid errors: %w", err)
//			}
//
//			log.Info("invalid errors saved", slog.Int("count", len(invalidErrorData)))
//		}
//
//		// Сохраняем MissingErrors
//		if missingErrors != nil && len(*missingErrors) > 0 {
//			missingErrorData := make([]repository.MissingErrorData, 0, len(*missingErrors))
//			for _, err := range *missingErrors {
//				missingErrorData = append(missingErrorData, repository.MissingErrorData{
//					ID:           uuid.New(),
//					ErrorID:      int(err.Id),
//					ErrorIDStr:   err.IdStr,
//					GroupID:      err.GroupID,
//					ErrorCode:    err.ErrorCode,
//					Analysis:     err.Analysis,
//					Critique:     err.Critique,
//					Verification: err.Verification,
//					SuggestedFix: err.SuggestedFix,
//					Rationale:    err.Rationale,
//					CreatedAt:    now,
//				})
//			}
//
//			missingReq := &repository.CreateMissingErrorsRequest{
//				VersionID: versionID,
//				Errors:    missingErrorData,
//			}
//
//			err = tz.repo.CreateMissingErrors(ctx, missingReq)
//			if err != nil {
//				return fmt.Errorf("failed to create missing errors: %w", err)
//			}
//
//			log.Info("missing errors saved", slog.Int("count", len(missingErrorData)))
//		}
//
//		// Сохраняем Errors
//		if errors != nil && len(*errors) > 0 {
//			errorData := make([]repository.ErrorData, 0, len(*errors))
//			for _, err := range *errors {
//				instancesJSON, jsonErr := json.Marshal(err.Instances)
//				if jsonErr != nil {
//					return fmt.Errorf("failed to marshal instances: %w", jsonErr)
//				}
//
//				errorData = append(errorData, repository.ErrorData{
//					ID:                  err.ID,
//					GroupID:             &err.GroupID,
//					ErrorCode:           &err.ErrorCode,
//					PreliminaryNotes:    err.PreliminaryNotes,
//					OverallCritique:     err.OverallCritique,
//					Verdict:             &err.Verdict,
//					ProcessAnalysis:     err.ProcessAnalysis,
//					ProcessCritique:     err.ProcessCritique,
//					ProcessVerification: err.ProcessVerification,
//					ProcessRetrieval:    err.ProcessRetrieval,
//					Instances:           instancesJSON,
//				})
//			}
//
//			errorsReq := &repository.CreateErrorsRequest{
//				VersionID: versionID,
//				Errors:    errorData,
//			}
//
//			err = tz.repo.CreateErrors(ctx, errorsReq)
//			if err != nil {
//				return fmt.Errorf("failed to create errors: %w", err)
//			}
//
//			log.Info("errors saved", slog.Int("count", len(errorData)))
//		}
//
//		log.Info("technical specification data saved successfully")
//		return nil
//	}
type VersionMe struct {
	ID                         uuid.UUID `db:"id"`
	TechnicalSpecificationName string    `db:"technical_specification_name"`
	VersionNumber              int       `db:"version_number"`
	CreatedAt                  time.Time `db:"created_at"`
	OriginalFileID             string    `db:"original_file_id"`
	OriginalFileLink           string    `db:"original_file_link"`
	ReportFileID               *string
	ReportFileLink             *string
	Status                     string
	Progress                   int
}

func (tz *Tz) GetVersionsMe(ctx context.Context, userID uuid.UUID) ([]*VersionMe, error) {
	const op = "Tz.GetTechnicalSpecificationVersions"

	log := tz.log.With(
		slog.String("op", op),
		slog.String("userID", userID.String()),
	)

	log.Info("getting technical specification versions")

	versions, err := tz.repo.GetVersionsMeByUserID(ctx, userID)
	if err != nil {
		log.Error("failed to get versions by user ID: ", sl.Err(err))
		return nil, fmt.Errorf("failed to get versions by user ID: %w", err)
	}

	for i := range versions {
		if versions[i].ReportFileID != nil && *versions[i].ReportFileID != "" {
			reportFileLink := "https://s3.timuroid.ru/reports/" + *versions[i].ReportFileID + ".docx"
			versions[i].ReportFileLink = &reportFileLink
		}

		if versions[i].OriginalFileID != "" {
			versions[i].OriginalFileLink = "https://s3.timuroid.ru/docx/" + versions[i].OriginalFileID + ".docx"
		}
	}

	log.Info("technical specification versions retrieved successfully", slog.Int("count", len(versions)))
	return versions, nil
}

type VersionAdminDashboard struct {
	ID                         uuid.UUID
	TechnicalSpecificationName string
	UserID                     uuid.UUID
	VersionNumber              int
	AllTokens                  int64
	AllRubs                    float64
	NumberOfErrors             int64
	InspectionTime             time.Duration
	OriginalFileSize           int64
	NumberOfPages              int
	CreatedAt                  time.Time
	OriginalFileId             string
	OriginalFileLink           string
	ReportFileId               string
	ReportFileLink             string
}

func (tz *Tz) GetAllVersionsAdminDashboard(ctx context.Context, userID uuid.UUID) ([]*VersionAdminDashboard, error) {
	const op = "Tz.GetAllVersions"

	log := tz.log.With(
		slog.String("op", op),
	)

	log.Info("getting all versions with error counts")

	versions, err := tz.repo.GetAllVersionsAdminDashboard(ctx, userID)
	if err != nil {
		log.Error("failed to get all versions: ", sl.Err(err))
		return nil, fmt.Errorf("failed to get all versions: %w", err)
	}

	for i := range versions {
		versions[i].OriginalFileLink = "https://s3.timuroid.ru/docx/" + versions[i].OriginalFileId + ".docx"
		versions[i].ReportFileLink = "https://s3.timuroid.ru/reports/" + versions[i].ReportFileId + ".docx"
	}

	log.Info("all versions retrieved successfully", slog.Int("count", len(versions)))
	return versions, nil
}

func (tz *Tz) GetVersionStatistics(ctx context.Context) (*modelrepo.VersionStatistics, error) {
	const op = "Tz.GetVersionStatistics"

	log := tz.log.With(
		slog.String("op", op),
	)

	log.Info("getting version statistics")

	stats, err := tz.repo.GetVersionStatistics(ctx)
	if err != nil {
		log.Error("failed to get version statistics: ", sl.Err(err))
		return nil, fmt.Errorf("failed to get version statistics: %w", err)
	}

	log.Info("version statistics retrieved successfully",
		slog.Int64("total_versions", stats.TotalVersions),
		slog.Any("total_tokens", stats.TotalTokens),
		slog.Any("total_rubs", stats.TotalRubs),
		slog.Any("average_inspection_time", stats.AverageInspectionTime))

	return stats, nil
}

func (tz *Tz) GetVersionsDateRange(ctx context.Context) (string, string, error) {
	const op = "Tz.GetVersionsDateRange"

	log := tz.log.With(
		slog.String("op", op),
	)

	log.Info("getting versions date range")

	minDate, maxDate, err := tz.repo.GetVersionsDateRange(ctx)
	if err != nil {
		log.Error("failed to get versions date range: ", sl.Err(err))
		return "", "", fmt.Errorf("failed to get versions date range: %w", err)
	}

	log.Info("versions date range retrieved successfully",
		slog.String("min_date", minDate),
		slog.String("max_date", maxDate))

	return minDate, maxDate, nil
}

func (tz *Tz) GetFeedbacks(ctx context.Context, userID *string) ([]*FeedbackInstance, error) {
	const op = "Tz.GetFeedbacks"

	log := tz.log.With(
		slog.String("op", op),
	)

	if userID != nil {
		log = log.With(slog.String("user_id", *userID))
	}

	log.Info("getting feedbacks")

	feedbacks, err := tz.repo.GetFeedbacks(ctx, userID)
	if err != nil {
		log.Error("failed to get feedbacks: ", sl.Err(err))
		return nil, fmt.Errorf("failed to get feedbacks: %w", err)
	}

	// Сортируем feedbacks по CreatedAt (элементы с nil в конце)
	sort.Slice(feedbacks, func(i, j int) bool {
		// Если CreatedAt == nil у элемента i, он должен быть после j
		if feedbacks[i].CreatedAt == nil {
			return false
		}
		// Если CreatedAt == nil у элемента j, элемент i должен быть перед ним
		if feedbacks[j].CreatedAt == nil {
			return true
		}
		// Оба поля не nil - сравниваем даты (сортировка по возрастанию)
		return feedbacks[i].CreatedAt.After(*feedbacks[j].CreatedAt)
	})

	log.Info("feedbacks retrieved successfully",
		slog.Int("feedbacks_count", len(feedbacks)))

	return feedbacks, nil
}

func (tz *Tz) GetDailyAnalytics(ctx context.Context, fromDate, toDate, timezone string, metrics []string) ([]*DailyAnalyticsPoint, error) {
	const op = "Tz.GetDailyAnalytics"

	log := tz.log.With(
		slog.String("op", op),
		slog.String("from_date", fromDate),
		slog.String("to_date", toDate),
		slog.String("timezone", timezone),
		slog.Int("metrics_count", len(metrics)),
	)

	log.Info("getting daily analytics")

	points, err := tz.repo.GetDailyAnalytics(ctx, fromDate, toDate, timezone, metrics)
	if err != nil {
		log.Error("failed to get daily analytics: ", sl.Err(err))
		return nil, fmt.Errorf("failed to get daily analytics: %w", err)
	}

	log.Info("daily analytics retrieved successfully",
		slog.Int("points_count", len(points)))

	return points, nil
}

func (tz *Tz) GetVersion(ctx context.Context, versionID uuid.UUID) (string, time.Time, float64, int64, time.Duration, string, string, string, *[]Error, *[]OutInvalidError, string, int64, int, string, int, error) {
	const op = "Tz.GetVersion"

	log := tz.log.With(
		slog.String("op", op),
		slog.String("versionID", versionID.String()),
	)

	log.Info("getting version with errors")

	version, err := tz.repo.GetVersion(ctx, versionID)
	if err != nil {
		log.Error("failed to get version: ", sl.Err(err))
		return "", time.Time{}, 0, 0, 0, "", "", "", nil, nil, "", 0, 0, "", 0, err
	}

	if version.Status == "in_progress" {
		return "in_progress", time.Time{}, 0, 0, 0, "", "", "", nil, nil, "", 0, 0, "", version.Progress, err
	}

	errorsInTz, err := tz.repo.GetErrorsByVersionID(ctx, versionID)
	if err != nil {
		log.Error("failed to get version errors: ", sl.Err(err))
		return "", time.Time{}, 0, 0, 0, "", "", "", nil, nil, "", 0, 0, "", 0, err
	}

	invalidInstances := make([]OutInvalidError, 0)
	for i := range *errorsInTz {
		invalidInstancesFromDb, err := tz.repo.GetInvalidInstancesByErrorID(ctx, (*errorsInTz)[i].ID)
		if err != nil {
			log.Error("failed to get version errors: ", sl.Err(err))
		} else {
			for j := range *invalidInstancesFromDb {
				(*invalidInstancesFromDb)[j].HtmlIDStr = strconv.Itoa(int((*invalidInstancesFromDb)[j].HtmlID))
			}
			invalidInstances = append(invalidInstances, *invalidInstancesFromDb...)
			(*errorsInTz)[i].InvalidInstances = invalidInstancesFromDb
		}

		missingInstances, err := tz.repo.GetMissingInstancesByErrorID(ctx, (*errorsInTz)[i].ID)
		if err != nil {
			log.Error("failed to get version missing instances: ", sl.Err(err))
		} else {
			(*errorsInTz)[i].MissingInstances = missingInstances
		}
	}

	SortOutInvalidErrorsByOrderNumber(&invalidInstances)

	//version, invalidErrors, missingErrors, err := tz.repo.GetVersionWithErrors(ctx, versionID)
	//if err != nil {
	//	log.Error("failed to get version with errors: ", sl.Err(err))
	//	return "", "", "", nil, nil, "", fmt.Errorf("failed to get version with errors: %w", err)
	//}
	//
	//// Конвертируем InvalidError в OutInvalidError
	//outInvalidErrors := make([]OutInvalidError, len(invalidErrors))
	//for i, invErr := range invalidErrors {
	//	outInvalidErrors[i] = OutInvalidError{
	//		Id:           uint32(invErr.ErrorID),
	//		IdStr:        invErr.ErrorIDStr,
	//		GroupID:      invErr.GroupID,
	//		ErrorCode:    invErr.ErrorCode,
	//		Quote:        invErr.Quote,
	//		Analysis:     invErr.Analysis,
	//		Critique:     invErr.Critique,
	//		Verification: invErr.Verification,
	//		SuggestedFix: invErr.SuggestedFix,
	//		Rationale:    invErr.Rationale,
	//	}
	//}
	//
	//// Конвертируем MissingError в OutMissingError
	//outMissingErrors := make([]OutMissingError, len(missingErrors))
	//for i, missErr := range missingErrors {
	//	outMissingErrors[i] = OutMissingError{
	//		Id:           uint32(missErr.ErrorID),
	//		IdStr:        missErr.ErrorIDStr,
	//		GroupID:      missErr.GroupID,
	//		ErrorCode:    missErr.ErrorCode,
	//		Analysis:     missErr.Analysis,
	//		Critique:     missErr.Critique,
	//		Verification: missErr.Verification,
	//		SuggestedFix: missErr.SuggestedFix,
	//		Rationale:    missErr.Rationale,
	//	}
	//}

	//log.Info("version with errors retrieved successfully",
	//	slog.Int("invalid_errors_count", len(outInvalidErrors)),
	//	slog.Int("missing_errors_count", len(outMissingErrors)))

	return "completed", version.CreatedAt, *version.AllRubs, *version.AllTokens, *version.InspectionTime, version.OutHTML, version.CSS, version.CheckedFileID, errorsInTz, &invalidInstances, "", *version.OriginalFileSize, int(*version.NumberOfErrors), version.LlmReport, 0, nil
}

func SortOutInvalidErrorsByOrderNumber(errors *[]OutInvalidError) {
	if errors == nil {
		return
	}

	sort.Slice(*errors, func(i, j int) bool {
		return (*errors)[i].OrderNumber < (*errors)[j].OrderNumber
	})
}

func (tz *Tz) NewFeedbackError(ctx context.Context, instanceID uuid.UUID, instanceType string, feedbackMark *bool, feedbackComment *string, userID uuid.UUID) error {
	const op = "Tz.NewFeedbackError"

	log := tz.log.With(
		slog.String("op", op),
		slog.String("instance_id", instanceID.String()),
		slog.String("instance_type", instanceType),
		slog.String("user_id", userID.String()),
	)

	log.Info("creating new feedback")

	if feedbackMark == nil {
		log.Error("feedbackMark not exists")
		return fmt.Errorf("failed to update instance feedback: feedbackMark not exists")
	}

	if *feedbackMark == false && (feedbackComment == nil || *feedbackComment == "") {
		log.Error("feedbackComment not exists for bad feedback")
		return fmt.Errorf("failed to update instance feedback: feedbackComment not exists for bad feedback")
	}

	switch instanceType {
	case "invalid":
		err := tz.repo.UpdateInvalidInstanceFeedback(ctx, instanceID, feedbackMark, feedbackComment, userID)
		if err != nil {
			log.Error("failed to update invalid instance feedback", slog.String("error", err.Error()))
			return fmt.Errorf("failed to update invalid instance feedback: %w", err)
		}
	case "missing":
		err := tz.repo.UpdateMissingInstanceFeedback(ctx, instanceID, feedbackMark, feedbackComment, userID)
		if err != nil {
			log.Error("failed to update missing instance feedback", slog.String("error", err.Error()))
			return fmt.Errorf("failed to update missing instance feedback: %w", err)
		}
	default:
		log.Error("invalid instance type", slog.String("instance_type", instanceType))
		return fmt.Errorf("invalid instance type: %s", instanceType)
	}

	log.Info("feedback created successfully")
	return nil
}

func (tz *Tz) NewVerificationFeedbackError(ctx context.Context, instanceID uuid.UUID, instanceType string, feedbackMark *bool, feedbackComment *string, userID uuid.UUID) error {
	const op = "Tz.NewFeedbackError"

	log := tz.log.With(
		slog.String("op", op),
		slog.String("instance_id", instanceID.String()),
		slog.String("instance_type", instanceType),
		slog.String("user_id", userID.String()),
	)

	log.Info("creating new feedback")

	if feedbackMark == nil {
		log.Error("feedbackMark not exists")
		return fmt.Errorf("failed to update instance feedback: feedbackMark not exists")
	}

	if *feedbackMark == false && (feedbackComment == nil || *feedbackComment == "") {
		log.Error("feedbackComment not exists for bad feedback")
		return fmt.Errorf("failed to update instance feedback: feedbackComment not exists for bad feedback")
	}

	switch instanceType {
	case "invalid":
		err := tz.repo.UpdateInvalidInstanceVerificationFeedback(ctx, instanceID, feedbackMark, feedbackComment, userID)
		if err != nil {
			log.Error("failed to update invalid instance feedback", slog.String("error", err.Error()))
			return fmt.Errorf("failed to update invalid instance feedback: %w", err)
		}
	case "missing":
		err := tz.repo.UpdateMissingInstanceVerificationFeedback(ctx, instanceID, feedbackMark, feedbackComment, userID)
		if err != nil {
			log.Error("failed to update missing instance feedback", slog.String("error", err.Error()))
			return fmt.Errorf("failed to update missing instance feedback: %w", err)
		}
	default:
		log.Error("invalid instance type", slog.String("instance_type", instanceType))
		return fmt.Errorf("invalid instance type: %s", instanceType)
	}

	log.Info("feedback created successfully")
	return nil
}

// SetGGID изменяет ggID и возвращает текущее значение
func (tz *Tz) SetGGID(newGGID int) int {
	tz.mu.Lock()
	defer tz.mu.Unlock()
	tz.ggID = newGGID
	return tz.ggID
}

// GetGGID возвращает текущий ggID
func (tz *Tz) GetGGID() int {
	tz.mu.RLock()
	defer tz.mu.RUnlock()
	return tz.ggID
}

// SetUseLlmCache изменяет useLlmCache и возвращает текущее значение
func (tz *Tz) SetUseLlmCache(useLlmCache bool) bool {
	tz.mu.Lock()
	defer tz.mu.Unlock()
	tz.useLlmCache = useLlmCache
	return tz.useLlmCache
}

// GetUseLlmCache возвращает текущее значение useLlmCache
func (tz *Tz) GetUseLlmCache() bool {
	tz.mu.RLock()
	defer tz.mu.RUnlock()
	return tz.useLlmCache
}
