package tg_client

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
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
