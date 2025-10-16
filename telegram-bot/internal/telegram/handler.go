package telegram

import (
	"context"
	"fmt"
	"log"

	"repairCopilotBot/telegram-bot/internal/domain/models"
	"repairCopilotBot/telegram-bot/internal/service"
	"repairCopilotBot/telegram-bot/internal/statemachine"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	bot     *tgbotapi.BotAPI
	service *service.Service
	sm      *statemachine.Manager
	km      *KeyboardManager
}

func NewHandler(botToken string, svc *service.Service, repo statemachine.Repository) (*Handler, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	return &Handler{
		bot:     bot,
		service: svc,
		sm:      statemachine.NewManager(repo),
		km:      NewKeyboardManager(),
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
		h.sendMessageWithoutKeyboard(message.Chat.ID, "Произошла ошибка. Попробуйте позже.")
		log.Printf("Error getting or creating user: %v", err)
		return
	}

	// Обрабатываем команды
	if message.IsCommand() {
		h.handleCommand(ctx, message, state)
		return
	}

	// Проверяем, является ли текст командой от кнопки
	if command := ParseButtonCommand(message.Text); command != "" {
		// Создаем временное сообщение с командой
		fakeMessage := *message
		fakeMessage.Text = "/" + command
		fakeMessage.Entities = []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: len(command) + 1}}
		h.handleCommand(ctx, &fakeMessage, state)
		return
	}

	// Обрабатываем текстовые сообщения в зависимости от состояния
	h.handleText(ctx, message, state)
}

// handleCommand обрабатывает команды бота
func (h *Handler) handleCommand(ctx context.Context, message *tgbotapi.Message, state *models.UserState) {
	command := message.Command()

	// Проверяем, разрешена ли команда в текущем состоянии
	allowed, reason := h.sm.IsCommandAllowed(state.State, command)
	if !allowed {
		h.sendMessage(message.Chat.ID, reason, state.State)
		return
	}

	switch command {
	case "start":
		h.handleStart(ctx, message, state)
	case "login":
		h.handleLoginCommand(ctx, message, state)
	case "startchat":
		h.handleStartChat(ctx, message, state)
	case "endchat":
		h.handleEndChat(ctx, message, state)
	case "cancel":
		h.handleCancel(ctx, message, state)
	default:
		h.sendMessage(message.Chat.ID, "Неизвестная команда. Используйте /start для начала.", state.State)
	}
}

// handleStart обрабатывает команду /start
func (h *Handler) handleStart(ctx context.Context, message *tgbotapi.Message, state *models.UserState) {
	// Обрабатываем событие StartCommand
	newState, err := h.sm.HandleEvent(ctx, message.From.ID, state.State, statemachine.EventStartCommand)
	if err != nil {
		log.Printf("Error handling start command event: %v", err)
	}

	// Если была незавершенная авторизация, сбрасываем её
	if state.State == models.StateAwaitingLogin || state.State == models.StateAwaitingPassword {
		if err := h.sm.OnStateEnter(ctx, message.From.ID, newState); err != nil {
			log.Printf("Error on state enter: %v", err)
		}
	}

	welcomeText := "Добро пожаловать!\n\n"

	if newState == models.StateUnauthorized || newState == models.StateAwaitingLogin || newState == models.StateAwaitingPassword {
		welcomeText += "Для начала работы вам нужно авторизоваться.\n"
		welcomeText += "Используйте команду /login для входа в систему."
	} else if newState == models.StateInChat {
		welcomeText += "У вас активный чат.\n"
		welcomeText += "Для завершения используйте /endchat"
	} else {
		welcomeText += "Вы уже авторизованы!\n"
		welcomeText += "Используйте /startchat для начала чата с нейросетью."
	}

	h.sendMessage(message.Chat.ID, welcomeText, newState)
}

