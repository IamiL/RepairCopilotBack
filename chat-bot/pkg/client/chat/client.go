package chat

import (
	"context"
	"fmt"
	"time"

	pb "repairCopilotBot/chat-bot/pkg/chat-bot/api/proto/chat/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	client pb.ChatServiceClient
	conn   *grpc.ClientConn
}

type Config struct {
	Address string        `yaml:"address" env-default:"localhost:50053"`
	Timeout time.Duration `yaml:"timeout" env-default:"10s"`
}

func New(cfg *Config) (*Client, error) {
	conn, err := grpc.NewClient(cfg.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to chat service: %w", err)
	}

	client := pb.NewChatServiceClient(conn)

	return &Client{
		client: client,
		conn:   conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) CreateNewMessage(ctx context.Context, chatID *string, userID string, message string) (string, string, error) {
	req := &pb.CreateNewMessageRequest{
		ChatId:  chatID,
		UserId:  userID,
		Message: message,
	}

	resp, err := c.client.CreateNewMessage(ctx, req)
	if err != nil {
		return "", "", fmt.Errorf("failed to create new message: %w", err)
	}

	var chatIDResp string
	if resp.ChatId != nil {
		chatIDResp = *resp.ChatId
	}

	return chatIDResp, resp.Message, nil
}

type Chat struct {
	*pb.Chat
	CreatedAt time.Time `json:"created_at"`
}

func (c *Client) GetChats(ctx context.Context, userID *string) ([]Chat, error) {
	req := &pb.GetChatsRequest{
		UserID: userID,
	}

	resp, err := c.client.GetChats(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get chats: %w", err)
	}

	chats := make([]Chat, 0)
	for _, chatItem := range resp.Chats {
		chats = append(chats, Chat{
			Chat:      chatItem,
			CreatedAt: chatItem.CreatedAt.AsTime(),
		})
	}

	return chats, nil
}

type Message struct {
	*pb.Message
	CreatedAt time.Time `json:"created_at"`
}

func (c *Client) GetMessages(ctx context.Context, chatID string) ([]Message, error) {
	req := &pb.GetMessagesRequest{
		ChatId: chatID,
	}

	resp, err := c.client.GetMessages(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	messages := make([]Message, 0)
	for _, msgItem := range resp.Messages {
		messages = append(messages, Message{
			Message:   msgItem,
			CreatedAt: msgItem.CreatedAt.AsTime(),
		})
	}

	return messages, nil
}

func (c *Client) FinishChat(ctx context.Context, chatID string, userID string) (string, error) {
	req := &pb.FinishChatRequest{
		ChatId: chatID,
		UserId: userID,
	}

	resp, err := c.client.FinishChat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to finish chat: %w", err)
	}

	return resp.Msg, nil
}
