package postgres

import (
	"context"
	"encoding/json"
	"time"

	chatmodel "repairCopilotBot/chat-bot/internal/domain/model/chat"

	"github.com/google/uuid"
)

func (r *Repository) CreateChat(ctx context.Context, chatId uuid.UUID, userID uuid.UUID, isFinished bool, isProcessing bool, enclosure int, createdAt time.Time, updatedAt time.Time) error {
	query := `
		INSERT INTO chats (id, user_id, is_finished, is_processing, enclosure, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(ctx, query, chatId, userID, isFinished, isProcessing, enclosure, createdAt, updatedAt)
	return err
}

func (r *Repository) Chats(ctx context.Context) ([]chatmodel.Chat, error) {
	query := `
		SELECT c.id, c.user_id, c.created_at, c.is_finished, c.conclusion, c.is_processing, c.enclosure,
		       COALESCE(COUNT(m.id), 0) as messages_count
		FROM chats c
		LEFT JOIN messages m ON c.id = m.chat_id
		GROUP BY c.id, c.user_id, c.created_at, c.is_finished, c.is_processing, c.enclosure
		ORDER BY c.created_at DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []chatmodel.Chat
	for rows.Next() {
		var chat chatmodel.Chat
		var conclusion *string
		err := rows.Scan(
			&chat.Id,
			&chat.UserID,
			&chat.CreatedAt,
			&chat.IsFinished,
			&conclusion,
			&chat.IsProcessing,
			&chat.Enclosure,
			&chat.MessagesCount,
		)
		if err != nil {
			return nil, err
		}
		if conclusion != nil {
			chat.Conclusion = *conclusion
		}
		chats = append(chats, chat)
	}

	return chats, rows.Err()
}

func (r *Repository) ChatsForUser(ctx context.Context, userID uuid.UUID) ([]chatmodel.Chat, error) {
	query := `
		SELECT c.id, c.user_id, c.created_at, c.is_finished, c.conclusion, c.is_processing, c.enclosure,
		       COALESCE(COUNT(m.id), 0) as messages_count
		FROM chats c
		LEFT JOIN messages m ON c.id = m.chat_id
		WHERE c.user_id = $1
		GROUP BY c.id, c.user_id, c.created_at, c.is_finished, c.is_processing, c.enclosure
		ORDER BY c.created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []chatmodel.Chat
	for rows.Next() {
		var chat chatmodel.Chat
		var conclusion *string
		err := rows.Scan(
			&chat.Id,
			&chat.UserID,
			&chat.CreatedAt,
			&chat.IsFinished,
			&conclusion,
			&chat.IsProcessing,
			&chat.Enclosure,
			&chat.MessagesCount,
		)
		if err != nil {
			return nil, err
		}
		if conclusion != nil {
			chat.Conclusion = *conclusion
		}
		chats = append(chats, chat)
	}

	return chats, rows.Err()
}

func (r *Repository) ChatShortInfo(ctx context.Context, chatID uuid.UUID) (uuid.UUID, bool, bool, error) {
	query := `
		SELECT user_id, is_finished, is_processing
		FROM chats
		WHERE id = $1
	`

	var userID uuid.UUID
	var isFinished, isProcessing bool

	err := r.db.QueryRow(ctx, query, chatID).Scan(&userID, &isFinished, &isProcessing)
	if err != nil {
		return uuid.Nil, false, false, err
	}

	return userID, isFinished, isProcessing, nil
}

func (r *Repository) FinishChat(ctx context.Context, chatID uuid.UUID, conclusion string) error {
	query := `
		UPDATE chats
		SET is_finished = true, conclusion = $1, updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, conclusion, chatID)
	return err
}

func (r *Repository) GetChatTree(ctx context.Context, chatID uuid.UUID) (json.RawMessage, error) {
	query := `
		SELECT tree
		FROM chats
		WHERE id = $1
	`

	var tree json.RawMessage
	err := r.db.QueryRow(ctx, query, chatID).Scan(&tree)
	if err != nil {
		return nil, err
	}

	return tree, nil
}

func (r *Repository) UpdateChatTree(ctx context.Context, chatID uuid.UUID, tree json.RawMessage) error {
	query := `
		UPDATE chats
		SET tree = $1, updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, tree, chatID)
	return err
}
