package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrTechnicalSpecificationNotFound = errors.New("technical specification not found")
	ErrVersionNotFound                = errors.New("version not found")
	ErrDuplicateVersion               = errors.New("version with this number already exists for this technical specification")
)

// TechnicalSpecificationRepository defines the interface for technical specification operations
type TechnicalSpecificationRepository interface {
	// CreateTechnicalSpecification creates a new technical specification
	CreateTechnicalSpecification(ctx context.Context, req *CreateTechnicalSpecificationRequest) (*TechnicalSpecification, error)

	// GetTechnicalSpecification retrieves a technical specification by ID
	GetTechnicalSpecification(ctx context.Context, id uuid.UUID) (*TechnicalSpecification, error)

	// GetTechnicalSpecificationsByUserID retrieves all technical specifications for a user
	GetTechnicalSpecificationsByUserID(ctx context.Context, userID uuid.UUID) ([]*TechnicalSpecification, error)

	// UpdateTechnicalSpecification updates a technical specification
	UpdateTechnicalSpecification(ctx context.Context, id uuid.UUID, name string, updatedAt time.Time) error

	// DeleteTechnicalSpecification deletes a technical specification and all its versions
	DeleteTechnicalSpecification(ctx context.Context, id uuid.UUID) error
}

// VersionRepository defines the interface for version operations
type VersionRepository interface {
	// CreateVersion creates a new version for a technical specification
	CreateVersion(ctx context.Context, req *CreateVersionRequest) (*Version, error)

	// GetVersion retrieves a version by ID
	GetVersion(ctx context.Context, id uuid.UUID) (*Version, error)

	// GetVersionWithErrors retrieves a version with all associated errors by ID
	GetVersionWithErrors(ctx context.Context, versionID uuid.UUID) (*Version, []*InvalidError, []*MissingError, error)

	// GetVersionsByTechnicalSpecificationID retrieves all versions for a technical specification
	GetVersionsByTechnicalSpecificationID(ctx context.Context, technicalSpecificationID uuid.UUID) ([]*Version, error)

	// GetVersionsByUserID retrieves all versions with minimal data for a user
	GetVersionsByUserID(ctx context.Context, userID uuid.UUID) ([]*VersionSummary, error)

	// GetLatestVersion retrieves the latest version for a technical specification
	GetLatestVersion(ctx context.Context, technicalSpecificationID uuid.UUID) (*Version, error)

	// UpdateVersion updates a version
	UpdateVersion(ctx context.Context, id uuid.UUID, outHTML, css, checkedFileID string, updatedAt time.Time) error

	// DeleteVersion deletes a version and all its errors
	DeleteVersion(ctx context.Context, id uuid.UUID) error
}

// InvalidErrorRepository defines the interface for invalid error operations
type InvalidErrorRepository interface {
	// CreateInvalidErrors creates multiple invalid errors for a version
	CreateInvalidErrors(ctx context.Context, req *CreateInvalidErrorsRequest) error

	// GetInvalidErrorsByVersionID retrieves all invalid errors for a version
	GetInvalidErrorsByVersionID(ctx context.Context, versionID uuid.UUID) ([]*InvalidError, error)

	// DeleteInvalidErrorsByVersionID deletes all invalid errors for a version
	DeleteInvalidErrorsByVersionID(ctx context.Context, versionID uuid.UUID) error
}

// MissingErrorRepository defines the interface for missing error operations
type MissingErrorRepository interface {
	// CreateMissingErrors creates multiple missing errors for a version
	CreateMissingErrors(ctx context.Context, req *CreateMissingErrorsRequest) error

	// GetMissingErrorsByVersionID retrieves all missing errors for a version
	GetMissingErrorsByVersionID(ctx context.Context, versionID uuid.UUID) ([]*MissingError, error)

	// DeleteMissingErrorsByVersionID deletes all missing errors for a version
	DeleteMissingErrorsByVersionID(ctx context.Context, versionID uuid.UUID) error
}

// Repository combines all repository interfaces
type Repository interface {
	TechnicalSpecificationRepository
	VersionRepository
	InvalidErrorRepository
	MissingErrorRepository
}
