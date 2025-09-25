package postgres

import (
	"context"
	"time"

	messagemodel "repairCopilotBot/chat-bot/internal/domain/model/message"

	"github.com/google/uuid"
)

func (r *Repository) CreateMessage(ctx context.Context, chat_id uuid.UUID, role string, content string, nestingLevel int, createdAt time.Time, updatedAt time.Time) error {
	query := `
		INSERT INTO messages (id, chat_id, role, content, nesting_level, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	messageId := uuid.New()
	_, err := r.db.Exec(ctx, query, messageId, chat_id, role, content, nestingLevel, createdAt, updatedAt)
	return err
}

func (r *Repository) Messages(ctx context.Context, chatID uuid.UUID) ([]messagemodel.Message, error) {
	query := `
		SELECT id, chat_id, role, content, created_at
		FROM messages
		WHERE chat_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []messagemodel.Message
	for rows.Next() {
		var msg messagemodel.Message
		err := rows.Scan(
			&msg.Id,
			&msg.ChatId,
			&msg.Role,
			&msg.Content,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}