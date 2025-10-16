package server

import (
	"fmt"
	"log/slog"
	"net"
	"time"

	"repairCopilotBot/search-bot/internal/app/grpc/handlers"
	chatservice "repairCopilotBot/search-bot/internal/service/chat"
	userservice "repairCopilotBot/search-bot/internal/service/user"
	pb "repairCopilotBot/search-bot/pkg/search-bot/api/proto/search/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type Config struct {
	Port string `yaml:"port" env-default:":50053"`
}

type ChatGRPCServer struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       string
}

func NewChatGRPCServer(
	log *slog.Logger,
	chatService *chatservice.ChatService,
	userService *userservice.UserService,
	config *Config,
) *ChatGRPCServer {
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

	chatHandlers := handlers.NewChatHandlers(log, chatService)
	userHandlers := handlers.NewUserHandlers(log, userService)

	pb.RegisterChatServiceServer(gRPCServer, chatHandlers)
	pb.RegisterUserServiceServer(gRPCServer, userHandlers)

	return &ChatGRPCServer{
		log:        log,
		gRPCServer: gRPCServer,
		port:       config.Port,
	}
}

func (s *ChatGRPCServer) MustRun() {
	if err := s.Run(); err != nil {
		panic(err)
	}
}

func (s *ChatGRPCServer) Run() error {
	const op = "grpc.server.Run"

	log := s.log.With(slog.String("op", op), slog.String("port", s.port))

	l, err := net.Listen("tcp", s.port)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("grpc server started", slog.String("addr", l.Addr().String()))

	if err := s.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *ChatGRPCServer) Stop() {
	const op = "grpc.server.Stop"

	s.log.With(slog.String("op", op)).Info("stopping gRPC server")

	s.gRPCServer.GracefulStop()
}
