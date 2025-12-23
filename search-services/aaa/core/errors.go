package core

import "errors"

var (
	ErrUserNotFound       = errors.New("User not found")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidUserID      = errors.New("invalid user id")
)
