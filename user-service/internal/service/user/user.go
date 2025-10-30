package userservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"repairCopilotBot/user-service/internal/domain/models"
	"repairCopilotBot/user-service/internal/pkg/logger/sl"
	"repairCopilotBot/user-service/internal/repository"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/resend/resend-go/v2"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	log         *slog.Logger
	usrSaver    UserSaver
	usrProvider UserProvider
	mailToken   string
	mailDomen   string
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
		inspectionsLeftForToday int,
		inspectionsCount int,
		errorFeedbacksCount int,
		isConfirmed bool,
		confirmationCode string,
	) error
}

type UserProvider interface {
	User(ctx context.Context, userID uuid.UUID) (*models.User, error)
	LoginById(ctx context.Context, uid string) (string, error)
	GetAllUsers(ctx context.Context) ([]UserShortInfo, error)
	//GetUserInfo(ctx context.Context, userID string) (UserDetailedInfo, error)
	GetUserDetailsById(ctx context.Context, userID string) (*UserFullDetails, error)
	GetUserIDByLogin(ctx context.Context, login string) (uuid.UUID, error)
	GetUserAuthDataByLogin(ctx context.Context, login string) (*models.User, error)
	UpdateInspectionsPerDay(ctx context.Context, userID string, inspectionsPerDay int) (int64, error)
	UpdateInspectionsLeftForToday(ctx context.Context, userID string, inspectionsLeftForToday int) (int64, error)
	GetFullNamesById(ctx context.Context, ids []string) (map[string]FullName, error)
	UpdateLastVisit(ctx context.Context, userID string) error
	GetConfirmationCodeByUserId(ctx context.Context, userID uuid.UUID) (string, error)
	UpdateConfirmStatusByUserId(ctx context.Context, userID uuid.UUID, status bool) error
	IncrementInspectionsForToday(ctx context.Context, userID string) error
	DecrementInspectionsForToday(ctx context.Context, userID string) error
	GetInspectionsLeftForToday(ctx context.Context, userID string) (int, error)
	IncrementInspectionsLeftForToday(ctx context.Context, userID string) error
	DecrementInspectionsLeftForToday(ctx context.Context, userID string) error
	UpdateIsAdmin1(ctx context.Context, userID string, isAdmin bool) error
	GetUserIDByEmail(ctx context.Context, email string) (uuid.UUID, error)
	UpdateLoginAndPassword(ctx context.Context, userID uuid.UUID, login string, passHash []byte) error
}

