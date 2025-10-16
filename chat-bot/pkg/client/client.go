package client

import (
	"fmt"
	"time"

	chatClient "repairCopilotBot/chat-bot/pkg/client/chat"
	userClient "repairCopilotBot/chat-bot/pkg/client/user"
)

type Config struct {
	Address string        `yaml:"address" env-default:"localhost:50053"`
	Timeout time.Duration `yaml:"timeout" env-default:"1000s"`
}

type ChatBotClient struct {
	Chat *chatClient.Client
	User *userClient.Client
}

func New(cfg *Config) (*ChatBotClient, error) {
	chatClientConfig := &chatClient.Config{
		Address: cfg.Address,
		Timeout: cfg.Timeout,
	}

	userClientConfig := &userClient.Config{
		Address: cfg.Address,
		Timeout: cfg.Timeout,
	}

	chat, err := chatClient.New(chatClientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create search client: %w", err)
	}

	user, err := userClient.New(userClientConfig)
	if err != nil {
		chat.Close()
		return nil, fmt.Errorf("failed to create user client: %w", err)
	}

	return &ChatBotClient{
		Chat: chat,
		User: user,
	}, nil
}

func (c *ChatBotClient) Close() error {
	var errs []error

	if err := c.Chat.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close search client: %w", err))
	}

	if err := c.User.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close user client: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing clients: %v", errs)
	}

	return nil
}
