package userserviceclient

import (
	"context"
	"fmt"
	pb "repairCopilotBot/user-service/pkg/user/v1"
	"time"

	"github.com/google/uuid"
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
func (c *UserClient) RegisterUser(ctx context.Context, email, firstName, lastName, login, password string) (uuid.UUID, error) {
	req := &pb.RegisterUserRequest{
		Login:     login,
		Password:  password,
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
	}

	resp, err := c.client.RegisterUser(ctx, req)
	if err != nil {
		// Обработка gRPC статусов
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.InvalidArgument:
				return uuid.Nil, fmt.Errorf("invalid input: %s", st.Message())
			case codes.AlreadyExists:
				return uuid.Nil, fmt.Errorf("user already exists")
			case codes.Internal:
				return uuid.Nil, fmt.Errorf("internal server error")
			default:
				return uuid.Nil, fmt.Errorf("registration failed: %s", st.Message())
			}
		}
		return uuid.Nil, err
	}

	return uuid.MustParse(resp.UserId), nil
}

// LoginResponse содержит данные пользователя после аутентификации
//
//	type LoginResponse struct {
//		UserID   string
//		Login    string
//		IsAdmin1 bool
//		IsAdmin2 bool
//	}
type LoginResponse struct {
	*pb.LoginResponse
	RegisteredAt time.Time `json:"created_at"`
	LastVisitAt  time.Time `json:"last_visit_at"`
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
		LoginResponse: resp,
		RegisteredAt:  resp.RegisteredAt.AsTime(),
		LastVisitAt:   resp.LastVisitAt.AsTime(),
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
func (c *UserClient) GetAllUsers(ctx context.Context) ([]*pb.UserInfo, error) {
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

	//users := make([]UserInfo, len(resp.Users))
	//for i, user := range resp.Users {
	//	users[i] = UserInfo{
	//		UserID:   user.UserId,
	//		Login:    user.Login,
	//		IsAdmin1: user.IsAdmin1,
	//		IsAdmin2: user.IsAdmin2,
	//	}
	//}

	return resp.Users, nil
}

type GetUserInfoResponse struct {
	*pb.GetUserInfoResponse
	RegisteredAt time.Time `json:"registered_at"`
	LastVisitAt  time.Time `json:"last_visit_at"`
}

// GetUserInfo получает подробную информацию о пользователе по ID
func (c *UserClient) GetUserInfo(ctx context.Context, userID uuid.UUID) (*GetUserInfoResponse, error) {
	req := &pb.GetUserInfoRequest{
		UserId: userID.String(),
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

	return &GetUserInfoResponse{
		GetUserInfoResponse: resp,
		RegisteredAt:        resp.RegisteredAt.AsTime(),
		LastVisitAt:         resp.LastVisitAt.AsTime(),
	}, nil
}

// FullName представляет полное имя пользователя
type FullName struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// GetFullNamesById получает полные имена пользователей по их ID
func (c *UserClient) GetFullNamesById(ctx context.Context, userIDs []string) (map[string]FullName, error) {
	if len(userIDs) == 0 {
		return make(map[string]FullName), nil
	}

	// Преобразуем массив ID в map для protobuf
	idsMap := make(map[string]*pb.Empty, len(userIDs))
	for _, id := range userIDs {
		idsMap[id] = &pb.Empty{}
	}

	req := &pb.GetFullNamesByIdRequest{
		Ids: idsMap,
	}

	resp, err := c.client.GetFullNamesById(ctx, req)
	if err != nil {
		// Обработка gRPC статусов
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.Internal:
				return nil, fmt.Errorf("internal server error")
			default:
				return nil, fmt.Errorf("failed to get full names: %s", st.Message())
			}
		}
		return nil, err
	}

	// Преобразуем protobuf ответ в обычный map
	result := make(map[string]FullName, len(resp.Users))
	for id, fullName := range resp.Users {
		result[id] = FullName{
			FirstName: fullName.FirstName,
			LastName:  fullName.LastName,
		}
	}

	return result, nil
}

// UpdateInspectionsPerDay изменяет inspections_per_day для пользователя или всех пользователей
func (c *UserClient) UpdateInspectionsPerDay(ctx context.Context, userID string, inspectionsPerDay uint32) (*pb.UpdateInspectionsPerDayResponse, error) {
	req := &pb.UpdateInspectionsPerDayRequest{
		UserId:            userID,
		InspectionsPerDay: inspectionsPerDay,
	}

	resp, err := c.client.UpdateInspectionsPerDay(ctx, req)
	if err != nil {
		// Обработка gRPC статусов
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.InvalidArgument:
				return nil, fmt.Errorf("invalid input: %s", st.Message())
			case codes.NotFound:
				return nil, fmt.Errorf("user not found")
			case codes.Internal:
				return nil, fmt.Errorf("internal server error")
			default:
				return nil, fmt.Errorf("failed to update inspections per day: %s", st.Message())
			}
		}
		return nil, err
	}

	return resp, nil
}

// RegisterVisit регистрирует посещение пользователя
func (c *UserClient) RegisterVisit(ctx context.Context, userID string) error {
	req := &pb.RegisterVisitRequest{
		UserId: userID,
	}

	_, err := c.client.RegisterVisit(ctx, req)
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
				return fmt.Errorf("failed to register visit: %s", st.Message())
			}
		}
		return err
	}

	return nil
}
