package grpcapp

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"unicode/utf8"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
	gRPCServer := grpc.NewServer()

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

	htmlText, css, docId, invalidErrors, missingErrors, fileId, err := s.tzService.CheckTz(ctx, req.File, req.Filename, requestID)
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

	// Конвертация OutInvalidError в proto сообщения
	grpcInvalidErrors := make([]*tzv1.OutInvalidError, len(*invalidErrors))
	for i, invalidError := range *invalidErrors {
		var startLine, endLine *int32
		if invalidError.StartLineNumber != nil {
			val := int32(*invalidError.StartLineNumber)
			startLine = &val
		}
		if invalidError.EndLineNumber != nil {
			val := int32(*invalidError.EndLineNumber)
			endLine = &val
		}

		grpcInvalidErrors[i] = &tzv1.OutInvalidError{
			Id:                   invalidError.Id,
			IdStr:                sanitizeString(invalidError.IdStr),
			GroupId:              sanitizeString(invalidError.GroupID),
			ErrorCode:            sanitizeString(invalidError.ErrorCode),
			Quote:                sanitizeString(invalidError.Quote),
			Analysis:             sanitizeString(invalidError.Analysis),
			Critique:             sanitizeString(invalidError.Critique),
			Verification:         sanitizeString(invalidError.Verification),
			SuggestedFix:         sanitizeString(invalidError.SuggestedFix),
			Rationale:            sanitizeString(invalidError.Rationale),
			UntilTheEndOfSentence: invalidError.UntilTheEndOfSentence,
			StartLineNumber:      startLine,
			EndLineNumber:        endLine,
		}
	}

	// Конвертация OutMissingError в proto сообщения
	grpcMissingErrors := make([]*tzv1.OutMissingError, len(*missingErrors))
	for i, missingError := range *missingErrors {
		grpcMissingErrors[i] = &tzv1.OutMissingError{
			Id:           missingError.Id,
			IdStr:        sanitizeString(missingError.IdStr),
			GroupId:      sanitizeString(missingError.GroupID),
			ErrorCode:    sanitizeString(missingError.ErrorCode),
			Analysis:     sanitizeString(missingError.Analysis),
			Critique:     sanitizeString(missingError.Critique),
			Verification: sanitizeString(missingError.Verification),
			SuggestedFix: sanitizeString(missingError.SuggestedFix),
			Rationale:    sanitizeString(missingError.Rationale),
		}
	}

	log.Info("CheckTz request processed successfully", slog.Int("invalid_errors_count", len(*invalidErrors)), slog.Int("missing_errors_count", len(*missingErrors)))

	return &tzv1.CheckTzResponse{
		HtmlText:      sanitizeString(htmlText),
		InvalidErrors: grpcInvalidErrors,
		MissingErrors: grpcMissingErrors,
		FileId:        fileId,
		Css:           css,
		DocId:         docId,
	}, nil
}

func sanitizeString(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	var out []rune
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			// некорректный байт: можете заменить на '\uFFFD' (�) или на пробел
			out = append(out, ' ')
			i++
		} else {
			out = append(out, r)
			i += size
		}
	}
	return string(out)
}
