package repository

import (
	"context"
	"github.com/google/uuid"
	"time"
)

type ActionLog struct {
	ID       int       `db:"id"`
	Action   string    `db:"action"`
	UserID   uuid.UUID `db:"user_id"`
	CreateAt time.Time `db:"created_at"`
}

type ActionLogRepository interface {
	CreateActionLog(ctx context.Context, action string, userID uuid.UUID) error
	GetAllActionLogs(ctx context.Context) ([]ActionLog, error)
}