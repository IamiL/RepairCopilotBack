package handlers

import (
	"context"
	"log/slog"

	userservice "repairCopilotBot/chat-bot/internal/service/user"
	pb "repairCopilotBot/chat-bot/pkg/chat-bot/api/proto/chat/v1"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserHandlers struct {
	log         *slog.Logger
	userService *userservice.UserService
	pb.UnimplementedUserServiceServer
}

func NewUserHandlers(log *slog.Logger, userService *userservice.UserService) *UserHandlers {
	return &UserHandlers{
		log:         log,
		userService: userService,
	}
}

func (h *UserHandlers) CreateNewUser(ctx context.Context, req *pb.CreateNewUserRequest) (*pb.CreateNewUserResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	err = h.userService.Create(ctx, userID)
	if err != nil {
		h.log.Error("failed to create new user", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to create new user")
	}

	return &pb.CreateNewUserResponse{}, nil
}
