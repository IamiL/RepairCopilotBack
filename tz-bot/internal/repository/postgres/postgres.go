package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	repo "repairCopilotBot/tz-bot/internal/repository"
)

type Storage struct {
	db *pgxpool.Pool
}

func (c *Config) ConnString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		c.User,
		c.Pass,
		c.Host,
		c.Port,
		c.DBName,
	)
}

func NewConnPool(config *Config) (*pgxpool.Pool, error) {
	pgxPollConfig, err := pgxpool.ParseConfig(config.ConnString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	pgxPollConfig.MaxConns = int32(config.MaxConns)

	pool, err := pgxpool.NewWithConfig(context.Background(), pgxPollConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	err = pool.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return pool, nil
}

func New(pool *pgxpool.Pool) (*Storage, error) {
	return &Storage{db: pool}, nil
}

// TechnicalSpecification operations
func (s *Storage) CreateTechnicalSpecification(ctx context.Context, req *repo.CreateTechnicalSpecificationRequest) (*repo.TechnicalSpecification, error) {
	query := `
		INSERT INTO technical_specifications (id, name, user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, user_id, created_at, updated_at`

	var ts repo.TechnicalSpecification
	err := s.db.QueryRow(ctx, query, req.ID, req.Name, req.UserID, req.CreatedAt, req.UpdatedAt).
		Scan(&ts.ID, &ts.Name, &ts.UserID, &ts.CreatedAt, &ts.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create technical specification: %w", err)
	}

	return &ts, nil
}

func (s *Storage) GetTechnicalSpecification(ctx context.Context, id uuid.UUID) (*repo.TechnicalSpecification, error) {
	query := `SELECT id, name, user_id, created_at, updated_at FROM technical_specifications WHERE id = $1`

	var ts repo.TechnicalSpecification
	err := s.db.QueryRow(ctx, query, id).
		Scan(&ts.ID, &ts.Name, &ts.UserID, &ts.CreatedAt, &ts.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrTechnicalSpecificationNotFound
		}
		return nil, fmt.Errorf("failed to get technical specification: %w", err)
	}

	return &ts, nil
}

func (s *Storage) GetTechnicalSpecificationsByUserID(ctx context.Context, userID uuid.UUID) ([]*repo.TechnicalSpecification, error) {
	query := `SELECT id, name, user_id, created_at, updated_at FROM technical_specifications WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get technical specifications by user ID: %w", err)
	}
	defer rows.Close()

	var specifications []*repo.TechnicalSpecification
	for rows.Next() {
		var ts repo.TechnicalSpecification
		err := rows.Scan(&ts.ID, &ts.Name, &ts.UserID, &ts.CreatedAt, &ts.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan technical specification: %w", err)
		}
		specifications = append(specifications, &ts)
	}

	return specifications, nil
}

func (s *Storage) UpdateTechnicalSpecification(ctx context.Context, id uuid.UUID, name string, updatedAt time.Time) error {
	query := `UPDATE technical_specifications SET name = $1, updated_at = $2 WHERE id = $3`

	result, err := s.db.Exec(ctx, query, name, updatedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update technical specification: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repo.ErrTechnicalSpecificationNotFound
	}

	return nil
}

func (s *Storage) DeleteTechnicalSpecification(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM technical_specifications WHERE id = $1`

	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete technical specification: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repo.ErrTechnicalSpecificationNotFound
	}

	return nil
}

// Version operations
func (s *Storage) CreateVersion(ctx context.Context, req *repo.CreateVersionRequest) (*repo.Version, error) {
	query := `
		INSERT INTO versions (id, technical_specification_id, version_number, created_at, updated_at, original_file_id, out_html, css, checked_file_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, technical_specification_id, version_number, created_at, updated_at, original_file_id, out_html, css, checked_file_id`

	var version repo.Version
	err := s.db.QueryRow(ctx, query, req.ID, req.TechnicalSpecificationID, req.VersionNumber, req.CreatedAt, req.UpdatedAt,
		req.OriginalFileID, req.OutHTML, req.CSS, req.CheckedFileID).
		Scan(&version.ID, &version.TechnicalSpecificationID, &version.VersionNumber,
			&version.CreatedAt, &version.UpdatedAt, &version.OriginalFileID,
			&version.OutHTML, &version.CSS, &version.CheckedFileID)
	if err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	return &version, nil
}

