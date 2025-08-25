package client

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	tzv1 "repairCopilotBot/tz-bot/pkg/tz/v1"
)

type Client struct {
	api tzv1.TzServiceClient
	cc  *grpc.ClientConn
}

type Config struct {
	Addr string `yaml:"addr" env-default:"localhost:9090"`
}

type OutInvalidError struct {
	Id                    uint32
	IdStr                 string
	GroupID               string
	ErrorCode             string
	Quote                 string
	Analysis              string
	Critique              string
	Verification          string
	SuggestedFix          string
	Rationale             string
	OriginalQuote         string
	QuoteLines            *[]string
	UntilTheEndOfSentence bool
	StartLineNumber       *int
	EndLineNumber         *int
}

type OutMissingError struct {
	Id           uint32
	IdStr        string
	GroupID      string
	ErrorCode    string
	Analysis     string
	Critique     string
	Verification string
	SuggestedFix string
	Rationale    string
}

type CheckTzResult struct {
	HtmlText      string
	InvalidErrors []OutInvalidError
	MissingErrors []OutMissingError
	FileId        string
	Css           string
	DocId         string
}

type TechnicalSpecificationVersion struct {
	VersionId                  string
	TechnicalSpecificationName string
	VersionNumber              int32
	CreatedAt                  string
}

type VersionWithErrorCounts struct {
	VersionId                  string
	TechnicalSpecificationId   string
	TechnicalSpecificationName string
	UserId                     string
	VersionNumber              int32
	CreatedAt                  string
	UpdatedAt                  string
	OriginalFileId             string
	OutHtml                    string
	Css                        string
	CheckedFileId              string
	AllRubs                    *float64
	AllTokens                  *int64
	InspectionTimeNanoseconds  *int64
	InvalidErrorCount          int32
	MissingErrorCount          int32
}

type VersionStatistics struct {
	TotalVersions                    int64
	TotalTokens                      *int64
	TotalRubs                        *float64
	AverageInspectionTimeNanoseconds *int64
}

func New(ctx context.Context, addr string) (*Client, error) {
	const op = "tz_client.New"

	cc, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	fmt.Println("создан клиент grpc tz-bot к адресу: ", addr)

	return &Client{
		api: tzv1.NewTzServiceClient(cc),
		cc:  cc,
	}, nil
}

