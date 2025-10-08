package models

import (
	"time"

	"github.com/google/uuid"
)

// TelegramUser представляет пользователя Telegram в системе
type TelegramUser struct {
	ID        int64      `db:"id"`
	TgUserID  int64      `db:"tg_user_id"`
	UserID    *uuid.UUID `db:"user_id"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
}