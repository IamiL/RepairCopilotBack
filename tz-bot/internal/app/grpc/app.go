package grpcapp

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	tzservice "repairCopilotBot/tz-bot/internal/service/tz"
	tzv1 "repairCopilotBot/tz-bot/pkg/tz/v1"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       string
}

type Config struct {
	Port string `yaml:"port" env-default:":50051"`
}

type serverAPI struct {
	tzv1.UnimplementedTzServiceServer
	tzService *tzservice.Tz
	log       *slog.Logger
}

func New(log *slog.Logger, tzService *tzservice.Tz, config *Config) *App {
	// Настраиваем gRPC сервер с увеличенными таймаутами
	gRPCServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 30 * time.Minute,
			Time:              30 * time.Minute,
			Timeout:           30 * time.Minute,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             30 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.ConnectionTimeout(30*time.Minute),
	)

	tzv1.RegisterTzServiceServer(gRPCServer, &serverAPI{
		tzService: tzService,
		log:       log,
	})

	return &App{
		log:        log,
		gRPCServer: gRPCServer,
		port:       config.Port,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *App) Run() error {
	const op = "grpcapp.Run"

	log := a.log.With(slog.String("op", op), slog.String("port", a.port))

	l, err := net.Listen("tcp", a.port)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("grpc server started", slog.String("addr", l.Addr().String()))

	if err := a.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *App) Stop() {
	const op = "grpcapp.Stop"

	a.log.With(slog.String("op", op)).Info("stopping gRPC server")

	a.gRPCServer.GracefulStop()
}

func (s *serverAPI) CheckTz(ctx context.Context, req *tzv1.CheckTzRequest) (*tzv1.CheckTzResponse, error) {
	const op = "grpc.tz.CheckTz"

	log := s.log.With(
		slog.String("op", op),
		slog.String("request_id", req.RequestId),
	)

	log.Info("processing CheckTz request")

	requestID, err := uuid.Parse(req.RequestId)
	if err != nil {
		log.Error("invalid request ID format", slog.String("error", err.Error()))
		return nil, status.Error(codes.InvalidArgument, "invalid request ID format")
	}

	if len(req.File) == 0 {
		log.Error("empty file provided")
		return nil, status.Error(codes.InvalidArgument, "file cannot be empty")
	}

	if req.Filename == "" {
		log.Error("empty filename provided")
		return nil, status.Error(codes.InvalidArgument, "filename cannot be empty")
	}

	htmlText, css, docId, errors, invalidInstances, fileId, err := s.tzService.CheckTz(ctx, req.File, req.Filename, requestID)
	if err != nil {
		log.Error("failed to check tz", slog.String("error", err.Error()))

		switch err {
		case tzservice.ErrConvertWordFile:
			return nil, status.Error(codes.InvalidArgument, "failed to convert word file")
		case tzservice.ErrLlmAnalyzeFile:
			return nil, status.Error(codes.Internal, "failed to analyze file with LLM")
		default:
			return nil, status.Error(codes.Internal, "internal server error")
		}
	}

	//htmlText, css, docId, errors, invalidInstances, fileId, err := s.tzService.GetVersion(ctx, versionID)
	//if err != nil {
	//	log.Error("failed to get version", slog.String("error", err.Error()))
	//	return nil, status.Error(codes.Internal, "failed to get version")
	//}

	errorsResp := make([]*tzv1.Error, 0, len(*errors))

	for i := range *errors {
		var processRetrieval []string

		if (*errors)[i].ProcessRetrieval != nil {
			processRetrieval = *(*errors)[i].ProcessRetrieval
		}
		errorsResp = append(errorsResp, &tzv1.Error{
			Id:                  (*errors)[i].ID.String(),
			GroupId:             (*errors)[i].GroupID,
			ErrorCode:           (*errors)[i].ErrorCode,
			PreliminaryNotes:    (*errors)[i].PreliminaryNotes,
			OverallCritique:     (*errors)[i].OverallCritique,
			Verdict:             (*errors)[i].Verdict,
			ProcessAnalysis:     (*errors)[i].ProcessAnalysis,
			ProcessCritique:     (*errors)[i].ProcessCritique,
			ProcessVerification: (*errors)[i].ProcessVerification,
			ProcessRetrieval:    processRetrieval,
			InvalidInstances:    convertInvalidInstances((*errors)[i].InvalidInstances, nil),
			MissingInstances:    convertMissingInstances((*errors)[i].MissingInstances),
		})
	}

	resp := &tzv1.CheckTzResponse{
		InvalidInstances: convertInvalidInstances(invalidInstances, errorsResp),
		Errors:           errorsResp,
		HtmlText:         htmlText,
		Css:              css,
		DocId:            docId,
		FileId:           fileId,
	}

	return resp, nil
}

