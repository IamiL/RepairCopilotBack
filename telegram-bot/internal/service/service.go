package service

import (
	"context"
	"fmt"

	chatclient "repairCopilotBot/chat-bot/pkg/client/chat"
	"repairCopilotBot/telegram-bot/internal/domain/models"
	"repairCopilotBot/telegram-bot/internal/repository"
	userserviceclient "repairCopilotBot/user-service/client"

	"github.com/google/uuid"
)

type Service struct {
	repo        repository.Repository
	userClient  *userserviceclient.UserClient
	chatClient  *chatclient.Client
}

func NewService(
	repo repository.Repository,
	userClient *userserviceclient.UserClient,
	chatClient *chatclient.Client,
) *Service {
	return &Service{
		repo:       repo,
		userClient: userClient,
		chatClient: chatClient,
	}
}

// GetOrCreateUser получает или создает пользователя Telegram и его состояние
func (s *Service) GetOrCreateUser(ctx context.Context, tgUserID int64) (*models.TelegramUser, *models.UserState, error) {
	// Получаем или создаем пользователя
	user, err := s.repo.GetTelegramUser(ctx, tgUserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get telegram user: %w", err)
	}

	if user == nil {
		user, err = s.repo.CreateTelegramUser(ctx, tgUserID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create telegram user: %w", err)
		}
	}

	// Получаем или создаем состояние пользователя
	state, err := s.repo.GetUserState(ctx, tgUserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user state: %w", err)
	}

	if state == nil {
		state, err = s.repo.CreateUserState(ctx, tgUserID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create user state: %w", err)
		}
	}

	return user, state, nil
}

// UpdateState обновляет состояние пользователя
func (s *Service) UpdateState(ctx context.Context, tgUserID int64, state models.UserStateEnum) error {
	return s.repo.UpdateUserState(ctx, tgUserID, state)
}

// SaveLoginAttempt сохраняет введенный логин
func (s *Service) SaveLoginAttempt(ctx context.Context, tgUserID int64, login string) error {
	return s.repo.UpdateLoginAttempt(ctx, tgUserID, login)
}

// AuthenticateUser выполняет аутентификацию пользователя через user-service
func (s *Service) AuthenticateUser(ctx context.Context, tgUserID int64, login, password string) error {
	// Авторизация через user-service
	loginResp, err := s.userClient.Login(ctx, login, password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Сохраняем user_id в БД
	userID, err := uuid.Parse(loginResp.UserId)
	if err != nil {
		return fmt.Errorf("invalid user_id from user-service: %w", err)
	}

	if err := s.repo.UpdateUserID(ctx, tgUserID, userID); err != nil {
		return fmt.Errorf("failed to update user_id: %w", err)
	}

	// Обновляем состояние на "авторизован"
	if err := s.repo.UpdateUserState(ctx, tgUserID, models.StateAuthorized); err != nil {
		return fmt.Errorf("failed to update state to authorized: %w", err)
	}

	return nil
}

// StartChat начинает новый чат или возвращает существующий
func (s *Service) StartChat(ctx context.Context, tgUserID int64) (*uuid.UUID, error) {
	// Получаем пользователя
	user, err := s.repo.GetTelegramUser(ctx, tgUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram user: %w", err)
	}

	if user.UserID == nil {
		return nil, fmt.Errorf("user is not authenticated")
	}

	// Создаем новый чат через chat-service (отправляем пустое сообщение для инициализации)
	chatIDStr, _, err := s.chatClient.CreateNewMessage(ctx, nil, user.UserID.String(), "")
	if err != nil {
		return nil, fmt.Errorf("failed to create chat: %w", err)
	}

	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid chat_id from chat-service: %w", err)
	}

	// Сохраняем chat_id в состоянии пользователя
	if err := s.repo.UpdateCurrentChatID(ctx, tgUserID, &chatID); err != nil {
		return nil, fmt.Errorf("failed to update current chat_id: %w", err)
	}

	// Обновляем состояние на "в чате"
	if err := s.repo.UpdateUserState(ctx, tgUserID, models.StateInChat); err != nil {
		return nil, fmt.Errorf("failed to update state to in_chat: %w", err)
	}

	return &chatID, nil
}

// SendMessage отправляет сообщение в текущий чат
func (s *Service) SendMessage(ctx context.Context, tgUserID int64, message string) (string, error) {
	// Получаем состояние пользователя
	state, err := s.repo.GetUserState(ctx, tgUserID)
	if err != nil {
		return "", fmt.Errorf("failed to get user state: %w", err)
	}

	if state.CurrentChatID == nil {
		return "", fmt.Errorf("no active chat")
	}

	// Получаем пользователя для user_id
	user, err := s.repo.GetTelegramUser(ctx, tgUserID)
	if err != nil {
		return "", fmt.Errorf("failed to get telegram user: %w", err)
	}

	if user.UserID == nil {
		return "", fmt.Errorf("user is not authenticated")
	}

	// Отправляем сообщение через chat-service
	chatIDStr := state.CurrentChatID.String()
	_, responseMessage, err := s.chatClient.CreateNewMessage(ctx, &chatIDStr, user.UserID.String(), message)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	return responseMessage, nil
}

// FinishChat завершает текущий чат
func (s *Service) FinishChat(ctx context.Context, tgUserID int64) error {
	// Получаем состояние пользователя
	state, err := s.repo.GetUserState(ctx, tgUserID)
	if err != nil {
		return fmt.Errorf("failed to get user state: %w", err)
	}

	if state.CurrentChatID == nil {
		return fmt.Errorf("no active chat")
	}

	// Получаем пользователя для user_id
	user, err := s.repo.GetTelegramUser(ctx, tgUserID)
	if err != nil {
		return fmt.Errorf("failed to get telegram user: %w", err)
	}

	if user.UserID == nil {
		return fmt.Errorf("user is not authenticated")
	}

	// Завершаем чат через chat-service
	_, err = s.chatClient.FinishChat(ctx, state.CurrentChatID.String(), user.UserID.String())
	if err != nil {
		return fmt.Errorf("failed to finish chat: %w", err)
	}

	// Очищаем current_chat_id
	if err := s.repo.UpdateCurrentChatID(ctx, tgUserID, nil); err != nil {
		return fmt.Errorf("failed to clear current chat_id: %w", err)
	}

	// Возвращаем состояние на "авторизован"
	if err := s.repo.UpdateUserState(ctx, tgUserID, models.StateAuthorized); err != nil {
		return fmt.Errorf("failed to update state to authorized: %w", err)
	}

	return nil
}

// GetUserState возвращает текущее состояние пользователя
func (s *Service) GetUserState(ctx context.Context, tgUserID int64) (*models.UserState, error) {
	return s.repo.GetUserState(ctx, tgUserID)
}