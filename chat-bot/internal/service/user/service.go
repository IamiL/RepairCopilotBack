package userservice

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type UserService struct {
	log      *slog.Logger
	usrSaver UserSaver
}

type UserSaver interface {
	CreateUser(ctx context.Context, id uuid.UUID, messagesPerDay int, messagesLeftForToday int, createdAt time.Time, updatedAt time.Time) error
}
