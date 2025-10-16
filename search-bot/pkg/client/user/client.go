package user

import (
	"context"
	"fmt"
	"time"

	pb "repairCopilotBot/search-bot/pkg/search-bot/api/proto/search/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	client pb.UserServiceClient
	conn   *grpc.ClientConn
}

type Config struct {
	Address string        `yaml:"address" env-default:"localhost:50053"`
	Timeout time.Duration `yaml:"timeout" env-default:"10s"`
}

func New(cfg *Config) (*Client, error) {
	conn, err := grpc.NewClient(cfg.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user service: %w", err)
	}

	client := pb.NewUserServiceClient(conn)

	return &Client{
		client: client,
		conn:   conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) CreateNewUser(ctx context.Context, userID string) error {
	req := &pb.CreateNewUserRequest{
		UserId: userID,
	}

	_, err := c.client.CreateNewUser(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create new user: %w", err)
	}

	return nil
}