// handleLoginCommand начинает процесс авторизации
func (h *Handler) handleLoginCommand(ctx context.Context, message *tgbotapi.Message, state *models.UserState) {
	// Обрабатываем событие LoginCommand
	newState, err := h.sm.HandleEvent(ctx, message.From.ID, state.State, statemachine.EventLoginCommand)
	if err != nil {
		h.sendMessage(message.Chat.ID, fmt.Sprintf("Ошибка: %v", err), state.State)
		log.Printf("Error handling login command event: %v", err)
		return
	}

	// Вызываем OnStateEnter для очистки старого login_attempt
	if err := h.sm.OnStateEnter(ctx, message.From.ID, newState); err != nil {
		log.Printf("Error on state enter: %v", err)
	}

	h.sendMessage(message.Chat.ID, "Введите ваш логин:", newState)
}

// handleStartChat начинает новый чат с нейросетью
func (h *Handler) handleStartChat(ctx context.Context, message *tgbotapi.Message, state *models.UserState) {
	// Обрабатываем событие StartChatCommand
	newState, err := h.sm.HandleEvent(ctx, message.From.ID, state.State, statemachine.EventStartChatCommand)
	if err != nil {
		h.sendMessage(message.Chat.ID, fmt.Sprintf("Ошибка: %v", err), state.State)
		log.Printf("Error handling startchat command event: %v", err)
		return
	}

	// Вызываем service.StartChat (очищает current_chat_id)
	if err := h.service.StartChat(ctx, message.From.ID); err != nil {
		h.sendMessage(message.Chat.ID, "Не удалось начать чат. Попробуйте позже.", state.State)
		log.Printf("Error starting search: %v", err)
		// Откатываем состояние назад
		_ = h.sm.TransitionTo(ctx, message.From.ID, models.StateInChat, models.StateAuthorized)
		return
	}

	h.sendMessage(message.Chat.ID, "Чат начат!\nТеперь вы можете отправлять сообщения.\nДля завершения чата используйте /endchat", newState)
}

// handleEndChat завершает текущий чат
func (h *Handler) handleEndChat(ctx context.Context, message *tgbotapi.Message, state *models.UserState) {
	// Показываем "typing..." индикатор
	typingAction := tgbotapi.NewChatAction(message.Chat.ID, tgbotapi.ChatTyping)
	if _, err := h.bot.Request(typingAction); err != nil {
		log.Printf("Error sending typing action: %v", err)
	}

	// Завершаем чат через service и получаем суммаризацию
	summary, err := h.service.FinishChat(ctx, message.From.ID)
	if err != nil {
		h.sendMessage(message.Chat.ID, "Не удалось завершить чат. Попробуйте позже.", state.State)
		log.Printf("Error finishing search: %v", err)
		return
	}

	// Отправляем суммаризацию
	msg := tgbotapi.NewMessage(message.Chat.ID, summary+"\n\n─────────────\nИспользуйте /startchat для начала нового чата.")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = h.km.GetKeyboard(models.StateAuthorized)
	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("Error sending summary message: %v", err)
		// Fallback без форматирования
		h.sendMessageWithoutKeyboard(message.Chat.ID, summary+"\n\nИспользуйте /startchat для начала нового чата.")
	}
}

// handleCancel отменяет текущую операцию
func (h *Handler) handleCancel(ctx context.Context, message *tgbotapi.Message, state *models.UserState) {
	// Обрабатываем событие CancelCommand
	newState, err := h.sm.HandleEvent(ctx, message.From.ID, state.State, statemachine.EventCancelCommand)
	if err != nil {
		h.sendMessage(message.Chat.ID, fmt.Sprintf("Ошибка: %v", err), state.State)
		log.Printf("Error handling cancel command event: %v", err)
		return
	}

	// Очищаем данные при входе в новое состояние
	if err := h.sm.OnStateEnter(ctx, message.From.ID, newState); err != nil {
		log.Printf("Error on state enter: %v", err)
	}

	h.sendMessage(message.Chat.ID, "Операция отменена. Используйте /start для начала.", newState)
}

