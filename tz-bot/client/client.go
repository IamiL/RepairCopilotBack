package client

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"time"

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
	Id                   uint32
	IdStr                string
	GroupID              string
	ErrorCode            string
	Quote                string
	Analysis             string
	Critique             string
	Verification         string
	SuggestedFix         string
	Rationale            string
	OriginalQuote        string
	QuoteLines           *[]string
	UntilTheEndOfSentence bool
	StartLineNumber      *int
	EndLineNumber        *int
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

func (c *Client) CheckTz(ctx context.Context, file []byte, filename string, requestID uuid.UUID) (*CheckTzResult, error) {
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

	// Конвертация OutInvalidError из proto в клиентские структуры
	invalidErrors := make([]OutInvalidError, len(resp.InvalidErrors))
	for i, grpcError := range resp.InvalidErrors {
		var startLine, endLine *int
		if grpcError.StartLineNumber != nil {
			val := int(*grpcError.StartLineNumber)
			startLine = &val
		}
		if grpcError.EndLineNumber != nil {
			val := int(*grpcError.EndLineNumber)
			endLine = &val
		}

		// Обработка QuoteLines (массив строк из proto)
		var quoteLines *[]string
		if len(grpcError.QuoteLines) > 0 {
			quoteLinesSlice := make([]string, len(grpcError.QuoteLines))
			copy(quoteLinesSlice, grpcError.QuoteLines)
			quoteLines = &quoteLinesSlice
		}

		invalidErrors[i] = OutInvalidError{
			Id:                   grpcError.Id,
			IdStr:                grpcError.IdStr,
			GroupID:              grpcError.GroupId,
			ErrorCode:            grpcError.ErrorCode,
			Quote:                grpcError.Quote,
			Analysis:             grpcError.Analysis,
			Critique:             grpcError.Critique,
			Verification:         grpcError.Verification,
			SuggestedFix:         grpcError.SuggestedFix,
			Rationale:            grpcError.Rationale,
			OriginalQuote:        grpcError.OriginalQuote,
			QuoteLines:           quoteLines,
			UntilTheEndOfSentence: grpcError.UntilTheEndOfSentence,
			StartLineNumber:      startLine,
			EndLineNumber:        endLine,
		}
	}

	// Конвертация OutMissingError из proto в клиентские структуры
	missingErrors := make([]OutMissingError, len(resp.MissingErrors))
	for i, grpcError := range resp.MissingErrors {
		missingErrors[i] = OutMissingError{
			Id:           grpcError.Id,
			IdStr:        grpcError.IdStr,
			GroupID:      grpcError.GroupId,
			ErrorCode:    grpcError.ErrorCode,
			Analysis:     grpcError.Analysis,
			Critique:     grpcError.Critique,
			Verification: grpcError.Verification,
			SuggestedFix: grpcError.SuggestedFix,
			Rationale:    grpcError.Rationale,
		}
	}

	fmt.Println("точка 13")

	return &CheckTzResult{
		HtmlText:      resp.HtmlText,
		InvalidErrors: invalidErrors,
		MissingErrors: missingErrors,
		FileId:        resp.FileId,
		Css:           resp.Css,
		DocId:         resp.DocId,
	}, nil
}

func (c *Client) Close() error {
	return c.cc.Close()
}
