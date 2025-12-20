package words

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"yadro.com/course/api/core"
	wordspb "yadro.com/course/proto/words"
)

type Client struct {
	log    *slog.Logger
	client wordspb.WordsClient
	conn   *grpc.ClientConn
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client := wordspb.NewWordsClient(conn)
	return &Client{
		log:    log.With("component", "words_client"),
		client: client,
		conn:   conn,
	}, nil
}

func (c *Client) Norm(ctx context.Context, phrase string) ([]string, error) {
	resp, err := c.client.Norm(ctx, &wordspb.WordsRequest{Phrase: phrase})
	if err != nil {
		switch status.Code(err) {
		case codes.ResourceExhausted:
			return nil, core.ErrBadArguments
		case codes.Unavailable:
			return nil, core.ErrServiceUnavailable
		default:
			return nil, core.ErrInternal
		}
	}
	return resp.Words, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{})
	if err != nil {
		return core.ErrServiceUnavailable
	}
	return nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
