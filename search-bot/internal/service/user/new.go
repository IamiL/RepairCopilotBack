package userservice

import (
	"log/slog"
)

func New(
	logger *slog.Logger,
	userSaver UserSaver,
) *UserService {
	return &UserService{
		log:      logger,
		usrSaver: userSaver,
	}
}
