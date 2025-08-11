package userservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"repairCopilotBot/user-service/internal/pkg/logger/sl"
	"repairCopilotBot/user-service/internal/repository"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	postgresUser "repairCopilotBot/user-service/internal/repository/postgres/user"
)

type User struct {
	log         *slog.Logger
	usrSaver    UserSaver
	usrProvider UserProvider
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
		isAdmin1 bool,
		isAdmin2 bool,
		uid uuid.UUID,
	) error
}

type UserProvider interface {
	User(ctx context.Context, login string) (uuid.UUID, []byte, bool, bool, error)
	LoginById(ctx context.Context, uid string) (string, error)
	GetAllUsers(ctx context.Context) ([]postgresUser.UserInfo, error)
}

func New(
	log *slog.Logger,
	userSaver UserSaver,
	userProvider UserProvider,
) *User {
	return &User{
		usrSaver:    userSaver,
		usrProvider: userProvider,
		log:         log,
	}
}

func (u *User) RegisterNewUser(ctx context.Context, login string, pass string) (uuid.UUID, error) {
	const op = "User.RegisterNewUser"

	log := u.log.With(
		slog.String("op", op),
		slog.String("login", login),
	)

	log.Info("registering user")

	_, _, _, _, err := u.usrProvider.User(ctx, login)
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

	err = u.usrSaver.SaveUser(ctx, login, passHash, false, false, uid)
	if err != nil {
		log.Error("failed to save user", sl.Err(err))

		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return uid, nil
}

func (u *User) Login(ctx context.Context, login string, password string) (uuid.UUID, bool, bool, error) {
	const op = "User.Login"

	log := u.log.With(
		slog.String("op", op),
		slog.String("username", login),
	)

	log.Info("attempting to login user")

	uid, passHash, isAdmin1, isAdmin2, err := u.usrProvider.User(ctx, login)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			u.log.Warn("user not found", sl.Err(err))

			return uuid.Nil, false, false, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}

		u.log.Error("failed to get user", sl.Err(err))

		return uuid.Nil, false, false, fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(passHash, []byte(password)); err != nil {
		u.log.Info("invalid credentials", sl.Err(err))

		return uuid.Nil, false, false, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	log.Info("user logged in successfully")

	return uid, isAdmin1, isAdmin2, nil
}

func (u *User) GetLoginById(ctx context.Context, userId string) (string, error) {
	const op = "User.GetLoginById"

	log := u.log.With(
		slog.String("op", op),
		slog.String("userId", userId),
	)

	log.Info("getting login by user id")

	login, err := u.usrProvider.LoginById(ctx, userId)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			u.log.Warn("user not found", sl.Err(err))
			return "", fmt.Errorf("%s: user not found", op)
		}

		u.log.Error("failed to get login by id", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	log.Info("login retrieved successfully")

	return login, nil
}

func (u *User) GetAllUsers(ctx context.Context) ([]postgresUser.UserInfo, error) {
	const op = "User.GetAllUsers"

	log := u.log.With(
		slog.String("op", op),
	)

	log.Info("getting all users")

	users, err := u.usrProvider.GetAllUsers(ctx)
	if err != nil {
		u.log.Error("failed to get all users", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("all users retrieved successfully", slog.Int("count", len(users)))

	return users, nil
}
