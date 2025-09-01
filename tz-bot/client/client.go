package client

import (
	"context"
	"fmt"
	"time"

	tzv1 "repairCopilotBot/tz-bot/pkg/tz/v1"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

type CheckTzResponse struct {
	*tzv1.CheckTzResponse
	СreatedAt time.Time `json:"created_at"`
}

func (c *Client) CheckTz(ctx context.Context, file []byte, filename string, requestID uuid.UUID) (*CheckTzResponse, error) {
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

	return &CheckTzResponse{
		CheckTzResponse: resp,
		СreatedAt:       resp.CreatedAt.AsTime(),
	}, nil

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

type GetVersionMeResponse struct {
	*tzv1.VersionMe
	CreatedAt time.Time `json:"created_at"`
}

func (c *Client) GetVersionsMe(ctx context.Context, userID uuid.UUID) ([]*GetVersionMeResponse, error) {
	const op = "tz_client.GetTechnicalSpecificationVersions"

	resp, err := c.api.GetVersionsMe(ctx, &tzv1.GetVersionsMeRequest{
		UserId: userID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	versions := make([]*GetVersionMeResponse, 0, len(resp.Versions))

	for _, version := range resp.Versions {
		versions = append(versions, &GetVersionMeResponse{
			VersionMe: version,
			CreatedAt: version.CreatedAt.AsTime(),
		})
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

	return versions, nil
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

type VersionAdminDashboard struct {
	tzv1.VersionAdminDashboard
	CreatedAt time.Time `json:"created_at"`
	User      struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
}

func (c *Client) GetAllVersionsAdminDashboard(ctx context.Context) ([]*VersionAdminDashboard, error) {
	const op = "tz_client.GetAllVersions"

	resp, err := c.api.GetAllVersionsAdminDashboard(ctx, &tzv1.GetAllVersionsAdminDashboardRequest{})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	versions := make([]*VersionAdminDashboard, 0, len(resp.Versions))
	for _, version := range resp.Versions {
		versions = append(versions, &VersionAdminDashboard{
			VersionAdminDashboard: *version,
			CreatedAt:             version.CreatedAt.AsTime(),
		})
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

func (c *Client) NewFeedbackError(ctx context.Context, instanceID uuid.UUID, instanceType string, feedbackMark *bool, feedbackComment *string, userID uuid.UUID, isVerification bool) error {
	const op = "tz_client.NewFeedbackError"

	_, err := c.api.NewFeedbackError(ctx, &tzv1.NewFeedbackErrorRequest{
		InstanceId:      instanceID.String(),
		InstanceType:    instanceType,
		FeedbackMark:    feedbackMark,
		FeedbackComment: feedbackComment,
		UserId:          userID.String(),
		IsVerification:  isVerification,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// DateRange представляет диапазон дат
type DateRange struct {
	MinDate string `json:"min_date"` // Формат: 2024-01-01
	MaxDate string `json:"max_date"` // Формат: 2024-01-01
}

// GetVersionsDateRange получает минимальную и максимальную даты создания версий
func (c *Client) GetVersionsDateRange(ctx context.Context) (*DateRange, error) {
	const op = "tz_client.GetVersionsDateRange"

	resp, err := c.api.GetVersionsDateRange(ctx, &tzv1.GetVersionsDateRangeRequest{})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &DateRange{
		MinDate: resp.MinDate,
		MaxDate: resp.MaxDate,
	}, nil
}

// DailyAnalyticsPoint представляет одну точку в ежедневной аналитике
type DailyAnalyticsPoint struct {
	Date        string   `json:"date"`
	Consumption *int64   `json:"consumption,omitempty"`
	ToPay       *float64 `json:"toPay,omitempty"`
	Tz          *int32   `json:"tz,omitempty"`
}

// DailyAnalyticsResponse представляет ответ с ежедневной аналитикой
type DailyAnalyticsResponse struct {
	Series []*DailyAnalyticsPoint `json:"series"`
}

// GetDailyAnalytics получает ежедневную аналитику за указанный период
func (c *Client) GetDailyAnalytics(ctx context.Context, fromDate, toDate, timezone string, metrics []string) (*DailyAnalyticsResponse, error) {
	const op = "tz_client.GetDailyAnalytics"

	req := &tzv1.GetDailyAnalyticsRequest{
		FromDate: fromDate,
		ToDate:   toDate,
		Metrics:  metrics,
	}

	if timezone != "" {
		req.Timezone = &timezone
	}

	resp, err := c.api.GetDailyAnalytics(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Преобразуем protobuf ответ в Go структуры
	points := make([]*DailyAnalyticsPoint, len(resp.Series))
	for i, pbPoint := range resp.Series {
		point := &DailyAnalyticsPoint{
			Date: pbPoint.Date,
		}

		if pbPoint.Consumption != nil {
			point.Consumption = pbPoint.Consumption
		}
		if pbPoint.ToPay != nil {
			point.ToPay = pbPoint.ToPay
		}
		if pbPoint.Tz != nil {
			point.Tz = pbPoint.Tz
		}

		points[i] = point
	}

	return &DailyAnalyticsResponse{
		Series: points,
	}, nil
}

func (c *Client) Close() error {
	return c.cc.Close()
}

type GetFeedbacksFeedbackResponse struct {
	*tzv1.FeedbackInstance
	User struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	} `json:"user"`
}

func (c *Client) GetFeedbacks(ctx context.Context, userID uuid.UUID) (*[]*GetFeedbacksFeedbackResponse, error) {
	const op = "tz_client.GetFeedbacks"
	var userIDStr *string
	if userID != uuid.Nil {
		uid := userID.String()
		userIDStr = &uid
	}
	resp, err := c.api.GetFeedbacks(ctx, &tzv1.GetFeedbacksRequest{
		UserId: userIDStr,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	feedbacksResp := make([]*GetFeedbacksFeedbackResponse, len(resp.Feedbacks))
	for i, pbFeedback := range resp.Feedbacks {
		feedbacksResp[i] = &GetFeedbacksFeedbackResponse{
			FeedbackInstance: pbFeedback,
		}
	}
	return &feedbacksResp, nil
}
