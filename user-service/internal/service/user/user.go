package userservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/smtp"
	"repairCopilotBot/user-service/internal/domain/models"
	"repairCopilotBot/user-service/internal/pkg/logger/sl"
	"repairCopilotBot/user-service/internal/repository"
	"strconv"
	"time"

	postgresUser "repairCopilotBot/user-service/internal/repository/postgres/user"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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
		uid uuid.UUID,
		login string,
		passHash []byte,
		firstName string,
		lastName string,
		email string,
		isAdmin1 bool,
		isAdmin2 bool,
		createdAt time.Time,
		updatedAt time.Time,
		lastVisitAt time.Time,
		inspectionsPerDay int,
		inspectionsForToday int,
		inspectionsCount int,
		errorFeedbacksCount int,
		isConfirmed bool,
		confirmationCode string,
	) error
}

type UserProvider interface {
	User(ctx context.Context, userID uuid.UUID) (*models.User, error)
	LoginById(ctx context.Context, uid string) (string, error)
	GetAllUsers(ctx context.Context) ([]postgresUser.UserInfo, error)
	GetUserInfo(ctx context.Context, userID string) (*postgresUser.UserDetailedInfo, error)
	GetUserDetailsById(ctx context.Context, userID string) (*postgresUser.UserFullDetails, error)
	GetUserIDByLogin(ctx context.Context, login string) (uuid.UUID, error)
	GetUserAuthDataByLogin(ctx context.Context, login string) (*postgresUser.UserAuthData, error)
	UpdateInspectionsPerDay(ctx context.Context, userID string, inspectionsPerDay int) (int64, error)
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

func (u *User) RegisterNewUser(ctx context.Context, login string, pass string, firstName string, lastName string, email string) (uuid.UUID, error) {
	const op = "User.RegisterNewUser"

	log := u.log.With(
		slog.String("op", op),
		slog.String("login", login),
		slog.String("firstName", firstName),
		slog.String("lastName", lastName),
	)

	log.Info("registering user")

	_, err := u.usrProvider.GetUserIDByLogin(ctx, login)
	if err == nil {
		u.log.Error("user already exists")

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

	// Генерируем 6-значный код подтверждения
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	confirmationCode := strconv.Itoa(r.Intn(900000) + 100000)

	err = u.usrSaver.SaveUser(
		ctx,
		uid,
		login,
		passHash,
		firstName,
		lastName,
		email,
		false,
		false,
		time.Now(),
		time.Now(),
		time.Now(),
		3,
		0,
		0,
		0,
		false,
		confirmationCode,
	)
	if err != nil {
		log.Error("failed to save user", sl.Err(err))

		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	// Отправляем код подтверждения на почту
	err = u.sendConfirmationEmail(email, confirmationCode)
	if err != nil {
		log.Error("failed to send confirmation email", sl.Err(err))
		// Не возвращаем ошибку, так как пользователь уже создан
	}

	log.Info("user registered successfully, confirmation code sent")

	return uid, nil
}

func (u *User) Login(ctx context.Context, login string, password string) (uuid.UUID, bool, bool, error) {
	const op = "User.Login"

	log := u.log.With(
		slog.String("op", op),
		slog.String("username", login),
	)

	log.Info("attempting to login user")

	authData, err := u.usrProvider.GetUserAuthDataByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			u.log.Warn("user not found", sl.Err(err))

			return uuid.Nil, false, false, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}

		u.log.Error("failed to get user", sl.Err(err))

		return uuid.Nil, false, false, fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(authData.PassHash, []byte(password)); err != nil {
		u.log.Info("invalid credentials", sl.Err(err))

		return uuid.Nil, false, false, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	log.Info("user logged in successfully")

	return authData.ID, authData.IsAdmin1, authData.IsAdmin2, nil
}

func (u *User) sendConfirmationEmail(email, confirmationCode string) error {
	// Конфигурация SMTP для Gmail
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	auth := smtp.PlainAuth("", "ivan2011avatar@gmail.com", "tsep nuqs bmvy dcbr", smtpHost)

	// Адрес отправителя
	from := "ivan2011avatar@gmail.com"

	// Тело письма
	subject := "Код подтверждения регистрации"
	body := fmt.Sprintf("Ваш код подтверждения: %s\n\nИспользуйте этот код для завершения регистрации.", confirmationCode)
	message := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s\r\n", email, subject, body))

	// Отправка письма через Gmail
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{email}, message)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
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

func (u *User) User(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	const op = "User.User"

	log := u.log.With(
		slog.String("op", op),
		slog.String("userID", userID.String()),
	)

	log.Info("getting user info")

	user, err := u.usrProvider.User(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			u.log.Warn("user not found", sl.Err(err))
			return nil, fmt.Errorf("%s: user not found", op)
		}

		u.log.Error("failed to get user info", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user info retrieved successfully", slog.String("login", user.Login))

	return user, nil
}

//func (u *User) GetUserByLogin(ctx context.Context, login string) (uuid.UUID, string, string, string, string, bool, bool, error) {
//	const op = "User.GetUserByLogin"
//
//	log := u.log.With(
//		slog.String("op", op),
//		slog.String("login", login),
//	)
//
//	log.Info("getting user by login")
//
//	uid, _, name, surname, email, isAdmin1, isAdmin2, err := u.usrProvider.User(ctx, login)
//	if err != nil {
//		if errors.Is(err, repository.ErrUserNotFound) {
//			u.log.Warn("user not found", sl.Err(err))
//			return uuid.Nil, "", "", "", "", false, false, fmt.Errorf("%s: user not found", op)
//		}
//
//		u.log.Error("failed to get user by login", sl.Err(err))
//		return uuid.Nil, "", "", "", "", false, false, fmt.Errorf("%s: %w", op, err)
//	}
//
//	log.Info("user retrieved successfully by login")
//
//	return uid, login, name, surname, email, isAdmin1, isAdmin2, nil
//}

func (u *User) GetUserDetailsById(ctx context.Context, userID string) (*postgresUser.UserFullDetails, error) {
	const op = "User.GetUserDetailsById"

	log := u.log.With(
		slog.String("op", op),
		slog.String("userID", userID),
	)

	log.Info("getting user details by id")

	userDetails, err := u.usrProvider.GetUserDetailsById(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			u.log.Warn("user not found", sl.Err(err))
			return nil, fmt.Errorf("%s: user not found", op)
		}

		u.log.Error("failed to get user details by id", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user details retrieved successfully", slog.String("login", userDetails.Login))

	return userDetails, nil
}

func (u *User) UpdateInspectionsPerDay(ctx context.Context, userID string, inspectionsPerDay int) (int64, error) {
	const op = "User.UpdateInspectionsPerDay"

	log := u.log.With(
		slog.String("op", op),
		slog.String("userID", userID),
		slog.Int("inspectionsPerDay", inspectionsPerDay),
	)

	if userID == "" {
		log.Info("updating inspections_per_day for all users")
	} else {
		log.Info("updating inspections_per_day for specific user")
	}

	rowsAffected, err := u.usrProvider.UpdateInspectionsPerDay(ctx, userID, inspectionsPerDay)
	if err != nil {
		log.Error("failed to update inspections_per_day", sl.Err(err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("inspections_per_day updated successfully", slog.Int64("rowsAffected", rowsAffected))

	return rowsAffected, nil
}
