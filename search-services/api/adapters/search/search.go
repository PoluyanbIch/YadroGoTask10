package search

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"yadro.com/course/api/core"
	searchpb "yadro.com/course/proto/search"
)

type Client struct {
	log    *slog.Logger
	client searchpb.SearchClient
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		client: searchpb.NewSearchClient(conn),
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

func (c Client) Search(ctx context.Context, phrase string, limit int) (core.SearchReply, error) {
	resp, err := c.client.Search(ctx, &searchpb.SearchRequest{Phrase: phrase, Limit: int64(limit)})
	if err != nil {
		c.log.Error("search call failed", "error", err)
		return core.SearchReply{}, core.ErrInternal
	}
	var comics []core.SearchResult
	for _, c := range resp.Comics {
		comics = append(comics, core.SearchResult{Id: int(c.Id), Url: c.Url})
	}
	return core.SearchReply{Comics: comics, Total: int(resp.Total)}, nil
}

func (c Client) ISearch(ctx context.Context, phrase string, limit int) (core.SearchReply, error) {
	resp, err := c.client.ISearch(ctx, &searchpb.SearchRequest{Phrase: phrase, Limit: int64(limit)})
	if err != nil {
		c.log.Error("isearch call failed", "error", err)
		return core.SearchReply{}, core.ErrInternal
	}
	var comics []core.SearchResult
	for _, c := range resp.Comics {
		comics = append(comics, core.SearchResult{Id: int(c.Id), Url: c.Url})
	}
	return core.SearchReply{Comics: comics, Total: int(resp.Total)}, nil
}

func (c Client) GetLatestComics(ctx context.Context, limit int) (core.SearchReply, error) {
	resp, err := c.client.GetLatestComics(ctx, &searchpb.LatestComicsRequest{Limit: int64(limit)})
	if err != nil {
		c.log.Error("get latest comics call failed", "error", err)
		return core.SearchReply{}, core.ErrInternal
	}
	var comics []core.SearchResult
	for _, c := range resp.Comics {
		comics = append(comics, core.SearchResult{Id: int(c.Id), Url: c.Url})
	}
	return core.SearchReply{Comics: comics, Total: len(comics)}, nil
}

func (c Client) GetComic(ctx context.Context, comicID int) (core.SearchResult, error) {
	resp, err := c.client.GetComic(ctx, &searchpb.GetComicRequest{Id: int64(comicID)})
	if err != nil {
		c.log.Error("get comic call failed", "error", err)
		return core.SearchResult{}, core.ErrInternal
	}
	return core.SearchResult{Id: int(resp.Comic.Id), Url: resp.Comic.Url}, nil
}

func (c Client) GetRecommendations(ctx context.Context, searchHistory []string, viewedComics []int, limit int) (core.SearchReply, error) {
	var viewedComicIDs []int64
	for _, id := range viewedComics {
		viewedComicIDs = append(viewedComicIDs, int64(id))
	}
	resp, err := c.client.GetRecommendations(ctx, &searchpb.RecommendationsRequest{
		SearchHistory: searchHistory,
		ViewHistory:   viewedComicIDs,
		Limit:         int64(limit),
	})
	if err != nil {
		c.log.Error("get recommendations call failed", "error", err)
		return core.SearchReply{}, core.ErrInternal
	}
	var comics []core.SearchResult
	for _, c := range resp.Comics {
		comics = append(comics, core.SearchResult{Id: int(c.Id), Url: c.Url})
	}
	return core.SearchReply{Comics: comics, Total: len(comics)}, nil
}
