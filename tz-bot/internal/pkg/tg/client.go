package tg_client

import (
	"bytes"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"repairCopilotBot/tz-bot/internal/pkg/llm"
	"strings"
)

func NewBot(token string) (*tgbotapi.BotAPI, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания бота: %v", err)
	}

	return bot, nil
}

type Config struct {
	Token  string `yaml:"token"`
	ChatID int64  `yaml:"chat_id"`
}

type Client struct {
	bot    *tgbotapi.BotAPI
	chatId int64
}

func New(bot *tgbotapi.BotAPI, chatId int64) *Client {
	return &Client{
		bot:    bot,
		chatId: chatId,
	}
}

func (c *Client) SendMessage(message string) error {

	// Создаем сообщение
	msg := tgbotapi.NewMessage(c.chatId, message)

	// Отправляем сообщение
	_, err := c.bot.Send(msg)

	if err != nil {
		log.Print("ошибка отправки сообщения в тг: %v", err.Error())
	}

	return nil
}

func (c *Client) SendMessages(messages []string) error {
	for _, message := range messages {
		// Создаем сообщение
		c.SendMessage(message)
	}

	return nil
}

// FormatForTelegram форматирует SuccessResponse в массив строк для отправки через Telegram
func FormatForTelegram(response *tz_llm_client.SuccessResponse) []string {
	const maxMessageLength = 4000 // Оставляем запас от лимита в 4096 символов

	var messages []string
	var currentMessage strings.Builder

	// Добавляем информацию о токенах в начало
	tokensInfo := fmt.Sprintf("📊 *Статистика обработки:*\n"+
		"• Prompt токены: `%d`\n"+
		"• Completion токены: `%d`\n"+
		"• Всего токенов: `%d`\n\n",
		response.Tokens.Prompt,
		response.Tokens.Completion,
		response.Tokens.Total)

	currentMessage.WriteString(tokensInfo)

	// Если нет ошибок
	if len(response.Errors) == 0 {
		currentMessage.WriteString("✅ *Ошибок не обнаружено*")
		messages = append(messages, currentMessage.String())
		return messages
	}

	// Добавляем заголовок для ошибок
	errorsHeader := fmt.Sprintf("🚨 *Обнаружено ошибок: %d*\n\n", len(response.Errors))
	currentMessage.WriteString(errorsHeader)

	// Обрабатываем каждую ошибку
	for errorIndex, err := range response.Errors {
		errorBlock := formatError(errorIndex+1, &err)

		// Проверяем, поместится ли ошибка в текущее сообщение
		if currentMessage.Len()+len(errorBlock) > maxMessageLength {
			// Сохраняем текущее сообщение и начинаем новое
			if currentMessage.Len() > 0 {
				messages = append(messages, currentMessage.String())
				currentMessage.Reset()
			}

			// Если одна ошибка слишком большая, разбиваем её по findings
			if len(errorBlock) > maxMessageLength {
				errorMessages := formatLargeError(errorIndex+1, &err, maxMessageLength)
				messages = append(messages, errorMessages...)
				continue
			}
		}

		currentMessage.WriteString(errorBlock)
	}

	// Добавляем последнее сообщение, если есть содержимое
	if currentMessage.Len() > 0 {
		messages = append(messages, currentMessage.String())
	}

	return messages
}

// formatError форматирует одну ошибку
func formatError(index int, err *tz_llm_client.Error) string {
	var builder strings.Builder

	// Заголовок ошибки
	builder.WriteString(fmt.Sprintf("🔴 *Ошибка #%d*\n", index))
	builder.WriteString(fmt.Sprintf("**Код:** `%s`\n", err.Code))
	builder.WriteString(fmt.Sprintf("**Заголовок:** %s\n", err.Title))
	builder.WriteString(fmt.Sprintf("**Тип:** `%s`\n\n", err.Kind))

	// Обрабатываем findings
	if len(err.Findings) > 0 {
		builder.WriteString("📋 *Детали:*\n")
		for findingIndex, finding := range err.Findings {
			builder.WriteString(formatFinding(findingIndex+1, &finding))
		}
	}

	builder.WriteString("━━━━━━━━━━━━━━━━━━━━\n\n")

	return builder.String()
}

// formatFinding форматирует один finding
func formatFinding(index int, finding *tz_llm_client.Finding) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("  *%d.* ", index))

	if finding.Paragraph != "" {
		builder.WriteString(fmt.Sprintf("**Параграф:** %s\n", finding.Paragraph))
	}

	if finding.Quote != "" {
		builder.WriteString(fmt.Sprintf("     💬 *Цитата:* ||%s||\n", finding.Quote))
	}

	if finding.Advice != "" {
		builder.WriteString(fmt.Sprintf("     💡 *Рекомендация:* _%s_\n", finding.Advice))
	}

	builder.WriteString("\n")

	return builder.String()
}

// formatLargeError разбивает большую ошибку на несколько сообщений
func formatLargeError(index int, err *tz_llm_client.Error, maxLength int) []string {
	var messages []string
	var currentMessage strings.Builder

	// Заголовок ошибки
	errorHeader := fmt.Sprintf("🔴 *Ошибка #%d*\n", index)
	errorHeader += fmt.Sprintf("**Код:** `%s`\n", err.Code)
	errorHeader += fmt.Sprintf("**Заголовок:** %s\n", err.Title)
	errorHeader += fmt.Sprintf("**Тип:** `%s`\n\n", err.Kind)

	currentMessage.WriteString(errorHeader)

	if len(err.Findings) > 0 {
		currentMessage.WriteString("📋 *Детали:*\n")

		for findingIndex, finding := range err.Findings {
			findingText := formatFinding(findingIndex+1, &finding)

			// Проверяем, поместится ли finding
			if currentMessage.Len()+len(findingText) > maxLength {
				// Сохраняем текущее сообщение
				currentMessage.WriteString("━━━━━━━━━━━━━━━━━━━━\n")
				messages = append(messages, currentMessage.String())

				// Начинаем новое сообщение с продолжением
				currentMessage.Reset()
				currentMessage.WriteString(fmt.Sprintf("🔴 *Ошибка #%d (продолжение)*\n\n", index))
			}

			currentMessage.WriteString(findingText)
		}
	}

	currentMessage.WriteString("━━━━━━━━━━━━━━━━━━━━\n")

	if currentMessage.Len() > 0 {
		messages = append(messages, currentMessage.String())
	}

	return messages
}

func (c *Client) SendFile(fileData []byte, filename string) error {
	reader := bytes.NewReader(fileData)
	
	file := tgbotapi.FileReader{
		Name:   filename,
		Reader: reader,
	}
	
	document := tgbotapi.NewDocument(c.chatId, file)
	
	_, err := c.bot.Send(document)
	if err != nil {
		log.Printf("ошибка отправки файла в тг: %v", err)
		return err
	}
	
	return nil
}
