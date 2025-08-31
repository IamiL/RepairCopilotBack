package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"repairCopilotBot/api-gateway-service/internal/repository"
	"repairCopilotBot/tz-bot/client"
	userserviceclient "repairCopilotBot/user-service/client"
	v1 "repairCopilotBot/user-service/pkg/user/v1"
	"sync"
	"time"

	"github.com/google/uuid"
)

type AdminDashboardUserInfo struct {
	UserID   string `json:"user_id"`
	Login    string `json:"login"`
	IsAdmin1 bool   `json:"is_admin1"`
	IsAdmin2 bool   `json:"is_admin2"`
}

type AdminDashboardVersionWithErrorCounts struct {
	VersionId                  string   `json:"version_id"`
	TechnicalSpecificationId   string   `json:"technical_specification_id"`
	TechnicalSpecificationName string   `json:"technical_specification_name"`
	UserId                     string   `json:"user_id"`
	VersionNumber              int32    `json:"version_number"`
	CreatedAt                  string   `json:"created_at"`
	UpdatedAt                  string   `json:"updated_at"`
	OriginalFileId             string   `json:"original_file_id"`
	OutHtml                    string   `json:"out_html"`
	Css                        string   `json:"css"`
	CheckedFileId              string   `json:"checked_file_id"`
	AllRubs                    *float64 `json:"all_rubs"`
	AllTokens                  *int64   `json:"all_tokens"`
	InspectionTimeNanoseconds  *int64   `json:"inspection_time_nanoseconds"`
	InvalidErrorCount          int32    `json:"invalid_error_count"`
	MissingErrorCount          int32    `json:"missing_error_count"`
}

type AdminDashboardVersionStatistics struct {
	TotalVersions                    int64    `json:"total_versions"`
	TotalTokens                      *int64   `json:"total_tokens"`
	TotalRubs                        *float64 `json:"total_rubs"`
	AverageInspectionTimeNanoseconds *int64   `json:"average_inspection_time_nanoseconds"`
}

type AdminDashboardActionLog struct {
	ID         int       `json:"id"`
	Action     string    `json:"action"`
	UserID     uuid.UUID `json:"user_id"`
	CreateAt   time.Time `json:"created_at"`
	ActionType int       `json:"action_type"`
}

type AdminDashboardResponse struct {
	Users      []*v1.UserInfo                          `json:"users"`
	Versions   []*client.VersionAdminDashboard         `json:"versions"`
	Statistics *AdminDashboardVersionStatistics        `json:"statistics"`
	ActionLogs []AdminDashboardActionLog               `json:"action_logs"`
	Feedbacks  *[]*client.GetFeedbacksFeedbackResponse `json:"feedbacks"`
}

