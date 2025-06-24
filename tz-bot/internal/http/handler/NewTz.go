package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	tz_llm_client "repairCopilotBot/tz-bot/package/llm"
	tg_client "repairCopilotBot/tz-bot/package/tg"
	word_parser_client "repairCopilotBot/tz-bot/package/word-parser"
	"strings"
)

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

		writeStringToFile(response.Text, "1.txt")

		result, err := llmClient.Analyze(response.Text)
		if err != nil {
			log.Error("Error: \n", err)
		}

		if result == nil {
			log.Info("пустой ответ от llm")
		}

		tgMessages := tg_client.FormatForTelegram(result)

		err = tgClient.SendMessage(tgMessages)
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
