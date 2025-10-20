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

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type App struct {
	//pb.UnimplementedUserServiceServer
	log  *slog.Logger
	port string
}

type serverAPI struct {
	log         *slog.Logger
	userService *service.User
	pb.UnimplementedUserServiceServer
}

type Config struct {
	Port string `env:"PORT" env-default:":50052"`
}

// UserGRPCServer реализует UserServiceServer
type UserGRPCServer struct {
	log         *slog.Logger
	gRPCServer  *grpc.Server
	userService *service.User // Ваш существующий сервис
	port        string
}

// NewUserGRPCServer создает новый gRPC сервер
func NewUserGRPCServer(log *slog.Logger, userService *service.User, config *Config) *UserGRPCServer {
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

	pb.RegisterUserServiceServer(gRPCServer, &serverAPI{
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
func (s *serverAPI) RegisterUser(ctx context.Context, req *pb.RegisterUserRequest) (*pb.RegisterUserResponse, error) {
	// Валидация входных данных
	if req.Login == "" {
		return nil, status.Error(codes.InvalidArgument, "login is required")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}
	if req.FirstName == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.LastName == "" {
		return nil, status.Error(codes.InvalidArgument, "surname is required")
	}
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	// Вызов вашего существующего метода
	userID, err := s.userService.RegisterNewUser(ctx, req.Login, req.Password, req.FirstName, req.LastName, req.Email)
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
func (s *serverAPI) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// Валидация входных данных
	if req.Login == "" {
		return nil, status.Error(codes.InvalidArgument, "login is required")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	// Вызов вашего существующего метода
	user, err := s.userService.Login(ctx, req.Login, req.Password)
	if err != nil {
		// Обработка ошибок аутентификации
		if errors.Is(err, service.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}

		// Для остальных ошибок возвращаем Internal Server Error
		return nil, status.Error(codes.Internal, "failed to authenticate user")
	}

	return &pb.LoginResponse{
		UserId:      user.ID.String(),
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		IsAdmin1:    user.IsAdmin1,
		IsAdmin2:    user.IsAdmin2,
		IsConfirmed: user.IsConfirmed,
	}, nil
}

func (s *serverAPI) GetLoginById(ctx context.Context, req *pb.GetLoginByIdRequest) (*pb.GetLoginByIdResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	login, err := s.userService.GetLoginById(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get login")
	}

	return &pb.GetLoginByIdResponse{
		Login: login,
	}, nil
}

//func (s *serverAPI) GetUserByLogin(ctx context.Context, req *pb.GetUserByLoginRequest) (*pb.GetUserByLoginResponse, error) {
//	if req.Login == "" {
//		return nil, status.Error(codes.InvalidArgument, "login is required")
//	}
//
//	userID, login, name, surname, email, isAdmin1, isAdmin2, err := s.userService.GetUserByLogin(ctx, req.Login)
//	if err != nil {
//		return nil, status.Error(codes.Internal, "failed to get user by login")
//	}
//
//	return &pb.GetUserByLoginResponse{
//		UserId:   userID.String(),
//		Login:    login,
//		Name:     name,
//		Surname:  surname,
//		Email:    email,
//		IsAdmin1: isAdmin1,
//		IsAdmin2: isAdmin2,
//	}, nil
//}

func (s *serverAPI) GetAllUsers(ctx context.Context, req *pb.GetAllUsersRequest) (*pb.GetAllUsersResponse, error) {
	users, err := s.userService.GetAllUsers(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get users")
	}

	var pbUsers []*pb.UserInfo
	for _, user := range users {
		pbUsers = append(pbUsers, &pb.UserInfo{
			UserId:    user.ID.String(),
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
			IsAdmin1:  user.IsAdmin1,
			IsAdmin2:  user.IsAdmin2,
		})
	}

	return &pb.GetAllUsersResponse{
		Users: pbUsers,
	}, nil
}

func (s *serverAPI) GetUserInfo(ctx context.Context, req *pb.GetUserInfoRequest) (*pb.GetUserInfoResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	user, err := s.userService.User(ctx, uuid.MustParse(req.UserId))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user info")
	}

	return &pb.GetUserInfoResponse{
		FirstName:           user.FirstName,
		LastName:            user.LastName,
		Email:               user.Email,
		Login:               user.Login,
		IsAdmin1:            user.IsAdmin1,
		IsAdmin2:            user.IsAdmin2,
		IsConfirmed:         user.IsConfirmed,
		RegisteredAt:        timestamppb.New(user.CreatedAt),
		LastVisitAt:         timestamppb.New(user.LastVisitAt),
		InspectionsCount:    uint32(user.InspectionsCount),
		ErrorFeedbackCount:  uint32(user.ErrorFeedbacksCount),
		InspectionsPerDay:   uint32(user.InspectionsPerDay),
		InspectionsForToday: uint32(user.InspectionsForToday),
	}, nil
}

func (s *serverAPI) GetUserDetailsById(ctx context.Context, req *pb.GetUserDetailsByIdRequest) (*pb.GetUserDetailsByIdResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userDetails, err := s.userService.GetUserDetailsById(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user details")
	}

	return &pb.GetUserDetailsByIdResponse{
		UserId:    userDetails.ID.String(),
		Login:     userDetails.Login,
		Name:      userDetails.Name,
		Surname:   userDetails.Surname,
		Email:     userDetails.Email,
		IsAdmin1:  userDetails.IsAdmin1,
		IsAdmin2:  userDetails.IsAdmin2,
		CreatedAt: userDetails.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: userDetails.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *serverAPI) UpdateInspectionsPerDay(ctx context.Context, req *pb.UpdateInspectionsPerDayRequest) (*pb.UpdateInspectionsPerDayResponse, error) {
	//if req.InspectionsPerDay == 0 {
	//	return nil, status.Error(codes.InvalidArgument, "inspections_per_day must be greater than 0")
	//}

	rowsAffected, err := s.userService.UpdateInspectionsPerDay(ctx, req.UserId, int(req.InspectionsPerDay))
	if err != nil {
		s.log.Error("failed to update inspections_per_day", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to update inspections_per_day")
	}

	var message string
	if req.UserId == "" {
		message = fmt.Sprintf("Successfully updated inspections_per_day for all users")
	} else {
		message = fmt.Sprintf("Successfully updated inspections_per_day for user %s", req.UserId)
	}

	return &pb.UpdateInspectionsPerDayResponse{
		Success:      true,
		Message:      message,
		UpdatedCount: uint32(rowsAffected),
	}, nil
}

func (s *serverAPI) GetFullNamesById(ctx context.Context, req *pb.GetFullNamesByIdRequest) (*pb.GetFullNamesByIdResponse, error) {
	if req == nil || len(req.Ids) == 0 {
		return &pb.GetFullNamesByIdResponse{
			Users: make(map[string]*pb.FullName),
		}, nil
	}

	ids := make([]string, 0, len(req.Ids))
	for id := range req.Ids {
		ids = append(ids, id)
	}

	fullNames, err := s.userService.GetFullNamesById(ctx, ids)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get full names")
	}

	pbUsers := make(map[string]*pb.FullName, len(fullNames))
	for id, fullName := range fullNames {
		pbUsers[id] = &pb.FullName{
			FirstName: fullName.FirstName,
			LastName:  fullName.LastName,
		}
	}

	return &pb.GetFullNamesByIdResponse{
		Users: pbUsers,
	}, nil
}

func (s *serverAPI) RegisterVisit(ctx context.Context, req *pb.RegisterVisitRequest) (*pb.RegisterVisitResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	err := s.userService.RegisterVisit(ctx, req.UserId)
	if err != nil {
		s.log.Error("failed to register visit", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to register visit")
	}

	return &pb.RegisterVisitResponse{}, nil
}

func (s *serverAPI) ConfirmEmail(ctx context.Context, req *pb.ConfirmEmailRequest) (*pb.ConfirmEmailResponse, error) {
	err := s.userService.ConfirmEmail(ctx, req.UserId, req.Code)
	if err != nil {
		s.log.Error("failed to confirm email", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to confirm email")
	}

	return &pb.ConfirmEmailResponse{}, nil
}

func (s *serverAPI) IncrementInspectionsForTodayByUserId(ctx context.Context, req *pb.IncrementInspectionsForTodayByUserIdRequest) (*pb.IncrementInspectionsForTodayByUserIdResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	err := s.userService.IncrementInspectionsForToday(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, service.ErrInspectionLimitExceeded) {
			return nil, status.Error(codes.ResourceExhausted, "daily inspection limit exceeded")
		}
		s.log.Error("failed to increment inspections for today", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to increment inspections for today")
	}

	return &pb.IncrementInspectionsForTodayByUserIdResponse{}, nil
}

func (s *serverAPI) DecrementInspectionsForTodayByUserId(ctx context.Context, req *pb.DecrementInspectionsForTodayByUserIdRequest) (*pb.DecrementInspectionsForTodayByUserIdResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	err := s.userService.DecrementInspectionsForToday(ctx, req.UserId)
	if err != nil {
		s.log.Error("failed to decrement inspections for today", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to decrement inspections for today")
	}

	return &pb.DecrementInspectionsForTodayByUserIdResponse{}, nil
}

func (s *serverAPI) CheckInspectionLimit(ctx context.Context, req *pb.CheckInspectionLimitRequest) (*pb.CheckInspectionLimitResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	inspectionsLeft, err := s.userService.CheckInspectionLimit(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, service.ErrInspectionLimitExceeded) {
			return nil, status.Error(codes.ResourceExhausted, "лимит исчерпан")
		}
		s.log.Error("failed to check inspection limit", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to check inspection limit")
	}

	return &pb.CheckInspectionLimitResponse{
		InspectionsLeft: uint32(inspectionsLeft),
	}, nil
}

func (s *serverAPI) ChangeUserRole(ctx context.Context, req *pb.ChangeUserRoleRequest) (*pb.ChangeUserRoleResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	err := s.userService.ChangeUserRole(ctx, req.UserId, req.IsAdmin)
	if err != nil {
		s.log.Error("failed to change user role", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to change user role")
	}

	roleStr := "user"
	if req.IsAdmin {
		roleStr = "admin"
	}

	return &pb.ChangeUserRoleResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully changed role for user %s to %s", req.UserId, roleStr),
	}, nil
}

func (s *serverAPI) Recovery(ctx context.Context, req *pb.RecoveryRequest) (*pb.RecoveryResponse, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	err := s.userService.Recovery(ctx, req.Email)
	if err != nil {
		s.log.Error("failed to recover account", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to recover account")
	}

	return &pb.RecoveryResponse{
		Success: true,
		Message: "Recovery email sent successfully. Please check your email for new credentials.",
	}, nil
}

//func (s *serverAPI) mustEmbedUnimplementedUserServiceServer() {
//	s.log.Error("GetLoginById not implemented")
//}

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
