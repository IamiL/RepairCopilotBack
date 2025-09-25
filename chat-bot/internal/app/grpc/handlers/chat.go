package handlers

import (
	"context"
	"fmt"
	"log/slog"
	chatmodel "repairCopilotBot/chat-bot/internal/domain/model/chat"

	chatservice "repairCopilotBot/chat-bot/internal/service/chat"
	pb "repairCopilotBot/chat-bot/pkg/chat-bot/api/proto/chat/v1"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ChatHandlers struct {
	log         *slog.Logger
	chatService *chatservice.ChatService
	pb.UnimplementedChatServiceServer
}

func NewChatHandlers(log *slog.Logger, chatService *chatservice.ChatService) *ChatHandlers {
	return &ChatHandlers{
		log:         log,
		chatService: chatService,
	}
}

func (h *ChatHandlers) CreateNewMessage(ctx context.Context, req *pb.CreateNewMessageRequest) (*pb.CreateNewMessageResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Message == "" {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	var chatID uuid.UUID
	if req.ChatId != nil && *req.ChatId != "" {
		chatID, err = uuid.Parse(*req.ChatId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid chat_id format")
		}
	}
	fmt.Println("userID: " + userID.String())
	responseMessage, chatIDStr, err := h.chatService.NewMessage(ctx, userID, chatID, req.Message)
	if err != nil {
		h.log.Error("failed to create new message", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to create new message")
	}

	chatIdRespStr := chatIDStr.String()

	return &pb.CreateNewMessageResponse{
		ChatId:  &chatIdRespStr,
		Message: responseMessage,
	}, nil
}

func (h *ChatHandlers) GetChats(ctx context.Context, req *pb.GetChatsRequest) (*pb.GetChatsResponse, error) {
	var userID uuid.UUID
	var err error

	if req.UserID != nil && *req.UserID != "" {
		userID, err = uuid.Parse(*req.UserID)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
		}
	}

	var (
		chats           []chatmodel.Chat
		chatsServiceErr error
	)

	if userID != uuid.Nil {
		chats, chatsServiceErr = h.chatService.ChatsForUser(ctx, userID)
	} else {
		chats, chatsServiceErr = h.chatService.Chats(ctx)
	}

	if chatsServiceErr != nil {
		h.log.Error("failed to get chats", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to get chats")
	}

	var pbChats []*pb.Chat
	for _, chat := range chats {
		pbChats = append(pbChats, &pb.Chat{
			Id:            chat.Id.String(),
			UserId:        chat.UserID.String(),
			CreatedAt:     timestamppb.New(chat.CreatedAt),
			MessagesCount: uint32(chat.MessagesCount),
			IsFinished:    chat.IsFinished,
			Conclusion:    chat.Conclusion,
			IsProcessing:  chat.IsProcessing,
			Enclosure:     uint32(chat.Enclosure),
		})
	}

	return &pb.GetChatsResponse{
		Chats: pbChats,
	}, nil
}

func (h *ChatHandlers) GetMessages(ctx context.Context, req *pb.GetMessagesRequest) (*pb.GetMessagesResponse, error) {
	if req.ChatId == "" {
		return nil, status.Error(codes.InvalidArgument, "chat_id is required")
	}

	chatID, err := uuid.Parse(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id format")
	}

	messages, err := h.chatService.Messages(ctx, chatID)
	if err != nil {
		h.log.Error("failed to get messages", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to get messages")
	}

	var pbMessages []*pb.Message
	for _, msg := range messages {
		pbMessages = append(pbMessages, &pb.Message{
			Id:        msg.Id.String(),
			Message:   msg.Content,
			Role:      msg.Role,
			CreatedAt: timestamppb.New(msg.CreatedAt),
		})
	}

	return &pb.GetMessagesResponse{
		Messages: pbMessages,
	}, nil
}

func (h *ChatHandlers) FinishChat(ctx context.Context, req *pb.FinishChatRequest) (*pb.FinishChatResponse, error) {
	if req.ChatId == "" {
		return nil, status.Error(codes.InvalidArgument, "chat_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	chatID, err := uuid.Parse(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id format")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id format")
	}

	msg, err := h.chatService.FinishChat(ctx, userID, chatID)
	if err != nil {
		h.log.Error("failed to finish chat", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to finish chat")
	}

	return &pb.FinishChatResponse{
		Msg: msg,
	}, nil
}
