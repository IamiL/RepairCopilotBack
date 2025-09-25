package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
)

func (r *Repository) CreateAction(ctx context.Context, actionId uuid.UUID, actionType int, userID uuid.UUID, message string, createdAt time.Time) error {
	query := `
		INSERT INTO actions (id, type, user_id, message, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Exec(ctx, query, actionId, actionType, userID, message, createdAt)
	return err
}