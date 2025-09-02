package user_service_client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	pb "repairCopilotBot/user-service/pkg/user/v1"
)

// Client клиент для user-service
type Client struct {
	client pb.UserServiceClient
	conn   *grpc.ClientConn
}

// NewClient создает новый клиент для user-service
func NewClient(address string) (*Client, error) {
	conn, err := grpc.Dial(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(time.Second*10),
	)
	if err != nil {
		return nil, err
	}

	client := pb.NewUserServiceClient(conn)

	return &Client{
		client: client,
		conn:   conn,
	}, nil
}

// Close закрывает соединение
func (c *Client) Close() error {
	return c.conn.Close()
}

// IncrementInspectionsForToday увеличивает счетчик проверок за сегодня для пользователя
func (c *Client) IncrementInspectionsForToday(ctx context.Context, userID string) error {
	req := &pb.IncrementInspectionsForTodayByUserIdRequest{
		UserId: userID,
	}

	_, err := c.client.IncrementInspectionsForTodayByUserId(ctx, req)
	if err != nil {
		// Обработка gRPC статусов
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.InvalidArgument:
				return fmt.Errorf("invalid user_id: %s", st.Message())
			case codes.ResourceExhausted:
				return fmt.Errorf("daily inspection limit exceeded")
			case codes.NotFound:
				return fmt.Errorf("user not found")
			case codes.Internal:
				return fmt.Errorf("internal server error")
			default:
				return fmt.Errorf("failed to increment inspections for today: %s", st.Message())
			}
		}
		return err
	}

	return nil
}

// DecrementInspectionsForToday уменьшает счетчик проверок за сегодня для пользователя
func (c *Client) DecrementInspectionsForToday(ctx context.Context, userID string) error {
	req := &pb.DecrementInspectionsForTodayByUserIdRequest{
		UserId: userID,
	}

	_, err := c.client.DecrementInspectionsForTodayByUserId(ctx, req)
	if err != nil {
		// Обработка gRPC статусов
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.InvalidArgument:
				return fmt.Errorf("invalid user_id: %s", st.Message())
			case codes.NotFound:
				return fmt.Errorf("user not found")
			case codes.Internal:
				return fmt.Errorf("internal server error")
			default:
				return fmt.Errorf("failed to decrement inspections for today: %s", st.Message())
			}
		}
		return err
	}

	return nil
}