package tzservice

import (
	"context"
	modelrepo "repairCopilotBot/tz-bot/internal/repository/models"
	"time"

	"github.com/google/uuid"
)

// DailyAnalyticsPoint represents a single point in daily analytics
type DailyAnalyticsPoint struct {
	Date        string
	Consumption *int64
	ToPay       *float64
	Tz          *int32
}

// FeedbackInstance represents a feedback instance with full context information
type FeedbackInstance struct {
	InstanceID                 string
	InstanceType               string // "invalid" or "missing"
	FeedbackMark               bool
	FeedbackComment            string
	FeedbackUser               string
	ErrorID                    string
	VersionID                  string
	TechnicalSpecificationName string
}

// TechnicalSpecificationRepository defines the interface for technical specification operations
type TechnicalSpecificationRepository interface {
	// CreateTechnicalSpecification creates a new technical specification
	CreateTechnicalSpecification(ctx context.Context, req *modelrepo.CreateTechnicalSpecificationRequest) (*modelrepo.TechnicalSpecification, error)

	// GetTechnicalSpecification retrieves a technical specification by ID
	GetTechnicalSpecification(ctx context.Context, id uuid.UUID) (*modelrepo.TechnicalSpecification, error)

	// GetTechnicalSpecificationsByUserID retrieves all technical specifications for a user
	GetTechnicalSpecificationsByUserID(ctx context.Context, userID uuid.UUID) ([]*modelrepo.TechnicalSpecification, error)

	// UpdateTechnicalSpecification updates a technical specification
	UpdateTechnicalSpecification(ctx context.Context, id uuid.UUID, name string, updatedAt time.Time) error

	// DeleteTechnicalSpecification deletes a technical specification and all its versions
	DeleteTechnicalSpecification(ctx context.Context, id uuid.UUID) error
}

// VersionRepository defines the interface for version operations
type VersionRepository interface {
	// CreateVersion creates a new version for a technical specification
	CreateVersion(ctx context.Context, req *modelrepo.CreateVersionRequest) error

	// GetVersion retrieves a version by ID
	GetVersion(ctx context.Context, id uuid.UUID) (*modelrepo.Version, error)

	// GetVersionWithErrors retrieves a version with all associated errors by ID
	GetVersionWithErrors(ctx context.Context, versionID uuid.UUID) (*modelrepo.Version, []*modelrepo.InvalidError, []*modelrepo.MissingError, error)

	// GetVersionsByTechnicalSpecificationID retrieves all versions for a technical specification
	GetVersionsByTechnicalSpecificationID(ctx context.Context, technicalSpecificationID uuid.UUID) ([]*modelrepo.Version, error)

	// GetVersionsByUserID retrieves all versions with minimal data for a user
	GetVersionsMeByUserID(ctx context.Context, userID uuid.UUID) ([]*VersionMe, error)

	// GetAllVersions retrieves all versions with complete data and error counts
	GetAllVersionsAdminDashboard(ctx context.Context) ([]*VersionAdminDashboard, error)

	// GetVersionStatistics retrieves aggregated statistics for all versions
	GetVersionStatistics(ctx context.Context) (*modelrepo.VersionStatistics, error)

	// GetVersionsDateRange retrieves min and max dates from versions table
	GetVersionsDateRange(ctx context.Context) (string, string, error)

	// GetDailyAnalytics retrieves daily analytics for versions
	GetDailyAnalytics(ctx context.Context, fromDate, toDate, timezone string, metrics []string) ([]*DailyAnalyticsPoint, error)

	// GetFeedbacks retrieves all feedbacks from invalid_instances and missing_instances
	GetFeedbacks(ctx context.Context, userID *string) ([]*FeedbackInstance, error)

	// GetLatestVersion retrieves the latest version for a technical specification
	GetLatestVersion(ctx context.Context, technicalSpecificationID uuid.UUID) (*modelrepo.Version, error)

	// UpdateVersion updates a version
	UpdateVersion(ctx context.Context, req *modelrepo.UpdateVersionRequest) error

	// DeleteVersion deletes a version and all its errors
	DeleteVersion(ctx context.Context, id uuid.UUID) error
}

