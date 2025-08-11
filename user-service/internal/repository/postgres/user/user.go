package postgresUser

import (
	"context"
	"errors"
	"fmt"
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
	login string,
	passHash []byte,
	isAdmin1 bool,
	isAdmin2 bool,
	uid uuid.UUID,
) error {
	_, err := s.db.Exec(
		ctx,
		"INSERT INTO users(id, login, pass_hash, is_admin1, is_admin2, created_at, updated_at) VALUES($1, $2, $3, $4, $5, $6, $7)",
		uid,
		login,
		passHash,
		isAdmin1,
		isAdmin2,
		time.Now(),
		time.Now(),
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) User(ctx context.Context, login string) (
	uuid.UUID,
	[]byte,
	bool,
	bool,
	error,
) {
	query := `SELECT id, pass_hash, is_admin1, is_admin2 FROM users WHERE login = $1`

	var id string
	var passHash []byte
	var isAdmin1, isAdmin2 bool

	err := s.db.QueryRow(ctx, query, login).Scan(&id, &passHash, &isAdmin1, &isAdmin2)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, nil, false, false, repo.ErrUserNotFound
		}
		return uuid.Nil, nil, false, false, fmt.Errorf("database error: %w", err)
	}

	uid, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, nil, false, false, fmt.Errorf("database error: %w", err)
	}

	return uid, passHash, isAdmin1, isAdmin2, nil
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
	IsAdmin1 bool
	IsAdmin2 bool
}

func (s *Storage) GetAllUsers(ctx context.Context) ([]UserInfo, error) {
	query := `SELECT id, login, is_admin1, is_admin2 FROM users ORDER BY login`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()

	var users []UserInfo
	for rows.Next() {
		var user UserInfo
		err := rows.Scan(&user.ID, &user.Login, &user.IsAdmin1, &user.IsAdmin2)
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
	IsAdmin1  bool
	IsAdmin2  bool
	CreatedAt time.Time
}

func (s *Storage) GetUserInfo(ctx context.Context, userID string) (*UserDetailedInfo, error) {
	query := `SELECT id, login, is_admin1, is_admin2, created_at FROM users WHERE id = $1`

	var user UserDetailedInfo
	err := s.db.QueryRow(ctx, query, userID).Scan(&user.ID, &user.Login, &user.IsAdmin1, &user.IsAdmin2, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrUserNotFound
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &user, nil
}
