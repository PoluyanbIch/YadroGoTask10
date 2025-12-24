package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	searchpb "yadro.com/course/proto/search"
	"yadro.com/course/search/core"
)

type Server struct {
	searchpb.UnimplementedSearchServer
	service core.Searcher
}

func NewServer(service core.Searcher) *Server {
	return &Server{service: service}
}

func (s *Server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if s.service == nil {
		return nil, status.Error(codes.Internal, "service not init")
	}
	return nil, nil
}

func (s *Server) Search(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchReply, error) {
	comics, err := s.service.Search(ctx, req.Phrase, int(req.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search error: %v", err)
	}
	var pbComics []*searchpb.Comics
	for _, comic := range comics {
		pbComics = append(pbComics, &searchpb.Comics{
			Id:  int64(comic.ID),
			Url: comic.URL,
		})
	}

	return &searchpb.SearchReply{
		Comics: pbComics,
		Total:  int64(len(pbComics)),
	}, nil
}

func (s *Server) ISearch(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchReply, error) {
	comics, err := s.service.ISearch(ctx, req.Phrase, int(req.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "isearch error: %v", err)
	}
	var pbComics []*searchpb.Comics
	for _, comic := range comics {
		pbComics = append(pbComics, &searchpb.Comics{
			Id:  int64(comic.ID),
			Url: comic.URL,
		})
	}

	return &searchpb.SearchReply{
		Comics: pbComics,
		Total:  int64(len(pbComics)),
	}, nil
}

func (s *Server) GetComic(ctx context.Context, req *searchpb.GetComicRequest) (*searchpb.GetComicReply, error) {
	comic, err := s.service.GetComic(ctx, int(req.Id))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get comic error: %v", err)
	}
	return &searchpb.GetComicReply{
		Comic: &searchpb.Comics{
			Id:  int64(comic.ID),
			Url: comic.URL,
		},
	}, nil
}

func (s *Server) GetRecommendations(ctx context.Context, req *searchpb.RecommendationsRequest) (*searchpb.RecommendationsReply, error) {
	comics, err := s.service.GetRecommendations(ctx, req.SearchHistory, convertInt64SliceToIntSlice(req.ViewHistory), int(req.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get recommendations error: %v", err)
	}
	var pbComics []*searchpb.Comics
	for _, comic := range comics {
		pbComics = append(pbComics, &searchpb.Comics{
			Id:  int64(comic.ID),
			Url: comic.URL,
		})
	}
	return &searchpb.RecommendationsReply{
		Comics: pbComics,
	}, nil
}

func (s *Server) GetLatestComics(ctx context.Context, req *searchpb.LatestComicsRequest) (*searchpb.LatestComicsReply, error) {
	comics, err := s.service.GetLatestComics(ctx, int(req.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get latest comics error: %v", err)
	}
	var pbComics []*searchpb.Comics
	for _, comic := range comics {
		pbComics = append(pbComics, &searchpb.Comics{
			Id:  int64(comic.ID),
			Url: comic.URL,
		})
	}
	return &searchpb.LatestComicsReply{
		Comics: pbComics,
	}, nil
}

func convertInt64SliceToIntSlice(input []int64) []int {
	var output []int
	for _, v := range input {
		output = append(output, int(v))
	}
	return output
}
