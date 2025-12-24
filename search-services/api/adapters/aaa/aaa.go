package aaa

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"yadro.com/course/api/core"
	aaapb "yadro.com/course/proto/aaa"
)

type Client struct {
	log    *slog.Logger
	client aaapb.AAAClient
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		client: aaapb.NewAAAClient(conn),
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

func (c Client) Login(ctx context.Context, login, password string) (string, string, int64, core.User, error) {
	resp, err := c.client.Login(ctx, &aaapb.LoginRequest{Login: login, Password: password})
	if err != nil {
		c.log.Error("login call failed", "error", err)
		return "", "", 0, core.User{}, core.ErrInternal
	}
	user := core.User{
		ID:      resp.User.Id,
		Login:   resp.User.Login,
		IsAdmin: resp.User.IsAdmin,
	}
	return resp.AccessToken, resp.RefreshToken, resp.ExpiresAt, user, nil
}

func (c Client) Register(ctx context.Context, login, password string) (core.User, error) {
	resp, err := c.client.Register(ctx, &aaapb.RegisterRequest{Login: login, Password: password})
	if err != nil {
		c.log.Error("register call failed", "error", err)
		return core.User{}, core.ErrInternal
	}
	user := core.User{
		ID:      resp.User.Id,
		Login:   resp.User.Login,
		IsAdmin: resp.User.IsAdmin,
	}
	return user, nil
}

func (c Client) ValidateToken(ctx context.Context, token string) (int64, bool, error) {
	resp, err := c.client.ValidateToken(ctx, &aaapb.ValidateTokenRequest{Token: token})
	if err != nil {
		c.log.Error("validate token call failed", "error", err)
		return 0, false, core.ErrInternal
	}
	if !resp.Valid {
		return 0, false, core.ErrInvalidToken
	}
	return resp.Id, resp.IsAdmin, nil
}

func (c Client) RefreshToken(ctx context.Context, token string) (string, string, int64, error) {
	resp, err := c.client.RefreshToken(ctx, &aaapb.RefreshTokenRequest{RefreshToken: token})
	if err != nil {
		c.log.Error("refresh token call failed", "error", err)
		return "", "", 0, core.ErrInternal
	}
	return resp.AccessToken, resp.RefreshToken, resp.ExpiresAt, nil
}

func (c Client) TrackSearch(ctx context.Context, userID int64, query string) error {
	_, err := c.client.TrackSearch(ctx, &aaapb.TrackSearchRequest{UserId: userID, Query: query})
	if err != nil {
		c.log.Error("track search call failed", "error", err)
		return core.ErrInternal
	}
	return nil
}

func (c Client) TrackComicView(ctx context.Context, userID int64, comicID int) error {
	_, err := c.client.TrackComicView(ctx, &aaapb.TrackComicViewRequest{UserId: userID, ComicId: int64(comicID)})
	if err != nil {
		c.log.Error("track comic view call failed", "error", err)
		return core.ErrInternal
	}
	return nil
}

func (c Client) GetRecentSearches(ctx context.Context, userID int64, limit int) ([]string, error) {
	resp, err := c.client.GetRecentSearches(ctx, &aaapb.GetRecentSearchesRequest{UserId: userID, Limit: int64(limit)})
	if err != nil {
		c.log.Error("get recent searches call failed", "error", err)
		return nil, core.ErrInternal
	}
	return resp.Queries, nil
}

func (c Client) GetRecentViews(ctx context.Context, userID int64, limit int) ([]int, error) {
	resp, err := c.client.GetRecentViews(ctx, &aaapb.GetRecentViewsRequest{UserId: userID, Limit: int64(limit)})
	if err != nil {
		c.log.Error("get recent views call failed", "error", err)
		return nil, core.ErrInternal
	}
	var comicIDs []int
	for _, id := range resp.ComicIds {
		comicIDs = append(comicIDs, int(id))
	}
	return comicIDs, nil
}
