package update

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"yadro.com/course/api/core"
	updatepb "yadro.com/course/proto/update"
)

type Client struct {
	log    *slog.Logger
	client updatepb.UpdateClient
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		client: updatepb.NewUpdateClient(conn),
		log:    log,
	}, nil
}

func (c Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{})
	if err != nil {
		return core.ErrServiceUnavailable
	}
	return nil
}

func (c Client) Status(ctx context.Context) (core.UpdateStatus, error) {
	resp, err := c.client.Status(ctx, &emptypb.Empty{})
	if err != nil {
		c.log.Error("status call failed", "error", err)
		return core.StatusUpdateUnknown, core.ErrInternal
	}
	switch resp.Status {
	case updatepb.Status_STATUS_IDLE:
		return core.StatusUpdateIdle, nil
	case updatepb.Status_STATUS_RUNNING:
		return core.StatusUpdateRunning, nil
	default:
		return core.StatusUpdateUnknown, nil
	}
}

func (c Client) Stats(ctx context.Context) (core.UpdateStats, error) {
	resp, err := c.client.Stats(ctx, &emptypb.Empty{})
	if err != nil {
		c.log.Error("stats call failed", "error", err)
		return core.UpdateStats{}, core.ErrInternal
	}
	return core.UpdateStats{
		WordsTotal:    int(resp.WordsTotal),
		WordsUnique:   int(resp.WordsUnique),
		ComicsFetched: int(resp.ComicsFetched),
		ComicsTotal:   int(resp.ComicsTotal),
	}, nil
}

func (c Client) Update(ctx context.Context) error {
	if _, err := c.client.Update(ctx, &emptypb.Empty{}); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return core.ErrUpdateInProgress
		}
		c.log.Error("update call failed", "error", err)
		return core.ErrInternal
	}
	return nil
}

func (c Client) Drop(ctx context.Context) error {
	if _, err := c.client.Drop(ctx, &emptypb.Empty{}); err != nil {
		c.log.Error("drop call failed", "error", err)
		return core.ErrInternal
	}
	return nil
}
