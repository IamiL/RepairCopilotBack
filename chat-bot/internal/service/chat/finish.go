package chatservice

import (
	"context"
	"errors"
	"log/slog"
	"repairCopilotBot/chat-bot/internal/pkg/logger/sl"
	"time"

	"github.com/google/uuid"
)

func (c *ChatService) FinishChat(ctx context.Context, userID uuid.UUID, chatID uuid.UUID) (string, error) {
	op := "chat.FinishChat"

	log := c.log.With(
		slog.String("op", op),
		slog.String("userID", userID.String()),
		slog.String("chatID", chatID.String()),
	)

	log.Info("processing finish chat")

	chatUserID, isFinished, isProcessing, err := c.chatProvider.ChatShortInfo(ctx, chatID)

	if chatUserID != userID {
		log.Error("userID does not match")
		return "", errors.New("userID does not match")
	}

	if isFinished {
		log.Info("chat already finished")
		return "", errors.New("chat already finished")
	}

	if isProcessing {
		log.Info("chat is processing")
		return "", errors.New("chat is processing")
	}

	now := time.Now()

	err = c.msgSaver.CreateMessage(ctx, chatID, "user", "Завершить чат", 0, now, now)
	if err != nil {
		log.Error("Error in saving message", sl.Err(err))
		return "", errors.New("internal server error")
	}

	msg, err := c.llmClient.FinishChat(chatID)
	if err != nil {
		log.Error("Error in finishing chat", sl.Err(err))
		return "", errors.New("internal server error")
	}

	now = time.Now()

	//err = c.msgSaver.CreateMessage(ctx, chatID, "llm", msg, 0, now, now)
	//if err != nil {
	//	log.Error("Error in saving message", sl.Err(err))
	//	return "", errors.New("internal server error")
	//}

	err = c.chatSaver.FinishChat(ctx, chatID, msg)
	if err != nil {
		log.Error("Error in finishing chat", sl.Err(err))
		return "", errors.New("internal server error")
	}

	return msg, nil
}
