package aaa

import (
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"yadro.com/course/api/core"
)

const secretKey = "something secret here" // token sign key
const adminRole = "superuser"             // token subject

// Authentication, Authorization, Accounting
type AAA struct {
	users    map[string]string
	tokenTTL time.Duration
	log      *slog.Logger
}

func New(tokenTTL time.Duration, log *slog.Logger, adminUser string, adminPass string) AAA {
	return AAA{
		users:    map[string]string{adminUser: adminPass},
		tokenTTL: tokenTTL,
		log:      log,
	}
}

func (a AAA) Login(name, password string) (string, error) {
	if p, ok := a.users[name]; !ok || p != password {
		a.log.Warn("Login failed ", "user", name)
		return "", core.ErrInvalidCredentials
	}
	claims := &jwt.RegisteredClaims{
		Subject:   adminRole,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.tokenTTL)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		a.log.Error("Failed signing token", "error", err)
		return "", err
	}
	return tokenString, nil
}

func (a AAA) Verify(tokenString string) error {
	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})
	if err != nil || !token.Valid {
		a.log.Error("invalid token", "error", err)
		return core.ErrInvalidToken
	}
	if claims.Subject != adminRole {
		a.log.Warn("Token subject check failed", "expected", adminRole, "got", claims.Subject)
		return core.ErrUnauthorised
	}
	return nil
}
