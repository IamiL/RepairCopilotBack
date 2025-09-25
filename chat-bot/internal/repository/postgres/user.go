package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
)

func (r *Repository) CreateUser(ctx context.Context, id uuid.UUID, messagesPerDay int, messagesLeftForToday int, createdAt time.Time, updatedAt time.Time) error {
	query := `
		INSERT INTO users (id, messages_per_day, messages_left_for_today, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Exec(ctx, query, id, messagesPerDay, messagesLeftForToday, createdAt, updatedAt)
	return err
}

func (r *Repository) GetUserConfirmAndLimitsInfo(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `
		SELECT messages_left_for_today
		FROM users
		WHERE id = $1
	`

	var messagesLeft int
	err := r.db.QueryRow(ctx, query, userID).Scan(&messagesLeft)
	if err != nil {
		return 0, err
	}

	return messagesLeft, nil
}