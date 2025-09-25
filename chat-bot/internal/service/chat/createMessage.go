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

func (c *ChatService) NewMessage(ctx context.Context, userID uuid.UUID, chatID uuid.UUID, text string) (string, uuid.UUID, error) {
	op := "chat.NewMessage"

	log := c.log.With(
		slog.String("op", op),
		slog.String("userID", userID.String()),
	)

	log.Info("processing new message", slog.String("message", text))

	messagesLeftForToday, err := c.usrProvider.GetUserConfirmAndLimitsInfo(ctx, userID)
	if err != nil {
		log.Error("Error in getting user confirmed and limits info", sl.Err(err))
		return "", uuid.Nil, errors.New("internal server error")
	}

	if messagesLeftForToday == 0 {
		log.Info("no messages left for today")
		return "", uuid.Nil, errors.New("no messages left for today")
	}

	now := time.Now()

	if chatID == uuid.Nil {
		log.Info("chatID is Nil. Creating new chat")
		newChatId := uuid.New()

		err = c.chatSaver.CreateChat(ctx, newChatId, userID, false, false, 0, now, now)
		if err != nil {
			log.Error("Error in creating new chat", sl.Err(err))
			return "", uuid.Nil, errors.New("internal server error")
		}

		chatID = newChatId

		err := c.llmClient.StartDialog(chatID)
		if err != nil {
			log.Error("Error in starting dialog", sl.Err(err))
			return "", uuid.Nil, errors.New("internal server error")
		}

		go func() {
			newActionID := uuid.New()
			actionMessage := "user " + userID.String() + " создал(а) новый чат"

			err = c.actionSaver.CreateAction(ctx, newActionID, actionmodel.CreateChatActionType, userID, actionMessage, now)
			if err != nil {
				log.Error("Error in creating new action log", sl.Err(err))
			}
		}()
	}

	err = c.msgSaver.CreateMessage(ctx, chatID, "user", text, 0, now, now)
	if err != nil {
		log.Error("Error in creating new message", sl.Err(err))
		return "", uuid.Nil, errors.New("internal server error")
	}

	llmResp, err := c.llmClient.SendMessage(chatID, text)
	if err != nil {
		log.Error("Error in sending message to llm", sl.Err(err))
		return "", uuid.Nil, errors.New("internal server error")
	}

	now = time.Now()

	err = c.msgSaver.CreateMessage(ctx, chatID, "llm", llmResp, 0, now, now)
	if err != nil {
		log.Error("Error in creating new message", sl.Err(err))
		return "", uuid.Nil, errors.New("internal server error")
	}

	return llmResp, chatID, nil
}