func New(
	log *slog.Logger,
	userSaver UserSaver,
	userProvider UserProvider,
	mailToken string,
	mailDomen string,
) *User {
	return &User{
		usrSaver:    userSaver,
		usrProvider: userProvider,
		mailToken:   mailToken,
		mailDomen:   mailDomen,
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
		3,
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

type UserAuthData struct {
	ID       uuid.UUID
	PassHash []byte
	IsAdmin1 bool
	IsAdmin2 bool
}

func (u *User) Login(ctx context.Context, login string, password string) (*models.User, error) {
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

			return nil, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}

		u.log.Error("failed to get user", sl.Err(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(authData.PassHash, []byte(password)); err != nil {
		u.log.Info("invalid credentials", sl.Err(err))

		return nil, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	log.Info("user logged in successfully")

	return authData, nil
}

func (u *User) sendConfirmationEmail(email, confirmationCode string) error {
	//err := mailerclient.SendMailViaMailer(context.Background(), email, fmt.Sprintf("Ваш код подтверждения: %s\n\nИспользуйте этот код для завершения регистрации.", confirmationCode))
	apiKey := u.mailToken
	if u.mailToken == "" {
		apiKey = "re_bSW5sxCn_2wqChs3bGbexf7FM69Updray"
	}

	mailDomen := u.mailDomen
	if u.mailDomen == "" {
		mailDomen = "mail.iamil.ru"
	}

	client := resend.NewClient(apiKey)

	params := &resend.SendEmailRequest{
		From:    "intbis@" + mailDomen,
		To:      []string{email},
		Subject: "Код подтверждения регистрации",
		Html:    "<p>" + fmt.Sprintf("Ваш код подтверждения: %s\n\nИспользуйте этот код для завершения регистрации на intbis.ru.", confirmationCode) + "</p>",
	}

	_, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	return nil
	//// Конфигурация SMTP для Gmail
	//smtpHost := "smtp.gmail.com"
	//smtpPort := 465 // Используем SSL порт вместо TLS 587
	//from := "ivan2011avatar@gmail.com"
	//password := "tsep nuqs bmvy dcbr"
	//
	//// Создаем письмо
	//m := gomail.NewMessage()
	//m.SetHeader("From", from)
	//m.SetHeader("To", email)
	//m.SetHeader("Subject", "Код подтверждения регистрации")
	//
	//body := fmt.Sprintf("Ваш код подтверждения: %s\n\nИспользуйте этот код для завершения регистрации.", confirmationCode)
	//m.SetBody("text/plain", body)
	//
	//// Настраиваем SMTP диалер с SSL (порт 465)
	//d := gomail.NewDialer(smtpHost, smtpPort, from, password)
	//
	//// Отправка письма
	//if err := d.DialAndSend(m); err != nil {
	//	return fmt.Errorf("failed to send email: %w", err)
	//}
	//
	//return nil
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

type UserShortInfo struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
	Email     string
	IsAdmin1  bool
	IsAdmin2  bool
}

func (u *User) GetAllUsers(ctx context.Context) ([]UserShortInfo, error) {
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

type UserFullDetails struct {
	ID        uuid.UUID
	Login     string
	Name      string
	Surname   string
	Email     string
	IsAdmin1  bool
	IsAdmin2  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (u *User) GetUserDetailsById(ctx context.Context, userID string) (*UserFullDetails, error) {
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

	rowsAffectedInspectionsPerDay, err := u.usrProvider.UpdateInspectionsPerDay(ctx, userID, inspectionsPerDay)
	if err != nil {
		log.Error("failed to update inspections_per_day", sl.Err(err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	_, err = u.usrProvider.UpdateInspectionsLeftForToday(ctx, userID, inspectionsPerDay)
	if err != nil {
		log.Error("failed to update inspections_per_day", sl.Err(err))
	}

	//if userID != "" {
	//	inspectionsLeftForToday
	//}

	log.Info("inspections_per_day updated successfully", slog.Int64("rowsAffected", rowsAffectedInspectionsPerDay))

	return rowsAffectedInspectionsPerDay, nil
}

type FullName struct {
	FirstName string
	LastName  string
}

func (u *User) GetFullNamesById(ctx context.Context, ids []string) (map[string]FullName, error) {
	const op = "User.GetFullNamesById"

	log := u.log.With(
		slog.String("op", op),
		slog.Int("ids_count", len(ids)),
	)

	log.Info("getting full names by ids")

	fullNames, err := u.usrProvider.GetFullNamesById(ctx, ids)
	if err != nil {
		log.Error("failed to get full names by ids", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("full names retrieved successfully", slog.Int("result_count", len(fullNames)))

	return fullNames, nil
}

func (u *User) RegisterVisit(ctx context.Context, userID string) error {
	const op = "User.RegisterVisit"

	log := u.log.With(
		slog.String("op", op),
		slog.String("userID", userID),
	)

	log.Info("registering user visit")

	err := u.usrProvider.UpdateLastVisit(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			u.log.Warn("user not found", sl.Err(err))
			return fmt.Errorf("%s: user not found", op)
		}

		u.log.Error("failed to update last visit", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user visit registered successfully")

	return nil
}

func (u *User) ConfirmEmail(ctx context.Context, userID string, codeReq string) error {
	const op = "User.ConfirmEmail"

	log := u.log.With(
		slog.String("op", op),
		slog.String("userID", userID),
	)

	log.Info("confirm email")

	code, err := u.usrProvider.GetConfirmationCodeByUserId(ctx, uuid.MustParse(userID))
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			u.log.Warn("user not found", sl.Err(err))
		}
		u.log.Error("failed to get user confirmation code", sl.Err(err))
	}

	if codeReq != code {
		u.log.Warn("user confirmation code mismatch")
		return errors.New("confirmation code mismatch")
	}

	err = u.usrProvider.UpdateConfirmStatusByUserId(ctx, uuid.MustParse(userID), true)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			u.log.Warn("user not found", sl.Err(err))
		}
		u.log.Error("failed to update user confirmation", sl.Err(err))
	}

	log.Info("user confirmation code retrieved successfully")

	return nil
}

var (
	ErrInspectionLimitExceeded = errors.New("daily inspection limit exceeded")
)

func (u *User) IncrementInspectionsForToday(ctx context.Context, userID string) error {
	const op = "User.IncrementInspectionsForToday"

	log := u.log.With(
		slog.String("op", op),
		slog.String("userID", userID),
	)

	log.Info("incrementing inspections for today")

	inspectionsLeft, err := u.usrProvider.GetInspectionsLeftForToday(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			u.log.Warn("user not found", sl.Err(err))
			return fmt.Errorf("%s: user not found", op)
		}
		u.log.Error("failed to get inspections left for today", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	if inspectionsLeft == 0 {
		u.log.Warn("daily inspection limit exceeded")
		return fmt.Errorf("%s: %w", op, ErrInspectionLimitExceeded)
	}

	//user, err := u.usrProvider.User(ctx, uuid.MustParse(userID))
	//if err != nil {
	//	if errors.Is(err, repository.ErrUserNotFound) {
	//		u.log.Warn("user not found", sl.Err(err))
	//		return fmt.Errorf("%s: user not found", op)
	//	}
	//	u.log.Error("failed to get user info", sl.Err(err))
	//	return fmt.Errorf("%s: %w", op, err)
	//}

	//if user.InspectionsForToday >= user.InspectionsPerDay {
	//	u.log.Warn("daily inspection limit exceeded",
	//		slog.Int("current", user.InspectionsForToday),
	//		slog.Int("limit", user.InspectionsPerDay))
	//	return fmt.Errorf("%s: %w", op, ErrInspectionLimitExceeded)
	//}

	err = u.usrProvider.IncrementInspectionsForToday(ctx, userID)
	if err != nil {
		log.Error("failed to increment inspections for today", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	err = u.usrProvider.DecrementInspectionsLeftForToday(ctx, userID)
	if err != nil {
		log.Error("failed to decrement inspections left for today", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("inspections for today incremented successfully")
	return nil
}

func (u *User) DecrementInspectionsForToday(ctx context.Context, userID string) error {
	const op = "User.DecrementInspectionsForToday"

	log := u.log.With(
		slog.String("op", op),
		slog.String("userID", userID),
	)

	log.Info("decrementing inspections for today")

	err := u.usrProvider.DecrementInspectionsForToday(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			u.log.Warn("user not found", sl.Err(err))
			return fmt.Errorf("%s: user not found", op)
		}
		log.Error("failed to decrement inspections for today", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	err = u.usrProvider.IncrementInspectionsLeftForToday(ctx, userID)
	if err != nil {
		log.Error("failed to increment inspections left for today", sl.Err(err))
	}

	log.Info("inspections for today decremented successfully")
	return nil
}

func (u *User) CheckInspectionLimit(ctx context.Context, userID string) (int, error) {
	const op = "User.CheckInspectionLimit"

	log := u.log.With(
		slog.String("op", op),
		slog.String("userID", userID),
	)

	log.Info("checking inspection limit")

	inspectionsLeft, err := u.usrProvider.GetInspectionsLeftForToday(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			u.log.Warn("user not found", sl.Err(err))
			return 0, fmt.Errorf("%s: user not found", op)
		}
		u.log.Error("failed to get inspections left for today", sl.Err(err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	if inspectionsLeft <= 0 {
		u.log.Warn("inspection limit exhausted", slog.Int("inspectionsLeft", inspectionsLeft))
		return 0, fmt.Errorf("%s: %w", op, ErrInspectionLimitExceeded)
	}

	log.Info("inspection limit checked successfully", slog.Int("inspectionsLeft", inspectionsLeft))
	return inspectionsLeft, nil
}

func (u *User) ChangeUserRole(ctx context.Context, userID string, isAdmin bool) error {
	const op = "User.ChangeUserRole"

	log := u.log.With(
		slog.String("op", op),
		slog.String("userID", userID),
		slog.Bool("isAdmin", isAdmin),
	)

	log.Info("changing user role")

	err := u.usrProvider.UpdateIsAdmin1(ctx, userID, isAdmin)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			u.log.Warn("user not found", sl.Err(err))
			return fmt.Errorf("%s: user not found", op)
		}
		log.Error("failed to update user role", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user role changed successfully")
	return nil
}

// generateRandomPassword генерирует случайный пароль из английских букв и цифр длиной 10 символов
func generateRandomPassword() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const passwordLength = 10

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	password := make([]byte, passwordLength)
	for i := range password {
		password[i] = charset[r.Intn(len(charset))]
	}
	return string(password)
}

// sendRecoveryEmail отправляет письмо с новыми данными для входа
func (u *User) sendRecoveryEmail(email, login, password string) error {
	//err := mailerclient.SendMailViaMailer(context.Background(), email, fmt.Sprintf("Здравствуйте. Система сгенерировала Вам следующие данные для входа: логин - %s, пароль - %s.", login, password))
	apiKey := u.mailToken
	if u.mailToken == "" {
		apiKey = "re_bSW5sxCn_2wqChs3bGbexf7FM69Updray"
	}

	mailDomen := u.mailDomen
	if u.mailDomen == "" {
		mailDomen = "mail.iamil.ru"
	}

	client := resend.NewClient(apiKey)

	params := &resend.SendEmailRequest{
		From:    "intbis@" + mailDomen,
		To:      []string{email},
		Subject: "Восстановление данных для входа",
		Html:    "<p>" + fmt.Sprintf("Здравствуйте. Система сгенерировала Вам следующие данные для входа: логин - %s, пароль - %s", login, password) + "</p>",
	}

	_, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	return nil
	//// Конфигурация SMTP для Gmail
	//smtpHost := "smtp.gmail.com"
	//smtpPort := 465 // Используем SSL порт вместо TLS 587
	//from := "ivan2011avatar@gmail.com"
	//emailPassword := "tsep nuqs bmvy dcbr"
	//
	//// Создаем письмо
	//m := gomail.NewMessage()
	//m.SetHeader("From", from)
	//m.SetHeader("To", email)
	//m.SetHeader("Subject", "Восстановление данных для входа")
	//
	//body := fmt.Sprintf("Здравствуйте. Система сгенерировала Вам следующие данные для входа: логин - %s, пароль - %s.", login, password)
	//m.SetBody("text/plain", body)
	//
	//// Настраиваем SMTP диалер с SSL (порт 465)
	//d := gomail.NewDialer(smtpHost, smtpPort, from, emailPassword)
	//
	//// Отправка письма
	//if err := d.DialAndSend(m); err != nil {
	//	return fmt.Errorf("failed to send email: %w", err)
	//}
	//
	//return nil
}

// Recovery восстанавливает логин и пароль пользователя по email
func (u *User) Recovery(ctx context.Context, email string) error {
	const op = "User.Recovery"

	log := u.log.With(
		slog.String("op", op),
		slog.String("email", email),
	)

	log.Info("starting account recovery")

	// Получаем ID пользователя по email
	userID, err := u.usrProvider.GetUserIDByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			u.log.Warn("user not found by email", sl.Err(err))
			return fmt.Errorf("%s: user not found", op)
		}
		u.log.Error("failed to get user by email", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	// Получаем логин пользователя по id
	login, err := u.usrProvider.LoginById(ctx, userID.String())
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			u.log.Warn("user not found by email", sl.Err(err))
			return fmt.Errorf("%s: user not found", op)
		}
		u.log.Error("failed to get login by userID", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	// Генерируем новый пароль
	newPassword := generateRandomPassword()

	// Хешируем новый пароль
	passHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	// Обновляем логин и пароль в базе данных
	err = u.usrProvider.UpdateLoginAndPassword(ctx, userID, login, passHash)
	if err != nil {
		log.Error("failed to update login and password", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	err = u.usrProvider.UpdateConfirmStatusByUserId(ctx, userID, true)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			u.log.Warn("user not found", sl.Err(err))
		}
		u.log.Error("failed to update user confirmation", sl.Err(err))
	}

	// Отправляем письмо с новыми данными
	err = u.sendRecoveryEmail(email, login, newPassword)
	if err != nil {
		log.Error("failed to send recovery email", sl.Err(err))
		// Не возвращаем ошибку, так как данные уже обновлены в базе
	}

	log.Info("account recovery completed successfully")

	return nil
}
