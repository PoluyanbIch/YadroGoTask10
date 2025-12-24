package core

import "context"

type Normalizer interface {
	Norm(context.Context, string) ([]string, error)
}

type Pinger interface {
	Ping(context.Context) error
}

type Updater interface {
	Update(context.Context) error
	Stats(context.Context) (UpdateStats, error)
	Status(context.Context) (UpdateStatus, error)
	Drop(context.Context) error
}

type Searcher interface {
	Search(context.Context, string, int) (SearchReply, error)
	ISearch(context.Context, string, int) (SearchReply, error)
	GetRecommendations(context.Context, []string, []int, int) (SearchReply, error)
	GetLatestComics(context.Context, int) (SearchReply, error)
	GetComic(context.Context, int) (SearchResult, error)
}

type AAA interface {
	Login(ctx context.Context, login, password string) (accessToken, refreshToken string, expiresAt int64, user User, err error)
	Register(ctx context.Context, login, password string) (user User, err error)
	ValidateToken(ctx context.Context, token string) (userID int64, isAdmin bool, err error)
	RefreshToken(ctx context.Context, token string) (newAccessToken string, newRefreshToken string, expiresAt int64, err error)

	TrackSearch(ctx context.Context, userID int64, query string) error
	TrackComicView(ctx context.Context, userID int64, comicID int) error
	GetRecentSearches(ctx context.Context, userID int64, limit int) ([]string, error)
	GetRecentViews(ctx context.Context, userID int64, limit int) ([]int, error)
}
