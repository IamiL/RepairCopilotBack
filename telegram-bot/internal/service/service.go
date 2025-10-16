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
	repo       repository.Repository
	userClient *userserviceclient.UserClient
	chatClient *chatclient.Client
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

// StartChat начинает новый чат локально (без создания чата в search-service)
func (s *Service) StartChat(ctx context.Context, tgUserID int64) error {
	// Получаем пользователя для проверки авторизации
	user, err := s.repo.GetTelegramUser(ctx, tgUserID)
	if err != nil {
		return fmt.Errorf("failed to get telegram user: %w", err)
	}

	if user.UserID == nil {
		return fmt.Errorf("user is not authenticated")
	}

	// Очищаем current_chat_id (чтобы при первом сообщении создался новый чат)
	if err := s.repo.UpdateCurrentChatID(ctx, tgUserID, nil); err != nil {
		return fmt.Errorf("failed to clear current chat_id: %w", err)
	}

	// Обновляем состояние на "в чате"
	if err := s.repo.UpdateUserState(ctx, tgUserID, models.StateInChat); err != nil {
		return fmt.Errorf("failed to update state to in_chat: %w", err)
	}

	return nil
}

// SendMessage отправляет сообщение в текущий чат или создает новый чат
func (s *Service) SendMessage(ctx context.Context, tgUserID int64, message string) (string, error) {
	// Получаем состояние пользователя
	state, err := s.repo.GetUserState(ctx, tgUserID)
	if err != nil {
		return "", fmt.Errorf("failed to get user state: %w", err)
	}

	// Получаем пользователя для user_id
	user, err := s.repo.GetTelegramUser(ctx, tgUserID)
	if err != nil {
		return "", fmt.Errorf("failed to get telegram user: %w", err)
	}

	if user.UserID == nil {
		return "", fmt.Errorf("user is not authenticated")
	}

	// Определяем chatIDStr для отправки
	var chatIDStr *string
	if state.CurrentChatID != nil {
		// Используем существующий чат
		chatIDValue := state.CurrentChatID.String()
		chatIDStr = &chatIDValue
	}
	// Если CurrentChatID == nil, то chatIDStr = nil - создастся новый чат

	// Отправляем сообщение через search-service
	responseChatIDStr, responseMessage, err := s.chatClient.CreateNewMessage(ctx, chatIDStr, user.UserID.String(), message)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	// Если чат был создан (CurrentChatID был nil), сохраняем новый chat_id
	if state.CurrentChatID == nil && responseChatIDStr != "" {
		chatID, err := uuid.Parse(responseChatIDStr)
		if err != nil {
			return "", fmt.Errorf("invalid chat_id from search-service: %w", err)
		}

		if err := s.repo.UpdateCurrentChatID(ctx, tgUserID, &chatID); err != nil {
			return "", fmt.Errorf("failed to update current chat_id: %w", err)
		}
	}

	return responseMessage, nil
}

// FinishChat завершает текущий чат и возвращает суммаризацию
func (s *Service) FinishChat(ctx context.Context, tgUserID int64) (string, error) {
	// Получаем состояние пользователя
	state, err := s.repo.GetUserState(ctx, tgUserID)
	if err != nil {
		return "", fmt.Errorf("failed to get user state: %w", err)
	}

	if state.CurrentChatID == nil {
		return "", fmt.Errorf("no active search")
	}

	// Получаем пользователя для user_id
	user, err := s.repo.GetTelegramUser(ctx, tgUserID)
	if err != nil {
		return "", fmt.Errorf("failed to get telegram user: %w", err)
	}

	if user.UserID == nil {
		return "", fmt.Errorf("user is not authenticated")
	}

	// Завершаем чат через search-service и получаем суммаризацию
	summary, err := s.chatClient.FinishChat(ctx, state.CurrentChatID.String(), user.UserID.String())
	if err != nil {
		return "", fmt.Errorf("failed to finish search: %w", err)
	}

	// Очищаем current_chat_id
	if err := s.repo.UpdateCurrentChatID(ctx, tgUserID, nil); err != nil {
		return "", fmt.Errorf("failed to clear current chat_id: %w", err)
	}

	// Возвращаем состояние на "авторизован"
	if err := s.repo.UpdateUserState(ctx, tgUserID, models.StateAuthorized); err != nil {
		return "", fmt.Errorf("failed to update state to authorized: %w", err)
	}

	return summary, nil
}

// GetUserState возвращает текущее состояние пользователя
func (s *Service) GetUserState(ctx context.Context, tgUserID int64) (*models.UserState, error) {
	return s.repo.GetUserState(ctx, tgUserID)
}
