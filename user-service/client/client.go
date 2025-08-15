package userserviceclient

import (
	"context"
	"fmt"
	pb "repairCopilotBot/user-service/pkg/user/v1"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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
func (c *UserClient) RegisterUser(ctx context.Context, email, firstName, lastName, login, password string) (string, error) {
	req := &pb.RegisterUserRequest{
		Login:    login,
		Password: password,
		Name:     firstName,
		Email:    email,
		Surname:  lastName,
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

// UserInfo информация о пользователе
type UserInfo struct {
	UserID   string `json:"user_id"`
	Login    string `json:"login"`
	IsAdmin1 bool   `json:"is_admin1"`
	IsAdmin2 bool   `json:"is_admin2"`
}

// GetAllUsers получает список всех пользователей
func (c *UserClient) GetAllUsers(ctx context.Context) ([]UserInfo, error) {
	req := &pb.GetAllUsersRequest{}

	resp, err := c.client.GetAllUsers(ctx, req)
	if err != nil {
		// Обработка gRPC статусов
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.Internal:
				return nil, fmt.Errorf("internal server error")
			default:
				return nil, fmt.Errorf("failed to get users: %s", st.Message())
			}
		}
		return nil, err
	}

	users := make([]UserInfo, len(resp.Users))
	for i, user := range resp.Users {
		users[i] = UserInfo{
			UserID:   user.UserId,
			Login:    user.Login,
			IsAdmin1: user.IsAdmin1,
			IsAdmin2: user.IsAdmin2,
		}
	}

	return users, nil
}

// UserDetailedInfo информация о пользователе с подробностями
type UserDetailedInfo struct {
	UserID    string `json:"user_id"`
	Login     string `json:"login"`
	IsAdmin1  bool   `json:"is_admin1"`
	IsAdmin2  bool   `json:"is_admin2"`
	CreatedAt string `json:"created_at"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// GetUserInfo получает подробную информацию о пользователе по ID
func (c *UserClient) GetUserInfo(ctx context.Context, userID string) (*UserDetailedInfo, error) {
	req := &pb.GetUserInfoRequest{
		UserId: userID,
	}

	resp, err := c.client.GetUserInfo(ctx, req)
	if err != nil {
		// Обработка gRPC статусов
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.InvalidArgument:
				return nil, fmt.Errorf("invalid user_id: %s", st.Message())
			case codes.Internal:
				return nil, fmt.Errorf("internal server error")
			default:
				return nil, fmt.Errorf("failed to get user info: %s", st.Message())
			}
		}
		return nil, err
	}

	return &UserDetailedInfo{
		UserID:    resp.UserId,
		Login:     resp.Login,
		IsAdmin1:  resp.IsAdmin1,
		IsAdmin2:  resp.IsAdmin2,
		CreatedAt: resp.CreatedAt,
		Email:     resp.Email,
		FirstName: resp.Name,
		LastName:  resp.Surname,
	}, nil
}
