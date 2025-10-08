package models

import (
	"time"

	"github.com/google/uuid"
)

// UserStateEnum представляет возможные состояния пользователя
type UserStateEnum string

const (
	StateUnauthorized     UserStateEnum = "unauthorized"
	StateAwaitingLogin    UserStateEnum = "awaiting_login"
	StateAwaitingPassword UserStateEnum = "awaiting_password"
	StateAuthorized       UserStateEnum = "authorized"
	StateInChat           UserStateEnum = "in_chat"
)

// UserState представляет состояние пользователя в Telegram боте
type UserState struct {
	ID             int64         `db:"id"`
	TgUserID       int64         `db:"tg_user_id"`
	State          UserStateEnum `db:"state"`
	LoginAttempt   *string       `db:"login_attempt"`
	CurrentChatID  *uuid.UUID    `db:"current_chat_id"`
	CreatedAt      time.Time     `db:"created_at"`
	UpdatedAt      time.Time     `db:"updated_at"`
}