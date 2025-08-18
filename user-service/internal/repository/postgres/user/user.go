package postgresUser

import (
	"context"
	"errors"
	"fmt"
	"repairCopilotBot/user-service/internal/domain/models"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	repo "repairCopilotBot/user-service/internal/repository"
)

type Storage struct {
	db *pgxpool.Pool
}

func New(pool *pgxpool.Pool) (*Storage, error) {
	return &Storage{db: pool}, nil
}

func (s *Storage) SaveUser(
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
) error {
	args := pgx.NamedArgs{
		"id":                  uid,
		"login":               login,
		"passHash":            passHash,
		"firstName":           firstName,
		"lastName":            lastName,
		"email":               email,
		"isAdmin1":            isAdmin1,
		"isAdmin2":            isAdmin2,
		"createdAt":           createdAt,
		"updatedAt":           updatedAt,
		"lastVisitAt":         lastVisitAt,
		"inspectionsPerDay":   inspectionsPerDay,
		"inspectionsForToday": inspectionsForToday,
		"inspectionsCount":    inspectionsCount,
		"errorFeedbacksCount": errorFeedbacksCount,
		"isConfirmed":         isConfirmed,
		"confirmationCode":    confirmationCode,
	}

	_, err := s.db.Exec(
		ctx,
		`INSERT INTO users(
                  id,
                  login,
                  pass_hash,
                  first_name,
                  last_name,
                  email,
                  is_admin1,
                  is_admin2,
                  created_at,
                  updated_at,
                  last_visit_at, 
                inspections_per_day,
                  inspections_for_today,
                  inspections_count,
                  error_feedbacks_count,
                  is_confirmed,
                  confirmation_code
                  ) VALUES(
                           @id,
                           @login,
                           @passHash,
                           @firstName,
                           @lastName,
                           @email,
                           @isAdmin1,
                           @isAdmin2,
                           @createdAt,
                           @updatedAt,
                           @lastVisitAt,
                           @inspectionsPerDay,
                           @inspectionsForToday,
                           @inspectionsCount,
                           @errorFeedbacksCount,
                           @isConfirmed,
                           @confirmationCode)`,
		args,
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) User(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	query := `SELECT login, first_name, last_name, email, is_admin1, is_admin2, created_at, last_visit_at, inspections_per_day, inspections_for_today, inspections_count, error_feedbacks_count FROM users WHERE id = $1`

	var user models.User

	err := s.db.QueryRow(ctx, query, userID.String()).Scan(&user.Login, &user.FirstName, &user.LastName, &user.Email, &user.IsAdmin1, &user.IsAdmin2, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrUserNotFound
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &user, nil
}

func (s *Storage) EditUser(
	ctx context.Context,
	uid string,
	login string,
	passHash string,
	isAdmin1 bool,
	isAdmin2 bool,
) error {
	query := `UPDATE users SET login = $1, pass_hash = $2, is_admin1 = $3, is_admin2 = $4 WHERE id = $5`

	_, err := s.db.Exec(ctx, query, login, passHash, isAdmin1, isAdmin2, uid)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) LoginById(ctx context.Context, uid string) (string, error) {
	query := `SELECT login FROM users WHERE id = $1`

	var login string

	err := s.db.QueryRow(ctx, query, uid).Scan(&login)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", repo.ErrUserNotFound
		}
		return "", fmt.Errorf("database error: %w", err)
	}

	return login, nil
}

type UserInfo struct {
	ID       string
	Login    string
	Name     string
	Surname  string
	Email    string
	IsAdmin1 bool
	IsAdmin2 bool
}

func (s *Storage) GetAllUsers(ctx context.Context) ([]UserInfo, error) {
	query := `SELECT id, login, name, surname, email, is_admin1, is_admin2 FROM users ORDER BY login`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()

	var users []UserInfo
	for rows.Next() {
		var user UserInfo
		err := rows.Scan(&user.ID, &user.Login, &user.Name, &user.Surname, &user.Email, &user.IsAdmin1, &user.IsAdmin2)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return users, nil
}

type UserDetailedInfo struct {
	ID        string
	Login     string
	Name      string
	Surname   string
	Email     string
	IsAdmin1  bool
	IsAdmin2  bool
	CreatedAt time.Time
}

type UserFullDetails struct {
	ID        string
	Login     string
	Name      string
	Surname   string
	Email     string
	IsAdmin1  bool
	IsAdmin2  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *Storage) GetUserInfo(ctx context.Context, userID string) (*UserDetailedInfo, error) {
	query := `SELECT id, login, name, surname, email, is_admin1, is_admin2, created_at FROM users WHERE id = $1`

	var user UserDetailedInfo
	err := s.db.QueryRow(ctx, query, userID).Scan(&user.ID, &user.Login, &user.Name, &user.Surname, &user.Email, &user.IsAdmin1, &user.IsAdmin2, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrUserNotFound
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &user, nil
}

func (s *Storage) GetUserDetailsById(ctx context.Context, userID string) (*UserFullDetails, error) {
	query := `SELECT id, login, name, surname, email, is_admin1, is_admin2, created_at, updated_at FROM users WHERE id = $1`

	var user UserFullDetails
	err := s.db.QueryRow(ctx, query, userID).Scan(&user.ID, &user.Login, &user.Name, &user.Surname, &user.Email, &user.IsAdmin1, &user.IsAdmin2, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrUserNotFound
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &user, nil
}

func (s *Storage) GetUserIDByLogin(ctx context.Context, login string) (uuid.UUID, error) {
	query := `SELECT id FROM users WHERE login = $1`

	var userID string
	err := s.db.QueryRow(ctx, query, login).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, repo.ErrUserNotFound
		}
		return uuid.Nil, fmt.Errorf("database error: %w", err)
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID format: %w", err)
	}

	return uid, nil
}

type UserAuthData struct {
	ID       uuid.UUID
	PassHash []byte
	IsAdmin1 bool
	IsAdmin2 bool
}

func (s *Storage) GetUserAuthDataByLogin(ctx context.Context, login string) (*UserAuthData, error) {
	query := `SELECT id, pass_hash, is_admin1, is_admin2 FROM users WHERE login = $1`

	var authData UserAuthData
	err := s.db.QueryRow(ctx, query, login).Scan(&authData.ID, &authData.PassHash, &authData.IsAdmin1, &authData.IsAdmin2)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrUserNotFound
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &authData, nil
}
