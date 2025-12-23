package core

import (
	"context"
)

type AAAService interface {
	Login(ctx context.Context, login, password string) (accessToken, refreshToken string, expiresAt int64, user *User, err error)
	Register(ctx context.Context, login, password string) (user *User, err error)
	ValidateToken(ctx context.Context, token string) (userID int64, isAdmin bool, err error)
	RefreshToken(ctx context.Context, token string) (newAccessToken string, newRefreshToken string, expiresAt int64, err error)

	TrackSearch(ctx context.Context, userID int64, query string) error
	TrackComicView(ctx context.Context, userID int64, comicID int) error
	GetRecentSearches(ctx context.Context, userID int64, limit int) ([]string, error)
	GetRecentViews(ctx context.Context, userID int64, limit int) ([]int, error)
}

type TokenManager interface {
	GenerateAccessToken(user *User) (string, int64, error) // token, expires_at
	GenerateRefreshToken(userID int64) (string, error)
	ParseAccessToken(token string) (*TokenClaims, error)
	ParseRefreshToken(token string) (userID int64, err error)
}

type DB interface {
	// users
	AddUser(context.Context, User) error
	GetUserByLogin(context.Context, string) (*User, error)
	GetUserByID(context.Context, int64) (*User, error)

	// comics_history
	AddComicView(ctx context.Context, userID int64, comicID int) error
	GetRecentViews(ctx context.Context, userID int64, limit int) ([]int, error)
	HasViewedComic(ctx context.Context, userID int64, comicID int) (bool, error)

	// search_history
	AddSearch(ctx context.Context, userID int64, query string) error
	GetRecentSearches(ctx context.Context, userID int64, limit int) ([]string, error)
}
