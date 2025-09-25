package chatservice

import (
	"log/slog"
	"repairCopilotBot/chat-bot/internal/pkg/llmClient"
)

func New(log *slog.Logger, userProvider UserProvider, chatProvider ChatProvider, chatSaver ChatSaver, actionSaver ActionSaver, messageProvider MessageProvider, messageSaver MessageSaver, llmClient *llmClient.Client) *ChatService {
	return &ChatService{
		log:          log,
		usrProvider:  userProvider,
		chatProvider: chatProvider,
		chatSaver:    chatSaver,
		actionSaver:  actionSaver,
		msgProvider:  messageProvider,
		msgSaver:     messageSaver,
		llmClient:    llmClient,
	}
}
