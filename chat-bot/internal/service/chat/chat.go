package chatservice

import (
	"context"
	"errors"
	"log/slog"
	"repairCopilotBot/chat-bot/internal/domain/model/chat"
	"repairCopilotBot/chat-bot/internal/pkg/logger/sl"

	"github.com/google/uuid"
)

func (c *ChatService) Chats(ctx context.Context) ([]chatmodel.Chat, error) {
	op := "chat.Chats"
	log := c.log.With(
		slog.String("op", op),
	)

	log.Info("getting chats")

	chats, err := c.chatProvider.Chats(ctx)
	if err != nil {
		log.Error("Error in getting chats", sl.Err(err))
		return nil, errors.New("internal server error")
	}

	return chats, nil
}

func (c *ChatService) ChatsForUser(ctx context.Context, userID uuid.UUID) ([]chatmodel.Chat, error) {
	op := "chat.ChatsForUser"
	log := c.log.With(
		slog.String("op", op),
		slog.String("userID", userID.String()),
	)

	log.Info("getting chats")

	chats, err := c.chatProvider.ChatsForUser(ctx, userID)
	if err != nil {
		log.Error("Error in getting chats", sl.Err(err))
		return nil, errors.New("internal server error")
	}

	return chats, nil
}
