package jwt

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"yadro.com/course/aaa/core"
)

type Manager struct {
	log           *slog.Logger
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
}

func NewManager(log *slog.Logger, accessSecret, refreshSecret string, accessTTL time.Duration) *Manager {
	return &Manager{
		log:           log,
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
	}
}

type AccessClaims struct {
	Login   string `json:"login"`
	IsAdmin bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

func (m *Manager) GenerateAccessToken(user *core.User) (string, int64, error) {
	expiresAt := time.Now().Add(m.accessTTL)
	claims := AccessClaims{
		Login:   user.Login,
		IsAdmin: user.IsAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(user.ID, 10),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.accessSecret)
	if err != nil {
		m.log.Error("signed token in generate access error", "error", err)
		return "", 0, nil
	}
	return signedToken, expiresAt.Unix(), nil
}

func (m *Manager) GenerateRefreshToken(userID int64) (string, error) {
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.refreshSecret)
}

func (m *Manager) ParseAccessToken(tokenString string) (*core.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessClaims{}, func(token *jwt.Token) (interface{}, error) {
		return m.accessSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	claims := token.Claims.(*AccessClaims)

	// Конвертируем subject (string) в int64
	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return nil, err
	}

	exp, _ := claims.GetExpirationTime()

	return &core.TokenClaims{
		UserID:  userID,
		Login:   claims.Login,
		IsAdmin: claims.IsAdmin,
		Exp:     exp.Unix(),
	}, nil
}

func (m *Manager) ParseRefreshToken(tokenString string) (int64, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return m.refreshSecret, nil
	})

	if err != nil || !token.Valid {
		return 0, err
	}

	claims := token.Claims.(*jwt.RegisteredClaims)
	return strconv.ParseInt(claims.Subject, 10, 64)
}
