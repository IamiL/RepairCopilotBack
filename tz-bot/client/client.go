package client

import (
	"context"
	"fmt"
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

type TzError struct {
	Id    int
	Title string
	Text  string
	Type  string
}

type CheckTzResult struct {
	HtmlText      string
	Errors        []TzError
	ErrorsMissing []TzError
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

	errors := make([]TzError, len(resp.Errors), len(resp.Errors))
	for i, grpcError := range resp.Errors {
		errors[i] = TzError{
			Id:    int(grpcError.Id),
			Title: grpcError.Title,
			Text:  grpcError.Text,
			Type:  grpcError.Type,
		}
	}

	errorsMissing := make([]TzError, len(resp.ErrorsMissing), len(resp.ErrorsMissing))
	for i, grpcError := range resp.ErrorsMissing {
		errorsMissing[i] = TzError{
			Id:    int(grpcError.Id),
			Title: grpcError.Title,
			Text:  grpcError.Text,
			Type:  grpcError.Type,
		}
	}

	fmt.Println("точка 13")

	return &CheckTzResult{
		HtmlText:      resp.HtmlText,
		Errors:        errors,
		FileId:        resp.FileId,
		ErrorsMissing: errorsMissing,
		Css:           resp.Css,
		DocId:         resp.DocId,
	}, nil
}

func (c *Client) Close() error {
	return c.cc.Close()
}
