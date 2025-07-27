package handler

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log/slog"
	"net/http"
	"os"
	"repairCopilotBot/tz-bot/client"
	"strings"
)

type NewTzResponse struct {
	Text        string               `json:"text"`
	Err         []NewTzErrorResponse `json:"errors"`
	ErrsMissing []NewTzErrorResponse `json:"errors_missing"`
}

type NewTzErrorResponse struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
	Text  string `json:"description"`
	Type  string `json:"type"`
}

func NewTzHandler(
	log *slog.Logger,
	tzBotClient *client.Client,
) func(
	w http.ResponseWriter, r *http.Request,
) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Info("NewTzHandler started2")

		// Парсим multipart form (максимум 10MB)
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, "Ошибка парсинга формы", http.StatusBadRequest)
			return
		}

		log.Info("парсинг multipart успешен")

		// Получаем файл из формы
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Ошибка получения файла", http.StatusBadRequest)
			return
		}
		defer file.Close()

		log.Info(fmt.Sprintf("Получен файл: %s, размер: %d байт\n", header.Filename, header.Size))

		// Читаем файл в байты
		fileBytes, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Ошибка чтения файла", http.StatusInternalServerError)
			return
		}

		filename := header.Filename

		fmt.Println("точка 1")

		requestID, err := uuid.NewUUID()
		if err != nil {
			http.Error(w, "error creating requestID", http.StatusInternalServerError)
		}

		fmt.Println("точка 2")

		checkTzResult, err := tzBotClient.CheckTz(r.Context(), fileBytes, filename, requestID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Println("точка 3-2")

		errorsResp := make([]NewTzErrorResponse, len(checkTzResult.Errors), len(checkTzResult.Errors))

		for i, e := range checkTzResult.Errors {
			errorsResp[i] = NewTzErrorResponse{
				Id:    e.Id,
				Title: e.Title,
				Text:  e.Text,
				Type:  e.Type,
			}
		}

		errorsMissing := make([]NewTzErrorResponse, len(checkTzResult.ErrorsMissing), len(checkTzResult.ErrorsMissing))
		for i, e := range checkTzResult.ErrorsMissing {
			errorsMissing[i] = NewTzErrorResponse{
				Id:    e.Id,
				Title: e.Title,
				Text:  e.Text,
				Type:  e.Type,
			}
		}

		log.Info("отдали ошибок в тексте: ", len(errorsResp), ", ошибок по документу: ", len(errorsMissing))

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(
			NewTzResponse{
				Text:        checkTzResult.HtmlText,
				Err:         errorsResp,
				ErrsMissing: errorsMissing,
			},
		); err != nil {
			log.Error("ошибка закодирования ответа json в http resp")
		}

		if err != nil {
			log.Error("Error: \n", err)
		}
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

// SortByIdOrderFiltered - альтернативная версия, которая возвращает только те элементы,
// ID которых есть во втором массиве, в точном порядке
func SortByIdOrderFiltered(responses []NewTzErrorResponse, idOrder []int) []NewTzErrorResponse {
	// Создаем map для быстрого поиска структур по ID
	idToResponse := make(map[int]NewTzErrorResponse)
	for _, response := range responses {
		idToResponse[response.Id] = response
	}

	// Создаем результирующий массив в нужном порядке
	var result []NewTzErrorResponse
	for _, id := range idOrder {
		if response, exists := idToResponse[id]; exists {
			result = append(result, response)
		}
	}

	return result
}