//func sanitizeString(s string) string {
//	if utf8.ValidString(s) {
//		return s
//	}
//	var out []rune
//	for i := 0; i < len(s); {
//		r, size := utf8.DecodeRuneInString(s[i:])
//		if r == utf8.RuneError && size == 1 {
//			// некорректный байт: можете заменить на '\uFFFD' (�) или на пробел
//			out = append(out, ' ')
//			i++
//		} else {
//			out = append(out, r)
//			i += size
//		}
//	}
//	return string(out)
//}

func (s *serverAPI) GetTechnicalSpecificationVersions(ctx context.Context, req *tzv1.GetTechnicalSpecificationVersionsRequest) (*tzv1.GetTechnicalSpecificationVersionsResponse, error) {
	const op = "grpc.tz.GetTechnicalSpecificationVersions"

	log := s.log.With(
		slog.String("op", op),
		slog.String("user_id", req.UserId),
	)

	log.Info("processing GetTechnicalSpecificationVersions request")

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		log.Error("invalid user ID format", slog.String("error", err.Error()))
		return nil, status.Error(codes.InvalidArgument, "invalid user ID format")
	}

	versions, err := s.tzService.GetTechnicalSpecificationVersions(ctx, userID)
	if err != nil {
		log.Error("failed to get technical specification versions", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to get technical specification versions")
	}

	// Конвертируем repository.VersionSummary в proto сообщения
	grpcVersions := make([]*tzv1.TechnicalSpecificationVersion, len(versions))
	for i, version := range versions {
		grpcVersions[i] = &tzv1.TechnicalSpecificationVersion{
			VersionId:                  version.ID.String(),
			TechnicalSpecificationName: version.TechnicalSpecificationName,
			VersionNumber:              int32(version.VersionNumber),
			CreatedAt:                  version.CreatedAt.Format(time.RFC3339),
		}
	}

	log.Info("GetTechnicalSpecificationVersions request processed successfully", slog.Int("versions_count", len(versions)))

	return &tzv1.GetTechnicalSpecificationVersionsResponse{
		Versions: grpcVersions,
	}, nil
}

