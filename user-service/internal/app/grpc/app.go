package grpcapp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	service "repairCopilotBot/user-service/internal/service/user"
	pb "repairCopilotBot/user-service/pkg/user/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

type serverAPI struct {
	pb.UnimplementedUserServiceServer
	userService *service.User
	log         *slog.Logger
}

type Config struct {
	Port string `yaml:"port" env-default:":50052"`
}

// UserGRPCServer реализует UserServiceServer
type UserGRPCServer struct {
	log         *slog.Logger
	gRPCServer  *grpc.Server
	userService *service.User // Ваш существующий сервис
	port        string
}

// NewUserGRPCServer создает новый gRPC сервер
func NewUserGRPCServer(log *slog.Logger, userService *service.User, config Config) *UserGRPCServer {
	gRPCServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 30 * time.Minute,
			Time:              30 * time.Minute,
			Timeout:           30 * time.Minute,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             30 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.ConnectionTimeout(30*time.Minute),
	)

	pb.RegisterUserServiceServer(gRPCServer, serverAPI{
		userService: userService,
		log:         log,
	})

	return &UserGRPCServer{
		log:        log,
		gRPCServer: gRPCServer,
		port:       config.Port,
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
	userID, isAdmin1, isAdmin2, err := s.userService.Login(ctx, req.Login, req.Password)
	if err != nil {
		// Обработка ошибок аутентификации
		if errors.Is(err, service.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}

		// Для остальных ошибок возвращаем Internal Server Error
		return nil, status.Error(codes.Internal, "failed to authenticate user")
	}

	return &pb.LoginResponse{
		UserId:   userID.String(),
		IsAdmin1: isAdmin1,
		IsAdmin2: isAdmin2,
	}, nil
}

func (a *UserGRPCServer) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *UserGRPCServer) Run() error {
	const op = "grpcapp.Run"

	log := a.log.With(slog.String("op", op), slog.String("port", a.port))

	l, err := net.Listen("tcp", a.port)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("grpc server started", slog.String("addr", l.Addr().String()))

	if err := a.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *UserGRPCServer) Stop() {
	const op = "grpcapp.Stop"

	a.log.With(slog.String("op", op)).Info("stopping gRPC server")

	a.gRPCServer.GracefulStop()
}