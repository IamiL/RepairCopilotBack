package repository

import (
	"context"

	"repairCopilotBot/telegram-bot/internal/domain/models"

	"github.com/google/uuid"
)

// Repository интерфейс для работы с базой данных
type Repository interface {
	// TelegramUser methods
	GetTelegramUser(ctx context.Context, tgUserID int64) (*models.TelegramUser, error)
	CreateTelegramUser(ctx context.Context, tgUserID int64) (*models.TelegramUser, error)
	UpdateUserID(ctx context.Context, tgUserID int64, userID uuid.UUID) error

	// UserState methods
	GetUserState(ctx context.Context, tgUserID int64) (*models.UserState, error)
	CreateUserState(ctx context.Context, tgUserID int64) (*models.UserState, error)
	UpdateUserState(ctx context.Context, tgUserID int64, state models.UserStateEnum) error
	UpdateLoginAttempt(ctx context.Context, tgUserID int64, login string) error
	UpdateCurrentChatID(ctx context.Context, tgUserID int64, chatID *uuid.UUID) error
}