func (s *serverAPI) GetAllVersions(ctx context.Context, _ *tzv1.GetAllVersionsRequest) (*tzv1.GetAllVersionsResponse, error) {
	const op = "grpc.tz.GetAllVersions"

	log := s.log.With(
		slog.String("op", op),
	)

	log.Info("processing GetAllVersions request")

	versions, err := s.tzService.GetAllVersions(ctx)
	if err != nil {
		log.Error("failed to get all versions", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to get all versions")
	}

	// Конвертируем repository.VersionWithErrorCounts в proto сообщения
	grpcVersions := make([]*tzv1.VersionWithErrorCounts, len(versions))
	for i, version := range versions {
		grpcVersion := &tzv1.VersionWithErrorCounts{
			VersionId:                  version.ID.String(),
			TechnicalSpecificationId:   version.TechnicalSpecificationID.String(),
			TechnicalSpecificationName: version.TechnicalSpecificationName,
			UserId:                     version.UserID.String(),
			VersionNumber:              int32(version.VersionNumber),
			CreatedAt:                  version.CreatedAt.Format(time.RFC3339),
			UpdatedAt:                  version.UpdatedAt.Format(time.RFC3339),
			OriginalFileId:             version.OriginalFileID,
			OutHtml:                    version.OutHTML,
			Css:                        version.CSS,
			CheckedFileId:              version.CheckedFileID,
			InvalidErrorCount:          int32(version.InvalidErrorCount),
			MissingErrorCount:          int32(version.MissingErrorCount),
		}

		// Устанавливаем опциональные поля
		if version.AllRubs != nil {
			grpcVersion.AllRubs = version.AllRubs
		}
		if version.AllTokens != nil {
			grpcVersion.AllTokens = version.AllTokens
		}
		if version.InspectionTime != nil {
			grpcVersion.InspectionTimeNanoseconds = (*int64)(version.InspectionTime)
		}

		grpcVersions[i] = grpcVersion
	}

	log.Info("GetAllVersions request processed successfully", slog.Int("versions_count", len(versions)))

	return &tzv1.GetAllVersionsResponse{
		Versions: grpcVersions,
	}, nil
}

func (s *serverAPI) GetVersionStatistics(ctx context.Context, _ *tzv1.GetVersionStatisticsRequest) (*tzv1.GetVersionStatisticsResponse, error) {
	const op = "grpc.tz.GetVersionStatistics"

	log := s.log.With(
		slog.String("op", op),
	)

	log.Info("processing GetVersionStatistics request")

	stats, err := s.tzService.GetVersionStatistics(ctx)
	if err != nil {
		log.Error("failed to get version statistics", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to get version statistics")
	}

	// Конвертируем repository.VersionStatistics в proto сообщение
	grpcStats := &tzv1.VersionStatistics{
		TotalVersions: stats.TotalVersions,
	}

	// Устанавливаем опциональные поля
	if stats.TotalTokens != nil {
		grpcStats.TotalTokens = stats.TotalTokens
	}
	if stats.TotalRubs != nil {
		grpcStats.TotalRubs = stats.TotalRubs
	}
	if stats.AverageInspectionTime != nil {
		grpcStats.AverageInspectionTimeNanoseconds = (*int64)(stats.AverageInspectionTime)
	}

	log.Info("GetVersionStatistics request processed successfully",
		slog.Int64("total_versions", stats.TotalVersions))

	return &tzv1.GetVersionStatisticsResponse{
		Statistics: grpcStats,
	}, nil
}

func (s *serverAPI) GetVersion(ctx context.Context, req *tzv1.GetVersionRequest) (*tzv1.GetVersionResponse, error) {
	const op = "grpc.tz.GetVersion"

	log := s.log.With(
		slog.String("op", op),
		slog.String("version_id", req.VersionId),
	)

	log.Info("processing GetVersion request")

	versionID, err := uuid.Parse(req.VersionId)
	if err != nil {
		log.Error("invalid version ID format", slog.String("error", err.Error()))
		return nil, status.Error(codes.InvalidArgument, "invalid version ID format")
	}

	htmlText, css, docId, errors, invalidInstances, fileId, err := s.tzService.GetVersion(ctx, versionID)
	if err != nil {
		log.Error("failed to get version", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to get version")
	}

	errorsResp := make([]*tzv1.Error, 0, len(*errors))

	for i := range *errors {
		var processRetrieval []string

		if (*errors)[i].ProcessRetrieval != nil {
			processRetrieval = *(*errors)[i].ProcessRetrieval
		}
		errorsResp = append(errorsResp, &tzv1.Error{
			Id:                  (*errors)[i].ID.String(),
			GroupId:             (*errors)[i].GroupID,
			ErrorCode:           (*errors)[i].ErrorCode,
			PreliminaryNotes:    (*errors)[i].PreliminaryNotes,
			OverallCritique:     (*errors)[i].OverallCritique,
			Verdict:             (*errors)[i].Verdict,
			ProcessAnalysis:     (*errors)[i].ProcessAnalysis,
			ProcessCritique:     (*errors)[i].ProcessCritique,
			ProcessVerification: (*errors)[i].ProcessVerification,
			ProcessRetrieval:    processRetrieval,
			InvalidInstances:    convertInvalidInstances((*errors)[i].InvalidInstances, nil),
			MissingInstances:    convertMissingInstances((*errors)[i].MissingInstances),
		})
	}

	resp := &tzv1.GetVersionResponse{
		InvalidInstances: convertInvalidInstances(invalidInstances, errorsResp),
		Errors:           errorsResp,
		HtmlText:         htmlText,
		Css:              css,
		DocId:            docId,
		FileId:           fileId,
	}

	return resp, nil
}

func convertInvalidInstances(invalidInstances *[]tzservice.OutInvalidError, errors []*tzv1.Error) []*tzv1.InvalidInstance {
	if invalidInstances == nil {
		return []*tzv1.InvalidInstance{}
	} else {
		respInvalidInstances := make([]*tzv1.InvalidInstance, 0, len(*invalidInstances))

		for i := range *invalidInstances {
			var quoteLines []string
			if (*invalidInstances)[i].QuoteLines != nil {
				quoteLines = *(*invalidInstances)[i].QuoteLines
			}

			var startLineNumber *int32
			if (*invalidInstances)[i].StartLineNumber != nil {
				startLineNumberTemp := int32(*(*invalidInstances)[i].StartLineNumber)
				startLineNumber = &startLineNumberTemp
			}

			var endLineNumber *int32
			if (*invalidInstances)[i].EndLineNumber != nil {
				endLineNumberTemp := int32(*(*invalidInstances)[i].EndLineNumber)
				endLineNumber = &endLineNumberTemp
			}

			var parentError *tzv1.Error
			if errors != nil {
				for j := range errors {
					if (*invalidInstances)[i].ErrorID.String() == (*errors[j]).Id {
						parentError = errors[j]
					}
				}
			}

			respInvalidInstances = append(respInvalidInstances,
				&tzv1.InvalidInstance{
					Id:                    (*invalidInstances)[i].ID.String(),
					HtmlId:                (*invalidInstances)[i].HtmlID,
					ErrorId:               (*invalidInstances)[i].ErrorID.String(),
					Quote:                 (*invalidInstances)[i].Quote,
					SuggestedFix:          (*invalidInstances)[i].SuggestedFix,
					OriginalQuote:         (*invalidInstances)[i].OriginalQuote,
					QuoteLines:            quoteLines,
					UntilTheEndOfSentence: (*invalidInstances)[i].UntilTheEndOfSentence,
					StartLineNumber:       startLineNumber,
					EndLineNumber:         endLineNumber,
					SystemComment:         (*invalidInstances)[i].SystemComment,
					OrderNumber:           int32((*invalidInstances)[i].OrderNumber),
					ParentError:           parentError,
				})
		}

		return respInvalidInstances
	}
}

func convertMissingInstances(missingInstances *[]tzservice.OutMissingError) []*tzv1.MissingInstance {
	if missingInstances == nil {
		return []*tzv1.MissingInstance{}
	} else {
		respMissingInstances := make([]*tzv1.MissingInstance, 0, len(*missingInstances))

		for i := range *missingInstances {

			respMissingInstances = append(respMissingInstances,
				&tzv1.MissingInstance{
					Id:           (*missingInstances)[i].ID.String(),
					HtmlId:       (*missingInstances)[i].HtmlID,
					ErrorId:      (*missingInstances)[i].ErrorID.String(),
					SuggestedFix: (*missingInstances)[i].SuggestedFix,
				})
		}

		return respMissingInstances
	}
}

func (s *serverAPI) NewFeedbackError(ctx context.Context, req *tzv1.NewFeedbackErrorRequest) (*tzv1.NewFeedbackErrorResponse, error) {
	const op = "grpc.tz.NewFeedbackError"

	log := s.log.With(
		slog.String("op", op),
		slog.String("version_id", req.VersionId),
		slog.String("error_id", req.ErrorId),
		slog.String("error_type", req.ErrorType),
		slog.String("user_id", req.UserId),
	)

	log.Info("processing NewFeedbackError request")

	versionID, err := uuid.Parse(req.VersionId)
	if err != nil {
		log.Error("invalid version ID format", slog.String("error", err.Error()))
		return nil, status.Error(codes.InvalidArgument, "invalid version ID format")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		log.Error("invalid user ID format", slog.String("error", err.Error()))
		return nil, status.Error(codes.InvalidArgument, "invalid user ID format")
	}

	if req.ErrorType != "invalid" && req.ErrorType != "missing" {
		log.Error("invalid error type", slog.String("error_type", req.ErrorType))
		return nil, status.Error(codes.InvalidArgument, "error type must be 'invalid' or 'missing'")
	}

	err = s.tzService.NewFeedbackError(ctx, versionID, req.ErrorId, req.ErrorType, req.FeedbackType, req.Comment, userID)
	if err != nil {
		log.Error("failed to create feedback error", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "failed to create feedback error")
	}

	log.Info("NewFeedbackError request processed successfully")

	return &tzv1.NewFeedbackErrorResponse{}, nil
}
