package postgresActionLog

import (
	"context"
	"fmt"
	"time"

	"repairCopilotBot/api-gateway-service/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	db *pgxpool.Pool
}

func New(pool *pgxpool.Pool) (*Storage, error) {
	return &Storage{db: pool}, nil
}

func (s *Storage) CreateActionLog(
	ctx context.Context,
	action string,
	userID uuid.UUID,
	actionType int,
) error {
	_, err := s.db.Exec(
		ctx,
		"INSERT INTO action_logs(action, user_id, created_at, type) VALUES($1, $2, $3, $4)",
		action,
		userID,
		time.Now(),
		actionType,
	)
	if err != nil {
		return fmt.Errorf("failed to create action log: %w", err)
	}

	return nil
}

func (s *Storage) GetAllActionLogs(ctx context.Context) ([]repository.ActionLog, error) {
	query := `SELECT id, action, user_id, created_at, type FROM action_logs ORDER BY created_at DESC`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()

	var logs []repository.ActionLog
	for rows.Next() {
		var actionType *int
		var log repository.ActionLog
		err := rows.Scan(&log.ID, &log.Action, &log.UserID, &log.CreateAt, &actionType)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		if actionType != nil {
			log.ActionType = *actionType
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return logs, nil
}
