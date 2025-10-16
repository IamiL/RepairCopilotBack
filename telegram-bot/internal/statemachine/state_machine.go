package statemachine

import (
	"context"
	"fmt"
	"sync"

	"repairCopilotBot/telegram-bot/internal/domain/models"
)

// Event представляет событие, которое может вызвать переход состояния
type Event string

const (
	EventStartCommand     Event = "start_command"
	EventLoginCommand     Event = "login_command"
	EventStartChatCommand Event = "startchat_command"
	EventEndChatCommand   Event = "endchat_command"
	EventCancelCommand    Event = "cancel_command"
	EventTextMessage      Event = "text_message"
	EventLoginSuccess     Event = "login_success"
	EventLoginFailure     Event = "login_failure"
	EventChatStarted      Event = "chat_started"
	EventChatEnded        Event = "chat_ended"
)

// Transition описывает переход из одного состояния в другое
type Transition struct {
	From models.UserStateEnum
	To   models.UserStateEnum
}

// StateMachine управляет переходами между состояниями для каждого пользователя
type StateMachine struct {
	transitions map[Transition]bool
	mu          sync.RWMutex // защита от race conditions
}

// NewStateMachine создает новую state machine с определенными переходами
func NewStateMachine() *StateMachine {
	sm := &StateMachine{
		transitions: make(map[Transition]bool),
	}

	// Определяем все разрешенные переходы
	allowedTransitions := []Transition{
		// Из Unauthorized
		{models.StateUnauthorized, models.StateAwaitingLogin},

		// Из AwaitingLogin
		{models.StateAwaitingLogin, models.StateAwaitingPassword},
		{models.StateAwaitingLogin, models.StateUnauthorized}, // cancel/reset

		// Из AwaitingPassword
		{models.StateAwaitingPassword, models.StateAuthorized},   // успешная авторизация
		{models.StateAwaitingPassword, models.StateUnauthorized}, // ошибка/cancel

		// Из Authorized
		{models.StateAuthorized, models.StateInChat},        // начать чат
		{models.StateAuthorized, models.StateAwaitingLogin}, // повторный логин (перелогин)

		// Из InChat
		{models.StateInChat, models.StateAuthorized}, // завершить чат
		{models.StateInChat, models.StateInChat},     // отправка сообщений (остаемся в том же состоянии)
	}

	for _, t := range allowedTransitions {
		sm.transitions[t] = true
	}

	return sm
}

// CanTransition проверяет, возможен ли переход из текущего состояния в новое
func (sm *StateMachine) CanTransition(from, to models.UserStateEnum) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.transitions[Transition{from, to}]
}

// Transition выполняет переход состояния с проверкой
func (sm *StateMachine) Transition(ctx context.Context, from, to models.UserStateEnum) error {
	if !sm.CanTransition(from, to) {
		return fmt.Errorf("invalid transition from %s to %s", from, to)
	}

	return nil
}

// GetAllowedTransitions возвращает список разрешенных переходов из текущего состояния
func (sm *StateMachine) GetAllowedTransitions(from models.UserStateEnum) []models.UserStateEnum {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var allowed []models.UserStateEnum
	for t := range sm.transitions {
		if t.From == from {
			allowed = append(allowed, t.To)
		}
	}

	return allowed
}

// HandleEvent определяет, в какое состояние нужно перейти на основе события и текущего состояния
func (sm *StateMachine) HandleEvent(currentState models.UserStateEnum, event Event) (models.UserStateEnum, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	switch currentState {
	case models.StateUnauthorized:
		switch event {
		case EventLoginCommand:
			return models.StateAwaitingLogin, nil
		case EventStartCommand, EventTextMessage:
			return models.StateUnauthorized, nil // остаемся в том же состоянии
		default:
			return currentState, fmt.Errorf("unexpected event %s in state %s", event, currentState)
		}

	case models.StateAwaitingLogin:
		switch event {
		case EventTextMessage:
			return models.StateAwaitingPassword, nil
		case EventCancelCommand, EventStartCommand:
			return models.StateUnauthorized, nil
		case EventLoginCommand:
			// Повторный /login - начинаем заново
			return models.StateAwaitingLogin, nil
		default:
			return currentState, fmt.Errorf("unexpected event %s in state %s", event, currentState)
		}

	case models.StateAwaitingPassword:
		switch event {
		case EventLoginSuccess:
			return models.StateAuthorized, nil
		case EventLoginFailure, EventCancelCommand, EventStartCommand:
			return models.StateUnauthorized, nil
		case EventLoginCommand:
			// Повторный /login во время ввода пароля - начинаем заново
			return models.StateAwaitingLogin, nil
		default:
			return currentState, fmt.Errorf("unexpected event %s in state %s", event, currentState)
		}

	case models.StateAuthorized:
		switch event {
		case EventStartChatCommand:
			return models.StateInChat, nil
		case EventLoginCommand:
			// Перелогин
			return models.StateAwaitingLogin, nil
		case EventStartCommand, EventTextMessage:
			return models.StateAuthorized, nil // остаемся в том же состоянии
		default:
			return currentState, fmt.Errorf("unexpected event %s in state %s", event, currentState)
		}

	case models.StateInChat:
		switch event {
		case EventEndChatCommand, EventChatEnded:
			return models.StateAuthorized, nil
		case EventTextMessage:
			return models.StateInChat, nil // продолжаем оставаться в чате
		case EventStartCommand:
			return models.StateInChat, nil // игнорируем /start в чате
		case EventLoginCommand:
			// Нельзя логиниться в чате - нужно сначала завершить чат
			return currentState, fmt.Errorf("cannot login while in search, use /endchat first")
		default:
			return currentState, fmt.Errorf("unexpected event %s in state %s", event, currentState)
		}

	default:
		return currentState, fmt.Errorf("unknown state: %s", currentState)
	}
}
