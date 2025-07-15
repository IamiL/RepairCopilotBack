package userservice

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"repairCopilotBot/user-service/internal/pkg/logger/sl"
	"repairCopilotBot/user-service/internal/repository"
	"time"
)

type User struct {
	log         *slog.Logger
	usrSaver    UserSaver
	usrProvider UserProvider
	tokenTTL    time.Duration
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserAlreadyExists  = errors.New("user already exists")
)

type UserSaver interface {
	SaveUser(
		ctx context.Context,
		login string,
		passHash []byte,
		uid uuid.UUID,
	) (err error)
}

type UserProvider interface {
	User(ctx context.Context, login string) (uuid.UUID, []byte, error)
}

func New(
	log *slog.Logger,
	userSaver UserSaver,
	userProvider UserProvider,
	tokenTTL time.Duration,
) *User {
	return &User{
		usrSaver:    userSaver,
		usrProvider: userProvider,
		log:         log,
		tokenTTL:    tokenTTL,
	}
}

func (u *User) RegisterNewUser(ctx context.Context, login string, pass string) (uuid.UUID, error) {
	const op = "User.RegisterNewUser"

	log := u.log.With(
		slog.String("op", op),
		slog.String("login", login),
	)

	log.Info("registering user")

	_, _, err := u.usrProvider.User(ctx, login)
	if err == nil {

		u.log.Error("user already exists", sl.Err(err))

		return uuid.Nil, ErrUserAlreadyExists
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", sl.Err(err))

		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	uid, err := uuid.NewUUID()
	if err != nil {
		log.Error("failed to generate uuid", sl.Err(err))
	}

	err = u.usrSaver.SaveUser(ctx, login, passHash, uid)
	if err != nil {
		log.Error("failed to save user", sl.Err(err))

		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return uid, nil
}

func (u *User) Login(ctx context.Context, login string, password string) (uuid.UUID, error) {
	const op = "User.Login"

	log := u.log.With(
		slog.String("op", op),
		slog.String("username", login),
	)

	log.Info("attempting to login user")

	uid, passHash, err := u.usrProvider.User(ctx, login)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			u.log.Warn("user not found", sl.Err(err))

			return uuid.Nil, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}

		u.log.Error("failed to get user", sl.Err(err))

		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(passHash, []byte(password)); err != nil {
		u.log.Info("invalid credentials", sl.Err(err))

		return uuid.Nil, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	log.Info("user logged in successfully")

	return uid, nil
}
