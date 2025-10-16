package chatservice

import (
	"context"
	"errors"
	"log/slog"
	"repairCopilotBot/chat-bot/internal/domain/model/action"
	"repairCopilotBot/chat-bot/internal/pkg/logger/sl"
	"time"

	"github.com/google/uuid"
)

func (c *ChatService) CreateChat(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	op := "search.CreateChat"
	log := c.log.With(
		slog.String("op", op),
		slog.String("userID", userID.String()),
	)

	log.Info("creating search")

	messagesLeftForToday, err := c.usrProvider.GetUserConfirmAndLimitsInfo(ctx, userID)
	if err != nil {
		log.Error("Error in getting user confirmed and limits info", sl.Err(err))
		return uuid.Nil, errors.New("internal server error")
	}

	//if !isConfirmed {
	//	log.Info("user is not confirmed")
	//	return uuid.Nil, errors.New("user is not confirmed")
	//}

	if messagesLeftForToday == 0 {
		log.Info("no messages left for today")
		return uuid.Nil, errors.New("no messages left for today")
	}

	newChatId := uuid.New()

	now := time.Now()

	err = c.chatSaver.CreateChat(ctx, newChatId, userID, false, false, 0, now, now)
	if err != nil {
		log.Error("Error in creating new search", sl.Err(err))
		return uuid.Nil, errors.New("internal server error")
	}

	go func() {
		newActionID := uuid.New()
		actionMessage := "user " + userID.String() + " создал(а) новый чат"

		err = c.actionSaver.CreateAction(ctx, newActionID, actionmodel.CreateChatActionType, userID, actionMessage, now)
		if err != nil {
			log.Error("Error in creating new action log", sl.Err(err))
		}
	}()

	return newChatId, nil
}
