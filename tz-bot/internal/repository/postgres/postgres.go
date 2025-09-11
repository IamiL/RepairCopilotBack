package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"repairCopilotBot/tz-bot/internal/repository/models"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	tz_llm_client "repairCopilotBot/tz-bot/internal/pkg/llm"
	repo "repairCopilotBot/tz-bot/internal/repository"
	tzservice "repairCopilotBot/tz-bot/internal/service/tz"
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
func (s *Storage) CreateTechnicalSpecification(ctx context.Context, req *modelrepo.CreateTechnicalSpecificationRequest) (*modelrepo.TechnicalSpecification, error) {
	query := `
		INSERT INTO technical_specifications (id, name, user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, user_id, created_at, updated_at`

	var ts modelrepo.TechnicalSpecification
	err := s.db.QueryRow(ctx, query, req.ID, req.Name, req.UserID, req.CreatedAt, req.UpdatedAt).
		Scan(&ts.ID, &ts.Name, &ts.UserID, &ts.CreatedAt, &ts.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create technical specification: %w", err)
	}

	return &ts, nil
}

func (s *Storage) GetTechnicalSpecification(ctx context.Context, id uuid.UUID) (*modelrepo.TechnicalSpecification, error) {
	query := `SELECT id, name, user_id, created_at, updated_at FROM technical_specifications WHERE id = $1`

	var ts modelrepo.TechnicalSpecification
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

func (s *Storage) GetTechnicalSpecificationsByUserID(ctx context.Context, userID uuid.UUID) ([]*modelrepo.TechnicalSpecification, error) {
	query := `SELECT id, name, user_id, created_at, updated_at FROM technical_specifications WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get technical specifications by user ID: %w", err)
	}
	defer rows.Close()

	var specifications []*modelrepo.TechnicalSpecification
	for rows.Next() {
		var ts modelrepo.TechnicalSpecification
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
func (s *Storage) CreateVersion(ctx context.Context, req *modelrepo.CreateVersionRequest) error {
	query := `
		INSERT INTO versions (id, technical_specification_id, version_number, created_at, updated_at, original_file_id, out_html, css, checked_file_id, all_rubs, all_tokens, inspection_time, original_file_size, number_of_errors, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	_, err := s.db.Exec(ctx, query, req.ID, req.TechnicalSpecificationID, req.VersionNumber, req.CreatedAt, req.UpdatedAt,
		req.OriginalFileID, req.OutHTML, req.CSS, req.CheckedFileID, &req.AllRubs, &req.AllTokens, int64(req.InspectionTime), req.OriginalFileSize, req.NumberOfErrors, req.Status)
	if err != nil {
		return fmt.Errorf("failed to create version: %w", err)
	}

	return nil
}

func (s *Storage) UpdateVersion(ctx context.Context, req *modelrepo.UpdateVersionRequest) error {
	query := `
		UPDATE versions 
		SET updated_at = $2, out_html = $3, css = $4, checked_file_id = $5, all_rubs = $6, all_tokens = $7, inspection_time = $8, number_of_errors = $9, status = $10, html_from_word_parser = $11, html_with_placeholder = $12, html_paragraphs = $13, markdown_from_markdown_service = $14, html_with_ids_from_markdown_service = $15, mappings_from_markdown_service = $16, promts_from_promt_builder = $17, group_reports_from_llm = $18, html_paragraphs_with_wrapped_errors = $19
		WHERE id = $1`

	result, err := s.db.Exec(ctx, query, req.ID, req.UpdatedAt, req.OutHTML, req.CSS, req.CheckedFileID,
		&req.AllRubs, &req.AllTokens, int64(req.InspectionTime), req.NumberOfErrors, req.Status, req.HtmlFromWordParser, req.HtmlWithPlacrholder, req.HtmlParagraphs, req.MarkdownFromMarkdownService, req.HtmlWithIdsFromMarkdownService, req.MappingsFromMarkdownService, req.PromtsFromPromtBuilder, req.GroupReportsFromLlm, req.HtmlParagraphsWithWrappesErrors)
	if err != nil {
		return fmt.Errorf("failed to update version: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repo.ErrVersionNotFound
	}

	return nil
}

func (s *Storage) GetVersion(ctx context.Context, id uuid.UUID) (*modelrepo.Version, error) {
	query := `SELECT id, technical_specification_id, version_number, created_at, updated_at, original_file_id, out_html, css, checked_file_id, all_rubs, all_tokens, inspection_time, original_file_size, number_of_errors, status FROM versions WHERE id = $1`

	var version modelrepo.Version
	err := s.db.QueryRow(ctx, query, id).
		Scan(&version.ID, &version.TechnicalSpecificationID, &version.VersionNumber,
			&version.CreatedAt, &version.UpdatedAt, &version.OriginalFileID,
			&version.OutHTML, &version.CSS, &version.CheckedFileID, &version.AllRubs, &version.AllTokens, &version.InspectionTime, &version.OriginalFileSize, &version.NumberOfErrors, &version.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrVersionNotFound
		}
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	return &version, nil
}

func (s *Storage) GetVersionWithErrors(ctx context.Context, versionID uuid.UUID) (*modelrepo.Version, []*modelrepo.InvalidError, []*modelrepo.MissingError, error) {
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

func (s *Storage) GetVersionsByTechnicalSpecificationID(ctx context.Context, technicalSpecificationID uuid.UUID) ([]*modelrepo.Version, error) {
	query := `SELECT id, technical_specification_id, version_number, created_at, updated_at, original_file_id, out_html, css, checked_file_id, all_rubs, all_tokens, inspection_time FROM versions WHERE technical_specification_id = $1 ORDER BY version_number DESC`

	rows, err := s.db.Query(ctx, query, technicalSpecificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get versions by technical specification ID: %w", err)
	}
	defer rows.Close()

	var versions []*modelrepo.Version
	for rows.Next() {
		var version modelrepo.Version
		err := rows.Scan(&version.ID, &version.TechnicalSpecificationID, &version.VersionNumber,
			&version.CreatedAt, &version.UpdatedAt, &version.OriginalFileID,
			&version.OutHTML, &version.CSS, &version.CheckedFileID, &version.AllRubs, &version.AllTokens, &version.InspectionTime)
		if err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		versions = append(versions, &version)
	}

	return versions, nil
}

func (s *Storage) GetLatestVersion(ctx context.Context, technicalSpecificationID uuid.UUID) (*modelrepo.Version, error) {
	query := `SELECT id, technical_specification_id, version_number, created_at, updated_at, original_file_id, out_html, css, checked_file_id, all_rubs, all_tokens, inspection_time FROM versions WHERE technical_specification_id = $1 ORDER BY version_number DESC LIMIT 1`

	var version modelrepo.Version
	err := s.db.QueryRow(ctx, query, technicalSpecificationID).
		Scan(&version.ID, &version.TechnicalSpecificationID, &version.VersionNumber,
			&version.CreatedAt, &version.UpdatedAt, &version.OriginalFileID,
			&version.OutHTML, &version.CSS, &version.CheckedFileID, &version.AllRubs, &version.AllTokens, &version.InspectionTime)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrVersionNotFound
		}
		return nil, fmt.Errorf("failed to get latest version: %w", err)
	}

	return &version, nil
}

func (s *Storage) GetVersionsMeByUserID(ctx context.Context, userID uuid.UUID) ([]*tzservice.VersionMe, error) {
	query := `
		SELECT v.id, ts.name, v.version_number, v.created_at, v.original_file_id, v.checked_file_id, v.status
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

	var versions []*tzservice.VersionMe
	for rows.Next() {
		var version tzservice.VersionMe
		err := rows.Scan(&version.ID, &version.TechnicalSpecificationName,
			&version.VersionNumber, &version.CreatedAt, &version.OriginalFileID, &version.ReportFileID, &version.Status)
		if err != nil {
			return nil, fmt.Errorf("failed to scan version summary: %w", err)
		}
		versions = append(versions, &version)
	}

	return versions, nil
}

func (s *Storage) GetAllVersionsAdminDashboard(ctx context.Context, userID uuid.UUID) ([]*tzservice.VersionAdminDashboard, error) {
	var query string
	if userID != uuid.Nil {
		query = `
		SELECT 
			v.id, 
			ts.name,
			ts.user_id,
			v.version_number, 
			v.all_tokens,
			v.all_rubs,
			v.number_of_errors,
			v.inspection_time,
			v.original_file_size,
			v.created_at,
			v.original_file_id,
			v.checked_file_id
		FROM versions v
		JOIN technical_specifications ts ON v.technical_specification_id = ts.id
		WHERE ts.user_id = $1
		ORDER BY v.created_at DESC
	`
	} else {
		query = `
		SELECT 
			v.id, 
			ts.name,
			ts.user_id,
			v.version_number, 
			v.all_tokens,
			v.all_rubs,
			v.number_of_errors,
			v.inspection_time,
			v.original_file_size,
			v.created_at,
			v.original_file_id,
			v.checked_file_id
		FROM versions v
		JOIN technical_specifications ts ON v.technical_specification_id = ts.id
		ORDER BY v.created_at DESC
	`
	}

	var rows pgx.Rows

	if userID != uuid.Nil {
		rowsResp, err := s.db.Query(ctx, query, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get all versions: %w", err)
		}
		rows = rowsResp
	} else {
		rowsResp, err := s.db.Query(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("failed to get all versions: %w", err)
		}
		rows = rowsResp
	}

	defer rows.Close()

	var versions []*tzservice.VersionAdminDashboard
	for rows.Next() {
		var numberOfErrors *int64
		var originalFileSize *int64
		var version tzservice.VersionAdminDashboard
		err := rows.Scan(
			&version.ID,
			&version.TechnicalSpecificationName,
			&version.UserID,
			&version.VersionNumber,
			&version.AllTokens,
			&version.AllRubs,
			&numberOfErrors,
			&version.InspectionTime,
			&originalFileSize,
			&version.CreatedAt,
			&version.OriginalFileId,
			&version.ReportFileId,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan version with error counts: %w", err)
		}
		if numberOfErrors != nil {
			version.NumberOfErrors = *numberOfErrors
		}
		if originalFileSize != nil {
			version.OriginalFileSize = *originalFileSize
		}

		versions = append(versions, &version)
	}

	return versions, nil
}

func (s *Storage) GetVersionStatistics(ctx context.Context) (*modelrepo.VersionStatistics, error) {
	query := `
		SELECT 
			COUNT(*) as total_versions,
			SUM(all_tokens) as total_tokens,
			SUM(all_rubs) as total_rubs,
			AVG(inspection_time) as average_inspection_time
		FROM versions
		WHERE all_tokens IS NOT NULL 
		   OR all_rubs IS NOT NULL 
		   OR inspection_time IS NOT NULL
	`

	var stats modelrepo.VersionStatistics
	var avgInspectionTime *float64 // временная переменная для сканирования среднего значения

	err := s.db.QueryRow(ctx, query).Scan(
		&stats.TotalVersions,
		&stats.TotalTokens,
		&stats.TotalRubs,
		&avgInspectionTime,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get version statistics: %w", err)
	}

	// Преобразуем среднее значение в time.Duration, если значение не nil
	if avgInspectionTime != nil {
		// Среднее значение в наносекундах, преобразуем в time.Duration
		duration := time.Duration(int64(*avgInspectionTime))
		stats.AverageInspectionTime = &duration
	}

	return &stats, nil
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
func (s *Storage) CreateInvalidErrors(ctx context.Context, req *modelrepo.CreateInvalidErrorsRequest) error {
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

func (s *Storage) GetInvalidErrorsByVersionID(ctx context.Context, versionID uuid.UUID) ([]*modelrepo.InvalidError, error) {
	query := `SELECT id, version_id, error_id, error_id_str, group_id, error_code, quote, analysis, critique, verification, suggested_fix, rationale,  created_at FROM invalid_errors WHERE version_id = $1 ORDER BY order_number`

	rows, err := s.db.Query(ctx, query, versionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invalid errors by version ID: %w", err)
	}
	defer rows.Close()

	var errors []*modelrepo.InvalidError
	for rows.Next() {
		var invErr modelrepo.InvalidError
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

func (s *Storage) GetUUIDByErrorID(ctx context.Context, errorID int) (uuid.UUID, error) {
	queryInvalid := `SELECT id FROM invalid_errors WHERE error_id = $1 LIMIT 1`
	queryMissing := `SELECT id FROM missing_errors WHERE error_id = $1 LIMIT 1`

	var errorUUID uuid.UUID

	err := s.db.QueryRow(ctx, queryInvalid, errorID).Scan(&errorUUID)
	if err == nil {
		return errorUUID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("failed to query invalid_errors: %w", err)
	}

	err = s.db.QueryRow(ctx, queryMissing, errorID).Scan(&errorUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, repo.ErrErrorNotFound
		}
		return uuid.Nil, fmt.Errorf("failed to query missing_errors: %w", err)
	}

	return errorUUID, nil
}

// CreateMissingErrors creates multiple missing errors for a version
func (s *Storage) CreateMissingErrors(ctx context.Context, req *modelrepo.CreateMissingErrorsRequest) error {
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

func (s *Storage) GetMissingErrorsByVersionID(ctx context.Context, versionID uuid.UUID) ([]*modelrepo.MissingError, error) {
	query := `SELECT id, version_id, error_id, error_id_str, group_id, error_code, analysis, critique, verification, suggested_fix, rationale, created_at FROM missing_errors WHERE version_id = $1 ORDER BY error_id`

	rows, err := s.db.Query(ctx, query, versionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get missing errors by version ID: %w", err)
	}
	defer rows.Close()

	var errors []*modelrepo.MissingError
	for rows.Next() {
		var missErr modelrepo.MissingError
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

// ErrorRepository implementation

// CreateErrors creates multiple errors for a version
func (s *Storage) CreateErrors(ctx context.Context, req *modelrepo.CreateErrorsRequest) error {
	if len(req.Errors) == 0 {
		return nil
	}

	query := `
		INSERT INTO errors (id, version_id, group_id, error_code, order_number, preliminary_notes, overall_critique, verdict, process_analysis, process_critique, process_verification, process_retrieval, instances, name, description, detector)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`

	for _, errorData := range req.Errors {
		_, err := s.db.Exec(ctx, query,
			errorData.ID, req.VersionID, errorData.GroupID, errorData.ErrorCode, errorData.OrderNumber,
			errorData.PreliminaryNotes, errorData.OverallCritique, errorData.Verdict,
			errorData.ProcessAnalysis, errorData.ProcessCritique, errorData.ProcessVerification,
			errorData.ProcessRetrieval, errorData.Instances, errorData.Name, errorData.Description, errorData.Detector)
		if err != nil {
			return fmt.Errorf("failed to create error: %w", err)
		}
	}

	return nil
}

// GetErrorsByVersionID retrieves all errors for a specific version
func (s *Storage) GetErrorsByVersionID(ctx context.Context, versionID uuid.UUID) (*[]tzservice.Error, error) {
	query := `
		SELECT id, version_id, group_id, error_code, order_number, preliminary_notes, overall_critique, verdict, process_analysis, process_critique, process_verification, process_retrieval, instances, name, description, detector
		FROM errors 
		WHERE version_id = @version_id
		ORDER BY order_number`

	args := pgx.NamedArgs{
		"version_id": versionID,
	}

	rows, err := s.db.Query(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get errors by version ID: %w", err)
	}
	defer rows.Close()
	var name *string
	var description *string
	var detector *string
	var errorsList []tzservice.Error
	for rows.Next() {
		var errorItem tzservice.Error
		var instancesJSON []byte
		err := rows.Scan(
			&errorItem.ID,
			&versionID,
			&errorItem.GroupID,
			&errorItem.ErrorCode,
			&errorItem.OrderNumber,
			&errorItem.PreliminaryNotes,
			&errorItem.OverallCritique,
			&errorItem.Verdict,
			&errorItem.ProcessAnalysis,
			&errorItem.ProcessCritique,
			&errorItem.ProcessVerification,
			&errorItem.ProcessRetrieval,
			&instancesJSON,
			&name,
			&description,
			&detector,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan error: %w", err)
		}

		// Десериализуем instances из JSON
		if len(instancesJSON) > 0 {
			var instances []tz_llm_client.Instance
			err := json.Unmarshal(instancesJSON, &instances)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal instances: %w", err)
			}
			errorItem.Instances = &instances
		}

		if name != nil {
			errorItem.Name = *name
		}

		if description != nil {
			errorItem.Description = *description
		}

		if detector != nil {
			errorItem.Detector = *detector
		}

		errorsList = append(errorsList, errorItem)
	}

	return &errorsList, nil
}

// InvalidInstanceRepository implementation

// SaveInvalidInstances saves multiple invalid instances
func (s *Storage) SaveInvalidInstances(ctx context.Context, invalidInstances *[]tzservice.OutInvalidError) error {
	if invalidInstances == nil || len(*invalidInstances) == 0 {
		return nil
	}

	query := `
		INSERT INTO invalid_instances (id, html_id, error_id, quote, suggested_fix, original_quote, quote_lines, until_the_end_of_sentence, start_line_number, end_line_number, system_comment, order_number, rationale, feedback_exists, feedback_verification_exists)
		VALUES (@id, @html_id, @error_id, @quote, @suggested_fix, @original_quote, @quote_lines, @until_the_end_of_sentence, @start_line_number, @end_line_number, @system_comment, @order_number, @rationale, @feedback_exists, @feedback_verification_exists)`

	for _, instance := range *invalidInstances {
		args := pgx.NamedArgs{
			"id":                           instance.ID,
			"html_id":                      instance.HtmlID,
			"error_id":                     instance.ErrorID,
			"quote":                        instance.Quote,
			"suggested_fix":                instance.SuggestedFix,
			"original_quote":               instance.OriginalQuote,
			"quote_lines":                  instance.QuoteLines,
			"until_the_end_of_sentence":    instance.UntilTheEndOfSentence,
			"start_line_number":            instance.StartLineNumber,
			"end_line_number":              instance.EndLineNumber,
			"system_comment":               instance.SystemComment,
			"order_number":                 instance.OrderNumber,
			"rationale":                    instance.Rationale,
			"feedback_exists":              false,
			"feedback_verification_exists": false,
		}

		_, err := s.db.Exec(ctx, query, args)
		if err != nil {
			return fmt.Errorf("failed to save invalid instance: %w", err)
		}
	}

	return nil
}

// GetInvalidInstancesByErrorID retrieves all invalid instances for a specific error
func (s *Storage) GetInvalidInstancesByErrorID(ctx context.Context, errorID uuid.UUID) (*[]tzservice.OutInvalidError, error) {
	query := `
		SELECT id, html_id, error_id, quote, suggested_fix, original_quote, quote_lines, until_the_end_of_sentence, start_line_number, end_line_number, system_comment, order_number, rationale, feedback_exists, feedback_mark, feedback_comment, feedback_user, feedback_verification_exists, feedback_verification_mark, feedback_verification_comment, feedback_verification_user
		FROM invalid_instances 
		WHERE error_id = @error_id 
		ORDER BY order_number`

	args := pgx.NamedArgs{
		"error_id": errorID,
	}

	rows, err := s.db.Query(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get invalid instances by error ID: %w", err)
	}
	defer rows.Close()

	var instances []tzservice.OutInvalidError
	for rows.Next() {
		var rationale *string
		var instance tzservice.OutInvalidError
		err := rows.Scan(
			&instance.ID,
			&instance.HtmlID,
			&instance.ErrorID,
			&instance.Quote,
			&instance.SuggestedFix,
			&instance.OriginalQuote,
			&instance.QuoteLines,
			&instance.UntilTheEndOfSentence,
			&instance.StartLineNumber,
			&instance.EndLineNumber,
			&instance.SystemComment,
			&instance.OrderNumber,
			&rationale,
			&instance.FeedbackExists,
			&instance.FeedbackMark,
			&instance.FeedbackComment,
			&instance.FeedbackUser,
			&instance.FeedbackVerificationExists,
			&instance.FeedbackVerificationMark,
			&instance.FeedbackVerificationComment,
			&instance.FeedbackVerificationUser,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invalid instance: %w", err)
		}

		if rationale != nil {
			instance.Rationale = *rationale
		}
		instances = append(instances, instance)
	}

	return &instances, nil
}

// MissingInstanceRepository implementation

// SaveMissingInstances saves multiple missing instances
func (s *Storage) SaveMissingInstances(ctx context.Context, missingInstances *[]tzservice.OutMissingError) error {
	if missingInstances == nil || len(*missingInstances) == 0 {
		return nil
	}

	query := `
		INSERT INTO missing_instances (id, html_id, error_id, suggested_fix, rationale, feedback_exists, feedback_verification_exists)
		VALUES (@id, @html_id, @error_id, @suggested_fix, @rationale, @feedback_exists, @feedback_verification_exists)`

	for _, instance := range *missingInstances {
		args := pgx.NamedArgs{
			"id":                           instance.ID,
			"html_id":                      instance.HtmlID,
			"error_id":                     instance.ErrorID,
			"suggested_fix":                instance.SuggestedFix,
			"rationale":                    instance.Rationale,
			"feedback_exists":              false,
			"feedback_verification_exists": false,
		}

		_, err := s.db.Exec(ctx, query, args)
		if err != nil {
			return fmt.Errorf("failed to save missing instance: %w", err)
		}
	}

	return nil
}

// GetMissingInstancesByErrorID retrieves all missing instances for a specific error
func (s *Storage) GetMissingInstancesByErrorID(ctx context.Context, errorID uuid.UUID) (*[]tzservice.OutMissingError, error) {
	query := `
		SELECT id, html_id, error_id, suggested_fix, rationale, feedback_exists, feedback_mark, feedback_comment, feedback_user, feedback_verification_exists, feedback_verification_mark, feedback_verification_comment, feedback_verification_user
		FROM missing_instances 
		WHERE error_id = @error_id`

	args := pgx.NamedArgs{
		"error_id": errorID,
	}

	rows, err := s.db.Query(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get missing instances by error ID: %w", err)
	}
	defer rows.Close()

	var instances []tzservice.OutMissingError
	for rows.Next() {
		var instance tzservice.OutMissingError
		err := rows.Scan(
			&instance.ID,
			&instance.HtmlID,
			&instance.ErrorID,
			&instance.SuggestedFix,
			&instance.Rationale,
			&instance.FeedbackExists,
			&instance.FeedbackMark,
			&instance.FeedbackComment,
			&instance.FeedbackUser,
			&instance.FeedbackVerificationExists,
			&instance.FeedbackVerificationMark,
			&instance.FeedbackVerificationComment,
			&instance.FeedbackVerificationUser,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan missing instance: %w", err)
		}
		instances = append(instances, instance)
	}

	return &instances, nil
}

// UpdateInvalidInstanceFeedback updates feedback for invalid instance
func (s *Storage) UpdateInvalidInstanceFeedback(ctx context.Context, instanceID uuid.UUID, feedbackMark *bool, feedbackComment *string, userID uuid.UUID) error {
	query := `
		UPDATE invalid_instances 
		SET feedback_exists = $2, feedback_mark = $3, feedback_comment = $4, feedback_user = $5
		WHERE id = $1`

	result, err := s.db.Exec(ctx, query, instanceID, true, feedbackMark, feedbackComment, userID)
	if err != nil {
		return fmt.Errorf("failed to update invalid instance feedback: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("invalid instance not found: %s", instanceID.String())
	}

	return nil
}

func (s *Storage) UpdateInvalidInstanceVerificationFeedback(ctx context.Context, instanceID uuid.UUID, feedbackMark *bool, feedbackComment *string, userID uuid.UUID) error {
	query := `
		UPDATE invalid_instances 
		SET feedback_verification_exists = $2, feedback_verification_mark = $3, feedback_verification_comment = $4, feedback_verification_user = $5
		WHERE id = $1`

	result, err := s.db.Exec(ctx, query, instanceID, true, feedbackMark, feedbackComment, userID)
	if err != nil {
		return fmt.Errorf("failed to update invalid instance feedback: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("invalid instance not found: %s", instanceID.String())
	}

	return nil
}

// UpdateMissingInstanceFeedback updates feedback for missing instance
func (s *Storage) UpdateMissingInstanceFeedback(ctx context.Context, instanceID uuid.UUID, feedbackMark *bool, feedbackComment *string, userID uuid.UUID) error {
	query := `
		UPDATE missing_instances 
		SET feedback_exists = $2, feedback_mark = $3, feedback_comment = $4, feedback_user = $5
		WHERE id = $1`

	result, err := s.db.Exec(ctx, query, instanceID, true, feedbackMark, feedbackComment, userID)
	if err != nil {
		return fmt.Errorf("failed to update missing instance feedback: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("missing instance not found: %s", instanceID.String())
	}

	return nil
}

// UpdateMissingInstanceFeedback updates feedback for missing instance
func (s *Storage) UpdateMissingInstanceVerificationFeedback(ctx context.Context, instanceID uuid.UUID, feedbackMark *bool, feedbackComment *string, userID uuid.UUID) error {
	query := `
		UPDATE missing_instances 
		SET feedback_verification_exists = $2, feedback_verification_mark = $3, feedback_verification_comment = $4, feedback_verification_user = $5
		WHERE id = $1`

	result, err := s.db.Exec(ctx, query, instanceID, true, feedbackMark, feedbackComment, userID)
	if err != nil {
		return fmt.Errorf("failed to update missing instance feedback: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("missing instance not found: %s", instanceID.String())
	}

	return nil
}

// CreateErrorFeedback creates new feedback for an error
func (s *Storage) CreateErrorFeedback(ctx context.Context, req *modelrepo.CreateErrorFeedbackRequest) (*modelrepo.ErrorFeedback, error) {
	query := `
		INSERT INTO error_feedback (id, version_id, error_id, error_type, user_id, feedback_type, comment, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, version_id, error_id, error_type, user_id, feedback_type, comment, created_at, updated_at`

	var feedback modelrepo.ErrorFeedback
	err := s.db.QueryRow(ctx, query, req.ID, req.VersionID, req.ErrorID, req.ErrorType, req.UserID, req.FeedbackType, req.Comment, req.CreatedAt, req.UpdatedAt).
		Scan(&feedback.ID, &feedback.VersionID, &feedback.ErrorID, &feedback.ErrorType, &feedback.UserID, &feedback.FeedbackType, &feedback.Comment, &feedback.CreatedAt, &feedback.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create error feedback: %w", err)
	}

	return &feedback, nil
}

// LLMCacheRepository implementation

func (s *Storage) GetCachedResponse(ctx context.Context, messagesHash string) (*modelrepo.LLMCache, error) {
	query := `SELECT id, messages_hash, response_data, created_at, updated_at FROM llm_cache WHERE messages_hash = $1 ORDER BY created_at DESC LIMIT 1`

	var cache modelrepo.LLMCache
	err := s.db.QueryRow(ctx, query, messagesHash).Scan(
		&cache.ID, &cache.MessagesHash, &cache.ResponseData, &cache.CreatedAt, &cache.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repo.ErrLLMCacheNotFound
		}
		return nil, fmt.Errorf("failed to get cached response: %w", err)
	}

	return &cache, nil
}

func (s *Storage) SaveCachedResponse(ctx context.Context, req *modelrepo.CreateLLMCacheRequest) (*modelrepo.LLMCache, error) {
	id := uuid.New()
	query := `INSERT INTO llm_cache (id, messages_hash, response_data, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, $5) RETURNING id, messages_hash, response_data, created_at, updated_at`

	var cache modelrepo.LLMCache
	err := s.db.QueryRow(ctx, query, id, req.MessagesHash, req.ResponseData, req.CreatedAt, req.UpdatedAt).Scan(
		&cache.ID, &cache.MessagesHash, &cache.ResponseData, &cache.CreatedAt, &cache.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to save cached response: %w", err)
	}

	return &cache, nil
}

func (s *Storage) GetVersionsDateRange(ctx context.Context) (string, string, error) {
	query := `
		SELECT 
			MIN(DATE(created_at)) as min_date,
			MAX(DATE(created_at)) as max_date
		FROM versions
		WHERE created_at IS NOT NULL`

	var minDate, maxDate *time.Time
	err := s.db.QueryRow(ctx, query).Scan(&minDate, &maxDate)
	if err != nil {
		return "", "", fmt.Errorf("failed to get versions date range: %w", err)
	}

	// Форматируем даты в формат 2024-01-01
	var minDateStr, maxDateStr string
	if minDate != nil {
		minDateStr = minDate.Format("2006-01-02")
	}
	if maxDate != nil {
		maxDateStr = maxDate.Format("2006-01-02")
	}

	return minDateStr, maxDateStr, nil
}

func (s *Storage) GetDailyAnalytics(ctx context.Context, fromDate, toDate, timezone string, metrics []string) ([]*tzservice.DailyAnalyticsPoint, error) {
	// Определяем, какие метрики нужно включить
	includeConsumption := len(metrics) == 0 || contains(metrics, "consumption")
	includeToPay := len(metrics) == 0 || contains(metrics, "toPay")
	includeTz := len(metrics) == 0 || contains(metrics, "tz")

	// Строим SELECT часть запроса
	selectParts := []string{"TO_CHAR(DATE(created_at"}
	if timezone != "" {
		selectParts[0] += fmt.Sprintf(" AT TIME ZONE '%s'", timezone)
	}
	selectParts[0] += "), 'YYYY-MM-DD') as date"

	if includeConsumption {
		selectParts = append(selectParts, "COALESCE(SUM(all_tokens), 0) as consumption")
	}
	if includeToPay {
		selectParts = append(selectParts, "COALESCE(SUM(all_rubs), 0) as to_pay")
	}
	if includeTz {
		selectParts = append(selectParts, "COUNT(*) as tz")
	}

	query := fmt.Sprintf(`
		SELECT %s
		FROM versions 
		WHERE DATE(created_at%s) >= $1 
		  AND DATE(created_at%s) <= $2
		  AND created_at IS NOT NULL
		GROUP BY DATE(created_at%s)
		ORDER BY DATE(created_at%s)`,
		strings.Join(selectParts, ", "),
		getTimezoneClause(timezone),
		getTimezoneClause(timezone),
		getTimezoneClause(timezone),
		getTimezoneClause(timezone))

	rows, err := s.db.Query(ctx, query, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily analytics: %w", err)
	}
	defer rows.Close()

	var points []*tzservice.DailyAnalyticsPoint
	for rows.Next() {
		point := &tzservice.DailyAnalyticsPoint{}

		// Создаём slice для сканирования значений
		values := make([]interface{}, 1) // дата всегда есть
		values[0] = &point.Date

		if includeConsumption {
			var consumption int64
			point.Consumption = &consumption
			values = append(values, &consumption)
		}
		if includeToPay {
			var toPay float64
			point.ToPay = &toPay
			values = append(values, &toPay)
		}
		if includeTz {
			var tz int32
			point.Tz = &tz
			values = append(values, &tz)
		}

		err := rows.Scan(values...)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily analytics row: %w", err)
		}

		points = append(points, point)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return points, nil
}

func getTimezoneClause(timezone string) string {
	if timezone != "" {
		return fmt.Sprintf(" AT TIME ZONE '%s'", timezone)
	}
	return ""
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (s *Storage) GetFeedbacks(ctx context.Context, userID *string) ([]*tzservice.FeedbackInstance, error) {
	// SQL запрос объединяет обе таблицы через UNION ALL и делает JOIN с errors, versions, technical_specifications
	query := `
		WITH feedbacks AS (
			-- Invalid instances with feedback
			SELECT 
				ii.id as instance_id,
				'invalid' as instance_type,
				ii.feedback_mark,
				ii.feedback_comment,
				ii.feedback_user,
				ii.error_id
			FROM invalid_instances ii
			WHERE ii.feedback_exists = true
			
			UNION ALL
			
			-- Missing instances with feedback  
			SELECT 
				mi.id as instance_id,
				'missing' as instance_type,
				mi.feedback_mark,
				mi.feedback_comment,
				mi.feedback_user,
				mi.error_id
			FROM missing_instances mi
			WHERE mi.feedback_exists = true
		)
		SELECT 
			f.instance_id,
			f.instance_type,
			f.feedback_mark,
			f.feedback_comment,
			f.feedback_user,
			f.error_id,
			e.error_code,
			v.id as version_id,
			ts.name as technical_specification_name
		FROM feedbacks f
		JOIN errors e ON f.error_id = e.id
		JOIN versions v ON e.version_id = v.id  
		JOIN technical_specifications ts ON v.technical_specification_id = ts.id`

	args := []interface{}{}

	// Добавляем фильтрацию по user_id если он передан
	if userID != nil && *userID != "" {
		query += " WHERE f.feedback_user = $1"
		args = append(args, *userID)
	}

	query += " ORDER BY ts.name, v.id, f.instance_type, f.instance_id"

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get feedbacks: %w", err)
	}
	defer rows.Close()

	var feedbacks []*tzservice.FeedbackInstance
	for rows.Next() {
		var feedbackComment *string
		var feedback tzservice.FeedbackInstance
		err := rows.Scan(
			&feedback.InstanceID,
			&feedback.InstanceType,
			&feedback.FeedbackMark,
			&feedbackComment,
			&feedback.FeedbackUser,
			&feedback.ErrorID,
			&feedback.ErrorCode,
			&feedback.VersionID,
			&feedback.TechnicalSpecificationName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feedback instance: %w", err)
		}

		if feedbackComment != nil {
			feedback.FeedbackComment = *feedbackComment
		}
		feedbacks = append(feedbacks, &feedback)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return feedbacks, nil
}