// InvalidErrorRepository defines the interface for invalid error operations
//type InvalidErrorRepository interface {
//	// CreateInvalidErrors creates multiple invalid errors for a version
//	CreateInvalidErrors(ctx context.Context, req *modelrepo.CreateInvalidErrorsRequest) error
//
//	// GetInvalidErrorsByVersionID retrieves all invalid errors for a version
//	GetInvalidErrorsByVersionID(ctx context.Context, versionID uuid.UUID) ([]*modelrepo.InvalidError, error)
//
//	// DeleteInvalidErrorsByVersionID deletes all invalid errors for a version
//	DeleteInvalidErrorsByVersionID(ctx context.Context, versionID uuid.UUID) error
//
//	// GetUUIDByErrorID retrieves UUID by numeric error ID from both invalid and missing errors
//	GetUUIDByErrorID(ctx context.Context, errorID int) (uuid.UUID, error)
//}

// MissingErrorRepository defines the interface for missing error operations
//type MissingErrorRepository interface {
//	// CreateMissingErrors creates multiple missing errors for a version
//	CreateMissingErrors(ctx context.Context, req *modelrepo.CreateMissingErrorsRequest) error
//
//	// GetMissingErrorsByVersionID retrieves all missing errors for a version
//	GetMissingErrorsByVersionID(ctx context.Context, versionID uuid.UUID) ([]*modelrepo.MissingError, error)
//
//	// DeleteMissingErrorsByVersionID deletes all missing errors for a version
//	DeleteMissingErrorsByVersionID(ctx context.Context, versionID uuid.UUID) error
//}

// ErrorRepository defines the interface for error operations
type ErrorRepository interface {
	// CreateErrors creates multiple errors for a version
	CreateErrors(ctx context.Context, req *modelrepo.CreateErrorsRequest) error
	GetErrorsByVersionID(ctx context.Context, versionID uuid.UUID) (*[]Error, error)
}

// InvalidInstanceRepository defines the interface for invalid instance operations
type InvalidInstanceRepository interface {
	// SaveInvalidInstances saves multiple invalid instances
	SaveInvalidInstances(ctx context.Context, invalidInstances *[]OutInvalidError) error
	GetInvalidInstancesByErrorID(ctx context.Context, errorID uuid.UUID) (*[]OutInvalidError, error)
	// UpdateInvalidInstanceFeedback updates feedback for invalid instance
	UpdateInvalidInstanceFeedback(ctx context.Context, instanceID uuid.UUID, feedbackMark *bool, feedbackComment *string, userID uuid.UUID) error
}

type MissingInstanceRepository interface {
	// SaveInvalidInstances saves multiple invalid instances
	SaveMissingInstances(ctx context.Context, invalidInstances *[]OutMissingError) error
	GetMissingInstancesByErrorID(ctx context.Context, errorID uuid.UUID) (*[]OutMissingError, error)
	// UpdateMissingInstanceFeedback updates feedback for missing instance
	UpdateMissingInstanceFeedback(ctx context.Context, instanceID uuid.UUID, feedbackMark *bool, feedbackComment *string, userID uuid.UUID) error
}

// ErrorFeedbackRepository defines the interface for error feedback operations
type ErrorFeedbackRepository interface {
	// CreateErrorFeedback creates new feedback for an error
	CreateErrorFeedback(ctx context.Context, req *modelrepo.CreateErrorFeedbackRequest) (*modelrepo.ErrorFeedback, error)
}

// Repository combines all repository interfaces
type Repository interface {
	TechnicalSpecificationRepository
	VersionRepository
	//InvalidErrorRepository
	//MissingErrorRepository
	ErrorRepository
	InvalidInstanceRepository
	MissingInstanceRepository
	ErrorFeedbackRepository
	//LLMCacheRepository

	// GetUUIDByErrorID retrieves UUID by numeric error ID from both invalid and missing errors
	GetUUIDByErrorID(ctx context.Context, errorID int) (uuid.UUID, error)
}
