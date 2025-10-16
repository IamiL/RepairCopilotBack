package chatservice

import (
	"context"
	"errors"
	"log/slog"
	messagemodel "repairCopilotBot/search-bot/internal/domain/model/message"
	"repairCopilotBot/search-bot/internal/pkg/logger/sl"

	"github.com/google/uuid"
)

func (c *ChatService) Messages(ctx context.Context, chatID uuid.UUID) ([]messagemodel.Message, error) {
	op := "ChatService.Messages"

	log := c.log.With(
		slog.String("op", op),
		slog.String("chatId", chatID.String()))

	log.Info("getting messages")

	messages, err := c.msgProvider.Messages(ctx, chatID)
	if err != nil {
		log.Error("error getting messages", sl.Err(err))
		return nil, errors.New("internal server error")
	}

	return messages, nil
}
