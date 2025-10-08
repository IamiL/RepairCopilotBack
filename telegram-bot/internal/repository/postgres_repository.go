package repository

import (
	"context"
	"database/sql"
	"fmt"

	"repairCopilotBot/telegram-bot/internal/domain/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PostgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

// GetTelegramUser получает пользователя Telegram по tg_user_id
func (r *PostgresRepository) GetTelegramUser(ctx context.Context, tgUserID int64) (*models.TelegramUser, error) {
	var user models.TelegramUser
	query := `SELECT id, tg_user_id, user_id, created_at, updated_at FROM telegram_users WHERE tg_user_id = $1`

	err := r.db.GetContext(ctx, &user, query, tgUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get telegram user: %w", err)
	}

	return &user, nil
}

// CreateTelegramUser создает нового пользователя Telegram
func (r *PostgresRepository) CreateTelegramUser(ctx context.Context, tgUserID int64) (*models.TelegramUser, error) {
	var user models.TelegramUser
	query := `
		INSERT INTO telegram_users (tg_user_id, created_at, updated_at)
		VALUES ($1, NOW(), NOW())
		RETURNING id, tg_user_id, user_id, created_at, updated_at
	`

	err := r.db.GetContext(ctx, &user, query, tgUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram user: %w", err)
	}

	return &user, nil
}

// UpdateUserID обновляет user_id для пользователя Telegram
func (r *PostgresRepository) UpdateUserID(ctx context.Context, tgUserID int64, userID uuid.UUID) error {
	query := `UPDATE telegram_users SET user_id = $1, updated_at = NOW() WHERE tg_user_id = $2`

	_, err := r.db.ExecContext(ctx, query, userID, tgUserID)
	if err != nil {
		return fmt.Errorf("failed to update user_id: %w", err)
	}

	return nil
}

// GetUserState получает состояние пользователя по tg_user_id
func (r *PostgresRepository) GetUserState(ctx context.Context, tgUserID int64) (*models.UserState, error) {
	var state models.UserState
	query := `SELECT id, tg_user_id, state, login_attempt, current_chat_id, created_at, updated_at FROM user_states WHERE tg_user_id = $1`

	err := r.db.GetContext(ctx, &state, query, tgUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user state: %w", err)
	}

	return &state, nil
}

// CreateUserState создает новое состояние для пользователя
func (r *PostgresRepository) CreateUserState(ctx context.Context, tgUserID int64) (*models.UserState, error) {
	var state models.UserState
	query := `
		INSERT INTO user_states (tg_user_id, state, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		RETURNING id, tg_user_id, state, login_attempt, current_chat_id, created_at, updated_at
	`

	err := r.db.GetContext(ctx, &state, query, tgUserID, models.StateUnauthorized)
	if err != nil {
		return nil, fmt.Errorf("failed to create user state: %w", err)
	}

	return &state, nil
}

// UpdateUserState обновляет состояние пользователя
func (r *PostgresRepository) UpdateUserState(ctx context.Context, tgUserID int64, state models.UserStateEnum) error {
	query := `UPDATE user_states SET state = $1, updated_at = NOW() WHERE tg_user_id = $2`

	_, err := r.db.ExecContext(ctx, query, state, tgUserID)
	if err != nil {
		return fmt.Errorf("failed to update user state: %w", err)
	}

	return nil
}

// UpdateLoginAttempt обновляет попытку входа (сохраняет введенный логин)
func (r *PostgresRepository) UpdateLoginAttempt(ctx context.Context, tgUserID int64, login string) error {
	query := `UPDATE user_states SET login_attempt = $1, updated_at = NOW() WHERE tg_user_id = $2`

	_, err := r.db.ExecContext(ctx, query, login, tgUserID)
	if err != nil {
		return fmt.Errorf("failed to update login attempt: %w", err)
	}

	return nil
}

// UpdateCurrentChatID обновляет текущий chat_id пользователя
func (r *PostgresRepository) UpdateCurrentChatID(ctx context.Context, tgUserID int64, chatID *uuid.UUID) error {
	query := `UPDATE user_states SET current_chat_id = $1, updated_at = NOW() WHERE tg_user_id = $2`

	_, err := r.db.ExecContext(ctx, query, chatID, tgUserID)
	if err != nil {
		return fmt.Errorf("failed to update current chat_id: %w", err)
	}

	return nil
}