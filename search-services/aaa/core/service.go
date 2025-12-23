package core

import (
	"context"
	"errors"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	log          *slog.Logger
	db           DB
	tokenManager TokenManager
}

func NewService(log *slog.Logger, db DB, tokenManager TokenManager) *Service {
	return &Service{
		log:          log,
		db:           db,
		tokenManager: tokenManager,
	}
}

func (s *Service) Login(ctx context.Context, login, password string) (string, string, int64, *User, error) {
	user, err := s.db.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			s.log.Debug("user not found", "login", login)
			return "", "", 0, nil, ErrInvalidCredentials
		}
		s.log.Error("failed to get user", "login", login, "error", err)
		return "", "", 0, nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PassHash), []byte(password)); err != nil {
		s.log.Debug("invalid password", "login", login)
		return "", "", 0, nil, ErrInvalidCredentials
	}

	accessToken, expiresAt, err := s.tokenManager.GenerateAccessToken(user)
	if err != nil {
		s.log.Error("failed to generate access token", "user_id", user.ID, "error", err)
		return "", "", 0, nil, err
	}

	refreshToken, err := s.tokenManager.GenerateRefreshToken(user.ID)
	if err != nil {
		s.log.Error("failed to generate refresh token", "user_id", user.ID, "error", err)
		return "", "", 0, nil, err
	}

	s.log.Info("user logged in", "user_id", user.ID, "login", login)
	return accessToken, refreshToken, expiresAt, user, nil
}

func (s *Service) Register(ctx context.Context, login, password string) (*User, error) {
	existing, err := s.db.GetUserByLogin(ctx, login)
	if err == nil && existing != nil {
		s.log.Debug("user already exists", "login", login)
		return nil, ErrUserExists
	}
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		s.log.Error("failed to check user existence", "login", login, "error", err)
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("failed to hash password", "login", login, "error", err)
		return nil, err
	}

	user := &User{
		Login:    login,
		PassHash: string(hashedPassword),
		IsAdmin:  false,
	}

	if err := s.db.AddUser(ctx, *user); err != nil {
		s.log.Error("failed to add user", "login", login, "error", err)
		return nil, err
	}

	createdUser, err := s.db.GetUserByLogin(ctx, login)
	if err != nil {
		s.log.Error("failed to get created user", "login", login, "error", err)
		return nil, err
	}

	s.log.Info("user registered", "user_id", createdUser.ID, "login", login)
	return createdUser, nil
}

func (s *Service) ValidateToken(ctx context.Context, token string) (int64, bool, error) {
	claims, err := s.tokenManager.ParseAccessToken(token)
	if err != nil {
		s.log.Debug("invalid token", "error", err)
		return 0, false, ErrInvalidToken
	}

	user, err := s.db.GetUserByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			s.log.Debug("user from token not found", "user_id", claims.UserID)
			return 0, false, ErrInvalidToken
		}
		s.log.Error("failed to get user by id", "user_id", claims.UserID, "error", err)
		return 0, false, err
	}

	return claims.UserID, user.IsAdmin, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (string, string, int64, error) {
	userID, err := s.tokenManager.ParseRefreshToken(refreshToken)
	if err != nil {
		s.log.Debug("invalid refresh token", "error", err)
		return "", "", 0, ErrInvalidToken
	}

	user, err := s.db.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			s.log.Debug("user from refresh token not found", "user_id", userID)
			return "", "", 0, ErrInvalidToken
		}
		s.log.Error("failed to get user by id", "user_id", userID, "error", err)
		return "", "", 0, err
	}

	accessToken, expiresAt, err := s.tokenManager.GenerateAccessToken(user)
	if err != nil {
		s.log.Error("failed to generate access token", "user_id", userID, "error", err)
		return "", "", 0, err
	}

	newRefreshToken, err := s.tokenManager.GenerateRefreshToken(user.ID)
	if err != nil {
		s.log.Error("failed to generate refresh token", "user_id", userID, "error", err)
		return "", "", 0, err
	}

	s.log.Info("tokens refreshed", "user_id", userID)
	return accessToken, newRefreshToken, expiresAt, nil
}

func (s *Service) TrackSearch(ctx context.Context, userID int64, query string) error {
	if userID <= 0 {
		return ErrInvalidUserID
	}

	_, err := s.db.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			s.log.Debug("user not found for tracking search", "user_id", userID)
			return ErrInvalidUserID
		}
		s.log.Error("failed to get user for tracking", "user_id", userID, "error", err)
		return err
	}

	go func() {
		ctx := context.Background()
		if err := s.db.AddSearch(ctx, userID, query); err != nil {
			s.log.Error("failed to track search", "user_id", userID, "query", query, "error", err)
		} else {
			s.log.Debug("search tracked", "user_id", userID, "query", query)
		}
	}()

	return nil
}

func (s *Service) TrackComicView(ctx context.Context, userID int64, comicID int) error {
	if userID <= 0 {
		return ErrInvalidUserID
	}

	_, err := s.db.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			s.log.Debug("user not found for tracking view", "user_id", userID)
			return ErrInvalidUserID
		}
		s.log.Error("failed to get user for tracking", "user_id", userID, "error", err)
		return err
	}

	go func() {
		ctx := context.Background()
		if err := s.db.AddComicView(ctx, userID, comicID); err != nil {
			s.log.Error("failed to track comic view", "user_id", userID, "comic_id", comicID, "error", err)
		} else {
			s.log.Debug("comic view tracked", "user_id", userID, "comic_id", comicID)
		}
	}()

	return nil
}

func (s *Service) GetRecentSearches(ctx context.Context, userID int64, limit int) ([]string, error) {
	if userID <= 0 {
		return nil, ErrInvalidUserID
	}

	queries, err := s.db.GetRecentSearches(ctx, userID, limit)
	if err != nil {
		s.log.Error("failed to get recent searches", "user_id", userID, "limit", limit, "error", err)
		return nil, err
	}

	s.log.Debug("retrieved recent searches", "user_id", userID, "count", len(queries))
	return queries, nil
}

func (s *Service) GetRecentViews(ctx context.Context, userID int64, limit int) ([]int, error) {
	if userID <= 0 {
		return nil, ErrInvalidUserID
	}

	comicIDs, err := s.db.GetRecentViews(ctx, userID, limit)
	if err != nil {
		s.log.Error("failed to get recent views", "user_id", userID, "limit", limit, "error", err)
		return nil, err
	}

	s.log.Debug("retrieved recent views", "user_id", userID, "count", len(comicIDs))
	return comicIDs, nil
}