func GetAdminDashboardHandler(
	log *slog.Logger,
	userServiceClient *userserviceclient.UserClient,
	tzBotClient *client.Client,
	sessionRepo *repository.SessionRepository,
	actionLogRepo repository.ActionLogRepository,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.GetAdminDashboardHandler"

		log := log.With(slog.String("op", op))
		log.Info("admin dashboard request started")

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

		// Создаем контекст с таймаутом для всех запросов
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		// Структуры для хранения результатов
		var users []*v1.UserInfo
		var versions []*client.VersionAdminDashboard
		var statistics *client.VersionStatistics
		var actionLogs []repository.ActionLog
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errors []error

		// Функция для добавления ошибок потокобезопасно
		addError := func(err error) {
			mu.Lock()
			defer mu.Unlock()
			errors = append(errors, err)
		}

		// Запрос 1: GetAllUsers из user-service
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Info("fetching users from user-service")

			usersList, err := userServiceClient.GetAllUsers(ctx)
			if err != nil || usersList == nil {
				log.Error("failed to get users", slog.String("error", err.Error()))
				addError(err)
				return
			}

			mu.Lock()
			users = usersList
			mu.Unlock()
			log.Info("users fetched successfully", slog.Int("count", len(users)))
		}()

		// Запрос 2: GetAllVersions из tz-bot
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Info("fetching versions from tz-bot")

			versionsList, err := tzBotClient.GetAllVersionsAdminDashboard(ctx)
			if err != nil {
				log.Error("failed to get versions", slog.String("error", err.Error()))
				addError(err)
				return
			}

			mu.Lock()
			versions = versionsList
			mu.Unlock()
			log.Info("versions fetched successfully", slog.Int("count", len(versionsList)))
		}()

		// Запрос 3: GetVersionStatistics из tz-bot
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Info("fetching statistics from tz-bot")

			stats, err := tzBotClient.GetVersionStatistics(ctx)
			if err != nil {
				log.Error("failed to get statistics", slog.String("error", err.Error()))
				addError(err)
				return
			}

			mu.Lock()
			statistics = stats
			mu.Unlock()
			log.Info("statistics fetched successfully", slog.Int64("total_versions", stats.TotalVersions))
		}()

		// Запрос 4: GetAllActionLogs из action log repository
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Info("fetching action logs")

			logs, err := actionLogRepo.GetAllActionLogs(ctx)
			if err != nil {
				log.Error("failed to get action logs", slog.String("error", err.Error()))
				addError(err)
				return
			}

			mu.Lock()
			actionLogs = logs
			mu.Unlock()
			log.Info("action logs fetched successfully", slog.Int("count", len(logs)))
		}()

		// Ждем завершения всех запросов
		wg.Wait()

		// Проверяем наличие критических ошибок
		if len(errors) > 0 {
			log.Error("errors occurred during data fetching", slog.Int("error_count", len(errors)))
			// Возвращаем ошибку только если все запросы провалились
			if len(errors) == 4 {
				http.Error(w, "Failed to fetch dashboard data", http.StatusInternalServerError)
				return
			}
		}

		// Обогащаем версии именами пользователей
		if versions != nil && len(versions) > 0 {
			// Собираем уникальные ID пользователей из versions
			userIDsMap := make(map[string]struct{})
			for _, version := range versions {
				if version.UserId != "" {
					userIDsMap[version.UserId] = struct{}{}
				}
			}

			// Преобразуем в slice
			userIDs := make([]string, 0, len(userIDsMap))
			for id := range userIDsMap {
				userIDs = append(userIDs, id)
			}

			// Получаем имена пользователей если есть ID
			if len(userIDs) > 0 {
				log.Info("fetching user names", slog.Int("user_ids_count", len(userIDs)))

				fullNames, err := userServiceClient.GetFullNamesById(ctx, userIDs)
				if err != nil {
					log.Error("failed to get user names", slog.String("error", err.Error()))
				} else {
					log.Info("user names fetched successfully", slog.Int("names_count", len(fullNames)))

					// Обогащаем versions именами
					for _, version := range versions {
						if fullName, exists := fullNames[version.UserId]; exists {
							version.User.FirstName = fullName.FirstName
							version.User.LastName = fullName.LastName
						}
					}
				}
			}
		}

		// Конвертируем пользователей
		//dashboardUsers := make([]AdminDashboardUserInfo, len(users))
		//for i, user := range users {
		//	dashboardUsers[i] = AdminDashboardUserInfo{
		//		UserID:   user.UserID,
		//		Login:    user.Login,
		//		IsAdmin1: user.IsAdmin1,
		//		IsAdmin2: user.IsAdmin2,
		//	}
		//}

		// Конвертируем версии
		//dashboardVersions := make([]AdminDashboardVersionWithErrorCounts, len(versions))
		//for i, version := range versions {
		//	dashboardVersions[i] = AdminDashboardVersionWithErrorCounts{
		//		VersionId:                  version.VersionId,
		//		TechnicalSpecificationId:   version.TechnicalSpecificationId,
		//		TechnicalSpecificationName: version.TechnicalSpecificationName,
		//		UserId:                     version.UserId,
		//		VersionNumber:              version.VersionNumber,
		//		CreatedAt:                  version.CreatedAt,
		//		UpdatedAt:                  version.UpdatedAt,
		//		OriginalFileId:             version.OriginalFileId,
		//		OutHtml:                    version.OutHtml,
		//		Css:                        version.Css,
		//		CheckedFileId:              version.CheckedFileId,
		//		AllRubs:                    version.AllRubs,
		//		AllTokens:                  version.AllTokens,
		//		InspectionTimeNanoseconds:  version.InspectionTimeNanoseconds,
		//		InvalidErrorCount:          version.InvalidErrorCount,
		//		MissingErrorCount:          version.MissingErrorCount,
		//	}
		//}

		// Конвертируем статистику
		var dashboardStatistics *AdminDashboardVersionStatistics
		if statistics != nil {
			dashboardStatistics = &AdminDashboardVersionStatistics{
				TotalVersions:                    statistics.TotalVersions,
				TotalTokens:                      statistics.TotalTokens,
				TotalRubs:                        statistics.TotalRubs,
				AverageInspectionTimeNanoseconds: statistics.AverageInspectionTimeNanoseconds,
			}
		}

		// Конвертируем логи действий
		dashboardActionLogs := make([]AdminDashboardActionLog, len(actionLogs))
		for i, log := range actionLogs {
			dashboardActionLogs[i] = AdminDashboardActionLog{
				ID:         log.ID,
				Action:     log.Action,
				UserID:     log.UserID,
				CreateAt:   log.CreateAt,
				ActionType: log.ActionType,
			}
		}

		feedbacks, err := tzBotClient.GetFeedbacks(ctx, uuid.Nil)
		if err != nil {
			log.Error("failed to get inspections", slog.String("error", err.Error()))
			//http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if feedbacks != nil && len(*feedbacks) > 0 {
			// Собираем уникальные ID пользователей из versions
			userIDsMap := make(map[string]struct{})
			for _, feedback := range *feedbacks {
				if feedback.FeedbackUser != "" {
					userIDsMap[feedback.FeedbackUser] = struct{}{}
				}
			}

			// Преобразуем в slice
			userIDs := make([]string, 0, len(userIDsMap))
			for id := range userIDsMap {
				userIDs = append(userIDs, id)
			}

			// Получаем имена пользователей если есть ID
			if len(userIDs) > 0 {
				log.Info("fetching user names", slog.Int("user_ids_count", len(userIDs)))

				fullNames, err := userServiceClient.GetFullNamesById(ctx, userIDs)
				if err != nil {
					log.Error("failed to get user names", slog.String("error", err.Error()))
				} else {
					log.Info("user names fetched successfully", slog.Int("names_count", len(fullNames)))

					// Обогащаем versions именами
					for _, feedback := range *feedbacks {
						if fullName, exists := fullNames[feedback.FeedbackUser]; exists {
							feedback.User.FirstName = fullName.FirstName
							feedback.User.LastName = fullName.LastName
						}
					}
				}
			}
		}

		response := AdminDashboardResponse{
			Users:      users,
			Versions:   versions,
			Statistics: dashboardStatistics,
			ActionLogs: dashboardActionLogs,
			Feedbacks:  feedbacks,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
			return
		}

		log.Info("admin dashboard request completed successfully",
			slog.Int("users_count", len(users)),
			slog.Int("versions_count", len(versions)),
			slog.Bool("statistics_available", statistics != nil),
			slog.Int("action_logs_count", len(actionLogs)))
	}
}
