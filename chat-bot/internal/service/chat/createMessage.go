package chatservice

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"repairCopilotBot/chat-bot/internal/domain/model/action"
	messagemodel "repairCopilotBot/chat-bot/internal/domain/model/message"
	"repairCopilotBot/chat-bot/internal/pkg/llmClient"
	"repairCopilotBot/chat-bot/internal/pkg/logger/sl"
	"time"

	"github.com/google/uuid"
)

func (c *ChatService) NewMessage(ctx context.Context, userID uuid.UUID, chatID uuid.UUID, text string) (string, uuid.UUID, error) {
	op := "search.NewMessage"
	ctx = context.Background()

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

	isNewChat := chatID == uuid.Nil

	if isNewChat {
		log.Info("chatID is Nil. Creating new search")
		newChatId := uuid.New()

		err = c.chatSaver.CreateChat(ctx, newChatId, userID, false, false, 0, now, now)
		if err != nil {
			log.Error("Error in creating new search", sl.Err(err))
			return "", uuid.Nil, errors.New("internal server error")
		}

		chatID = newChatId

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

	var llmResp string
	var newState *llmClient.ChatState
	var state *llmClient.ChatState

	if isNewChat {
		// Новый чат - используем пустое состояние
		state = &llmClient.ChatState{
			History: []llmClient.MessagePair{},
			Tree:    make(map[string]interface{}),
		}
	} else {
		// Существующий чат - получаем tree и messages
		tree, err := c.chatProvider.GetChatTree(ctx, chatID)
		if err != nil {
			log.Error("Error in getting search tree", sl.Err(err))
			return "", uuid.Nil, errors.New("internal server error")
		}

		messages, err := c.msgProvider.Messages(ctx, chatID)
		if err != nil {
			log.Error("Error in getting messages", sl.Err(err))
			return "", uuid.Nil, errors.New("internal server error")
		}

		// Конвертируем сообщения в пары {user, bot}
		history := convertMessagesToHistory(messages)

		// Десериализуем tree
		var treeMap map[string]interface{}
		if len(tree) > 0 {
			if err := json.Unmarshal(tree, &treeMap); err != nil {
				log.Error("Error in unmarshaling tree", sl.Err(err))
				return "", uuid.Nil, errors.New("internal server error")
			}
		} else {
			treeMap = make(map[string]interface{})
		}

		state = &llmClient.ChatState{
			History: history,
			Tree:    treeMap,
		}
	}

	// Отправляем сообщение в LLM
	llmResp, newState, err = c.llmClient.SendMessage(state, text)
	if err != nil {
		log.Error("Error in sending message to llm", sl.Err(err))
		return "", uuid.Nil, errors.New("internal server error")
	}

	// Сохраняем обновленное дерево в БД
	treeJSON, err := json.Marshal(newState.Tree)
	if err != nil {
		log.Error("Error in marshaling tree", sl.Err(err))
		return "", uuid.Nil, errors.New("internal server error")
	}

	err = c.chatSaver.UpdateChatTree(context.Background(), chatID, treeJSON)
	if err != nil {
		log.Error("Error in updating search tree", sl.Err(err))
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

// convertMessagesToHistory конвертирует список сообщений в пары {user, bot}
func convertMessagesToHistory(messages []messagemodel.Message) []llmClient.MessagePair {
	var history []llmClient.MessagePair
	var currentPair *llmClient.MessagePair

	for _, msg := range messages {
		if msg.Role == "user" {
			if currentPair != nil {
				// Если есть незавершенная пара, добавляем её
				history = append(history, *currentPair)
			}
			currentPair = &llmClient.MessagePair{
				User: msg.Content,
			}
		} else if msg.Role == "llm" && currentPair != nil {
			currentPair.Bot = msg.Content
			history = append(history, *currentPair)
			currentPair = nil
		}
	}

	return history
}
