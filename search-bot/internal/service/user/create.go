package userservice

import (
	"context"
	"errors"
	"log/slog"
	"repairCopilotBot/search-bot/internal/pkg/logger/sl"
	"time"

	"github.com/google/uuid"
)

func (u *UserService) Create(ctx context.Context, userID uuid.UUID) error {
	op := "userService.Create"
	log := u.log.With(
		slog.String("op", op),
		slog.String("userID", userID.String()),
	)

	log.Info("creating user")

	now := time.Now()

	err := u.usrSaver.CreateUser(ctx, userID, 100, 100, now, now)
	if err != nil {
		log.Error("failed to create user", sl.Err(err))
		return errors.New("internal server error")
	}

	return nil
}
