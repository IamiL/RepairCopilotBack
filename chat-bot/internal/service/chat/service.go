package chatservice

import (
	"context"
	"encoding/json"
	"log/slog"
	"repairCopilotBot/chat-bot/internal/domain/model/chat"
	messagemodel "repairCopilotBot/chat-bot/internal/domain/model/message"
	"repairCopilotBot/chat-bot/internal/pkg/llmClient"
	"time"

	"github.com/google/uuid"
)

type ChatService struct {
	log          *slog.Logger
	usrProvider  UserProvider
	chatProvider ChatProvider
	chatSaver    ChatSaver
	actionSaver  ActionSaver
	msgProvider  MessageProvider
	msgSaver     MessageSaver
	llmClient    *llmClient.Client
}

type UserProvider interface {
	// Возвращает messages_left_for_today по userID
	GetUserConfirmAndLimitsInfo(ctx context.Context, userID uuid.UUID) (int, error)
}

type ChatProvider interface {
	Chats(ctx context.Context) ([]chatmodel.Chat, error)
	ChatsForUser(ctx context.Context, userID uuid.UUID) ([]chatmodel.Chat, error)
	GetChatTree(ctx context.Context, chatID uuid.UUID) (json.RawMessage, error)

	// возвращает userID, is_finished, is_processing,
	ChatShortInfo(ctx context.Context, chatID uuid.UUID) (uuid.UUID, bool, bool, error)
}

type ChatSaver interface {
	CreateChat(ctx context.Context, chatId uuid.UUID, userID uuid.UUID, isFinished bool, isProcessing bool, enclosure int, createdAt time.Time, updatedAt time.Time) error
	FinishChat(ctx context.Context, chatID uuid.UUID, conclusion string) error
	UpdateChatTree(ctx context.Context, chatID uuid.UUID, tree json.RawMessage) error
}

type ActionSaver interface {
	CreateAction(ctx context.Context, actionId uuid.UUID, actionType int, userID uuid.UUID, message string, createdAt time.Time) error
}

type MessageProvider interface {
	Messages(ctx context.Context, chatID uuid.UUID) ([]messagemodel.Message, error)
}

type MessageSaver interface {
	CreateMessage(ctx context.Context, chat_id uuid.UUID, role string, content string, nestingLevel int, createdAt time.Time, updatedAt time.Time) error
}
