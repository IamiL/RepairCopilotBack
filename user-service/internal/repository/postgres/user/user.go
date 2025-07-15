package postgresUser

import (
	"context"
	"errors"
	"fmt"

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

func (s *Storage) NewUser(
	ctx context.Context,
	uid string,
	login string,
	passHash string,
) error {
	_, err := s.db.Exec(
		ctx,
		"INSERT INTO users(id, login, pass_hash) VALUES($1, $2, $3)",
		uid,
		login,
		passHash,
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) User(ctx context.Context, login string) (
	string,
	string,
	error,
) {
	query := `SELECT id, pass_hash FROM users WHERE login = $1`

	var id, passHash string

	err := s.db.QueryRow(ctx, query, login).Scan(&id, &passHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", repo.ErrUserNotFound
		}
		return "", "", fmt.Errorf("database error: %w", err)
	}

	return id, passHash, nil
}

func (s *Storage) EditUser(
	ctx context.Context,
	uid string,
	login string,
	passHash string,
) error {
	query := `UPDATE users SET login = $1, pass_hash = $2 WHERE id = $3`

	_, err := s.db.Exec(ctx, query, login, passHash, uid)
	if err != nil {
		return err
	}

	return nil
}
