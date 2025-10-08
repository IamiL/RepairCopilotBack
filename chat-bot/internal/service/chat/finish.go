package chatservice

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"repairCopilotBot/chat-bot/internal/pkg/llmClient"
	"repairCopilotBot/chat-bot/internal/pkg/logger/sl"

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

	// Получаем tree и messages
	tree, err := c.chatProvider.GetChatTree(ctx, chatID)
	if err != nil {
		log.Error("Error in getting chat tree", sl.Err(err))
		return "", errors.New("internal server error")
	}

	messages, err := c.msgProvider.Messages(ctx, chatID)
	if err != nil {
		log.Error("Error in getting messages", sl.Err(err))
		return "", errors.New("internal server error")
	}

	// Конвертируем сообщения в пары {user, bot}
	history := convertMessagesToHistory(messages)

	// Десериализуем tree
	var treeMap map[string]interface{}
	if len(tree) > 0 {
		if err := json.Unmarshal(tree, &treeMap); err != nil {
			log.Error("Error in unmarshaling tree", sl.Err(err))
			return "", errors.New("internal server error")
		}
	} else {
		treeMap = make(map[string]interface{})
	}

	state := &llmClient.ChatState{
		History: history,
		Tree:    treeMap,
	}

	// Получаем summary от LLM
	summary, err := c.llmClient.FinishChat(state)
	if err != nil {
		log.Error("Error in finishing chat", sl.Err(err))
		return "", errors.New("internal server error")
	}

	// Сохраняем чат как завершенный с резюме
	err = c.chatSaver.FinishChat(ctx, chatID, summary)
	if err != nil {
		log.Error("Error in finishing chat", sl.Err(err))
		return "", errors.New("internal server error")
	}

	return summary, nil
}
