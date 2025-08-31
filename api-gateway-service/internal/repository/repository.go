package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ActionLog struct {
	ID         int       `db:"id" json:"id"`
	Action     string    `db:"action" json:"action"`
	UserID     uuid.UUID `db:"user_id" json:"user_id"`
	CreateAt   time.Time `db:"created_at" json:"create_at"`
	ActionType int       `db:"action_type" json:"action_type"`
}

type ActionLogRepository interface {
	CreateActionLog(ctx context.Context, action string, userID uuid.UUID, actionType int) error
	GetAllActionLogs(ctx context.Context) ([]ActionLog, error)
}
