package telegram

import (
	"context"
	"fmt"
	"log"

	"repairCopilotBot/telegram-bot/internal/domain/models"
	"repairCopilotBot/telegram-bot/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	bot     *tgbotapi.BotAPI
	service *service.Service
}

func NewHandler(botToken string, svc *service.Service) (*Handler, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	return &Handler{
		bot:     bot,
		service: svc,
	}, nil
}

// Start запускает обработку сообщений от Telegram
func (h *Handler) Start(ctx context.Context) error {
	log.Printf("Authorized on account %s", h.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := h.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update := <-updates:
			if update.Message == nil {
				continue
			}

			go h.handleMessage(ctx, update.Message)
		}
	}
}

// handleMessage обрабатывает входящее сообщение
func (h *Handler) handleMessage(ctx context.Context, message *tgbotapi.Message) {
	tgUserID := message.From.ID

	// Получаем или создаем пользователя и его состояние
	_, state, err := h.service.GetOrCreateUser(ctx, tgUserID)
	if err != nil {
		h.sendMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		log.Printf("Error getting or creating user: %v", err)
		return
	}

	// Обрабатываем команды
	if message.IsCommand() {
		h.handleCommand(ctx, message, state)
		return
	}

	// Обрабатываем текстовые сообщения в зависимости от состояния
	h.handleText(ctx, message, state)
}

// handleCommand обрабатывает команды бота
func (h *Handler) handleCommand(ctx context.Context, message *tgbotapi.Message, state *models.UserState) {
	switch message.Command() {
	case "start":
		h.handleStart(ctx, message, state)
	case "login":
		h.handleLoginCommand(ctx, message, state)
	case "startchat":
		h.handleStartChat(ctx, message, state)
	case "endchat":
		h.handleEndChat(ctx, message, state)
	default:
		h.sendMessage(message.Chat.ID, "Неизвестная команда. Используйте /start для начала.")
	}
}

// handleStart обрабатывает команду /start
func (h *Handler) handleStart(ctx context.Context, message *tgbotapi.Message, state *models.UserState) {
	welcomeText := "Добро пожаловать! Я бот для общения с нейросетью.\n\n"

	if state.State == models.StateUnauthorized || state.State == models.StateAwaitingLogin || state.State == models.StateAwaitingPassword {
		welcomeText += "Для начала работы вам нужно авторизоваться.\n"
		welcomeText += "Используйте команду /login для входа в систему."
	} else {
		welcomeText += "Вы уже авторизованы!\n"
		welcomeText += "Используйте /startchat для начала чата с нейросетью."
	}

	h.sendMessage(message.Chat.ID, welcomeText)
}

// handleLoginCommand начинает процесс авторизации
func (h *Handler) handleLoginCommand(ctx context.Context, message *tgbotapi.Message, state *models.UserState) {
	if state.State == models.StateAuthorized || state.State == models.StateInChat {
		h.sendMessage(message.Chat.ID, "Вы уже авторизованы!")
		return
	}

	// Переводим пользователя в состояние ожидания логина
	if err := h.service.UpdateState(ctx, message.From.ID, models.StateAwaitingLogin); err != nil {
		h.sendMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		log.Printf("Error updating state: %v", err)
		return
	}

	h.sendMessage(message.Chat.ID, "Введите ваш логин:")
}

// handleStartChat начинает новый чат с нейросетью
func (h *Handler) handleStartChat(ctx context.Context, message *tgbotapi.Message, state *models.UserState) {
	if state.State != models.StateAuthorized {
		h.sendMessage(message.Chat.ID, "Сначала необходимо авторизоваться. Используйте /login")
		return
	}

	chatID, err := h.service.StartChat(ctx, message.From.ID)
	if err != nil {
		h.sendMessage(message.Chat.ID, "Не удалось начать чат. Попробуйте позже.")
		log.Printf("Error starting chat: %v", err)
		return
	}

	msg := fmt.Sprintf("Чат начат! (ID: %s)\nТеперь вы можете отправлять сообщения.\nДля завершения чата используйте /endchat", chatID.String())
	h.sendMessage(message.Chat.ID, msg)
}

// handleEndChat завершает текущий чат
func (h *Handler) handleEndChat(ctx context.Context, message *tgbotapi.Message, state *models.UserState) {
	if state.State != models.StateInChat {
		h.sendMessage(message.Chat.ID, "У вас нет активного чата.")
		return
	}

	if err := h.service.FinishChat(ctx, message.From.ID); err != nil {
		h.sendMessage(message.Chat.ID, "Не удалось завершить чат. Попробуйте позже.")
		log.Printf("Error finishing chat: %v", err)
		return
	}

	h.sendMessage(message.Chat.ID, "Чат завершен. Используйте /startchat для начала нового чата.")
}

// handleText обрабатывает текстовые сообщения в зависимости от состояния
func (h *Handler) handleText(ctx context.Context, message *tgbotapi.Message, state *models.UserState) {
	switch state.State {
	case models.StateUnauthorized:
		h.sendMessage(message.Chat.ID, "Пожалуйста, авторизуйтесь с помощью /login")

	case models.StateAwaitingLogin:
		// Сохраняем введенный логин
		if err := h.service.SaveLoginAttempt(ctx, message.From.ID, message.Text); err != nil {
			h.sendMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
			log.Printf("Error saving login attempt: %v", err)
			return
		}

		// Переводим в состояние ожидания пароля
		if err := h.service.UpdateState(ctx, message.From.ID, models.StateAwaitingPassword); err != nil {
			h.sendMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
			log.Printf("Error updating state: %v", err)
			return
		}

		h.sendMessage(message.Chat.ID, "Введите ваш пароль:")

	case models.StateAwaitingPassword:
		// Получаем сохраненный логин
		currentState, err := h.service.GetUserState(ctx, message.From.ID)
		if err != nil || currentState.LoginAttempt == nil {
			h.sendMessage(message.Chat.ID, "Произошла ошибка. Попробуйте начать авторизацию заново с /login")
			log.Printf("Error getting user state: %v", err)
			return
		}

		// Пытаемся авторизоваться
		if err := h.service.AuthenticateUser(ctx, message.From.ID, *currentState.LoginAttempt, message.Text); err != nil {
			h.sendMessage(message.Chat.ID, "Неверный логин или пароль. Попробуйте снова с /login")
			log.Printf("Authentication failed: %v", err)

			// Возвращаем в состояние неавторизован
			_ = h.service.UpdateState(ctx, message.From.ID, models.StateUnauthorized)
			return
		}

		h.sendMessage(message.Chat.ID, "Вы успешно авторизованы! Используйте /startchat для начала чата с нейросетью.")

	case models.StateAuthorized:
		h.sendMessage(message.Chat.ID, "Используйте /startchat для начала чата с нейросетью.")

	case models.StateInChat:
		// Отправляем сообщение в чат
		response, err := h.service.SendMessage(ctx, message.From.ID, message.Text)
		if err != nil {
			h.sendMessage(message.Chat.ID, "Не удалось отправить сообщение. Попробуйте позже.")
			log.Printf("Error sending message: %v", err)
			return
		}

		h.sendMessage(message.Chat.ID, response)

	default:
		h.sendMessage(message.Chat.ID, "Неизвестное состояние. Используйте /start")
	}
}

// sendMessage отправляет сообщение пользователю
func (h *Handler) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}