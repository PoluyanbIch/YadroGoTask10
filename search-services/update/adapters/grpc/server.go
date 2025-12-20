package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	updatepb "yadro.com/course/proto/update"
	"yadro.com/course/update/core"
)

func NewServer(service core.Updater) *Server {
	return &Server{service: service}
}

type Server struct {
	updatepb.UnimplementedUpdateServer
	service core.Updater
}

func (s *Server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if s.service == nil {
		return nil, status.Error(codes.Internal, "service not init")
	}
	return nil, nil
}

func (s *Server) Status(ctx context.Context, _ *emptypb.Empty) (*updatepb.StatusReply, error) {
	if s.service.Status(ctx) == core.StatusIdle {
		return &updatepb.StatusReply{Status: updatepb.Status_STATUS_IDLE}, nil
	}
	return &updatepb.StatusReply{Status: updatepb.Status_STATUS_RUNNING}, nil
}

func (s *Server) Update(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Update(ctx); err != nil {
		if errors.Is(err, core.ErrUpdateInProgress) {
			return nil, status.Error(codes.AlreadyExists, "update already in progress")
		}
		return nil, status.Errorf(codes.Internal, "update error: %v", err)
	}
	return nil, nil
}

func (s *Server) Stats(ctx context.Context, _ *emptypb.Empty) (*updatepb.StatsReply, error) {
	stats, err := s.service.Stats(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "stats error: %v", err)
	}
	return &updatepb.StatsReply{
		WordsTotal:    int64(stats.WordsTotal),
		WordsUnique:   int64(stats.WordsUnique),
		ComicsFetched: int64(stats.ComicsFetched),
		ComicsTotal:   int64(stats.ComicsTotal),
	}, nil
}

func (s *Server) Drop(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Drop(ctx); err != nil {
		return nil, status.Errorf(codes.Internal, "drop error: %v", err)
	}
	return nil, nil
}