func (s *Storage) GetVersion(ctx context.Context, id uuid.UUID) (*repo.Version, error) {
	query := `SELECT id, technical_specification_id, version_number, created_at, updated_at, original_file_id, out_html, css, checked_file_id FROM versions WHERE id = $1`

	var version repo.Version
	err := s.db.QueryRow(ctx, query, id).
		Scan(&version.ID, &version.TechnicalSpecificationID, &version.VersionNumber,
			&version.CreatedAt, &version.UpdatedAt, &version.OriginalFileID,
			&version.OutHTML, &version.CSS, &version.CheckedFileID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrVersionNotFound
		}
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	return &version, nil
}

func (s *Storage) GetVersionWithErrors(ctx context.Context, versionID uuid.UUID) (*repo.Version, []*repo.InvalidError, []*repo.MissingError, error) {
	// Получаем версию
	version, err := s.GetVersion(ctx, versionID)
	if err != nil {
		return nil, nil, nil, err
	}

	// Получаем invalid errors
	invalidErrors, err := s.GetInvalidErrorsByVersionID(ctx, versionID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get invalid errors: %w", err)
	}

	// Получаем missing errors
	missingErrors, err := s.GetMissingErrorsByVersionID(ctx, versionID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get missing errors: %w", err)
	}

	return version, invalidErrors, missingErrors, nil
}

func (s *Storage) GetVersionsByTechnicalSpecificationID(ctx context.Context, technicalSpecificationID uuid.UUID) ([]*repo.Version, error) {
	query := `SELECT id, technical_specification_id, version_number, created_at, updated_at, original_file_id, out_html, css, checked_file_id FROM versions WHERE technical_specification_id = $1 ORDER BY version_number DESC`

	rows, err := s.db.Query(ctx, query, technicalSpecificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get versions by technical specification ID: %w", err)
	}
	defer rows.Close()

	var versions []*repo.Version
	for rows.Next() {
		var version repo.Version
		err := rows.Scan(&version.ID, &version.TechnicalSpecificationID, &version.VersionNumber,
			&version.CreatedAt, &version.UpdatedAt, &version.OriginalFileID,
			&version.OutHTML, &version.CSS, &version.CheckedFileID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		versions = append(versions, &version)
	}

	return versions, nil
}

func (s *Storage) GetLatestVersion(ctx context.Context, technicalSpecificationID uuid.UUID) (*repo.Version, error) {
	query := `SELECT id, technical_specification_id, version_number, created_at, updated_at, original_file_id, out_html, css, checked_file_id FROM versions WHERE technical_specification_id = $1 ORDER BY version_number DESC LIMIT 1`

	var version repo.Version
	err := s.db.QueryRow(ctx, query, technicalSpecificationID).
		Scan(&version.ID, &version.TechnicalSpecificationID, &version.VersionNumber,
			&version.CreatedAt, &version.UpdatedAt, &version.OriginalFileID,
			&version.OutHTML, &version.CSS, &version.CheckedFileID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrVersionNotFound
		}
		return nil, fmt.Errorf("failed to get latest version: %w", err)
	}

	return &version, nil
}

func (s *Storage) GetVersionsByUserID(ctx context.Context, userID uuid.UUID) ([]*repo.VersionSummary, error) {
	query := `
		SELECT v.id, ts.name, v.version_number, v.created_at
		FROM versions v
		JOIN technical_specifications ts ON v.technical_specification_id = ts.id
		WHERE ts.user_id = $1
		ORDER BY v.created_at DESC
	`

	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get versions by user ID: %w", err)
	}
	defer rows.Close()

	var versions []*repo.VersionSummary
	for rows.Next() {
		var version repo.VersionSummary
		err := rows.Scan(&version.ID, &version.TechnicalSpecificationName,
			&version.VersionNumber, &version.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan version summary: %w", err)
		}
		versions = append(versions, &version)
	}

	return versions, nil
}

func (s *Storage) UpdateVersion(ctx context.Context, id uuid.UUID, outHTML, css, checkedFileID string, updatedAt time.Time) error {
	query := `UPDATE versions SET out_html = $1, css = $2, checked_file_id = $3, updated_at = $4 WHERE id = $5`

	result, err := s.db.Exec(ctx, query, outHTML, css, checkedFileID, updatedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update version: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repo.ErrVersionNotFound
	}

	return nil
}

