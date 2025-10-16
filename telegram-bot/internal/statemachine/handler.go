package statemachine

import (
	"context"
	"fmt"
	"log"

	"repairCopilotBot/telegram-bot/internal/domain/models"

	"github.com/google/uuid"
)

// StateHandler определяет обработчик для каждого состояния
type StateHandler interface {
	HandleCommand(ctx context.Context, userState *models.UserState, command string, args []string) (string, error)
	HandleText(ctx context.Context, userState *models.UserState, text string) (string, error)
}

// Repository интерфейс для работы с состоянием пользователя
type Repository interface {
	UpdateUserState(ctx context.Context, tgUserID int64, state models.UserStateEnum) error
	UpdateLoginAttempt(ctx context.Context, tgUserID int64, login string) error
	ClearLoginAttempt(ctx context.Context, tgUserID int64) error
	UpdateCurrentChatID(ctx context.Context, tgUserID int64, chatID *uuid.UUID) error
}

// Manager управляет переходами состояний и вызывает соответствующие обработчики
type Manager struct {
	sm       *StateMachine
	repo     Repository
	handlers map[models.UserStateEnum]StateHandler
}

// NewManager создает новый менеджер состояний
func NewManager(repo Repository) *Manager {
	return &Manager{
		sm:       NewStateMachine(),
		repo:     repo,
		handlers: make(map[models.UserStateEnum]StateHandler),
	}
}

// RegisterHandler регистрирует обработчик для конкретного состояния
func (m *Manager) RegisterHandler(state models.UserStateEnum, handler StateHandler) {
	m.handlers[state] = handler
}

// TransitionTo выполняет переход в новое состояние с проверкой и сохранением в БД
func (m *Manager) TransitionTo(ctx context.Context, tgUserID int64, currentState, newState models.UserStateEnum) error {
	// Проверяем, разрешен ли переход
	if !m.sm.CanTransition(currentState, newState) {
		return fmt.Errorf("transition from %s to %s is not allowed", currentState, newState)
	}

	// Сохраняем новое состояние в БД
	if err := m.repo.UpdateUserState(ctx, tgUserID, newState); err != nil {
		return fmt.Errorf("failed to update state in DB: %w", err)
	}

	log.Printf("User %d: state transition %s -> %s", tgUserID, currentState, newState)
	return nil
}

// HandleEvent обрабатывает событие и выполняет переход состояния
func (m *Manager) HandleEvent(ctx context.Context, tgUserID int64, currentState models.UserStateEnum, event Event) (models.UserStateEnum, error) {
	// Определяем новое состояние на основе события
	newState, err := m.sm.HandleEvent(currentState, event)
	if err != nil {
		return currentState, err
	}

	// Если состояние изменилось, сохраняем в БД
	if newState != currentState {
		if err := m.TransitionTo(ctx, tgUserID, currentState, newState); err != nil {
			return currentState, err
		}
	}

	return newState, nil
}

// OnStateEnter вызывается при входе в новое состояние (для cleanup/setup)
func (m *Manager) OnStateEnter(ctx context.Context, tgUserID int64, state models.UserStateEnum) error {
	switch state {
	case models.StateUnauthorized:
		// Очищаем login_attempt при возврате в неавторизованное состояние
		return m.repo.ClearLoginAttempt(ctx, tgUserID)

	case models.StateAwaitingLogin:
		// Очищаем старый login_attempt при начале новой попытки
		return m.repo.ClearLoginAttempt(ctx, tgUserID)

	case models.StateInChat:
		// При входе в состояние InChat, current_chat_id должен быть очищен
		// (чтобы при первом сообщении создался новый чат)
		// Это делается в service.StartChat()
		return nil

	case models.StateAuthorized:
		// current_chat_id уже очищен в service.FinishChat()
		return nil

	default:
		return nil
	}
}

// OnStateExit вызывается при выходе из состояния (для cleanup)
func (m *Manager) OnStateExit(ctx context.Context, tgUserID int64, state models.UserStateEnum) error {
	switch state {
	case models.StateInChat:
		// При выходе из чата current_chat_id очищается в service.FinishChat()
		return nil

	case models.StateAwaitingPassword:
		// Очищаем login_attempt при выходе (кроме перехода в Authorized)
		// Это делается условно в HandleEvent
		return nil

	default:
		return nil
	}
}

// IsCommandAllowed проверяет, разрешена ли команда в текущем состоянии
func (m *Manager) IsCommandAllowed(currentState models.UserStateEnum, command string) (bool, string) {
	switch command {
	case "start":
		// /start всегда разрешен
		return true, ""

	case "login":
		if currentState == models.StateInChat {
			return false, "Нельзя авторизоваться во время активного чата. Используйте /endchat для завершения чата."
		}
		return true, ""

	case "startchat":
		if currentState != models.StateAuthorized {
			if currentState == models.StateInChat {
				return false, "У вас уже есть активный чат."
			}
			return false, "Сначала необходимо авторизоваться. Используйте /login"
		}
		return true, ""

	case "endchat":
		if currentState != models.StateInChat {
			return false, "У вас нет активного чата."
		}
		return true, ""

	case "cancel":
		// /cancel разрешен в состояниях ожидания
		if currentState == models.StateAwaitingLogin || currentState == models.StateAwaitingPassword {
			return true, ""
		}
		return false, "Нечего отменять."

	default:
		return true, ""
	}
}