func (c *Client) CheckTz(ctx context.Context, file []byte, filename string, requestID uuid.UUID) (*tzv1.CheckTzResponse, error) {
	const op = "tz_client.CheckTz"

	// Создаем контекст с таймаутом 30 минут для gRPC запроса
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	fmt.Println("точка 11")
	resp, err := c.api.CheckTz(ctx, &tzv1.CheckTzRequest{
		File:      file,
		Filename:  filename,
		RequestId: requestID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	fmt.Println("точка 12")

	return resp, nil

	// Конвертация OutInvalidError из proto в клиентские структуры
	//invalidErrors := make([]OutInvalidError, len(resp.InvalidErrors))
	//for i, grpcError := range resp.InvalidErrors {
	//	var startLine, endLine *int
	//	if grpcError.StartLineNumber != nil {
	//		val := int(*grpcError.StartLineNumber)
	//		startLine = &val
	//	}
	//	if grpcError.EndLineNumber != nil {
	//		val := int(*grpcError.EndLineNumber)
	//		endLine = &val
	//	}
	//
	//	// Обработка QuoteLines (массив строк из proto)
	//	var quoteLines *[]string
	//	if len(grpcError.QuoteLines) > 0 {
	//		quoteLinesSlice := make([]string, len(grpcError.QuoteLines))
	//		copy(quoteLinesSlice, grpcError.QuoteLines)
	//		quoteLines = &quoteLinesSlice
	//	}
	//
	//	invalidErrors[i] = OutInvalidError{
	//		Id:                    grpcError.Id,
	//		IdStr:                 grpcError.IdStr,
	//		GroupID:               grpcError.GroupId,
	//		ErrorCode:             grpcError.ErrorCode,
	//		Quote:                 grpcError.Quote,
	//		Analysis:              grpcError.Analysis,
	//		Critique:              grpcError.Critique,
	//		Verification:          grpcError.Verification,
	//		SuggestedFix:          grpcError.SuggestedFix,
	//		Rationale:             grpcError.Rationale,
	//		OriginalQuote:         grpcError.OriginalQuote,
	//		QuoteLines:            quoteLines,
	//		UntilTheEndOfSentence: grpcError.UntilTheEndOfSentence,
	//		StartLineNumber:       startLine,
	//		EndLineNumber:         endLine,
	//	}
	//}
	//
	//// Конвертация OutMissingError из proto в клиентские структуры
	//missingErrors := make([]OutMissingError, len(resp.MissingErrors))
	//for i, grpcError := range resp.MissingErrors {
	//	missingErrors[i] = OutMissingError{
	//		Id:           grpcError.Id,
	//		IdStr:        grpcError.IdStr,
	//		GroupID:      grpcError.GroupId,
	//		ErrorCode:    grpcError.ErrorCode,
	//		Analysis:     grpcError.Analysis,
	//		Critique:     grpcError.Critique,
	//		Verification: grpcError.Verification,
	//		SuggestedFix: grpcError.SuggestedFix,
	//		Rationale:    grpcError.Rationale,
	//	}
	//}
	//
	//fmt.Println("точка 13")
	//
	//return &CheckTzResult{
	//	HtmlText:      resp.HtmlText,
	//	InvalidErrors: invalidErrors,
	//	MissingErrors: missingErrors,
	//	FileId:        resp.FileId,
	//	Css:           resp.Css,
	//	DocId:         resp.DocId,
	//}, nil
}

func (c *Client) GetTechnicalSpecificationVersions(ctx context.Context, userID uuid.UUID) (*tzv1.GetTechnicalSpecificationVersionsResponse, error) {
	const op = "tz_client.GetTechnicalSpecificationVersions"

	resp, err := c.api.GetTechnicalSpecificationVersions(ctx, &tzv1.GetTechnicalSpecificationVersionsRequest{
		UserId: userID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	//versions := make([]TechnicalSpecificationVersion, len(resp.Versions))
	//for i, grpcVersion := range resp.Versions {
	//	versions[i] = TechnicalSpecificationVersion{
	//		VersionId:                  grpcVersion.VersionId,
	//		TechnicalSpecificationName: grpcVersion.TechnicalSpecificationName,
	//		VersionNumber:              grpcVersion.VersionNumber,
	//		CreatedAt:                  grpcVersion.CreatedAt,
	//	}
	//}

	return resp, nil
}

func (c *Client) GetVersion(ctx context.Context, versionID uuid.UUID) (*tzv1.GetVersionResponse, error) {
	const op = "tz_client.GetVersion"

	resp, err := c.api.GetVersion(ctx, &tzv1.GetVersionRequest{
		VersionId: versionID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return resp, nil
}

func (c *Client) GetAllVersions(ctx context.Context) ([]VersionWithErrorCounts, error) {
	const op = "tz_client.GetAllVersions"

	resp, err := c.api.GetAllVersions(ctx, &tzv1.GetAllVersionsRequest{})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	versions := make([]VersionWithErrorCounts, len(resp.Versions))
	for i, version := range resp.Versions {
		versions[i] = VersionWithErrorCounts{
			VersionId:                  version.VersionId,
			TechnicalSpecificationId:   version.TechnicalSpecificationId,
			TechnicalSpecificationName: version.TechnicalSpecificationName,
			UserId:                     version.UserId,
			VersionNumber:              version.VersionNumber,
			CreatedAt:                  version.CreatedAt,
			UpdatedAt:                  version.UpdatedAt,
			OriginalFileId:             version.OriginalFileId,
			OutHtml:                    version.OutHtml,
			Css:                        version.Css,
			CheckedFileId:              version.CheckedFileId,
			AllRubs:                    version.AllRubs,
			AllTokens:                  version.AllTokens,
			InspectionTimeNanoseconds:  version.InspectionTimeNanoseconds,
			InvalidErrorCount:          version.InvalidErrorCount,
			MissingErrorCount:          version.MissingErrorCount,
		}
	}

	return versions, nil
}

func (c *Client) GetVersionStatistics(ctx context.Context) (*VersionStatistics, error) {
	const op = "tz_client.GetVersionStatistics"

	resp, err := c.api.GetVersionStatistics(ctx, &tzv1.GetVersionStatisticsRequest{})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &VersionStatistics{
		TotalVersions:                    resp.Statistics.TotalVersions,
		TotalTokens:                      resp.Statistics.TotalTokens,
		TotalRubs:                        resp.Statistics.TotalRubs,
		AverageInspectionTimeNanoseconds: resp.Statistics.AverageInspectionTimeNanoseconds,
	}, nil
}

func (c *Client) NewFeedbackError(ctx context.Context, instanceID uuid.UUID, instanceType string, feedbackMark *bool, feedbackComment *string, userID uuid.UUID) error {
	const op = "tz_client.NewFeedbackError"

	_, err := c.api.NewFeedbackError(ctx, &tzv1.NewFeedbackErrorRequest{
		InstanceId:      instanceID.String(),
		InstanceType:    instanceType,
		FeedbackMark:    feedbackMark,
		FeedbackComment: feedbackComment,
		UserId:          userID.String(),
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (c *Client) Close() error {
	return c.cc.Close()
}
