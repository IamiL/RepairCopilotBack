package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"repairCopilotBot/api-gateway-service/internal/repository"
	"repairCopilotBot/tz-bot/client"
	userserviceclient "repairCopilotBot/user-service/client"
	"time"

	"github.com/google/uuid"
)

/*
API Endpoint: GET /api/admin/inspections

Описание: Возвращает список всех инспекций (версий технических заданий) с детальной информацией

Авторизация: Требуется cookie "auth_token"

HTTP Status Codes:
- 200: Успешный ответ
- 401: Неавторизован (отсутствует или невалидный токен)
- 500: Внутренняя ошибка сервера

Response JSON Schema:
Возвращает массив объектов VersionAdminDashboard:

[
  {
      "version_id": "uuid-string",                    // ID версии
      "technical_specification_name": "string",       // Название технического задания
      "user_id": "uuid-string",                       // ID пользователя
      "version_number": 1,                            // Номер версии (integer)
      "all_tokens": 1500,                             // Общее количество токенов (integer)
      "all_rubs": 25.50,                             // Общая стоимость в рублях (float)
      "number_of_errors": 12,                         // Количество ошибок (integer)
      "inspection_time": 1234567890,                  // Время инспекции в наносекундах (integer)
      "original_file_size": 2048,                     // Размер оригинального файла в байтах (integer)
      "number_of_pages": 5,                           // Количество страниц (integer)
      "created_at": "2025-01-15T10:30:00Z",          // Дата создания (RFC3339 timestamp)
      "original_file_link": "string",                 // Ссылка на оригинальный файл
      "report_file_link": "string"                    // Ссылка на файл отчета
      "created_at": "2025-01-15T10:30:00.123456789Z"    // Дата создания (Go time.Time в JSON)
  }
]

Пример ответа:
[
  {
            "version_id": "4f2745f0-0ba5-4edf-b0d6-6c1d4079a8e9",
            "technical_specification_name": "ТЗ_ИС ''MES НТПЗ''-15082025",
            "user_id": "b84cdfda-81cc-11f0-9bae-acde48001122",
            "version_number": 1,
            "all_tokens": 118402,
            "all_rubs": 59.201,
            "number_of_errors": 19,
            "inspection_time": 3738242573,
            "original_file_size": 49761,
            "original_file_link": "https://timuroid.ru/docx/f038cdd6-f53c-4c7c-b3f8-e22fb05c633f.docx",
            "report_file_link": "https://timuroid.ru/reports/.docx",
            "created_at": "2025-08-25T19:55:29.596257Z"
        },
]
*/

func GetInspectionsHandler(
	log *slog.Logger,
	tzBotClient *client.Client,
	userServiceClient *userserviceclient.UserClient,
	sessionRepo *repository.SessionRepository,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.GetInspectionsHandler"

		log := log.With(slog.String("op", op))
		log.Info("inspections request started")

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

		// Создаем контекст с таймаутом
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		// Вызываем gRPC метод GetAllVersionsAdminDashboard
		inspections, err := tzBotClient.GetAllVersionsAdminDashboard(ctx, uuid.Nil)
		if err != nil {
			log.Error("failed to get inspections", slog.String("error", err.Error()))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Обогащаем версии именами пользователей
		if inspections != nil && len(inspections) > 0 {
			// Собираем уникальные ID пользователей из versions
			userIDsMap := make(map[string]struct{})
			for _, version := range inspections {
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
					for _, version := range inspections {
						if fullName, exists := fullNames[version.UserId]; exists {
							version.User.FirstName = fullName.FirstName
							version.User.LastName = fullName.LastName
						}
					}
				}
			}
		}

		// Отдаем данные в JSON формате
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(inspections); err != nil {
			log.Error("failed to encode response", slog.String("error", err.Error()))
			return
		}

		log.Info("inspections request completed successfully", slog.Int("inspections_count", len(inspections)))
	}
}
