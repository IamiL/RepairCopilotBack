package tgapp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"repairCopilotBot/tz-bot/internal/pkg/logger/sl"
	"strconv"
	"time"

	"repairCopilotBot/tz-bot/internal/config"
	tzservice "repairCopilotBot/tz-bot/internal/service/tz"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type App struct {
	bot         *bot.Bot
	config      *config.TelegramBotConfig
	tzService   TZServiceInterface
	log         *slog.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	waitingGGID map[int64]bool
}

type TZServiceInterface interface {
	SetGGID(newGGID int) int
	GetGGID() int
	SetUseLlmCache(useLlmCache bool) bool
	GetUseLlmCache() bool
}

func New(log *slog.Logger, config *config.TelegramBotConfig, tzService *tzservice.Tz) (*App, error) {
	ctx, cancel := context.WithCancel(context.Background())

	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			log.Debug("Получено неизвестное сообщение",
				slog.Any("update", update))
		}),
	}

	b, err := bot.New(config.Token, opts...)
	if err != nil {
		log.Error("failed to create telegram bot", "error", sl.Err(err))
		cancel()
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	app := &App{
		bot:         b,
		config:      config,
		tzService:   tzService,
		log:         log,
		ctx:         ctx,
		cancel:      cancel,
		waitingGGID: make(map[int64]bool),
	}

	app.registerHandlers()

	return app, nil
}

func (a *App) Start() error {
	a.log.Info("Запуск Telegram-бота")

	// Отправляем сообщение о запуске
	err := a.sendStartupMessage()
	if err != nil {
		a.log.Error("Ошибка отправки сообщения о запуске", slog.Any("error", err))
	}

	if a.config.UseWebhooks {
		return a.startWebhook()
	}

	go a.bot.Start(a.ctx)
	a.log.Info("Telegram-бот запущен (long polling)")
	return nil
}

func (a *App) Stop() error {
	a.log.Info("Остановка Telegram-бота")

	// Отправляем сообщение об остановке
	err := a.sendShutdownMessage()
	if err != nil {
		a.log.Error("Ошибка отправки сообщения об остановке", slog.Any("error", err))
	}

	a.cancel()
	return nil
}

func (a *App) startWebhook() error {
	a.log.Info("Запуск webhook режима",
		slog.String("host", a.config.WebhookHost),
		slog.Int("port", a.config.WebhookPort),
		slog.String("path", a.config.WebhookPath))

	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", a.config.WebhookPort), a.bot.WebhookHandler())
		if err != nil {
			a.log.Error("Ошибка webhook сервера", slog.Any("error", err))
		}
	}()

	a.bot.StartWebhook(a.ctx)
	return nil
}

func (a *App) registerHandlers() {
	// Обработчик команды /start
	a.bot.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeCommand, a.handleStart)

	// Обработчик команды /status
	a.bot.RegisterHandler(bot.HandlerTypeMessageText, "/status", bot.MatchTypeCommand, a.handleStatus)

	// Обработчик callback кнопок
	a.bot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "", bot.MatchTypePrefix, a.handleCallbackQuery)

	// Обработчик текстовых сообщений (для ввода ggID)
	a.bot.RegisterHandler(bot.HandlerTypeMessageText, "", bot.MatchTypeContains, a.handleTextMessage)
}

func (a *App) handleStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "📊 Текущие настройки", CallbackData: "status"},
			},
			{
				{Text: "🔢 Изменить ggID", CallbackData: "change_ggid"},
				{Text: "💾 Переключить кэш", CallbackData: "toggle_cache"},
			},
		},
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: `🤖 <b>Управление системой TZ Bot</b>

Добро пожаловать в панель управления TZ Bot!

Доступные операции:
• Просмотр текущих настроек
• Изменение ggID для системы
• Включение/выключение кэша LLM

Выберите действие:`,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	if err != nil {
		a.log.Error("Ошибка отправки сообщения start", slog.Any("error", err))
	}
}

func (a *App) handleStatus(ctx context.Context, b *bot.Bot, update *models.Update) {
	currentGGID := a.tzService.GetGGID()
	currentCache := a.tzService.GetUseLlmCache()

	cacheIcon := "❌"
	cacheStatus := "Выключен"
	if currentCache {
		cacheIcon = "✅"
		cacheStatus = "Включен"
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "🔢 Изменить ggID", CallbackData: "change_ggid"},
				{Text: "💾 Переключить кэш", CallbackData: "toggle_cache"},
			},
		},
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: fmt.Sprintf(`📊 <b>Текущие настройки системы</b>

🔢 <b>ggID:</b> <code>%d</code>
%s <b>Кэш LLM:</b> %s

<i>Последнее обновление: %s</i>`,
			currentGGID,
			cacheIcon,
			cacheStatus,
			time.Now().Format("15:04:05 02.01.2006")),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	if err != nil {
		a.log.Error("Ошибка отправки статуса", slog.Any("error", err))
	}
}

func (a *App) handleCallbackQuery(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery

	// Получаем chat ID из callback query
	var chatID int64

	// Проверяем есть ли у callback.Message поле Message (доступное сообщение)
	if callback.Message.Message != nil {
		chatID = callback.Message.Message.Chat.ID
	} else {
		a.log.Error("Не удалось получить chat ID из callback message")
		return
	}

	// Отвечаем на callback query
	_, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callback.ID,
	})
	if err != nil {
		a.log.Error("Ошибка ответа на callback", slog.Any("error", err))
	}

	switch callback.Data {
	case "status":
		a.sendStatus(ctx, b, chatID)
	case "change_ggid":
		a.handleChangeGGID(ctx, b, chatID)
	case "toggle_cache":
		a.handleToggleCache(ctx, b, chatID)
	}
}

