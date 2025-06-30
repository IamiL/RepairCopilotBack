package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"repairCopilotBot/tz-bot/internal/service/tz"
	tz_llm_client "repairCopilotBot/tz-bot/package/llm"
	tg_client "repairCopilotBot/tz-bot/package/tg"
	word_parser_client "repairCopilotBot/tz-bot/package/word-parser"
	"strings"
)

type NewTzResponse struct {
	Text string               `json:"text"`
	Err  []NewTzErrorResponse `json:"errors"`
}

type NewTzErrorResponse struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
	Text  string `json:"description"`
	Type  string `json:"type"`
}

func NewTzHandler(
	log *slog.Logger,
	wordConverterClient *word_parser_client.Client,
	llmClient *tz_llm_client.Client,
	tgClient *tg_client.Client,
) func(
	w http.ResponseWriter, r *http.Request,
) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Парсим multipart form (максимум 10MB)
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, "Ошибка парсинга формы", http.StatusBadRequest)
			return
		}

		// Получаем файл из формы
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Ошибка получения файла", http.StatusBadRequest)
			return
		}
		defer file.Close()

		log.Info("Получен файл: %s, размер: %d байт\n", header.Filename, header.Size)

		// Отправляем файл в API
		response, err := wordConverterClient.Convert(file, header.Filename)
		if err != nil {
			log.Info("Ошибка отправки файла в API: %v\n" + err.Error())
			http.Error(w, "Ошибка обработки файла", http.StatusInternalServerError)
			return
		}

		// Выводим ответ в консоль
		log.Info("Ответ от API:\n")
		log.Info("Filename: %s\n", response.Filename)
		log.Info("Length: %d\n", response.Length)
		log.Info("Success: %t\n", response.Success)
		log.Info("Text: %s\n", response.Text)

		log.Info("отправляем запрос к llm")

		//writeStringToFile(response.Text, "1.txt")

		result, err := llmClient.Analyze(response.Text)
		if err != nil {
			log.Error("Error: \n", err)
		}

		if result == nil {
			log.Info("пустой ответ от llm")
		}

		tgMessages := tg_client.FormatForTelegram(result)

		err = tgClient.SendMessage(tgMessages)

		textToWeb := response.Text

		errorsResp_temp := make([]NewTzErrorResponse, 0, 100)

		errorId := 0

		for _, tzError := range result.Errors {
			for _, finding := range tzError.Findings {
				if len(finding.Quote) < 4 {
					continue
				}

				textToWeb = tz.HighlightPhraseIgnoreCase(textToWeb, finding.Quote, errorId)

				errorsResp_temp = append(errorsResp_temp, NewTzErrorResponse{
					Id:    errorId,
					Title: tzError.Code + " " + tzError.Title,
					Text:  finding.Advice,
					Type:  "error",
				})

				errorId++
			}
		}

		textToWeb = tz.FixHTMLTags(textToWeb)

		log.Info("ТЕКСТ НА ФРОНТ:")
		log.Info(textToWeb)
		log.Info("КОНЕЦ ТЕКСТА НА ФРОНТ")

		ids_temp := tz.ExtractErrorIds(textToWeb)

		ids, err := tz.StringsToInts(ids_temp)
		if err != nil {
			log.Error("ошибка преобразования массива ids_string в ids_int")
		}

		errorsResponse := SortByIdOrderFiltered(errorsResp_temp, ids)

		log.Info("ОШИБКИ НА ФРОНТ")
		for _, er := range errorsResponse {
			fmt.Println("errId: ", er.Id)
			fmt.Println("errTitle: ", er.Title)
			fmt.Println("_________________________________________________")
		}
		log.Info("КОНЕЦ ОШИБОК НА ФРОНТ")

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(
			NewTzResponse{
				Text: textToWeb,
				Err:  errorsResponse,
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
