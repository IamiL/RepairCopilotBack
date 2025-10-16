package chatservice

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
)

func (c *ChatService) FinishChat(ctx context.Context, userID uuid.UUID, chatID uuid.UUID) (string, error) {
	op := "search.FinishChat"

	log := c.log.With(
		slog.String("op", op),
		slog.String("userID", userID.String()),
		slog.String("chatID", chatID.String()),
	)

	log.Error("not implemented")

	return "", errors.New("not implemented")
}