func (s *Storage) DeleteVersion(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM versions WHERE id = $1`

	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete version: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repo.ErrVersionNotFound
	}

	return nil
}

// CreateInvalidErrors creates multiple invalid errors for a version
func (s *Storage) CreateInvalidErrors(ctx context.Context, req *repo.CreateInvalidErrorsRequest) error {
	if len(req.Errors) == 0 {
		return nil
	}

	query := `
		INSERT INTO invalid_errors (id, version_id, error_id, error_id_str, group_id, error_code, quote, analysis, critique, verification, suggested_fix, rationale, order_number, retrieval, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, errorData := range req.Errors {
		_, err := tx.Exec(ctx, query,
			errorData.ID, req.VersionID, errorData.ErrorID, errorData.ErrorIDStr,
			errorData.GroupID, errorData.ErrorCode, errorData.Quote,
			errorData.Analysis, errorData.Critique, errorData.Verification,
			errorData.SuggestedFix, errorData.Rationale, errorData.OrderNumber, errorData.Retrieval, errorData.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert invalid error: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (s *Storage) GetInvalidErrorsByVersionID(ctx context.Context, versionID uuid.UUID) ([]*repo.InvalidError, error) {
	query := `SELECT id, version_id, error_id, error_id_str, group_id, error_code, quote, analysis, critique, verification, suggested_fix, rationale,  created_at FROM invalid_errors WHERE version_id = $1 ORDER BY order_number`

	rows, err := s.db.Query(ctx, query, versionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invalid errors by version ID: %w", err)
	}
	defer rows.Close()

	var errors []*repo.InvalidError
	for rows.Next() {
		var invErr repo.InvalidError
		err := rows.Scan(&invErr.ID, &invErr.VersionID, &invErr.ErrorID, &invErr.ErrorIDStr,
			&invErr.GroupID, &invErr.ErrorCode, &invErr.Quote, &invErr.Analysis,
			&invErr.Critique, &invErr.Verification, &invErr.SuggestedFix,
			&invErr.Rationale, &invErr.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invalid error: %w", err)
		}
		errors = append(errors, &invErr)
	}

	return errors, nil
}

func (s *Storage) DeleteInvalidErrorsByVersionID(ctx context.Context, versionID uuid.UUID) error {
	query := `DELETE FROM invalid_errors WHERE version_id = $1`

	_, err := s.db.Exec(ctx, query, versionID)
	if err != nil {
		return fmt.Errorf("failed to delete invalid errors by version ID: %w", err)
	}

	return nil
}

// CreateMissingErrors creates multiple missing errors for a version
func (s *Storage) CreateMissingErrors(ctx context.Context, req *repo.CreateMissingErrorsRequest) error {
	if len(req.Errors) == 0 {
		return nil
	}

	query := `
		INSERT INTO missing_errors (id, version_id, error_id, error_id_str, group_id, error_code, analysis, critique, verification, suggested_fix, rationale, retrieval, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, errorData := range req.Errors {
		_, err := tx.Exec(ctx, query,
			errorData.ID, req.VersionID, errorData.ErrorID, errorData.ErrorIDStr,
			errorData.GroupID, errorData.ErrorCode, errorData.Analysis,
			errorData.Critique, errorData.Verification, errorData.SuggestedFix,
			errorData.Rationale, []string{"", ""}, errorData.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert missing error: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (s *Storage) GetMissingErrorsByVersionID(ctx context.Context, versionID uuid.UUID) ([]*repo.MissingError, error) {
	query := `SELECT id, version_id, error_id, error_id_str, group_id, error_code, analysis, critique, verification, suggested_fix, rationale, created_at FROM missing_errors WHERE version_id = $1 ORDER BY error_id`

	rows, err := s.db.Query(ctx, query, versionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get missing errors by version ID: %w", err)
	}
	defer rows.Close()

	var errors []*repo.MissingError
	for rows.Next() {
		var missErr repo.MissingError
		err := rows.Scan(&missErr.ID, &missErr.VersionID, &missErr.ErrorID, &missErr.ErrorIDStr,
			&missErr.GroupID, &missErr.ErrorCode, &missErr.Analysis, &missErr.Critique,
			&missErr.Verification, &missErr.SuggestedFix, &missErr.Rationale, &missErr.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan missing error: %w", err)
		}
		errors = append(errors, &missErr)
	}

	return errors, nil
}

func (s *Storage) DeleteMissingErrorsByVersionID(ctx context.Context, versionID uuid.UUID) error {
	query := `DELETE FROM missing_errors WHERE version_id = $1`

	_, err := s.db.Exec(ctx, query, versionID)
	if err != nil {
		return fmt.Errorf("failed to delete missing errors by version ID: %w", err)
	}

	return nil
}
