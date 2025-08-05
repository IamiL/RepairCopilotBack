package userserviceclient

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	pb "repairCopilotBot/user-service/pkg/user/v1"
	"time"
)

// UserClient обертка для gRPC клиента
type UserClient struct {
	client pb.UserServiceClient
	conn   *grpc.ClientConn
}

// NewUserClient создает новый клиент для user-service
func NewUserClient(address string) (*UserClient, error) {
	conn, err := grpc.Dial(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(time.Second*10),
	)
	if err != nil {
		return nil, err
	}

	client := pb.NewUserServiceClient(conn)

	return &UserClient{
		client: client,
		conn:   conn,
	}, nil
}

// Close закрывает соединение
func (c *UserClient) Close() error {
	return c.conn.Close()
}

// RegisterUser регистрирует нового пользователя
func (c *UserClient) RegisterUser(ctx context.Context, login, password string) (string, error) {
	req := &pb.RegisterUserRequest{
		Login:    login,
		Password: password,
	}

	resp, err := c.client.RegisterUser(ctx, req)
	if err != nil {
		// Обработка gRPC статусов
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.InvalidArgument:
				return "", fmt.Errorf("invalid input: %s", st.Message())
			case codes.AlreadyExists:
				return "", fmt.Errorf("user already exists")
			case codes.Internal:
				return "", fmt.Errorf("internal server error")
			default:
				return "", fmt.Errorf("registration failed: %s", st.Message())
			}
		}
		return "", err
	}

	return resp.UserId, nil
}

// LoginResponse содержит данные пользователя после аутентификации 
type LoginResponse struct {
	UserID   string
	Login    string
	IsAdmin1 bool
	IsAdmin2 bool
}

// Login выполняет аутентификацию пользователя
func (c *UserClient) Login(ctx context.Context, login, password string) (*LoginResponse, error) {
	req := &pb.LoginRequest{
		Login:    login,
		Password: password,
	}

	resp, err := c.client.Login(ctx, req)
	if err != nil {
		// Обработка gRPC статусов
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.InvalidArgument:
				return nil, fmt.Errorf("invalid input: %s", st.Message())
			case codes.Unauthenticated:
				return nil, fmt.Errorf("invalid credentials")
			case codes.Internal:
				return nil, fmt.Errorf("internal server error")
			default:
				return nil, fmt.Errorf("authentication failed: %s", st.Message())
			}
		}
		return nil, err
	}

	return &LoginResponse{
		UserID:   resp.UserId,
		Login:    login,
		IsAdmin1: resp.IsAdmin1,
		IsAdmin2: resp.IsAdmin2,
	}, nil
}
