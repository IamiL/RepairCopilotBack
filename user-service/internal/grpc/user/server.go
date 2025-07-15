package usergrpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	service "repairCopilotBot/user-service/internal/service/user"
	pb "repairCopilotBot/user-service/pkg/user/v1"
)

// UserGRPCServer реализует UserServiceServer
type UserGRPCServer struct {
	pb.UnimplementedUserServiceServer
	userService *service.User // Ваш существующий сервис
}

// NewUserGRPCServer создает новый gRPC сервер
func NewUserGRPCServer(userService *service.User) *UserGRPCServer {
	return &UserGRPCServer{
		userService: userService,
	}
}

// RegisterUser обрабатывает регистрацию пользователя
func (s *UserGRPCServer) RegisterUser(ctx context.Context, req *pb.RegisterUserRequest) (*pb.RegisterUserResponse, error) {
	// Валидация входных данных
	if req.Login == "" {
		return nil, status.Error(codes.InvalidArgument, "login is required")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	// Вызов вашего существующего метода
	userID, err := s.userService.RegisterNewUser(ctx, req.Login, req.Password)
	if err != nil {
		// Обработка различных типов ошибок
		if errors.Is(err, service.ErrUserAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		}

		// Для остальных ошибок возвращаем Internal Server Error
		return nil, status.Error(codes.Internal, "failed to register user")
	}

	return &pb.RegisterUserResponse{
		UserId: userID.String(),
	}, nil
}

// Login обрабатывает аутентификацию пользователя
func (s *UserGRPCServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// Валидация входных данных
	if req.Login == "" {
		return nil, status.Error(codes.InvalidArgument, "login is required")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	// Вызов вашего существующего метода
	userID, err := s.userService.Login(ctx, req.Login, req.Password)
	if err != nil {
		// Обработка ошибок аутентификации
		if errors.Is(err, service.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}

		// Для остальных ошибок возвращаем Internal Server Error
		return nil, status.Error(codes.Internal, "failed to authenticate user")
	}

	return &pb.LoginResponse{
		UserId: userID.String(),
	}, nil
}