// handleText обрабатывает текстовые сообщения в зависимости от состояния
func (h *Handler) handleText(ctx context.Context, message *tgbotapi.Message, state *models.UserState) {
	switch state.State {
	case models.StateUnauthorized:
		h.sendMessage(message.Chat.ID, "Пожалуйста, авторизуйтесь с помощью /login", state.State)

	case models.StateAwaitingLogin:
		// Сохраняем введенный логин
		if err := h.service.SaveLoginAttempt(ctx, message.From.ID, message.Text); err != nil {
			h.sendMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.", state.State)
			log.Printf("Error saving login attempt: %v", err)
			return
		}

		// Переводим в состояние ожидания пароля
		if err := h.service.UpdateState(ctx, message.From.ID, models.StateAwaitingPassword); err != nil {
			h.sendMessage(message.Chat.ID, "Произошла ошибка. Попробуйте позже.", state.State)
			log.Printf("Error updating state: %v", err)
			return
		}

		h.sendMessage(message.Chat.ID, "Введите ваш пароль:", models.StateAwaitingPassword)

	case models.StateAwaitingPassword:
		// Получаем сохраненный логин
		currentState, err := h.service.GetUserState(ctx, message.From.ID)
		if err != nil || currentState.LoginAttempt == nil {
			h.sendMessage(message.Chat.ID, "Произошла ошибка. Попробуйте начать авторизацию заново с /login", state.State)
			log.Printf("Error getting user state: %v", err)
			return
		}

		// Пытаемся авторизоваться
		if err := h.service.AuthenticateUser(ctx, message.From.ID, *currentState.LoginAttempt, message.Text); err != nil {
			h.sendMessage(message.Chat.ID, "Неверный логин или пароль. Попробуйте снова с /login", models.StateUnauthorized)
			log.Printf("Authentication failed: %v", err)

			// Возвращаем в состояние неавторизован
			_ = h.service.UpdateState(ctx, message.From.ID, models.StateUnauthorized)
			return
		}

		h.sendMessage(message.Chat.ID, "Вы успешно авторизованы! Используйте /startchat для начала чата с нейросетью.", models.StateAuthorized)

	case models.StateAuthorized:
		h.sendMessage(message.Chat.ID, "Используйте /startchat для начала чата с нейросетью.", state.State)

	case models.StateInChat:
		// Показываем "typing..." индикатор
		typingAction := tgbotapi.NewChatAction(message.Chat.ID, tgbotapi.ChatTyping)
		if _, err := h.bot.Request(typingAction); err != nil {
			log.Printf("Error sending typing action: %v", err)
		}

		// Отправляем "думающее" сообщение
		thinkingMsg := tgbotapi.NewMessage(message.Chat.ID, "🤔 Думаю...")
		sentThinkingMsg, err := h.bot.Send(thinkingMsg)
		if err != nil {
			log.Printf("Error sending thinking message: %v", err)
		}

		// Отправляем сообщение в search-service
		response, err := h.service.SendMessage(ctx, message.From.ID, message.Text)

		// Обновляем "думающее" сообщение на ответ или ошибку
		if err != nil {
			editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, sentThinkingMsg.MessageID, "❌ Не удалось отправить сообщение. Попробуйте позже.")
			editMsg.ParseMode = "HTML"
			if _, err := h.bot.Send(editMsg); err != nil {
				log.Printf("Error editing message: %v", err)
			}
			log.Printf("Error sending message: %v", err)
			return
		}

		editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, sentThinkingMsg.MessageID, response)
		editMsg.ParseMode = "HTML"
		if _, err := h.bot.Send(editMsg); err != nil {
			log.Printf("Error editing message: %v", err)
		}

	default:
		h.sendMessage(message.Chat.ID, "Неизвестное состояние. Используйте /start", state.State)
	}
}

// sendMessage отправляет сообщение пользователю с клавиатурой для текущего состояния
func (h *Handler) sendMessage(chatID int64, text string, state models.UserStateEnum) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = h.km.GetKeyboard(state)
	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

// sendMessageWithoutKeyboard отправляет сообщение без обновления клавиатуры
func (h *Handler) sendMessageWithoutKeyboard(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}
