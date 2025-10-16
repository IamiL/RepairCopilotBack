package telegram

import (
	"repairCopilotBot/telegram-bot/internal/domain/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// KeyboardManager управляет клавиатурами для разных состояний
type KeyboardManager struct {
	keyboards map[models.UserStateEnum]tgbotapi.ReplyKeyboardMarkup
}

// NewKeyboardManager создает менеджер клавиатур
func NewKeyboardManager() *KeyboardManager {
	km := &KeyboardManager{
		keyboards: make(map[models.UserStateEnum]tgbotapi.ReplyKeyboardMarkup),
	}

	// Клавиатура для неавторизованных пользователей
	km.keyboards[models.StateUnauthorized] = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔑 Войти"),
		),
	)

	// Клавиатура при вводе логина
	km.keyboards[models.StateAwaitingLogin] = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("❌ Отменить"),
		),
	)

	// Клавиатура при вводе пароля
	km.keyboards[models.StateAwaitingPassword] = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("❌ Отменить"),
		),
	)

	// Клавиатура для авторизованных пользователей
	km.keyboards[models.StateAuthorized] = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("💬 Начать чат"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔄 Сменить аккаунт"),
		),
	)

	// Клавиатура во время чата
	km.keyboards[models.StateInChat] = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🛑 Завершить чат"),
		),
	)

	// Настраиваем все клавиатуры
	for state := range km.keyboards {
		keyboard := km.keyboards[state]
		keyboard.ResizeKeyboard = true
		keyboard.OneTimeKeyboard = false
		km.keyboards[state] = keyboard
	}

	return km
}

// GetKeyboard возвращает клавиатуру для заданного состояния
func (km *KeyboardManager) GetKeyboard(state models.UserStateEnum) tgbotapi.ReplyKeyboardMarkup {
	if keyboard, exists := km.keyboards[state]; exists {
		return keyboard
	}
	// По умолчанию возвращаем клавиатуру для неавторизованных
	return km.keyboards[models.StateUnauthorized]
}

// RemoveKeyboard создает команду для удаления клавиатуры
func (km *KeyboardManager) RemoveKeyboard() tgbotapi.ReplyKeyboardRemove {
	return tgbotapi.NewRemoveKeyboard(true)
}

// ParseButtonCommand конвертирует текст кнопки в команду
func ParseButtonCommand(text string) string {
	buttonToCommand := map[string]string{
		"🔑 Войти":         "login",
		"💬 Начать чат":    "startchat",
		"🛑 Завершить чат": "endchat",
		"❌ Отменить":      "cancel",
		"🔄 Сменить аккаунт": "login",
	}

	if command, exists := buttonToCommand[text]; exists {
		return command
	}

	return ""
}