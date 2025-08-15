package repository

import (
	"time"

	"github.com/google/uuid"
)

// TechnicalSpecification represents a technical specification document
type TechnicalSpecification struct {
	ID        uuid.UUID `db:"id"`
	Name      string    `db:"name"`
	UserID    uuid.UUID `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Version represents a version of a technical specification
type Version struct {
	ID                       uuid.UUID     `db:"id"`
	TechnicalSpecificationID uuid.UUID     `db:"technical_specification_id"`
	VersionNumber            int           `db:"version_number"`
	CreatedAt                time.Time     `db:"created_at"`
	UpdatedAt                time.Time     `db:"updated_at"`
	OriginalFileID           string        `db:"original_file_id"`
	OutHTML                  string        `db:"out_html"`
	CSS                      string        `db:"css"`
	CheckedFileID            string        `db:"checked_file_id"`
	AllRubs                  *float64      `db:"all_rubs"`
	AllTokens                *int64        `db:"all_tokens"`
	InspectionTime           *time.Duration `db:"inspection_time"`
}

// VersionWithTechnicalSpec represents a version with technical specification info
type VersionWithTechnicalSpec struct {
	ID                         uuid.UUID      `db:"id"`
	TechnicalSpecificationID   uuid.UUID      `db:"technical_specification_id"`
	TechnicalSpecificationName string         `db:"technical_specification_name"`
	VersionNumber              int            `db:"version_number"`
	CreatedAt                  time.Time      `db:"created_at"`
	UpdatedAt                  time.Time      `db:"updated_at"`
	OriginalFileID             string         `db:"original_file_id"`
	OutHTML                    string         `db:"out_html"`
	CSS                        string         `db:"css"`
	CheckedFileID              string         `db:"checked_file_id"`
	AllRubs                    *float64       `db:"all_rubs"`
	AllTokens                  *int64         `db:"all_tokens"`
	InspectionTime             *time.Duration `db:"inspection_time"`
}

// VersionSummary represents minimal version data for API responses
type VersionSummary struct {
	ID                         uuid.UUID `db:"id"`
	TechnicalSpecificationName string    `db:"technical_specification_name"`
	VersionNumber              int       `db:"version_number"`
	CreatedAt                  time.Time `db:"created_at"`
}

// VersionWithErrorCounts represents a complete version with all fields and error counts
type VersionWithErrorCounts struct {
	ID                         uuid.UUID      `db:"id"`
	TechnicalSpecificationID   uuid.UUID      `db:"technical_specification_id"`
	TechnicalSpecificationName string         `db:"technical_specification_name"`
	UserID                     uuid.UUID      `db:"user_id"`
	VersionNumber              int            `db:"version_number"`
	CreatedAt                  time.Time      `db:"created_at"`
	UpdatedAt                  time.Time      `db:"updated_at"`
	OriginalFileID             string         `db:"original_file_id"`
	OutHTML                    string         `db:"out_html"`
	CSS                        string         `db:"css"`
	CheckedFileID              string         `db:"checked_file_id"`
	AllRubs                    *float64       `db:"all_rubs"`
	AllTokens                  *int64         `db:"all_tokens"`
	InspectionTime             *time.Duration `db:"inspection_time"`
	InvalidErrorCount          int            `db:"invalid_error_count"`
	MissingErrorCount          int            `db:"missing_error_count"`
}

// VersionStatistics represents aggregated statistics for all versions
type VersionStatistics struct {
	TotalVersions          int64           `db:"total_versions"`
	TotalTokens            *int64          `db:"total_tokens"`
	TotalRubs              *float64        `db:"total_rubs"`
	AverageInspectionTime  *time.Duration  `db:"average_inspection_time"`
}

// InvalidError represents an invalid error from the analysis
type InvalidError struct {
	ID           uuid.UUID `db:"id"`
	VersionID    uuid.UUID `db:"version_id"`
	ErrorID      int       `db:"error_id"`
	ErrorIDStr   string    `db:"error_id_str"`
	GroupID      string    `db:"group_id"`
	ErrorCode    string    `db:"error_code"`
	Quote        string    `db:"quote"`
	Analysis     string    `db:"analysis"`
	Critique     string    `db:"critique"`
	Verification string    `db:"verification"`
	SuggestedFix string    `db:"suggested_fix"`
	Rationale    string    `db:"rationale"`
	OrderNumber  int       `db:"order_number"`
	Retrieval    []string  `db:"retrieval"`
	CreatedAt    time.Time `db:"created_at"`
}

// MissingError represents a missing error from the analysis
type MissingError struct {
	ID           uuid.UUID `db:"id"`
	VersionID    uuid.UUID `db:"version_id"`
	ErrorID      int       `db:"error_id"`
	ErrorIDStr   string    `db:"error_id_str"`
	GroupID      string    `db:"group_id"`
	ErrorCode    string    `db:"error_code"`
	Analysis     string    `db:"analysis"`
	Critique     string    `db:"critique"`
	Verification string    `db:"verification"`
	SuggestedFix string    `db:"suggested_fix"`
	Rationale    string    `db:"rationale"`
	Retrieval    []string  `db:"retrieval"`
	CreatedAt    time.Time `db:"created_at"`
}

// CreateTechnicalSpecificationRequest represents request to create a new technical specification
type CreateTechnicalSpecificationRequest struct {
	ID        uuid.UUID
	Name      string
	UserID    uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateVersionRequest represents request to create a new version
type CreateVersionRequest struct {
	ID                       uuid.UUID
	TechnicalSpecificationID uuid.UUID
	VersionNumber            int
	CreatedAt                time.Time
	UpdatedAt                time.Time
	OriginalFileID           string
	OutHTML                  string
	CSS                      string
	CheckedFileID            string
	AllRubs                  float64
	AllTokens                int64
	InspectionTime           time.Duration
}

// CreateInvalidErrorsRequest represents request to create invalid errors
type CreateInvalidErrorsRequest struct {
	VersionID uuid.UUID
	Errors    []InvalidErrorData
}

// CreateMissingErrorsRequest represents request to create missing errors
type CreateMissingErrorsRequest struct {
	VersionID uuid.UUID
	Errors    []MissingErrorData
}

// InvalidErrorData represents data for creating an invalid error
type InvalidErrorData struct {
	ID           uuid.UUID
	ErrorID      int
	ErrorIDStr   string
	GroupID      string
	ErrorCode    string
	Quote        string
	Analysis     string
	Critique     string
	Verification string
	SuggestedFix string
	Rationale    string
	OrderNumber  int
	Retrieval    []string
	CreatedAt    time.Time
}

// MissingErrorData represents data for creating a missing error
type MissingErrorData struct {
	ID           uuid.UUID
	ErrorID      int
	ErrorIDStr   string
	GroupID      string
	ErrorCode    string
	Analysis     string
	Critique     string
	Verification string
	SuggestedFix string
	Rationale    string
	Retrieval    []string
	CreatedAt    time.Time
}

// ErrorType represents the type of error (invalid or missing)
type ErrorType string

const (
	ErrorTypeInvalid ErrorType = "invalid"
	ErrorTypeMissing ErrorType = "missing"
)

// ErrorFeedback represents user feedback on error analysis
type ErrorFeedback struct {
	ID          uuid.UUID `db:"id"`
	VersionID   uuid.UUID `db:"version_id"`
	ErrorID     uuid.UUID `db:"error_id"`
	ErrorType   ErrorType `db:"error_type"`
	IsGoodError bool      `db:"is_good_error"`
	Comment     *string   `db:"comment"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// CreateErrorFeedbackRequest represents request to create error feedback
type CreateErrorFeedbackRequest struct {
	ID          uuid.UUID
	VersionID   uuid.UUID
	ErrorID     uuid.UUID
	ErrorType   ErrorType
	IsGoodError bool
	Comment     *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// LLMCache represents a cached LLM request and response
type LLMCache struct {
	ID           uuid.UUID `db:"id"`
	MessagesHash string    `db:"messages_hash"`
	ResponseData []byte    `db:"response_data"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// CreateLLMCacheRequest represents request to create a new cache entry
type CreateLLMCacheRequest struct {
	MessagesHash string
	ResponseData []byte
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