func (a *App) handleTextMessage(ctx context.Context, b *bot.Bot, update *models.Update) {
	// Проверяем, ожидается ли ввод ggID от этого пользователя
	if !a.waitingGGID[update.Message.Chat.ID] {
		return
	}

	// Пытаемся преобразовать текст в число
	newGGID, err := strconv.Atoi(update.Message.Text)
	if err != nil || newGGID <= 0 {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Неверный формат ggID. Пожалуйста, введите положительное целое число:",
		})
		if err != nil {
			a.log.Error("Ошибка отправки сообщения об ошибке", slog.Any("error", err))
		}
		return
	}

	// Устанавливаем новый ggID
	actualGGID := a.tzService.SetGGID(newGGID)
	a.waitingGGID[update.Message.Chat.ID] = false

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "📊 Показать статус", CallbackData: "status"},
			},
		},
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: fmt.Sprintf(`✅ <b>ggID успешно изменен!</b>

🔢 <b>Новое значение:</b> <code>%d</code>

<i>Изменение вступило в силу немедленно.</i>`, actualGGID),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	if err != nil {
		a.log.Error("Ошибка отправки подтверждения", slog.Any("error", err))
	}

	// Логируем изменение
	a.log.Info("ggID изменен через Telegram бот",
		slog.Int("new_ggid", actualGGID),
		slog.Int64("chat_id", update.Message.Chat.ID))
}

func (a *App) sendStatus(ctx context.Context, b *bot.Bot, chatID int64) {
	currentGGID := a.tzService.GetGGID()
	currentCache := a.tzService.GetUseLlmCache()

	cacheIcon := "❌"
	cacheStatus := "Выключен"
	if currentCache {
		cacheIcon = "✅"
		cacheStatus = "Включен"
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "🔢 Изменить ggID", CallbackData: "change_ggid"},
				{Text: "💾 Переключить кэш", CallbackData: "toggle_cache"},
			},
		},
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text: fmt.Sprintf(`📊 <b>Текущие настройки системы</b>

🔢 <b>ggID:</b> <code>%d</code>
%s <b>Кэш LLM:</b> %s

<i>Последнее обновление: %s</i>`,
			currentGGID,
			cacheIcon,
			cacheStatus,
			time.Now().Format("15:04:05 02.01.2006")),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	if err != nil {
		a.log.Error("Ошибка отправки статуса", slog.Any("error", err))
	}
}

func (a *App) handleChangeGGID(ctx context.Context, b *bot.Bot, chatID int64) {
	a.waitingGGID[chatID] = true

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "📊 Показать текущий статус", CallbackData: "status"},
			},
		},
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text: `🔢 <b>Изменение ggID</b>

Пожалуйста, введите новое значение ggID (положительное целое число):

<i>Например: 1, 2, 3, etc.</i>`,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	if err != nil {
		a.log.Error("Ошибка запроса ggID", slog.Any("error", err))
	}
}

func (a *App) handleToggleCache(ctx context.Context, b *bot.Bot, chatID int64) {
	currentCache := a.tzService.GetUseLlmCache()
	newCache := a.tzService.SetUseLlmCache(!currentCache)

	cacheIcon := "❌"
	cacheStatus := "выключен"
	actionIcon := "🔴"
	if newCache {
		cacheIcon = "✅"
		cacheStatus = "включен"
		actionIcon = "🟢"
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "📊 Показать статус", CallbackData: "status"},
			},
		},
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text: fmt.Sprintf(`%s <b>Кэш LLM %s!</b>

%s <b>Статус кэша:</b> %s

<i>Изменение вступило в силу немедленно.</i>`,
			actionIcon, cacheStatus, cacheIcon, cacheStatus),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	if err != nil {
		a.log.Error("Ошибка отправки статуса кэша", slog.Any("error", err))
	}

	// Логируем изменение
	a.log.Info("Статус кэша LLM изменен через Telegram бот",
		slog.Bool("new_cache_status", newCache),
		slog.Int64("chat_id", chatID))
}

func (a *App) sendStartupMessage() error {
	chatID, err := strconv.ParseInt(a.config.ChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat_id format: %w", err)
	}

	currentGGID := a.tzService.GetGGID()
	currentCache := a.tzService.GetUseLlmCache()

	cacheIcon := "❌"
	cacheStatus := "Выключен"
	if currentCache {
		cacheIcon = "✅"
		cacheStatus = "Включен"
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "📊 Показать статус", CallbackData: "status"},
			},
		},
	}

	_, err = a.bot.SendMessage(context.Background(), &bot.SendMessageParams{
		ChatID: chatID,
		Text: fmt.Sprintf(`🚀 <b>TZ Bot запущен!</b>

📊 <b>Текущие настройки:</b>
🔢 <b>ggID:</b> <code>%d</code>
%s <b>Кэш LLM:</b> %s

<i>Время запуска: %s</i>

Бот готов к работе! 🤖`,
			currentGGID,
			cacheIcon,
			cacheStatus,
			time.Now().Format("15:04:05 02.01.2006")),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})

	return err
}

func (a *App) sendShutdownMessage() error {
	chatID, err := strconv.ParseInt(a.config.ChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat_id format: %w", err)
	}

	_, err = a.bot.SendMessage(context.Background(), &bot.SendMessageParams{
		ChatID: chatID,
		Text: fmt.Sprintf(`🔴 <b>TZ Bot остановлен</b>

<i>Время остановки: %s</i>`, time.Now().Format("15:04:05 02.01.2006")),
		ParseMode: models.ParseModeHTML,
	})

	return err
}
