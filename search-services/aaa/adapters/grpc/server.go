package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"yadro.com/course/aaa/core"
	aaapb "yadro.com/course/proto/aaa"
)

type Server struct {
	aaapb.UnimplementedAAAServer
	service core.AAAService
}

func NewServer(service core.AAAService) *Server {
	return &Server{
		service: service,
	}
}

func (s *Server) Login(ctx context.Context, req *aaapb.LoginRequest) (*aaapb.LoginResponse, error) {

	accessToken, refreshToken, expiresAt, user, err := s.service.Login(ctx, req.Login, req.Password)
	if err != nil {

		switch err {
		case core.ErrInvalidCredentials:
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		case core.ErrUserNotFound:
			return nil, status.Error(codes.NotFound, "user not found")
		default:
			return nil, status.Error(codes.Internal, "internal error")
		}
	}

	protoUser := &aaapb.User{
		Id:      user.ID,
		Login:   user.Login,
		IsAdmin: user.IsAdmin,
	}

	return &aaapb.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         protoUser,
	}, nil
}

func (s *Server) Register(ctx context.Context, req *aaapb.RegisterRequest) (*aaapb.RegisterResponse, error) {

	user, err := s.service.Register(ctx, req.Login, req.Password)
	if err != nil {

		switch err {
		case core.ErrUserExists:
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		default:
			return nil, status.Error(codes.Internal, "internal error")
		}
	}

	protoUser := &aaapb.User{
		Id:      user.ID,
		Login:   user.Login,
		IsAdmin: user.IsAdmin,
	}

	return &aaapb.RegisterResponse{
		User: protoUser,
	}, nil
}

func (s *Server) ValidateToken(ctx context.Context, req *aaapb.ValidateTokenRequest) (*aaapb.ValidateTokenResponse, error) {

	userID, isAdmin, err := s.service.ValidateToken(ctx, req.Token)
	if err != nil {

		switch err {
		case core.ErrInvalidToken:
			return &aaapb.ValidateTokenResponse{
				Valid: false,
			}, nil
		default:
			return nil, status.Error(codes.Internal, "internal error")
		}
	}

	return &aaapb.ValidateTokenResponse{
		Valid:   true,
		Id:      userID,
		IsAdmin: isAdmin,
	}, nil
}

func (s *Server) RefreshToken(ctx context.Context, req *aaapb.RefreshTokenRequest) (*aaapb.RefreshTokenResponse, error) {

	newAccessToken, newRefreshToken, expiresAt, err := s.service.RefreshToken(ctx, req.RefreshToken)
	if err != nil {

		switch err {
		case core.ErrInvalidToken:
			return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
		default:
			return nil, status.Error(codes.Internal, "internal error")
		}
	}

	return &aaapb.RefreshTokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

func (s *Server) TrackSearch(ctx context.Context, req *aaapb.TrackSearchRequest) (*emptypb.Empty, error) {

	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "empty search query")
	}

	err := s.service.TrackSearch(ctx, req.UserId, req.Query)
	if err != nil {

		switch err {
		case core.ErrInvalidUserID:
			return nil, status.Error(codes.NotFound, "user not found")
		default:
			return nil, status.Error(codes.Internal, "internal error")
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) TrackComicView(ctx context.Context, req *aaapb.TrackComicViewRequest) (*emptypb.Empty, error) {

	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if req.ComicId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid comic id")
	}

	err := s.service.TrackComicView(ctx, req.UserId, int(req.ComicId))
	if err != nil {

		switch err {
		case core.ErrInvalidUserID:
			return nil, status.Error(codes.NotFound, "user not found")
		default:
			return nil, status.Error(codes.Internal, "internal error")
		}
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) GetRecentSearches(ctx context.Context, req *aaapb.GetRecentSearchesRequest) (*aaapb.GetRecentSearchesResponse, error) {
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	queries, err := s.service.GetRecentSearches(ctx, req.UserId, int(req.Limit))
	if err != nil {

		switch err {
		case core.ErrInvalidUserID:
			return nil, status.Error(codes.NotFound, "user not found")
		default:
			return nil, status.Error(codes.Internal, "internal error")
		}
	}

	return &aaapb.GetRecentSearchesResponse{
		Queries: queries,
	}, nil
}

func (s *Server) GetRecentViews(ctx context.Context, req *aaapb.GetRecentViewsRequest) (*aaapb.GetRecentViewsResponse, error) {
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	comicIDs, err := s.service.GetRecentViews(ctx, req.UserId, int(req.Limit))
	if err != nil {
		switch err {
		case core.ErrInvalidUserID:
			return nil, status.Error(codes.NotFound, "user not found")
		default:
			return nil, status.Error(codes.Internal, "internal error")
		}
	}

	protoComicIDs := make([]int64, len(comicIDs))
	for i, id := range comicIDs {
		protoComicIDs[i] = int64(id)
	}

	return &aaapb.GetRecentViewsResponse{
		ComicIds: protoComicIDs,
	}, nil
}

// ========== PING (Health Check) ==========

func (s *Server) Ping(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
